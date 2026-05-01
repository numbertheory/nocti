package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the main configuration file with existing resources",
	Long:  `Scans immediate child directories for .nocti.json files and updates .nocti/nocti.json.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Error if inside a resource
		if _, err := os.Stat(".nocti.json"); err == nil {
			return fmt.Errorf("this command must be run in the nocti project root, not inside a resource")
		}

		// 2. Ensure we are in the root (check for .nocti/nocti.json)
		filename := filepath.Join(".nocti", "nocti.json")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return fmt.Errorf("nocti project not found in this directory (missing .nocti/nocti.json)")
		}

		// 3. Read existing config
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		var config FullConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		// 4. Scan immediate children
		entries, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		// Clear existing lists to rebuild them
		config.Notebooks = []Notebook{}
		config.Todos = []Todo{}
		config.Calendars = []Calendar{}

		foundCount := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			resourceConfigPath := filepath.Join(entry.Name(), ".nocti.json")
			if _, err := os.Stat(resourceConfigPath); err == nil {
				// Found a resource!
				resData, err := os.ReadFile(resourceConfigPath)
				if err != nil {
					fmt.Printf("Warning: could not read %s: %v\n", resourceConfigPath, err)
					continue
				}

				// We need to extract the type and then the full resource data
				var rawRes map[string]interface{}
				if err := json.Unmarshal(resData, &rawRes); err != nil {
					fmt.Printf("Warning: could not parse %s: %v\n", resourceConfigPath, err)
					continue
				}

				resType, _ := rawRes["type"].(string)

				// Marshal/Unmarshal into the correct struct to be safe
				var res Resource
				if err := json.Unmarshal(resData, &res); err != nil {
					fmt.Printf("Warning: could not map %s to Resource: %v\n", resourceConfigPath, err)
					continue
				}

				switch resType {
				case "notebook":
					config.Notebooks = append(config.Notebooks, Notebook(res))
				case "todo":
					config.Todos = append(config.Todos, Todo(res))
				case "calendar":
					config.Calendars = append(config.Calendars, Calendar(res))
				default:
					fmt.Printf("Warning: unknown resource type '%s' in %s\n", resType, resourceConfigPath)
					continue
				}
				foundCount++
			}
		}

		// 5. Save updated config
		updatedData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal updated config: %w", err)
		}

		if err := os.WriteFile(filename, updatedData, 0644); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Successfully updated %s. Found %d resources.\n", filename, foundCount)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(UpdateCmd)
}
