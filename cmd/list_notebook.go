package cmd

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func ScanNotebookFiles(searchDir string, showHidden bool) ([]string, error) {
	var results []string
	// Map to track folders that contain relevant files or are empty
	// but we'll use a more direct approach by walking twice or tracking.
	// Let's use a map to track which directories are "valid"
	validDirs := make(map[string]bool)

	// First pass: identify files and their parent directories
	err := filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == searchDir {
			return nil
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				if !showHidden || d.Name() != ".templates" {
					return filepath.SkipDir
				}
			}

			// Check if this is a sub-resource
			configPath := filepath.Join(path, ".nocti.json")
			if _, err := os.Stat(configPath); err == nil {
				if path != searchDir {
					// Read resource type
					data, err := os.ReadFile(configPath)
					if err == nil {
						var config struct {
							Type string `json:"type"`
						}
						if err := json.Unmarshal(data, &config); err == nil {
							if config.Type == "notebook" {
								// Recurse into nested notebooks
								validDirs[path] = true
								return nil
							} else {
								// For other resources, show them but don't recurse
								relPath, err := filepath.Rel(searchDir, path)
								if err == nil {
									results = append(results, relPath+string(os.PathSeparator))
								}
								return filepath.SkipDir
							}
						}
					}
					// If we can't read/parse it, default to skipping
					return filepath.SkipDir
				}
			}

			// Check if folder is empty
			entries, err := os.ReadDir(path)
			if err == nil && len(entries) == 0 {
				validDirs[path] = true
			}
			return nil
		}

		if showHidden && d.Name() == ".nocti.json" {
			relPath, err := filepath.Rel(searchDir, path)
			if err == nil {
				results = append(results, relPath)
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".txt" {
			relPath, err := filepath.Rel(searchDir, path)
			if err == nil {
				results = append(results, relPath)
				// Mark all parents as valid
				parent := filepath.Dir(path)
				for parent != searchDir && parent != "." {
					validDirs[parent] = true
					parent = filepath.Dir(parent)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Add valid empty folders or folders containing valid files to results
	// results already contains the files. We need to make sure BuildDisplayEntries
	// handles the folder structure correctly.
	// Actually, BuildDisplayEntries reconstructs the tree from file paths.
	// If a folder is empty, it won't have a file path to trigger its creation.
	// So we add "dummy" entries for empty folders.

	for dir := range validDirs {
		relDir, err := filepath.Rel(searchDir, dir)
		if err == nil {
			// Check if this dir is already represented by a file
			found := false
			for _, f := range results {
				if strings.HasPrefix(f, relDir+string(os.PathSeparator)) {
					found = true
					break
				}
			}
			if !found {
				// Add the directory itself as a result
				// We'll append a trailing separator to distinguish it if needed,
				// but BuildDisplayEntries should handle it if we are careful.
				results = append(results, relDir+string(os.PathSeparator))
			}
		}
	}

	return results, nil
}

func GetFilePreview(path string, width int) []string {
	file, err := os.Open(path)
	if err != nil {
		return []string{"Error opening file"}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > width {
			line = line[:width]
		}
		lines = append(lines, line)
	}
	return lines
}
