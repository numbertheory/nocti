package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type Config struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new nocti project",
	Long:  `Creates a nocti.json file in the current working directory with default configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := ".nocti"
		filename := dir + "/nocti.json"

		// Prompt for project name
		var projectName string
		fmt.Print("Enter project name: ")
		_, err := fmt.Scanln(&projectName)
		if err != nil && err.Error() != "unexpected newline" {
			return fmt.Errorf("failed to read project name: %w", err)
		}
		if projectName == "" {
			projectName = "my-nocti-project"
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
			Name:    projectName,
			Version: Version,
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
	rootCmd.AddCommand(initCmd)
}
