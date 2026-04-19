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
		filename := "nocti.json"

		// Check if the file already exists
		if _, err := os.Stat(filename); err == nil {
			return fmt.Errorf("%s already exists", filename)
		}

		config := Config{
			Name:    "my-nocti-project",
			Version: "1.0.0",
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
