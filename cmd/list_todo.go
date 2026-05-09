package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ScanTodoItems(searchDir string, showHidden bool) ([]string, error) {
	var markdownFiles []string
	var subResources []string

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Name() == ".nocti.json" || entry.Name() == "nocti.json" {
			if showHidden {
				markdownFiles = append(markdownFiles, entry.Name())
			}
			continue
		}

		if entry.IsDir() {
			// Check if it's a resource
			configPath := filepath.Join(searchDir, entry.Name(), ".nocti.json")
			if _, err := os.Stat(configPath); err == nil {
				subResources = append(subResources, entry.Name()+string(os.PathSeparator))
			}
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".md" {
			markdownFiles = append(markdownFiles, entry.Name())
		}
	}

	sort.Strings(markdownFiles)
	sort.Strings(subResources)

	var results []string
	results = append(results, markdownFiles...)
	results = append(results, subResources...)

	return results, nil
}

func GetTodoPreview(path string, width int) []string {
	// For now, use the same as FilePreview
	return GetFilePreview(path, width)
}
