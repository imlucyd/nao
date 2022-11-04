package cmd

import (
	"fmt"

	"github.com/luisnquin/nao/v2/internal/config"
	"github.com/luisnquin/nao/v2/internal/data"
	"github.com/luisnquin/nao/v2/internal/store"
	"github.com/luisnquin/nao/v2/internal/store/tagutils"
	"github.com/spf13/cobra"
)

type TagCmd struct {
	*cobra.Command
	config *config.AppConfig
	data   *data.Buffer
}

func BuildTag(config *config.AppConfig, data *data.Buffer) TagCmd {
	c := TagCmd{
		Command: &cobra.Command{
			Use:           "tag <old> <new>",
			Short:         "Rename the tag of any file",
			Args:          cobra.ExactArgs(2),
			SilenceUsage:  true,
			SilenceErrors: true,
			ValidArgsFunction: func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return SearchKeyTagsByPattern(toComplete, data), cobra.ShellCompDirectiveNoFileComp
			},
		},
		config: config,
		data:   data,
	}

	c.RunE = c.Main()

	return c
}

func (c *TagCmd) Main() scriptor {
	return func(cmd *cobra.Command, args []string) error {
		notesRepo := store.NewNotesRepository(c.data)
		tagutil := tagutils.New(c.data)

		err := tagutil.IsValidAsNew(args[1])
		if err != nil {
			return fmt.Errorf("tag %s is not valid: %w", args[1], err)
		}

		return notesRepo.ModifyTag(SearchKeyByPattern(args[0], c.data), args[1])
	}
}
