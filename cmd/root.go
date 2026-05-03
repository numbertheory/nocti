package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "development"

var RootCmd = &cobra.Command{
	Use:   "nocti",
	Short: "Nocti Note Taking CLI",
	Long:  `Nocti is a CLI tool for note-taking and knowledge management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		version, _ := cmd.Flags().GetBool("version")
		if version {
			fmt.Printf("Nocti version %s\n", Version)
			return nil
		}

		// Default to 'list' command if no args provided
		if len(args) == 0 {
			// Check if we are in the project root
			root, err := FindProjectRoot()
			wd, _ := os.Getwd()
			if err == nil && wd == root {
				return ListCmd.RunE(ListCmd, args)
			}

			// Check if we are in a notebook context
			_, resType, err := FindEnclosingResource()
			if err == nil && resType == "notebook" {
				return ListCmd.RunE(ListCmd, args)
			}
		}

		return cmd.Help()
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().BoolP("version", "v", false, "Print the version number of nocti")
}
