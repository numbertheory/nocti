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

type DisplayEntry struct {
	RelPath string
	IsFile  bool
	Depth   int
	Name    string
}

func buildDisplayEntries(files []string) []DisplayEntry {
	var entries []DisplayEntry
	seenDirs := make(map[string]bool)

	for _, f := range files {
		parts := strings.Split(f, string(os.PathSeparator))
		// Process parent directories
		for i := 0; i < len(parts)-1; i++ {
			dirPath := filepath.Join(parts[:i+1]...)
			if !seenDirs[dirPath] {
				entries = append(entries, DisplayEntry{
					RelPath: dirPath,
					IsFile:  false,
					Depth:   i,
					Name:    parts[i],
				})
				seenDirs[dirPath] = true
			}
		}
		// Process the file itself
		entries = append(entries, DisplayEntry{
			RelPath: f,
			IsFile:  true,
			Depth:   len(parts) - 1,
			Name:    parts[len(parts)-1],
		})
	}
	return entries
}

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

		entries := buildDisplayEntries(files)
		return runInteractiveList(entries, searchDir)
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

func runInteractiveList(entries []DisplayEntry, baseDir string) error {
	// Check if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		for _, e := range entries {
			if e.IsFile {
				fmt.Println(e.RelPath)
			}
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
		clearScreen    = "\033[2J"
		cursorHome     = "\033[H"
		hideCursor     = "\033[?25l"
		showCursor     = "\033[?25h"
		reverseOn      = "\033[7m"
		reverseOff     = "\033[27m"
		enterAltScreen = "\033[?1049h"
		exitAltScreen  = "\033[?1049l"
	)

	// Icons
	const (
		iconFolder   = " "
		iconText     = " "
		iconMarkdown = " "
	)

	fmt.Print(enterAltScreen + hideCursor)
	defer fmt.Print(showCursor + exitAltScreen)

	for {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return err
		}

		fmt.Print(clearScreen + cursorHome)

		listWidth := width / 3
		if listWidth < 30 {
			listWidth = 30
		}
		previewWidth := width - listWidth - 3
		displayHeight := height - 1

		for i := 0; i < displayHeight && i < len(entries); i++ {
			fmt.Printf("\033[%d;1H", i+1)

			entry := entries[i]
			indent := strings.Repeat("  ", entry.Depth)
			icon := iconFolder
			if entry.IsFile {
				if strings.HasSuffix(strings.ToLower(entry.Name), ".md") {
					icon = iconMarkdown
				} else {
					icon = iconText
				}
			}

			displayStr := fmt.Sprintf("%s%s%s", indent, icon, entry.Name)
			if len(displayStr) > listWidth {
				displayStr = displayStr[:listWidth-3] + "..."
			}

			if i == selectedIndex {
				fmt.Printf("%s%-*s%s", reverseOn, listWidth, displayStr, reverseOff)
			} else {
				fmt.Printf("%-*s", listWidth, displayStr)
			}

			fmt.Print(" | ")
		}

		// Preview
		var previewLines []string
		selected := entries[selectedIndex]
		if selected.IsFile {
			previewLines = getFilePreview(filepath.Join(baseDir, selected.RelPath), previewWidth, displayHeight)
		} else {
			previewLines = []string{"Directory: " + selected.RelPath}
		}

		for j, pLine := range previewLines {
			if j >= displayHeight {
				break
			}
			fmt.Printf("\033[%d;%dH%s", j+1, listWidth+4, pLine)
		}

		fmt.Printf("\033[%d;1H%sPress 'q' to exit | Use arrow keys to navigate%s", height, reverseOn, reverseOff)

		b := make([]byte, 3)
		n, err := os.Stdin.Read(b)
		if err != nil {
			return err
		}

		if n == 1 {
			if b[0] == 'q' || b[0] == 'Q' || b[0] == 3 {
				break
			}
		} else if n == 3 && b[0] == 27 && b[1] == 91 {
			switch b[2] {
			case 65: // Up
				if selectedIndex > 0 {
					selectedIndex--
				}
			case 66: // Down
				if selectedIndex < len(entries)-1 {
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
