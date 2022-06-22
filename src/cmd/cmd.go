package cmd

import (
	"fmt"
	"os"

	"github.com/luisnquin/nao/src/constants"
	"github.com/spf13/cobra"
)

// add support for fswatch?

var root = &cobra.Command{
	Use:   constants.AppName,
	Short: constants.AppName + " is a tool to manage your notes",
	Long: `A tool to manage your notes or other types of files without
		worry about the path where it is, agile and safe if you want`,
	Run: func(cmd *cobra.Command, args []string) {
		switch len(args) {
		case 0:
			mainCmd.Run(cmd, args)

		case 1:
			editCmd.Run(cmd, args)

		default:
			cmd.Usage()
		}
	},
	TraverseChildren: true,
}

func Execute() {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	root.AddCommand(newCmd, renderCmd, mergeCmd, lsCmd, mainCmd, editCmd, delCmd, cleanCmd)
}
