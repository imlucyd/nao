package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/luisnquin/nao/src/constants"
	"github.com/luisnquin/nao/src/data"
	"github.com/luisnquin/nao/src/helper"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{ // editor as a flag
	Use:   "new",
	Short: "Creates a new nao file",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		box := data.New()

		from, editor, tag := parseNewCmdFlags(cmd)

		f, remove, err := helper.NewCached()
		cobra.CheckErr(err)

		defer remove()

		if from != "" {
			_, set, err := box.SearchSetByKeyTagPattern(from)
			cobra.CheckErr(err)

			err = ioutil.WriteFile(f.Name(), []byte(set.Content), 0644)
			cobra.CheckErr(err)
		}

		run, err := helper.PrepareToRun(cmd.Context(), helper.EditorOptions{
			Path:   f.Name(),
			Editor: editor,
		})

		cobra.CheckErr(err)

		err = run()
		cobra.CheckErr(err)

		content, err := ioutil.ReadAll(f)
		cobra.CheckErr(err)

		if len(content) == 0 {
			fmt.Fprintln(os.Stderr, "Empty content, will not be saved!")
			os.Exit(1)
		}

		var k string

		if tag != "" {
			k, err = box.NewSetWithTag(string(content), constants.TypeDefault, tag)
		} else {
			k, err = box.NewSet(string(content), constants.TypeDefault)
		}

		cobra.CheckErr(err)

		fmt.Fprintln(os.Stdout, k[:10])
	},
}

func init() {
	newCmd.Flags().String("from", "", constants.AppName+" new --from=<hash>")
	newCmd.Flags().String("editor", "", constants.AppName+"new --editor=<?>")
	newCmd.Flags().String("tag", "", constants.AppName+"new --tag=<name>")
}

func parseNewCmdFlags(cmd *cobra.Command) (string, string, string) {
	editor, _ := cmd.Flags().GetString("editor")
	from, _ := cmd.Flags().GetString("from")
	tag, _ := cmd.Flags().GetString("tag")

	return from, editor, tag
}
