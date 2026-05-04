package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var RawOutput bool

type DisplayEntry struct {
	RelPath      string
	IsFile       bool
	Depth        int
	Name         string
	ResourceType string // "notebook", "todo", "calendar", or empty for normal folder
}

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

func BuildDisplayEntries(files []string, baseDir string) []DisplayEntry {
	var entries []DisplayEntry
	seenDirs := make(map[string]bool)

	// Sort files to ensure parents are processed before children
	sort.Strings(files)

	getResourceType := func(path string) string {
		configPath := filepath.Join(baseDir, path, ".nocti.json")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return ""
		}
		var config struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return ""
		}
		return config.Type
	}

	for _, f := range files {
		isDirOnly := strings.HasSuffix(f, string(os.PathSeparator))
		cleanPath := strings.TrimSuffix(f, string(os.PathSeparator))
		parts := strings.Split(cleanPath, string(os.PathSeparator))

		// Process parent directories
		for i := 0; i < len(parts)-1; i++ {
			dirPath := filepath.Join(parts[:i+1]...)
			if !seenDirs[dirPath] {
				entries = append(entries, DisplayEntry{
					RelPath:      dirPath,
					IsFile:       false,
					Depth:        i,
					Name:         parts[i],
					ResourceType: getResourceType(dirPath),
				})
				seenDirs[dirPath] = true
			}
		}

		if isDirOnly {
			// Process the empty folder itself
			dirPath := cleanPath
			if !seenDirs[dirPath] {
				entries = append(entries, DisplayEntry{
					RelPath:      dirPath,
					IsFile:       false,
					Depth:        len(parts) - 1,
					Name:         parts[len(parts)-1],
					ResourceType: getResourceType(dirPath),
				})
				seenDirs[dirPath] = true
			}
		} else {
			// Process the file itself
			entries = append(entries, DisplayEntry{
				RelPath: f,
				IsFile:  true,
				Depth:   len(parts) - 1,
				Name:    parts[len(parts)-1],
			})
		}
	}
	return entries
}

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

func loadColorsAndEditor(searchDir string) (*ColorsConfig, string) {
	var colors *ColorsConfig
	var editorCmd string

	// Read colors and editor: check local .nocti.json first, then fallback to main config
	localConfigFile := filepath.Join(searchDir, ".nocti.json")
	if data, err := os.ReadFile(localConfigFile); err == nil {
		var config struct {
			Colors *ColorsConfig `json:"colors"`
			Editor string        `json:"editor"`
		}
		if err := json.Unmarshal(data, &config); err == nil {
			if config.Colors != nil {
				colors = config.Colors
			}
			if config.Editor != "" {
				editorCmd = config.Editor
			}
		}
	}

	if colors == nil || editorCmd == "" {
		if root, err := FindProjectRoot(); err == nil {
			configFile := filepath.Join(root, ".nocti/nocti.json")
			if data, err := os.ReadFile(configFile); err == nil {
				var config FullConfig
				if err := json.Unmarshal(data, &config); err == nil {
					if colors == nil {
						colors = config.Colors
					}
					if editorCmd == "" {
						editorCmd = config.Editor
					}
				}
			}
		}
	}

	if editorCmd == "" {
		editorCmd = "nvim"
	}

	return colors, editorCmd
}

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files in a notebook resource",
	RunE: func(cmd *cobra.Command, args []string) error {
		var searchDir string
		var isProjectRoot bool

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
			// Check if we are in the project root
			root, err := FindProjectRoot()
			wd, _ := os.Getwd()
			if err == nil && wd == root {
				isProjectRoot = true
				searchDir = "."
			} else {
				// Detect if we are inside a nocti resource and if it's a notebook
				_, resourceType, err := FindEnclosingResource()
				if err != nil {
					return fmt.Errorf("not inside a nocti resource and no resource name provided: %w", err)
				}

				if resourceType != "notebook" {
					return fmt.Errorf("the 'list' command is only available inside a notebook resource (current type: %s)", resourceType)
				}
				searchDir = "."
			}
		}

		var files []string
		var err error
		if isProjectRoot {
			files, err = ScanProjectResources(searchDir)
		} else {
			files, err = ScanNotebookFiles(searchDir, false)
		}
		if err != nil {
			return err
		}

		if RawOutput {
			for _, f := range files {
				fmt.Println(f)
			}
			return nil
		}

		if len(files) == 0 && !term.IsTerminal(int(os.Stdout.Fd())) {
			if isProjectRoot {
				fmt.Println("No resources found in project root.")
			} else {
				fmt.Println("No markdown or text files found.")
			}
			return nil
		}

		if len(files) == 0 && RawOutput {
			return nil
		}

		colors, editorCmd := loadColorsAndEditor(searchDir)

		entries := BuildDisplayEntries(files, searchDir)
		return runInteractiveList(entries, searchDir, colors, editorCmd, isProjectRoot)
	},
}

