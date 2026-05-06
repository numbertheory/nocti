package cmd

import (
	"os"
	"path/filepath"
	"sort"
)

func ScanProjectResources(root string) ([]string, error) {
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
	sort.Strings(resources)
	return resources, nil
}
