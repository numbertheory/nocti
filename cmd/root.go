package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "development"

var rootCmd = &cobra.Command{
	Use:   "nocti",
	Short: "Nocti Note Taking CLI",
	Long:  `Nocti is a CLI tool for note-taking and knowledge management.`,
	Run: func(cmd *cobra.Command, args []string) {
		version, _ := cmd.Flags().GetBool("version")
		if version {
			fmt.Printf("Nocti version %s\n", Version)
			return
		}
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print the version number of nocti")
}
