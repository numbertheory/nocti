package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files in a notebook resource",
	RunE: func(cmd *cobra.Command, args []string) error {
		var searchDir string

		if len(args) > 0 {
			target := args[0]
			info, err := os.Stat(target)
			if err != nil || !info.IsDir() {
				return fmt.Errorf("target '%s' is not a directory", target)
			}

			configPath := filepath.Join(target, ".nocti.json")
			data, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("target '%s' is not a nocti resource (could not read .nocti.json)", target)
			}

			var config struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("failed to parse config in '%s': %w", target, err)
			}

			if config.Type != "notebook" {
				return fmt.Errorf("target '%s' is a %s, but 'list' only works on notebooks", target, config.Type)
			}
			searchDir = target
		} else {
			// Detect if we are inside a nocti resource and if it's a notebook
			_, resourceType, err := findEnclosingResource()
			if err != nil {
				return fmt.Errorf("not inside a nocti resource and no resource name provided: %w", err)
			}

			if resourceType != "notebook" {
				return fmt.Errorf("the 'list' command is only available inside a notebook resource (current type: %s)", resourceType)
			}
			searchDir = "."
		}

		err := filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip the starting directory itself from the resource check
			if path == searchDir {
				return nil
			}

			if d.IsDir() {
				// Ignore .git folders
				if d.Name() == ".git" {
					return filepath.SkipDir
				}

				// Check if this subdirectory is a nocti resource
				if _, err := os.Stat(filepath.Join(path, ".nocti.json")); err == nil {
					return filepath.SkipDir
				}
				return nil
			}

			// It's a file, check extension
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".md" || ext == ".txt" {
				relPath, err := filepath.Rel(searchDir, path)
				if err != nil {
					fmt.Println(path)
				} else {
					fmt.Println(relPath)
				}
			}

			return nil
		})

		return err
	},
}

func findEnclosingResource() (string, string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	for {
		configPath := filepath.Join(wd, ".nocti.json")
		if _, err := os.Stat(configPath); err == nil {
			// Found a resource config, read its type
			data, err := os.ReadFile(configPath)
			if err != nil {
				return "", "", err
			}

			var config struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(data, &config); err != nil {
				return "", "", err
			}

			return wd, config.Type, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return "", "", fmt.Errorf(".nocti.json not found in parents")
}

func init() {
	RootCmd.AddCommand(ListCmd)
}
