package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var RawOutput bool

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

		var files []string
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
				if err == nil {
					files = append(files, relPath)
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		if RawOutput {
			for _, f := range files {
				fmt.Println(f)
			}
			return nil
		}

		if len(files) == 0 {
			fmt.Println("No markdown or text files found.")
			return nil
		}

		return runInteractiveList(files, searchDir)
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

func runInteractiveList(files []string, baseDir string) error {
	// Check if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		for _, f := range files {
			fmt.Println(f)
		}
		return nil
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	selectedIndex := 0

	// ANSI escape codes
	const (
		clearScreen = "\033[2J"
		cursorHome  = "\033[H"
		hideCursor  = "\033[?25l"
		showCursor  = "\033[?25h"
		reverseOn   = "\033[7m"
		reverseOff  = "\033[27m"
	)

	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)

	for {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return err
		}

		fmt.Print(clearScreen + cursorHome)

		// Layout: Left column (files), Right column (preview)
		// Status bar at the bottom (last line)
		listWidth := width / 3
		if listWidth < 20 {
			listWidth = 20
		}
		previewWidth := width - listWidth - 3 // -3 for separator and margin
		displayHeight := height - 1

		for i := 0; i < displayHeight && i < len(files); i++ {
			// Move cursor to correct line
			fmt.Printf("\033[%d;1H", i+1)

			filename := files[i]
			if len(filename) > listWidth {
				filename = filename[:listWidth-3] + "..."
			}

			if i == selectedIndex {
				fmt.Printf("%s%-*s%s", reverseOn, listWidth, filename, reverseOff)
			} else {
				fmt.Printf("%-*s", listWidth, filename)
			}

			// Separator
			fmt.Print(" | ")
		}

		// Preview - show it once for the whole screen
		previewLines := getFilePreview(filepath.Join(baseDir, files[selectedIndex]), previewWidth, displayHeight)
		for j, pLine := range previewLines {
			if j >= displayHeight {
				break
			}
			// Move cursor to correct position for preview lines
			fmt.Printf("\033[%d;%dH%s", j+1, listWidth+4, pLine)
		}

		// Status bar
		fmt.Printf("\033[%d;1H%sPress 'q' to exit | Use arrow keys to navigate%s", height, reverseOn, reverseOff)

		// Input handling
		b := make([]byte, 3)
		n, err := os.Stdin.Read(b)
		if err != nil {
			return err
		}

		if n == 1 {
			if b[0] == 'q' || b[0] == 'Q' || b[0] == 3 { // 3 is Ctrl-C
				break
			}
		} else if n == 3 && b[0] == 27 && b[1] == 91 { // ESC [
			switch b[2] {
			case 65: // Up
				if selectedIndex > 0 {
					selectedIndex--
				}
			case 66: // Down
				if selectedIndex < len(files)-1 {
					selectedIndex++
				}
			}
		}
	}

	return nil
}

func getFilePreview(path string, width, height int) []string {
	file, err := os.Open(path)
	if err != nil {
		return []string{"Error opening file"}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for i := 0; i < height && scanner.Scan(); i++ {
		line := scanner.Text()
		if len(line) > width {
			line = line[:width]
		}
		lines = append(lines, line)
	}
	return lines
}

func init() {
	ListCmd.Flags().BoolVarP(&RawOutput, "raw", "r", false, "Standard output the list of files")
	RootCmd.AddCommand(ListCmd)
}
