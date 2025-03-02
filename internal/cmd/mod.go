package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/luisnquin/nao/v3/internal"
	"github.com/luisnquin/nao/v3/internal/config"
	"github.com/luisnquin/nao/v3/internal/data"
	"github.com/luisnquin/nao/v3/internal/models"
	"github.com/luisnquin/nao/v3/internal/note"
	"github.com/luisnquin/nao/v3/internal/ui"
	"github.com/luisnquin/nao/v3/internal/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type ModCmd struct {
	*cobra.Command

	log    *zerolog.Logger
	config *config.Core
	data   *data.Buffer
	latest bool
	editor string
}

func BuildMod(log *zerolog.Logger, config *config.Core, data *data.Buffer) ModCmd {
	c := ModCmd{
		Command: &cobra.Command{
			Use:               "mod [<id> | <tag>]",
			Short:             "Edit any file",
			Args:              cobra.MaximumNArgs(1),
			ValidArgsFunction: KeyTagCompletions(data),
			SilenceUsage:      true,
			SilenceErrors:     true,
		},
		config: config,
		data:   data,
		log:    log,
	}

	c.RunE = c.Main()
	log.Trace().Msg("the 'mod' command has been created")

	flags := c.Flags()
	if !c.latest {
		flags.BoolVarP(&c.latest, "latest", "l", false, "access the last modified file")
	}

	flags.StringVar(&c.editor, "editor", "", "change the default code editor (ignoring configuration file)")

	return c
}

func (c *ModCmd) Main() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		notesRepo := note.NewRepository(c.data)

		var nt models.Note

		switch {
		case c.latest:
			var err error

			c.log.Trace().Msg("the last note accessed has been requested")

			nt, err = notesRepo.LastAccessed()
			if err != nil {
				c.log.Err(err).Msg("error encountered when trying to access the last accessed note")

				return err
			}

		case len(args) == 1:
			c.log.Trace().Str("key/tag provided", args[0]).Send()

			key, err := note.SearchByPrefix(args[0], c.data)
			if err != nil {
				c.log.Err(err).Str("arg", args[0]).Msg("error with the argument supplied")

				return err
			}

			c.log.Trace().Str("key found", key).Send()

			nt, err = notesRepo.Get(key)
			if err != nil {
				c.log.Err(err).Msg("unexpected error trying to get a previously found note")

				return err
			}

		default:
			c.log.Trace().Msg("no argument supplied, returning usage")

			return cmd.Usage()
		}

		editorName := c.getEditorName()

		var editorArgs []string

		unlog, err := c.logKeyInUse(nt.Key)
		if err != nil {
			if !c.config.ReadOnlyOnConflict {
				return err
			}

			editorArgs = append(editorArgs, getReadOnlyFlag(editorName))
		} else {
			defer func() {
				if err := unlog(); err != nil {
					panic(err)
				}
			}()
		}

		c.log.Trace().Msg("creating temporary file")

		filePath, err := NewFileCached(c.config, nt.Key, nt.Content)
		if err != nil {
			return err
		}

		c.log.Trace().Str("temporary file path", filePath).Send()

		defer func() {
			c.log.Trace().Msg("deleting temporary file")

			if err := os.Remove(filePath); err != nil {
				c.log.Trace().Msg("unexpected error trying to delete temporary file")

				ui.Error(err.Error())
			}
		}()

		start := time.Now()

		c.log.Trace().Str("editor", editorName).Strs("flags", editorArgs).Msg("running editor...")

		err = RunEditor(cmd.Context(), editorName, filePath, editorArgs...)
		if err != nil {
			c.log.Err(err).Msg("error running the editor")

			return err
		}

		c.log.Trace().Msg("reading content of temporary file...")

		content, err := os.ReadFile(filePath)
		if err != nil {
			c.log.Err(err).Msg("error reading content of temporary file")

			return err
		}

		modifiers := []note.ModifyOption{note.WithSpentTime(time.Since(start))}

		if string(content) != nt.Content {
			modifiers = append(modifiers, note.WithContent(string(content)))
		} else {
			c.log.Trace().Msg("no new content was written to the temporary file, note will not be updated")
		}

		return notesRepo.Update(nt.Key, modifiers...)
	}
}

func (c ModCmd) openKeysInUseFile() (*os.File, error) {
	return os.OpenFile(
		path.Join(os.TempDir(), ".nao.keys"),
		os.O_CREATE|os.O_APPEND|os.O_RDWR, internal.PermReadWrite,
	)
}

func (c ModCmd) logKeyInUse(key string) (remove func() error, err error) {
	f, err := c.openKeysInUseFile()
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	separator := "\n"

	if utils.Contains(strings.Split(string(content), separator), key) {
		return nil, fmt.Errorf("key '%s' already in use", key)
	}

	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	f.WriteString(key + "\n")

	return func() error {
		f, err = c.openKeysInUseFile()
		if err != nil {
			return err
		}

		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		keys := strings.Split(string(content), separator)

		updatedKeys := make([]string, 0, len(keys)-1)

		for _, k := range keys {
			if k != key {
				updatedKeys = append(updatedKeys, k)
			}
		}

		if err := f.Truncate(0); err != nil {
			return err
		}

		f.WriteString(strings.Join(updatedKeys, separator))

		return f.Close()
	}, f.Close()
}

func (c *ModCmd) getEditorName() string {
	if c.editor != "" {
		return c.editor
	}

	if c.config.Editor.Name != "" {
		return c.config.Editor.Name
	}

	return internal.Nano
}
