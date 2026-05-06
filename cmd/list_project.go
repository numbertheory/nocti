package cmd

import (
	"os"
	"path/filepath"
	"sort"
)

func ScanProjectResources(root string, showHidden bool) ([]string, error) {
	var resources []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), ".nocti.json")); err == nil {
			resources = append(resources, entry.Name()+string(os.PathSeparator))
		}
	}

	if showHidden {
		configPath := filepath.Join(".nocti", "nocti.json")
		if _, err := os.Stat(filepath.Join(root, configPath)); err == nil {
			resources = append(resources, configPath)
		}
	}

	sort.Strings(resources)
	return resources, nil
}
