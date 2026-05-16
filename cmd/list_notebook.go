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
		if ext == ".md" || ext == ".txt" || ext == ".json" {
			if !showHidden && strings.HasPrefix(d.Name(), ".") {
				return nil
			}
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

func GetFilePreview(path string, width int) []PreviewLine {
	file, err := os.Open(path)
	if err != nil {
		return []PreviewLine{{Text: "Error opening file"}}
	}
	defer file.Close()

	var rawLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}

	formattedLines := FormatTables(rawLines)
	isJSON := strings.ToLower(filepath.Ext(path)) == ".json"

	var lines []PreviewLine
	lineNo := 1
	for _, line := range formattedLines {
		if isJSON {
			line = ColorizeJSON(line)
		}

		// Detect if this is a table row (starts and ends with box drawing or pipe)
		isTable := (strings.HasPrefix(line, "│") && strings.HasSuffix(line, "│")) ||
			(strings.HasPrefix(line, "┌") && strings.HasSuffix(line, "┐")) ||
			(strings.HasPrefix(line, "├") && strings.HasSuffix(line, "┤")) ||
			(strings.HasPrefix(line, "└") && strings.HasSuffix(line, "┘"))

		line = ProcessHighlights(line)

		if VisibleLenWithLinks(line) <= width {
			lines = append(lines, PreviewLine{Text: line, LineNo: lineNo})
			lineNo++
			continue
		}

		if isTable {
			// Truncate table rows instead of wrapping
			// We need to be careful with ANSI codes when truncating.
			vLen := VisibleLenWithLinks(line)
			if vLen > width {
				_, mapping := StripANSIWithMapping(line)

				// Ensure we don't exceed mapping bounds
				truncIdx := width - 3
				if truncIdx < 0 {
					truncIdx = 0
				}
				if truncIdx >= len(mapping) {
					truncIdx = len(mapping) - 1
				}

				truncated := line[:mapping[truncIdx]] + "..."
				// Close the table border if it was a data row
				if strings.HasSuffix(line, "│") {
					truncated += "│"
				} else if strings.HasSuffix(line, "┐") {
					truncated += "┐"
				} else if strings.HasSuffix(line, "┤") {
					truncated += "┤"
				} else if strings.HasSuffix(line, "┘") {
					truncated += "┘"
				}
				lines = append(lines, PreviewLine{Text: truncated, LineNo: lineNo})
			} else {
				lines = append(lines, PreviewLine{Text: line, LineNo: lineNo})
			}
			lineNo++
			continue
		}

		// Word wrap logic for non-table lines
		words := strings.Fields(line)
		if len(words) == 0 {
			lines = append(lines, PreviewLine{Text: "", LineNo: lineNo})
			lineNo++
			continue
		}

		currentLine := ""
		isFirst := true
		for _, word := range words {
			// If adding this word exceeds width
			if VisibleLenWithLinks(currentLine)+1+VisibleLenWithLinks(word) > width && currentLine != "" {
				lNo := 0
				if isFirst {
					lNo = lineNo
					isFirst = false
				}
				lines = append(lines, PreviewLine{Text: currentLine, LineNo: lNo})
				currentLine = ""
			}

			if VisibleLenWithLinks(word) > width {
				// Handle extremely long words by breaking them
				if currentLine != "" {
					lNo := 0
					if isFirst {
						lNo = lineNo
						isFirst = false
					}
					lines = append(lines, PreviewLine{Text: currentLine, LineNo: lNo})
					currentLine = ""
				}

				for VisibleLenWithLinks(word) > width {
					lNo := 0
					if isFirst {
						lNo = lineNo
						isFirst = false
					}

					// Simple break - might break ANSI or Markdown link
					// To be robust we'd need a more complex breaker, but for now:
					lines = append(lines, PreviewLine{Text: word[:width], LineNo: lNo})
					word = word[width:]
				}
				currentLine = word
			} else {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			}
		}
		if currentLine != "" {
			lNo := 0
			if isFirst {
				lNo = lineNo
			}
			lines = append(lines, PreviewLine{Text: currentLine, LineNo: lNo})
		}
		lineNo++
	}
	return lines
}
