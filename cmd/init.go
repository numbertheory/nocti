package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type Config struct {
	Name    string        `json:"name"`
	Version string        `json:"version"`
	Editor  string        `json:"editor"`
	Colors  *ColorsConfig `json:"colors,omitempty"`
}

var ProjectName string

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new nocti project",
	Long:  `Creates a .nocti/nocti.json file in the current working directory with default configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Block init if inside a nocti resource
		if _, err := os.Stat(".nocti.json"); err == nil {
			return fmt.Errorf("cannot run 'nocti init' inside a nocti resource directory")
		}

		dir := ".nocti"
		filename := dir + "/nocti.json"
		defaultProjectName := "my-nocti-project"

		// If flag is not set, prompt for project name
		if ProjectName == "" {
			fmt.Printf("Enter project name (%s): ", defaultProjectName)
			_, err := fmt.Scanln(&ProjectName)
			if err != nil && err.Error() != "unexpected newline" {
				return fmt.Errorf("failed to read project name: %w", err)
			}
			if ProjectName == "" {
				ProjectName = defaultProjectName
			}
		}

		// Create hidden directory if it doesn't exist
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Check if the file already exists
		if _, err := os.Stat(filename); err == nil {
			return fmt.Errorf("%s already exists", filename)
		}

		config := Config{
			Name:    ProjectName,
			Version: Version,
			Editor:  "nano",
			Colors: &ColorsConfig{
				FileList:     "blue",
				PreviewPane:  "orange",
				HelpBg:       "darkgray",
				HelpFg:       "white",
				HelpBorderBg: "black",
				HelpBorderFg: "gray",
			},
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		err = os.WriteFile(filename, data, 0644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("Initialized successfully: %s created\n", filename)
		return nil
	},
}

func init() {
	InitCmd.Flags().StringVarP(&ProjectName, "project", "p", "", "Name of the nocti project")
	RootCmd.AddCommand(InitCmd)
}