func FindEnclosingResource() (string, string, error) {
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

func GetFGColorCode(colorName string, defaultCode string) string {
	if strings.ToLower(colorName) == "default" {
		return "\033[39m" // Reset foreground
	}
	colors := map[string]string{
		"black":         "\033[38;5;0m",
		"red":           "\033[38;5;1m",
		"green":         "\033[38;5;2m",
		"yellow":        "\033[38;5;3m",
		"blue":          "\033[38;5;4m",
		"magenta":       "\033[38;5;5m",
		"cyan":          "\033[38;5;6m",
		"white":         "\033[38;5;7m",
		"gray":          "\033[38;5;244m",
		"darkgray":      "\033[38;5;236m",
		"lightgray":     "\033[38;5;250m",
		"silver":        "\033[38;5;7m",
		"brightred":     "\033[38;5;9m",
		"brightgreen":   "\033[38;5;10m",
		"brightyellow":  "\033[38;5;11m",
		"brightblue":    "\033[38;5;12m",
		"brightmagenta": "\033[38;5;13m",
		"brightcyan":    "\033[38;5;14m",
		"brightwhite":   "\033[38;5;15m",
		"orange":        "\033[38;5;208m",
		"darkorange":    "\033[38;5;166m",
		"pink":          "\033[38;5;205m",
		"hotpink":       "\033[38;5;198m",
		"purple":        "\033[38;5;93m",
		"violet":        "\033[38;5;129m",
		"brown":         "\033[38;5;94m",
		"navy":          "\033[38;5;18m",
		"teal":          "\033[38;5;30m",
		"olive":         "\033[38;5;58m",
		"maroon":        "\033[38;5;88m",
		"aqua":          "\033[38;5;51m",
		"fuchsia":       "\033[38;5;201m",
		"lime":          "\033[38;5;46m",
		"skyblue":       "\033[38;5;117m",
		"gold":          "\033[38;5;214m",
		"indigo":        "\033[38;5;54m",
		"coral":         "\033[38;5;209m",
		"turquoise":     "\033[38;5;45m",
		"plum":          "\033[38;5;96m",
		"orchid":        "\033[38;5;170m",
		"salmon":        "\033[38;5;210m",
	}

	if code, ok := colors[strings.ToLower(colorName)]; ok {
		return code
	}
	return defaultCode
}

func GetColorCode(colorName string, defaultCode string) string {
	if strings.ToLower(colorName) == "default" {
		return "\033[49m" // Reset background
	}
	colors := map[string]string{
		"black":         "\033[48;5;0m",
		"red":           "\033[48;5;1m",
		"green":         "\033[48;5;2m",
		"yellow":        "\033[48;5;3m",
		"blue":          "\033[48;5;4m",
		"magenta":       "\033[48;5;5m",
		"cyan":          "\033[48;5;6m",
		"white":         "\033[48;5;7m",
		"gray":          "\033[48;5;244m",
		"darkgray":      "\033[48;5;236m",
		"lightgray":     "\033[48;5;250m",
		"silver":        "\033[48;5;7m",
		"brightred":     "\033[48;5;9m",
		"brightgreen":   "\033[48;5;10m",
		"brightyellow":  "\033[48;5;11m",
		"brightblue":    "\033[48;5;12m",
		"brightmagenta": "\033[48;5;13m",
		"brightcyan":    "\033[48;5;14m",
		"brightwhite":   "\033[48;5;15m",
		"orange":        "\033[48;5;208m",
		"darkorange":    "\033[48;5;166m",
		"pink":          "\033[48;5;205m",
		"hotpink":       "\033[48;5;198m",
		"purple":        "\033[48;5;93m",
		"violet":        "\033[48;5;129m",
		"brown":         "\033[48;5;94m",
		"navy":          "\033[48;5;18m",
		"teal":          "\033[48;5;30m",
		"olive":         "\033[48;5;58m",
		"maroon":        "\033[48;5;88m",
		"aqua":          "\033[48;5;51m",
		"fuchsia":       "\033[48;5;201m",
		"lime":          "\033[48;5;46m",
		"skyblue":       "\033[48;5;117m",
		"gold":          "\033[48;5;214m",
		"indigo":        "\033[48;5;54m",
		"coral":         "\033[48;5;209m",
		"turquoise":     "\033[48;5;45m",
		"plum":          "\033[48;5;96m",
		"orchid":        "\033[48;5;170m",
		"salmon":        "\033[48;5;210m",
	}

	if code, ok := colors[strings.ToLower(colorName)]; ok {
		return code
	}
	return defaultCode
}

func runInteractiveList(entries []DisplayEntry, baseDir string, colors *ColorsConfig, editorCmd string, isProjectRoot bool) error {
	// Track initial state for "back to root" functionality
	initialDir := baseDir
	initialIsRoot := isProjectRoot
	showHidden := false

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
	previewOffset := 0
	focusList := true // true = List, false = Preview
	showHelp := false

	// Creation states
	showCreateType := false
	showCreateName := false
	createTypeSelected := 0 // 0 = File, 1 = Folder
	createInputName := ""

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
		boldOn         = "\033[1m"
		boldOff        = "\033[22m"
		reset          = "\033[0m"
	)

	// Icons
	const (
		iconFolder   = " "
		iconText     = " "
		iconMarkdown = " "
		iconNotebook = " "
		iconCalendar = " "
		iconTodo     = " "
	)

	fmt.Print(enterAltScreen + hideCursor)
	defer fmt.Print(showCursor + exitAltScreen)

	for {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return err
		}

		fmt.Print(reset + clearScreen + cursorHome)

		// Layout Constants
		headerHeight := 1
		statusHeight := 1
		contentHeight := height - headerHeight - statusHeight

		listWidth := width / 3
		if listWidth < 30 {
			listWidth = 30
		}
		previewWidth := width - listWidth - 5 // -3 for separator, -1 for margin, -1 for scrollbar

		// 1. Draw Header Bar
		listColor := "\033[44m"       // Default Blue
		prevColor := "\033[48;5;208m" // Default Orange
		if colors != nil {
			listColor = GetColorCode(colors.FileList, listColor)
			prevColor = GetColorCode(colors.PreviewPane, prevColor)
		}

		fmt.Printf("\033[1;1H")
		listHeader := " FILE LIST "
		if isProjectRoot {
			listHeader = " RESOURCES "
		}
		prevHeader := " PREVIEW "
		if focusList {
			listHeader = "[" + listHeader + "]"
		} else {
			prevHeader = "[" + prevHeader + "]"
		}

		// Draw File List Header (filled to listWidth)
		fmt.Printf("%s%-*s%s", listColor, listWidth, listHeader, reset)
		// Vertical separator in header row
		fmt.Printf(" │ ")
		// Draw Preview Header (filled to remaining width)
		fmt.Printf("%s%-*s%s", prevColor, width-listWidth-3, prevHeader, reset)

		// 2. Draw List Content
		for i := 0; i < contentHeight; i++ {
			row := i + headerHeight + 1
			fmt.Printf("\033[%d;1H", row)

			if i < len(entries) {
				entry := entries[i]
				indent := strings.Repeat("  ", entry.Depth)
				icon := iconFolder
				if entry.IsFile {
					if strings.HasSuffix(strings.ToLower(entry.Name), ".md") {
						icon = iconMarkdown
					} else if entry.Name == ".nocti.json" {
						icon = " " // Gear icon for settings
					} else {
						icon = iconText
					}
				} else {
					switch entry.ResourceType {
					case "notebook":
						icon = iconNotebook
					case "calendar":
						icon = iconCalendar
					case "todo":
						icon = iconTodo
					}
				}

				displayStr := fmt.Sprintf("%s%s%s", indent, icon, entry.Name)
				if len(displayStr) > listWidth {
					displayStr = displayStr[:listWidth-3] + "..."
				}

				// Apply resource-specific colors
				resFg := ""
				resBg := ""
				if entry.ResourceType != "" {
					resFg = "\033[36m" // Default Cyan foreground
					resBg = ""         // Default background
					if colors != nil {
						resFg = GetFGColorCode(colors.ResourceFg, resFg)
						resBg = GetColorCode(colors.ResourceBg, resBg)
					}
				}

				if i == selectedIndex {
					fmt.Print(reverseOn)
					if focusList {
						fmt.Print(boldOn)
					}
					fmt.Printf("%s%s%-*s", resFg, resBg, listWidth, displayStr)
					fmt.Print(reverseOff + boldOff + reset)
				} else {
					fmt.Printf("%s%s%-*s%s", resFg, resBg, listWidth, displayStr, reset)
				}
			} else {
				fmt.Printf("%-*s", listWidth, "")
			}

			// Separator
			fmt.Printf("\033[%d;%dH │ ", row, listWidth+1)
		}

		// 3. Preview Content
		var allPreviewLines []string
		if len(entries) > 0 {
			selected := entries[selectedIndex]
			if isProjectRoot {
				resConfigPath := filepath.Join(baseDir, selected.RelPath, ".nocti.json")
				data, err := os.ReadFile(resConfigPath)
				if err == nil {
					var config struct {
						ID   string `json:"id"`
						Name string `json:"name"`
						Type string `json:"type"`
					}
					json.Unmarshal(data, &config)

					allPreviewLines = append(allPreviewLines, "Resource: "+config.Name)
					allPreviewLines = append(allPreviewLines, "Type:     "+config.Type)
					allPreviewLines = append(allPreviewLines, "ID:       "+config.ID)
					allPreviewLines = append(allPreviewLines, "")

					if config.Type == "notebook" {
						files, _ := ScanNotebookFiles(filepath.Join(baseDir, selected.RelPath), false)
						allPreviewLines = append(allPreviewLines, fmt.Sprintf("Notes:    %d", len(files)))

					} else {
						allPreviewLines = append(allPreviewLines, "[ Not yet implemented ]")
					}
				} else {
					allPreviewLines = []string{"Error reading resource config"}
				}
			} else if selected.IsFile {
				allPreviewLines = GetFilePreview(filepath.Join(baseDir, selected.RelPath), previewWidth)
			} else {
				allPreviewLines = []string{"Directory: " + selected.RelPath}
			}
		} else {
			if isProjectRoot {
				allPreviewLines = []string{"No resources found in project root."}
			} else {
				allPreviewLines = []string{"No markdown or text files found.", "", "Press 'n' to create a new file."}
			}
		}

		// Bound check previewOffset
		if previewOffset < 0 {
			previewOffset = 0
		}
		if previewOffset > len(allPreviewLines)-contentHeight && len(allPreviewLines) > contentHeight {
			previewOffset = len(allPreviewLines) - contentHeight
		} else if len(allPreviewLines) <= contentHeight {
			previewOffset = 0
		}

		for j := 0; j < contentHeight && (j+previewOffset) < len(allPreviewLines); j++ {
			pLine := allPreviewLines[j+previewOffset]
			fmt.Printf("\033[%d;%dH%s", j+headerHeight+1, listWidth+5, pLine)
		}

		// 4. Draw Scrollbar
		if len(allPreviewLines) > contentHeight {
			scrollbarX := width
			thumbHeight := (contentHeight * contentHeight) / len(allPreviewLines)
			if thumbHeight < 1 {
				thumbHeight = 1
			}

			thumbPos := 0
			if len(allPreviewLines) > contentHeight {
				thumbPos = (previewOffset * (contentHeight - thumbHeight)) / (len(allPreviewLines) - contentHeight)
			}

			for i := 0; i < contentHeight; i++ {
				fmt.Printf("\033[%d;%dH", i+headerHeight+1, scrollbarX)
				if i >= thumbPos && i < thumbPos+thumbHeight {
					fmt.Print(reverseOn + " " + reverseOff)
				} else {
					fmt.Print("│")
				}
			}
		}

		// 5. Help Modal
		if showHelp {
			modalWidth := 50
			modalHeight := 16 // Increased for equal top/bottom buffer
			if width < modalWidth {
				modalWidth = width - 4
			}
			startX := (width - modalWidth) / 2
			startY := (height - modalHeight) / 2

			// Resolve colors with specific defaults
			hBg := "\033[44m"       // Default Blue
			hFg := "\033[38;5;15m"  // Default White
			hbBg := "\033[44m"      // Default Blue
			hbFg := "\033[38;5;15m" // Default White

			if colors != nil {
				hBg = GetColorCode(colors.HelpBg, hBg)
				hFg = GetFGColorCode(colors.HelpFg, hFg)
				hbBg = GetColorCode(colors.HelpBorderBg, hbBg)
				hbFg = GetFGColorCode(colors.HelpBorderFg, hbFg)
			}

			// Draw modal box background
			for i := 0; i < modalHeight; i++ {
				fmt.Printf("\033[%d;%dH%s%s%*s%s", startY+i, startX, hBg, hFg, modalWidth, "", reset)
			}

			// Draw Border
			fmt.Printf("\033[%d;%dH%s%s┌%s┐%s", startY, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)
			for i := 1; i < modalHeight-1; i++ {
				fmt.Printf("\033[%d;%dH%s%s│\033[%d;%dH%s%s│%s", startY+i, startX, hbBg, hbFg, startY+i, startX+modalWidth-1, hbBg, hbFg, reset)
			}
			fmt.Printf("\033[%d;%dH%s%s└%s┘%s", startY+modalHeight-1, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)

			// Content
			fmt.Printf("\033[%d;%dH%s%s%s HELP %s", startY+1, startX+(modalWidth-6)/2, hBg, hFg, boldOn+reverseOn, reset)

			helpLines := []string{
				"  Navigation:",
				"    ↑ / ↓      : Navigate List / Preview",
				"    TAB        : Switch Focus",
				"    PgUp/PgDn  : Page Preview",
				"",
				"  Actions:",
				"    n          : New File/Folder",
				"    Ctrl+T     : Toggle Settings/Templates",
				"    ENTER      : Edit File",
				"    q          : Quit",
				"    ESC        : Close Help",
			}

			for i, line := range helpLines {
				fmt.Printf("\033[%d;%dH%s%s%s", startY+3+i, startX+2, hBg, hFg, line)
			}
			fmt.Print(reset)
		}

		// 6. Create Type Modal
		if showCreateType {
			modalWidth := 40
			modalHeight := 8
			startX := (width - modalWidth) / 2
			startY := (height - modalHeight) / 2

			// Resolve colors with specific defaults
			hBg := "\033[44m"       // Default Blue
			hFg := "\033[38;5;15m"  // Default White
			hbBg := "\033[44m"      // Default Blue
			hbFg := "\033[38;5;15m" // Default White

			if colors != nil {
				hBg = GetColorCode(colors.HelpBg, hBg)
				hFg = GetFGColorCode(colors.HelpFg, hFg)
				hbBg = GetColorCode(colors.HelpBorderBg, hbBg)
				hbFg = GetFGColorCode(colors.HelpBorderFg, hbFg)
			}

			// Draw modal box background
			for i := 0; i < modalHeight; i++ {
				fmt.Printf("\033[%d;%dH%s%s%*s%s", startY+i, startX, hBg, hFg, modalWidth, "", reset)
			}

			// Draw Border
			fmt.Printf("\033[%d;%dH%s%s┌%s┐%s", startY, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)
			for i := 1; i < modalHeight-1; i++ {
				fmt.Printf("\033[%d;%dH%s%s│\033[%d;%dH%s%s│%s", startY+i, startX, hbBg, hbFg, startY+i, startX+modalWidth-1, hbBg, hbFg, reset)
			}
			fmt.Printf("\033[%d;%dH%s%s└%s┘%s", startY+modalHeight-1, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)

			fmt.Printf("\033[%d;%dH%s%s CREATE NEW %s", startY+1, startX+(modalWidth-12)/2, hBg, hFg+boldOn, reset)

			options := []string{" File ", " Folder "}
			for i, opt := range options {
				fmt.Printf("\033[%d;%dH", startY+3+i, startX+14)
				if i == createTypeSelected {
					fmt.Printf("%s%s%s", hBg+hFg+reverseOn, opt, reset+hBg+hFg)
				} else {
					fmt.Printf("%s%s%s", hBg, hFg+opt, reset)
				}
			}
			fmt.Printf("\033[%d;%dH%s%s ↑/↓ to select | ENTER to confirm %s", startY+6, startX+(modalWidth-32)/2, hBg, hFg, reset)
		}

		// 7. Create Name Modal
		if showCreateName {
			modalWidth := 60
			modalHeight := 8
			startX := (width - modalWidth) / 2
			startY := (height - modalHeight) / 2

			// Resolve colors with specific defaults
			hBg := "\033[44m"       // Default Blue
			hFg := "\033[38;5;15m"  // Default White
			hbBg := "\033[44m"      // Default Blue
			hbFg := "\033[38;5;15m" // Default White

			if colors != nil {
				hBg = GetColorCode(colors.HelpBg, hBg)
				hFg = GetFGColorCode(colors.HelpFg, hFg)
				hbBg = GetColorCode(colors.HelpBorderBg, hbBg)
				hbFg = GetFGColorCode(colors.HelpBorderFg, hbFg)
			}

			// Draw modal box background
			for i := 0; i < modalHeight; i++ {
				fmt.Printf("\033[%d;%dH%s%s%*s%s", startY+i, startX, hBg, hFg, modalWidth, "", reset)
			}

			// Draw Border
			fmt.Printf("\033[%d;%dH%s%s┌%s┐%s", startY, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)
			for i := 1; i < modalHeight-1; i++ {
				fmt.Printf("\033[%d;%dH%s%s│\033[%d;%dH%s%s│%s", startY+i, startX, hbBg, hbFg, startY+i, startX+modalWidth-1, hbBg, hbFg, reset)
			}
			fmt.Printf("\033[%d;%dH%s%s└%s┘%s", startY+modalHeight-1, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)

			typeStr := "FILE"
			if createTypeSelected == 1 {
				typeStr = "FOLDER"
			}
			fmt.Printf("\033[%d;%dH%s%s NEW %s NAME %s", startY+1, startX+(modalWidth-len(typeStr)-10)/2, hBg, hFg+boldOn, typeStr, reset)

			fmt.Printf("\033[%d;%dH%s%s > %s%s%s", startY+3, startX+4, hBg, hFg, reverseOn, createInputName, reset+hBg+hFg)
			fmt.Printf("\033[%d;%dH%s%s ENTER to create | ESC to cancel %s", startY+6, startX+(modalWidth-28)/2, hBg, hFg, reset)
		}

		// 8. Status bar
		fmt.Printf("\033[%d;1H%s%s Ctrl+H - help %s", height, reset, reverseOn, reverseOff)

		// Input handling
		b := make([]byte, 8)
		n, err := os.Stdin.Read(b)
		if err != nil {
			return err
		}

		if n == 1 {
			if b[0] == 20 { // Ctrl+T
				if !isProjectRoot {
					showHidden = !showHidden
					files, _ := ScanNotebookFiles(baseDir, showHidden)
					entries = BuildDisplayEntries(files, baseDir)
					if selectedIndex >= len(entries) {
						selectedIndex = len(entries) - 1
					}
					if selectedIndex < 0 {
						selectedIndex = 0
					}
				}
				continue
			}
			if b[0] == 'q' || b[0] == 'Q' || b[0] == 3 {
				if !isProjectRoot && initialIsRoot {
					// Return to project root
					isProjectRoot = true
					baseDir = initialDir
					files, _ := ScanProjectResources(baseDir)
					entries = BuildDisplayEntries(files, baseDir)
					colors, editorCmd = loadColorsAndEditor(baseDir)
					selectedIndex = 0
					previewOffset = 0
					focusList = true
					continue
				}
				break
			}

			if b[0] == 8 { // Ctrl+H
				showHelp = true
				continue
			}
			if b[0] == 27 { // ESC
				if showHelp || showCreateType || showCreateName {
					showHelp = false
					showCreateType = false
					showCreateName = false
					continue
				}
				if !isProjectRoot && initialIsRoot {
					// Return to project root
					isProjectRoot = true
					baseDir = initialDir
					files, _ := ScanProjectResources(baseDir)
					entries = BuildDisplayEntries(files, baseDir)
					colors, editorCmd = loadColorsAndEditor(baseDir)
					selectedIndex = 0
					previewOffset = 0
					focusList = true
					continue
				}
				break
			}

			if showHelp {
				continue
			}

			// Creation state handling
			if showCreateType {
				if b[0] == '\r' || b[0] == '\n' {
					showCreateType = false
					showCreateName = true
					createInputName = ""
					continue
				}
			} else if showCreateName {
				if b[0] == '\r' || b[0] == '\n' {
					// Perform creation
					if createInputName != "" {
						targetDir := baseDir
						if len(entries) > 0 {
							selected := entries[selectedIndex]
							if selected.IsFile {
								targetDir = filepath.Join(baseDir, filepath.Dir(selected.RelPath))
							} else {
								targetDir = filepath.Join(baseDir, selected.RelPath)
							}
						}

						newName := createInputName
						if createTypeSelected == 0 {
							// File
							ext := filepath.Ext(newName)
							if ext != ".md" && ext != ".txt" {
								newName += ".md"
							}
							os.WriteFile(filepath.Join(targetDir, newName), []byte(""), 0644)
						} else {
							// Folder
							os.MkdirAll(filepath.Join(targetDir, newName), 0755)
						}

						// Refresh
						files, _ := ScanNotebookFiles(baseDir, showHidden)
						entries = BuildDisplayEntries(files, baseDir)
						// Reset selection to something reasonable if it changed
						if selectedIndex >= len(entries) {
							selectedIndex = len(entries) - 1
						}
						if selectedIndex < 0 {
							selectedIndex = 0
						}
					}
					showCreateName = false
					continue
				} else if b[0] == 127 || b[0] == 8 { // Backspace
					if len(createInputName) > 0 {
						createInputName = createInputName[:len(createInputName)-1]
					}
					continue
				} else if b[0] >= 32 && b[0] <= 126 {
					createInputName += string(b[0])
					continue
				}
			}

			if !isProjectRoot && (b[0] == 'n' || b[0] == 'N') {
				showCreateType = true
				createTypeSelected = 0
				continue
			}

			if b[0] == '\t' {
				focusList = !focusList
			}
			if b[0] == '\r' || b[0] == '\n' {
				if isProjectRoot && len(entries) > 0 {
					selected := entries[selectedIndex]
					resConfigPath := filepath.Join(baseDir, selected.RelPath, ".nocti.json")
					data, err := os.ReadFile(resConfigPath)
					if err == nil {
						var config struct {
							Type string `json:"type"`
						}
						json.Unmarshal(data, &config)
						if config.Type == "notebook" {
							// Jump to notebook
							newBaseDir := filepath.Join(baseDir, selected.RelPath)
							showHidden = false // Reset hidden toggle when switching notebooks
							newFiles, _ := ScanNotebookFiles(newBaseDir, showHidden)
							newEntries := BuildDisplayEntries(newFiles, newBaseDir)

							// Refresh colors and editor for the new notebook
							colors, editorCmd = loadColorsAndEditor(newBaseDir)

							entries = newEntries
							baseDir = newBaseDir
							isProjectRoot = false
							selectedIndex = 0
							previewOffset = 0
							focusList = true
							continue
						}
					}
				} else if !isProjectRoot && len(entries) > 0 && entries[selectedIndex].IsFile {
					term.Restore(int(os.Stdin.Fd()), oldState)
					fmt.Print(showCursor + exitAltScreen)

					filePath := filepath.Join(baseDir, entries[selectedIndex].RelPath)
					cmd := exec.Command(editorCmd, filePath)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Run()

					// Re-enter raw mode and alt screen
					oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
					fmt.Print(enterAltScreen + hideCursor)
				}
			}
		} else if n >= 3 && b[0] == 27 && b[1] == 91 {
			if showCreateType {
				switch b[2] {
				case 65: // Up
					createTypeSelected = 0
				case 66: // Down
					createTypeSelected = 1
				}
				continue
			}

			if !showHelp && !showCreateName {
				if focusList {
					switch b[2] {
					case 65: // Up
						if selectedIndex > 0 {
							selectedIndex--
							previewOffset = 0
						}
					case 66: // Down
						if selectedIndex < len(entries)-1 {
							selectedIndex++
							previewOffset = 0
						}
					}
				} else {
					// Preview focus navigation
					switch b[2] {
					case 65: // Up
						if previewOffset > 0 {
							previewOffset--
						}
					case 66: // Down
						if previewOffset < len(allPreviewLines)-contentHeight {
							previewOffset++
						}
					case 53: // PgUp (ESC [ 5 ~)
						if n >= 4 && b[3] == 126 {
							previewOffset -= contentHeight
						}
					case 54: // PgDn (ESC [ 6 ~)
						if n >= 4 && b[3] == 126 {
							previewOffset += contentHeight
						}
					}
				}
			}
		}
	}

	return nil
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

func init() {
	ListCmd.Flags().BoolVarP(&RawOutput, "raw", "r", false, "Standard output the list of files")
	RootCmd.AddCommand(ListCmd)
}
