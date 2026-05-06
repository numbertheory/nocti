package cmd

import (
	"encoding/json"
	"fmt"
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

func BuildDisplayEntries(files []string, baseDir string, includeRoot bool) []DisplayEntry {
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

	depthOffset := 0
	if includeRoot {
		depthOffset = 1
		// Add the root resource itself
		configPath := filepath.Join(baseDir, ".nocti.json")
		projectConfigPath := filepath.Join(baseDir, ".nocti", "nocti.json")
		name := filepath.Base(baseDir)
		resType := "notebook"

		if data, err := os.ReadFile(configPath); err == nil {
			var config struct {
				Name string `json:"name"`
				Type string `json:"type"`
			}
			if err := json.Unmarshal(data, &config); err == nil {
				if config.Name != "" {
					name = config.Name
				}
				if config.Type != "" {
					resType = config.Type
				}
			}
		} else if data, err := os.ReadFile(projectConfigPath); err == nil {
			var config struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(data, &config); err == nil {
				if config.Name != "" {
					name = config.Name
				}
				resType = "" // Special case for project root
			}
		}

		entries = append(entries, DisplayEntry{
			RelPath:      ".",
			IsFile:       false,
			Depth:        0,
			Name:         name,
			ResourceType: resType,
		})
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
					Depth:        i + depthOffset,
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
					Depth:        len(parts) - 1 + depthOffset,
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
				Depth:   len(parts) - 1 + depthOffset,
				Name:    parts[len(parts)-1],
			})
		}
	}
	return entries
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
			files, err = ScanProjectResources(searchDir, false)
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

		entries := BuildDisplayEntries(files, searchDir, true)
		return runInteractiveList(entries, searchDir, colors, editorCmd, isProjectRoot)
	},
}

type ResourceConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func FindEnclosingResourceIn(startDir string) (*ResourceConfig, error) {
	wd := startDir
	for {
		configPath := filepath.Join(wd, ".nocti.json")
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return nil, err
			}

			var config ResourceConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return nil, err
			}

			return &config, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return nil, fmt.Errorf(".nocti.json not found in parents")
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
		"black":         "\033[30m",
		"red":           "\033[31m",
		"green":         "\033[32m",
		"yellow":        "\033[33m",
		"blue":          "\033[34m",
		"magenta":       "\033[35m",
		"cyan":          "\033[36m",
		"white":         "\033[37m",
		"gray":          "\033[38;5;244m",
		"darkgray":      "\033[38;5;236m",
		"lightgray":     "\033[38;5;250m",
		"silver":        "\033[38;5;7m",
		"brightred":     "\033[91m",
		"brightgreen":   "\033[92m",
		"brightyellow":  "\033[93m",
		"brightblue":    "\033[94m",
		"brightmagenta": "\033[95m",
		"brightcyan":    "\033[96m",
		"brightwhite":   "\033[97m",
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
		"black":         "\033[40m",
		"red":           "\033[41m",
		"green":         "\033[42m",
		"yellow":        "\033[43m",
		"blue":          "\033[44m",
		"magenta":       "\033[45m",
		"cyan":          "\033[46m",
		"white":         "\033[47m",
		"gray":          "\033[48;5;244m",
		"darkgray":      "\033[48;5;236m",
		"lightgray":     "\033[48;5;250m",
		"silver":        "\033[48;5;7m",
		"brightred":     "\033[101m",
		"brightgreen":   "\033[102m",
		"brightyellow":  "\033[103m",
		"brightblue":    "\033[104m",
		"brightmagenta": "\033[105m",
		"brightcyan":    "\033[106m",
		"brightwhite":   "\033[107m",
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
	type navState struct {
		dir           string
		entries       []DisplayEntry
		isProjectRoot bool
		colors        *ColorsConfig
		editorCmd     string
	}
	var navStack []navState

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
		iconProject  = " "
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
		listBg := "\033[48;5;208m" // Default Orange
		listFg := "\033[38;5;15m"  // Default White
		prevBg := "\033[44m"       // Default Blue
		prevFg := "\033[38;5;15m"  // Default White
		if colors != nil {
			listBg = GetColorCode(colors.FileListBg, listBg)
			listFg = GetFGColorCode(colors.FileListFg, listFg)
			prevBg = GetColorCode(colors.PreviewPaneBg, prevBg)
			prevFg = GetFGColorCode(colors.PreviewPaneFg, prevFg)
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
		fmt.Printf("%s%s%-*s%s", listBg, listFg, listWidth, listHeader, reset)
		// Vertical separator in header row
		fmt.Printf(" │ ")
		// Draw Preview Header (filled to remaining width)
		fmt.Printf("%s%s%-*s%s", prevBg, prevFg, width-listWidth-3, prevHeader, reset)

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
					} else if entry.Name == ".nocti.json" || entry.Name == "nocti.json" {
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
					default:
						if isProjectRoot && entry.RelPath == "." {
							icon = iconProject
						}
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
					switch entry.ResourceType {
					case "notebook":
						resFg = "\033[36m" // Default Cyan
						if colors != nil {
							resFg = GetFGColorCode(colors.NotebookFg, resFg)
							resBg = GetColorCode(colors.NotebookBg, resBg)
						}
					case "calendar":
						resFg = "\033[35m" // Default Magenta
						if colors != nil {
							resFg = GetFGColorCode(colors.CalendarFg, resFg)
							resBg = GetColorCode(colors.CalendarBg, resBg)
						}
					case "todo":
						resFg = "\033[32m" // Default Green
						if colors != nil {
							resFg = GetFGColorCode(colors.TodoFg, resFg)
							resBg = GetColorCode(colors.TodoBg, resBg)
						}
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
			if selected.IsFile {
				allPreviewLines = GetFilePreview(filepath.Join(baseDir, selected.RelPath), previewWidth)
			} else if isProjectRoot {
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
					// Check if it's just a regular directory
					if info, err := os.Stat(filepath.Join(baseDir, selected.RelPath)); err == nil && info.IsDir() {
						allPreviewLines = []string{"Directory: " + selected.RelPath}
					} else {
						allPreviewLines = []string{"Error reading resource config"}
					}
				}
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
				"    q / ESC    : Back to Parent / Quit",
				"",
				"  Actions:",
				"    n          : New File/Folder",
				"    Ctrl+T     : Toggle Settings/Templates",
				"    ENTER      : Edit / Enter Notebook",
				"    Ctrl+H     : Show Help",
			}

			for i, line := range helpLines {
				fmt.Printf("\033[%d;%dH%s%s%s", startY+3+i, startX+2, hBg, hFg, line)
			}
			fmt.Print(reset)
		}

		// 6. Create Type Modal
		if showCreateType {
			modalWidth := 40
			modalHeight := 11
			if isProjectRoot {
				modalHeight = 9
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

			fmt.Printf("\033[%d;%dH%s%s CREATE NEW %s", startY+1, startX+(modalWidth-12)/2, hBg, hFg+boldOn, reset)

			options := []string{" File ", " Folder ", " Notebook ", " Calendar ", " Todo "}
			displayIdx := 0
			for i, opt := range options {
				if isProjectRoot && i < 2 {
					continue
				}
				fmt.Printf("\033[%d;%dH", startY+3+displayIdx, startX+(modalWidth-len(opt))/2)
				if i == createTypeSelected {
					fmt.Printf("%s%s%s", hBg+hFg+reverseOn, opt, reset+hBg+hFg)
				} else {
					fmt.Printf("%s%s%s", hBg, hFg+opt, reset)
				}
				displayIdx++
			}
			fmt.Printf("\033[%d;%dH%s%s ↑/↓ to select | ENTER to confirm %s", startY+modalHeight-2, startX+(modalWidth-32)/2, hBg, hFg, reset)
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
			switch createTypeSelected {
			case 1:
				typeStr = "FOLDER"
			case 2:
				typeStr = "NOTEBOOK"
			case 3:
				typeStr = "CALENDAR"
			case 4:
				typeStr = "TODO"
			}
			fmt.Printf("\033[%d;%dH%s%s NEW %s NAME %s", startY+1, startX+(modalWidth-len(typeStr)-10)/2, hBg, hFg+boldOn, typeStr, reset)

			fmt.Printf("\033[%d;%dH%s%s > %s%s%s", startY+3, startX+4, hBg, hFg, reverseOn, createInputName, reset+hBg+hFg)
			fmt.Printf("\033[%d;%dH%s%s ENTER to create | ESC to cancel %s", startY+6, startX+(modalWidth-28)/2, hBg, hFg, reset)
		}

		// 8. Status bar
		// RULE: The status bar should NEVER have anything besides Ctrl+H for help.
		// All other navigation (like back) or actions should be documented in the help modal.
		fmt.Printf("\033[%d;1H%s%s Ctrl+H - help %s", height, reset, reverseOn, reverseOff)

		// Input handling
		b := make([]byte, 8)
		n, err := os.Stdin.Read(b)
		if err != nil {
			return err
		}

		if n == 1 {
			// Priority 1: Text Input Modal (Naming)
			if showCreateName {
				if b[0] == '\r' || b[0] == '\n' {
					// Perform creation
					if createInputName != "" {
						targetDir := baseDir
						if len(entries) > 0 {
							selected := entries[selectedIndex]
							if selected.IsFile || selected.ResourceType == "calendar" || selected.ResourceType == "todo" {
								targetDir = filepath.Join(baseDir, filepath.Dir(selected.RelPath))
							} else {
								targetDir = filepath.Join(baseDir, selected.RelPath)
							}
						}

						newName := createInputName
						switch createTypeSelected {
						case 0: // File
							ext := filepath.Ext(newName)
							if ext != ".md" && ext != ".txt" {
								newName += ".md"
							}
							os.WriteFile(filepath.Join(targetDir, newName), []byte(""), 0644)
						case 1: // Folder
							os.MkdirAll(filepath.Join(targetDir, newName), 0755)
						case 2, 3, 4: // Notebook, Calendar, Todo
							resType := "notebook"
							if createTypeSelected == 3 {
								resType = "calendar"
							} else if createTypeSelected == 4 {
								resType = "todo"
							}

							// Find parent resource if targetDir is inside one
							var parentID, parentName string
							pConfig, err := FindEnclosingResourceIn(targetDir)
							if err == nil {
								parentID = pConfig.ID
								parentName = pConfig.Name
							}

							CreateResource(resType, targetDir, newName, parentID, parentName)
						}

						// Refresh
						var files []string
						if isProjectRoot {
							files, _ = ScanProjectResources(baseDir, showHidden)
						} else {
							files, _ = ScanNotebookFiles(baseDir, showHidden)
						}
						entries = BuildDisplayEntries(files, baseDir, true)
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
				} else if b[0] == 27 { // ESC
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
				continue // Ignore other keys while naming
			}

			// Priority 2: Selection Modals (Help, Type Selection)
			if showHelp {
				if b[0] == 27 || b[0] == 'q' || b[0] == 'Q' || b[0] == 8 { // ESC, q, Ctrl+H
					showHelp = false
				}
				continue
			}
			if showCreateType {
				if b[0] == 27 { // ESC
					showCreateType = false
					continue
				}
				if b[0] == '\r' || b[0] == '\n' {
					showCreateType = false
					showCreateName = true
					createInputName = ""
					continue
				}
				// Arrow keys are handled in the n >= 3 block
				continue
			}

			// Priority 3: General Navigation
			if b[0] == 20 { // Ctrl+T
				showHidden = !showHidden
				var files []string
				if isProjectRoot {
					files, _ = ScanProjectResources(baseDir, showHidden)
				} else {
					files, _ = ScanNotebookFiles(baseDir, showHidden)
				}
				entries = BuildDisplayEntries(files, baseDir, true)
				if selectedIndex >= len(entries) {
					selectedIndex = len(entries) - 1
				}
				if selectedIndex < 0 {
					selectedIndex = 0
				}
				continue
			}
			if b[0] == 'q' || b[0] == 'Q' || b[0] == 3 {
				if len(navStack) > 0 {
					// Pop from stack
					last := navStack[len(navStack)-1]
					navStack = navStack[:len(navStack)-1]

					baseDir = last.dir
					entries = last.entries
					isProjectRoot = last.isProjectRoot
					colors = last.colors
					editorCmd = last.editorCmd
					showHidden = false // Reset hidden toggle when going back
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
				if len(navStack) > 0 {
					// Pop from stack
					last := navStack[len(navStack)-1]
					navStack = navStack[:len(navStack)-1]

					baseDir = last.dir
					entries = last.entries
					isProjectRoot = last.isProjectRoot
					colors = last.colors
					editorCmd = last.editorCmd
					showHidden = false // Reset hidden toggle when going back
					selectedIndex = 0
					previewOffset = 0
					focusList = true
					continue
				}
				break
			}

			if b[0] == 'n' || b[0] == 'N' {
				showCreateType = true
				if isProjectRoot {
					createTypeSelected = 2 // Start at Notebook
				} else {
					createTypeSelected = 0 // Start at File
				}
				continue
			}

			if b[0] == '\t' {
				focusList = !focusList
			}
			if b[0] == '\r' || b[0] == '\n' {
				if len(entries) > 0 {
					selected := entries[selectedIndex]
					if !selected.IsFile && selected.ResourceType == "notebook" {
						// Push current state to stack
						navStack = append(navStack, navState{
							dir:           baseDir,
							entries:       entries,
							isProjectRoot: isProjectRoot,
							colors:        colors,
							editorCmd:     editorCmd,
						})

						// Jump to notebook
						newBaseDir := filepath.Join(baseDir, selected.RelPath)
						showHidden = false // Reset hidden toggle when switching notebooks
						newFiles, _ := ScanNotebookFiles(newBaseDir, showHidden)
						newEntries := BuildDisplayEntries(newFiles, newBaseDir, true)
						// Refresh colors and editor for the new notebook
						colors, editorCmd = loadColorsAndEditor(newBaseDir)

						entries = newEntries
						baseDir = newBaseDir
						isProjectRoot = false
						selectedIndex = 0
						previewOffset = 0
						focusList = true
						continue
					} else if selected.IsFile {
						term.Restore(int(os.Stdin.Fd()), oldState)
						fmt.Print(showCursor + exitAltScreen)

						filePath := filepath.Join(baseDir, selected.RelPath)
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
			}
		} else if n >= 3 && b[0] == 27 && b[1] == 91 {
			if showCreateType {
				switch b[2] {
				case 65: // Up
					minIdx := 0
					if isProjectRoot {
						minIdx = 2
					}
					if createTypeSelected > minIdx {
						createTypeSelected--
					}
				case 66: // Down
					if createTypeSelected < 4 {
						createTypeSelected++
					}
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

func init() {
	ListCmd.Flags().BoolVarP(&RawOutput, "raw", "r", false, "Standard output the list of files")
	RootCmd.AddCommand(ListCmd)
}
