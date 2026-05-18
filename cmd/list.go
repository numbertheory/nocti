package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

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
	DoneTasks    int
	TotalTasks   int
}

type PreviewLine struct {
	Text   string
	LineNo int // 0 for continuation
}

func BuildDisplayEntries(files []string, baseDir string, includeRoot bool, skipSort bool, parentResType string) []DisplayEntry {
	var entries []DisplayEntry
	seenDirs := make(map[string]bool)

	// Sort files to ensure parents are processed before children, unless skipped
	if !skipSort {
		sort.Strings(files)
	}

	getResourceInfo := func(path string) (string, string) {
		physPath := GetPhysicalPath(path, baseDir, parentResType)

		configPath := filepath.Join(baseDir, physPath, ".nocti.json")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return "", ""
		}
		var config struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return "", ""
		}
		return config.Type, config.Name
	}

	addRootEntry := func() {
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

	depthOffset := 0
	if includeRoot {
		depthOffset = 1
	}

	rootAdded := false
	if includeRoot {
		addRootEntry()
		rootAdded = true
	}

	for _, f := range files {
		isDirOnly := strings.HasSuffix(f, string(os.PathSeparator))
		cleanPath := strings.TrimSuffix(f, string(os.PathSeparator))
		parts := strings.Split(cleanPath, string(os.PathSeparator))

		// Process parent directories
		for i := 0; i < len(parts)-1; i++ {
			dirPath := filepath.Join(parts[:i+1]...)
			if !seenDirs[dirPath] {
				depth := i + depthOffset
				resType, resName := getResourceInfo(dirPath)
				if (parentResType == "calendar" && resType != "" && resType != "event" && i == 0) ||
					(parentResType == "todo" && resType != "" && i == 0) {
					depth = 0
				}

				name := parts[i]
				if resName != "" {
					name = resName
				}

				entries = append(entries, DisplayEntry{
					RelPath:      dirPath,
					IsFile:       false,
					Depth:        depth,
					Name:         name,
					ResourceType: resType,
				})
				seenDirs[dirPath] = true
			}
		}

		if isDirOnly {
			// Process the empty folder itself
			dirPath := cleanPath
			if !seenDirs[dirPath] {
				depth := len(parts) - 1 + depthOffset
				resType, resName := getResourceInfo(dirPath)
				if (parentResType == "calendar" && resType != "" && resType != "event" && len(parts) == 1) ||
					(parentResType == "todo" && resType != "" && len(parts) == 1) {
					depth = 0
				}

				name := parts[len(parts)-1]
				if resName != "" {
					name = resName
				}

				entries = append(entries, DisplayEntry{
					RelPath:      dirPath,
					IsFile:       false,
					Depth:        depth,
					Name:         name,
					ResourceType: resType,
				})
				seenDirs[dirPath] = true
			}
		} else {
			// Process the file itself
			depth := len(parts) - 1 + depthOffset
			if parentResType == "calendar" && len(parts) == 1 && func() bool { t, _ := getResourceInfo(cleanPath); return t != "" }() {
				// This shouldn't really happen for files in a calendar but for robustness:
				depth = 0
			}

			done, total := 0, 0
			if parentResType == "todo" && strings.HasSuffix(strings.ToLower(f), ".md") {
				physPath := GetPhysicalPath(f, baseDir, parentResType)
				done, total = GetTaskStatus(filepath.Join(baseDir, physPath))
			}

			entries = append(entries, DisplayEntry{
				RelPath:    f,
				IsFile:     true,
				Depth:      depth,
				Name:       parts[len(parts)-1],
				DoneTasks:  done,
				TotalTasks: total,
			})
		}
	}

	// If root was never added (e.g. no days, only resources first), add it at the end
	if includeRoot && !rootAdded {
		addRootEntry()
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

			if config.Type != "notebook" && config.Type != "calendar" && config.Type != "todo" {
				return fmt.Errorf("target '%s' is a %s, but 'list' only works on notebooks, calendars, and todos", target, config.Type)
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
				// Detect if we are inside a nocti resource
				_, resourceType, err := FindEnclosingResource()
				if err != nil {
					return fmt.Errorf("not inside a nocti resource and no resource name provided: %w", err)
				}

				if resourceType != "notebook" && resourceType != "calendar" && resourceType != "todo" {
					return fmt.Errorf("the 'list' command is only available inside a notebook, calendar, or todo resource (current type: %s)", resourceType)
				}
				searchDir = "."
			}
		}

		// Get the specific resource type if not project root
		currentResType := ""
		if !isProjectRoot {
			config, err := FindEnclosingResourceIn(searchDir)
			if err == nil {
				currentResType = config.Type
			}
		}

		var files []string
		var err error
		if isProjectRoot {
			files, err = ScanProjectResources(searchDir, false)
		} else if currentResType == "calendar" {
			files, err = ScanCalendarDays(searchDir, false)
		} else if currentResType == "todo" {
			files, err = ScanTodoItems(searchDir, false)
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

		entries := BuildDisplayEntries(files, searchDir, true, currentResType == "calendar" || currentResType == "todo", currentResType)
		return runInteractiveList(entries, searchDir, colors, editorCmd, isProjectRoot, currentResType)
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

func FindEnclosingResourceRoot(startDir string) (string, *ResourceConfig, error) {
	wd, err := filepath.Abs(startDir)
	if err != nil {
		wd = startDir
	}
	for {
		configPath := filepath.Join(wd, ".nocti.json")
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return "", nil, err
			}

			var config ResourceConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return "", nil, err
			}

			return wd, &config, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", nil, fmt.Errorf(".nocti.json not found in parents")
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

func ProcessHighlights(text string) string {
	// Matches [:color1:color2: text] or [:color1: text]
	re := regexp.MustCompile(`\[:([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\s*(.*?)\]`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		var fgCode, bgCode, content string

		if parts[2] != "" {
			// Format: [:fg:bg: content]
			fgCode = GetFGColorCode(parts[1], "")
			bgCode = GetColorCode(parts[2], "")
			content = parts[3]
		} else {
			// Format: [:bg: content]
			bgCode = GetColorCode(parts[1], "")
			content = parts[3]
		}

		if fgCode == "" && bgCode == "" {
			return match
		}

		res := ""
		if fgCode != "" {
			res += fgCode
		}
		if bgCode != "" {
			res += bgCode
		}
		res += content

		// Reset both FG and BG
		if fgCode != "" {
			res += "\033[39m"
		}
		if bgCode != "" {
			res += "\033[49m"
		}
		return res
	})
}

func runInteractiveList(entries []DisplayEntry, baseDir string, colors *ColorsConfig, editorCmd string, isProjectRoot bool, currentResType string) error {
	type navState struct {
		dir             string
		entries         []DisplayEntry
		isProjectRoot   bool
		currentResType  string
		colors          *ColorsConfig
		editorCmd       string
		showLineNumbers bool
	}
	var navStack []navState

	showHidden := false
	showLineNumbers := false

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
	listOffset := 0
	previewOffset := 0
	selectedLinkIdx := -1
	focusList := true // true = List, false = Preview
	showHelp := false
	showSettings := false
	var settingsState *SettingsState

	// Creation states
	type createOption struct {
		Label    string
		Type     string // "file", "folder", "notebook", "calendar", "todo", "event", "template"
		Metadata string // template filename
	}
	var currentCreateOptions []createOption

	showCreateType := false
	showCreateName := false
	showCreateDays := false
	createTypeSelected := 0
	createInputName := ""
	createInputDays := ""

	showOverwriteConfirm := false
	showFolderExistsError := false
	showEditorError := false
	lastEditorError := ""
	pendingCreation := func() {}

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
		iconDay      = " "
	)

	fmt.Print(enterAltScreen + hideCursor)
	defer fmt.Print(showCursor + exitAltScreen)

	for {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return err
		}

		if showSettings {
			DrawSettingsPanel(width, height, settingsState)
			b := make([]byte, 8)
			n, _ := os.Stdin.Read(b)
			if n > 0 {
				handled, shouldRefresh := settingsState.HandleInput(b, n)
				if !handled {
					showSettings = false
					if shouldRefresh {
						colors, editorCmd = loadColorsAndEditor(baseDir)
					}
				}
			}
			continue
		}

		// Determine effective resource type for creation options based on selection
		effectiveResType := currentResType
		if isProjectRoot {
			effectiveResType = "project"
		}
		if len(entries) > 0 {
			selected := entries[selectedIndex]
			if isProjectRoot && selected.RelPath == "." {
				effectiveResType = "project"
			} else if !selected.IsFile && selected.ResourceType != "" {
				effectiveResType = selected.ResourceType
			}
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
		} else if currentResType == "calendar" {
			listHeader = " DAYS "
		}

		prevHeader := " PREVIEW "
		if currentResType == "calendar" {
			prevHeader = " EVENTS "
		}

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

			entryIdx := i + listOffset
			if entryIdx < len(entries) {
				entry := entries[entryIdx]
				indent := strings.Repeat("  ", entry.Depth)
				icon := iconFolder
				displayName := entry.Name

				if entry.IsFile {
					if currentResType == "calendar" && entry.Name != ".nocti.json" && entry.Name != "nocti.json" {
						icon = " "
						displayName = strings.TrimSuffix(displayName, ".md")
					} else if currentResType == "todo" && entry.Name != ".nocti.json" && entry.Name != "nocti.json" {
						// There should always be a space between the icon and the name of the todo list
						// nf-md-checkbox_blank_outline for incomplete tasks
						icon = "  "
						displayName = strings.TrimSuffix(displayName, ".md")

						if entry.TotalTasks > 0 {
							if entry.DoneTasks == entry.TotalTasks {
								// nf-md-check_bold for completed tasks
								icon = " 󰸞 "
							} else if entry.DoneTasks > 0 {
								displayName = fmt.Sprintf("%s (%d/%d)", displayName, entry.DoneTasks, entry.TotalTasks)
							}
						}
					} else if strings.HasSuffix(strings.ToLower(entry.Name), ".md") {
						icon = iconMarkdown
					} else if entry.Name == ".nocti.json" || entry.Name == "nocti.json" {
						icon = " " // Gear icon for settings
					} else {
						if currentResType == "calendar" {
							icon = iconDay
						} else {
							icon = iconText
						}
					}
				} else {

					switch entry.ResourceType {
					case "notebook":
						icon = iconNotebook
					case "calendar":
						icon = iconCalendar
					case "todo":
						icon = iconTodo
					case "event":
						icon = iconDay
					default:
						if isProjectRoot && entry.RelPath == "." {
							icon = iconProject
						} else if currentResType == "calendar" {
							// Check if it's a virtual day folder
							if _, err := GetDateFromRelPath(entry.RelPath, baseDir); err == nil {
								icon = iconDay
							}
						}
					}
				}
				displayStr := fmt.Sprintf("%s%s%s", indent, icon, displayName)
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
				} else if currentResType == "calendar" && entry.Name != ".nocti.json" && entry.Name != "nocti.json" {
					if IsHoliday(entry.RelPath, baseDir) {
						resFg = "\033[38;5;214m" // Gold
						if colors != nil {
							resFg = GetFGColorCode(colors.HolidayFg, resFg)
							resBg = GetColorCode(colors.HolidayBg, resBg)
						}
					}
				}

				if entryIdx == selectedIndex {
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
		var allPreviewLines []PreviewLine
		effectivePreviewWidth := previewWidth
		if showLineNumbers {
			effectivePreviewWidth -= 6
		}

		if len(entries) > 0 {
			selected := entries[selectedIndex]
			physPath := GetPhysicalPath(selected.RelPath, baseDir, currentResType)

			if currentResType == "calendar" {
				// Check if it's a date entry (either virtual or has a physical folder)
				isDate := false
				if _, err := GetDateFromRelPath(selected.RelPath, baseDir); err == nil {
					isDate = true
				}

				if isDate {
					allPreviewLines = GetCalendarDayPreview(selected.RelPath, baseDir)
				} else if selected.IsFile {
					fullPath := filepath.Join(baseDir, physPath)
					if _, err := os.Stat(fullPath); err == nil {
						allPreviewLines = GetFilePreview(fullPath, effectivePreviewWidth)
					} else {
						allPreviewLines = []PreviewLine{{Text: "[ File not found ]"}}
					}
				} else {
					// Fallthrough for resources within calendar
					resConfigPath := filepath.Join(baseDir, physPath, ".nocti.json")
					data, err := os.ReadFile(resConfigPath)
					if err == nil {
						var config struct {
							ID   string `json:"id"`
							Name string `json:"name"`
							Type string `json:"type"`
						}
						json.Unmarshal(data, &config)

						allPreviewLines = append(allPreviewLines, PreviewLine{Text: "Resource: " + config.Name})
						allPreviewLines = append(allPreviewLines, PreviewLine{Text: "Type:     " + config.Type})
						allPreviewLines = append(allPreviewLines, PreviewLine{Text: "ID:       " + config.ID})
						allPreviewLines = append(allPreviewLines, PreviewLine{Text: ""})

						if config.Type == "notebook" {
							files, _ := ScanNotebookFiles(filepath.Join(baseDir, selected.RelPath), false)
							allPreviewLines = append(allPreviewLines, PreviewLine{Text: fmt.Sprintf("Notes:    %d", len(files))})
						} else if config.Type == "calendar" {
							// Load calendar-specific config
							var calConfig struct {
								DaysLength int    `json:"daysLength"`
								CreatedAt  string `json:"created_at"`
							}
							json.Unmarshal(data, &calConfig)

							if calConfig.DaysLength <= 0 {
								calConfig.DaysLength = 30
							}

							centerDate := time.Now()
							if calConfig.CreatedAt != "" {
								if t, err := time.Parse(time.RFC3339, calConfig.CreatedAt); err == nil {
									centerDate = t
								}
							}

							startDate := centerDate.AddDate(0, 0, -calConfig.DaysLength)
							endDate := centerDate.AddDate(0, 0, calConfig.DaysLength)

							allPreviewLines = append(allPreviewLines, PreviewLine{Text: fmt.Sprintf("Range:    %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))})
							allPreviewLines = append(allPreviewLines, PreviewLine{Text: fmt.Sprintf("Days:     %d", calConfig.DaysLength*2+1)})
						} else {
							allPreviewLines = append(allPreviewLines, PreviewLine{Text: "[ Preview not implemented ]"})
						}
					} else {
						if info, err := os.Stat(filepath.Join(baseDir, selected.RelPath)); err == nil && info.IsDir() {
							allPreviewLines = []PreviewLine{{Text: "Directory: " + selected.RelPath}}
						} else {
							allPreviewLines = []PreviewLine{{Text: "Error reading resource config"}}
						}
					}
				}
			} else if selected.IsFile {
				fullPath := filepath.Join(baseDir, physPath)
				if _, err := os.Stat(fullPath); err == nil {
					allPreviewLines = GetFilePreview(fullPath, effectivePreviewWidth)
				} else {
					allPreviewLines = []PreviewLine{{Text: "[ File not found ]"}}
				}
			} else if isProjectRoot {
				resConfigPath := filepath.Join(baseDir, physPath, ".nocti.json")
				data, err := os.ReadFile(resConfigPath)
				if err == nil {
					var config struct {
						ID   string `json:"id"`
						Name string `json:"name"`
						Type string `json:"type"`
					}
					json.Unmarshal(data, &config)

					allPreviewLines = append(allPreviewLines, PreviewLine{Text: "Resource: " + config.Name})
					allPreviewLines = append(allPreviewLines, PreviewLine{Text: "Type:     " + config.Type})
					allPreviewLines = append(allPreviewLines, PreviewLine{Text: "ID:       " + config.ID})
					allPreviewLines = append(allPreviewLines, PreviewLine{Text: ""})

					if config.Type == "notebook" {
						files, _ := ScanNotebookFiles(filepath.Join(baseDir, selected.RelPath), false)
						allPreviewLines = append(allPreviewLines, PreviewLine{Text: fmt.Sprintf("Notes:    %d", len(files))})

					} else if config.Type == "calendar" {
						// Load calendar-specific config
						var calConfig struct {
							DaysLength int    `json:"daysLength"`
							CreatedAt  string `json:"created_at"`
						}
						json.Unmarshal(data, &calConfig)

						if calConfig.DaysLength <= 0 {
							calConfig.DaysLength = 30
						}

						centerDate := time.Now()
						if calConfig.CreatedAt != "" {
							if t, err := time.Parse(time.RFC3339, calConfig.CreatedAt); err == nil {
								centerDate = t
							}
						}

						startDate := centerDate.AddDate(0, 0, -calConfig.DaysLength)
						endDate := centerDate.AddDate(0, 0, calConfig.DaysLength)

						allPreviewLines = append(allPreviewLines, PreviewLine{Text: fmt.Sprintf("Range:    %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))})
						allPreviewLines = append(allPreviewLines, PreviewLine{Text: fmt.Sprintf("Days:     %d", calConfig.DaysLength*2+1)})
					} else {
						allPreviewLines = append(allPreviewLines, PreviewLine{Text: "[ Not yet implemented ]"})
					}
				} else {
					// Check if it's just a regular directory
					if info, err := os.Stat(filepath.Join(baseDir, selected.RelPath)); err == nil && info.IsDir() {
						allPreviewLines = []PreviewLine{{Text: "Directory: " + selected.RelPath}}
					} else {
						allPreviewLines = []PreviewLine{{Text: "Error reading resource config"}}
					}
				}
			} else {
				allPreviewLines = []PreviewLine{{Text: "Directory: " + selected.RelPath}}
			}
		} else {
			if isProjectRoot {
				allPreviewLines = []PreviewLine{{Text: "No resources found in project root."}}
			} else {
				allPreviewLines = []PreviewLine{{Text: "No markdown or text files found."}, {Text: ""}, {Text: "Press 'n' to create a new file."}}
			}
		}

		// Link detection for preview pane focus
		type previewLink struct {
			link   Link
			lineNo int
		}
		var allLinks []previewLink

		// Pre-process all lines to detect links and calculate their display length
		// This is important because Markdown links change the line length.
		type processedPreviewLine struct {
			display string
			links   []Link
			lineNo  int
		}
		var processedLines []processedPreviewLine

		globalLinkCount := 0
		for i, pLine := range allPreviewLines {
			lineLinks := DetectLinks(pLine.Text)
			processedLines = append(processedLines, processedPreviewLine{
				display: "", // will be set during rendering for current view
				links:   lineLinks,
				lineNo:  pLine.LineNo,
			})
			for _, l := range lineLinks {
				allLinks = append(allLinks, previewLink{l, i})
				globalLinkCount++
			}
		}

		if selectedLinkIdx >= len(allLinks) {
			selectedLinkIdx = len(allLinks) - 1
		}
		if selectedLinkIdx < -1 {
			selectedLinkIdx = -1
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

		// If a link is selected, ensure it's visible
		if selectedLinkIdx >= 0 && selectedLinkIdx < len(allLinks) {
			targetLine := allLinks[selectedLinkIdx].lineNo
			if targetLine < previewOffset {
				previewOffset = targetLine
			} else if targetLine >= previewOffset+contentHeight {
				previewOffset = targetLine - contentHeight + 1
			}
		}

		for j := 0; j < contentHeight && (j+previewOffset) < len(allPreviewLines); j++ {
			pIdx := j + previewOffset
			pLine := allPreviewLines[pIdx]

			globalLinkStartIdx := 0
			for k := 0; k < pIdx; k++ {
				globalLinkStartIdx += len(processedLines[k].links)
			}

			rendered := PrepareLineForDisplay(pLine.Text, !focusList, selectedLinkIdx, globalLinkStartIdx)

			if showLineNumbers {
				lineNoStr := ""
				if pLine.LineNo > 0 {
					lineNoStr = fmt.Sprintf("%3d │ ", pLine.LineNo)
				} else {
					lineNoStr = "    │ "
				}
				fmt.Printf("\033[%d;%dH%s%s", j+headerHeight+1, listWidth+5, lineNoStr, rendered.Display)
			} else {
				fmt.Printf("\033[%d;%dH%s", j+headerHeight+1, listWidth+5, rendered.Display)
			}
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
			modalHeight := 20 // Increased for equal top/bottom buffer
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
				"    Ctrl+↑ / ↓ : Jump 7 Days",
				"    Home / End : Start / End of List",
				"    TAB        : Switch Focus",
				"    PgUp/PgDn  : Page Preview",
				"    q / ESC    : Back to Parent / Quit",
				"",
				"  Actions:",
				"    n          : New File/Folder",
				"    Ctrl+T     : Toggle Settings/Templates",
				"    Ctrl+L     : Toggle Line Numbers",
				"    ENTER      : Edit / Enter Notebook",
				"    Ctrl+S     : Settings (Save/Discard)",
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
			modalHeight := 6 + len(currentCreateOptions)
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

			for i, opt := range currentCreateOptions {
				fmt.Printf("\033[%d;%dH", startY+3+i, startX+(modalWidth-len(opt.Label))/2)
				if i == createTypeSelected {
					fmt.Printf("%s%s%s", hBg+hFg+reverseOn, opt.Label, reset+hBg+hFg)
				} else {
					fmt.Printf("%s%s%s", hBg, hFg+opt.Label, reset)
				}
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
			if createTypeSelected < len(currentCreateOptions) {
				opt := currentCreateOptions[createTypeSelected]
				switch opt.Type {
				case "file", "template":
					if effectiveResType == "todo" {
						typeStr = "TODO LIST"
					} else {
						typeStr = "FILE"
					}
				case "folder":
					typeStr = "FOLDER"
				case "notebook":
					typeStr = "NOTEBOOK"
				case "calendar":
					typeStr = "CALENDAR"
				case "todo":
					typeStr = "TODO"
				case "event":
					typeStr = "EVENT"
				}
			}
			fmt.Printf("\033[%d;%dH%s%s NEW %s NAME %s", startY+1, startX+(modalWidth-len(typeStr)-10)/2, hBg, hFg+boldOn, typeStr, reset)

			fmt.Printf("\033[%d;%dH%s%s > %s%s%s", startY+3, startX+4, hBg, hFg, reverseOn, createInputName, reset+hBg+hFg)
			fmt.Printf("\033[%d;%dH%s%s ENTER to confirm | ESC to cancel %s", startY+6, startX+(modalWidth-28)/2, hBg, hFg, reset)
		}

		// 7.2 Overwrite Confirmation Modal
		if showOverwriteConfirm {
			modalWidth := 60
			modalHeight := 8
			startX := (width - modalWidth) / 2
			startY := (height - modalHeight) / 2

			hBg := "\033[41m"      // Red background for warning
			hFg := "\033[38;5;15m" // White foreground
			hbBg := "\033[41m"
			hbFg := "\033[38;5;15m"

			if colors != nil {
				hBg = GetColorCode(colors.HelpBg, hBg)
				hFg = GetFGColorCode(colors.HelpFg, hFg)
			}

			for i := 0; i < modalHeight; i++ {
				fmt.Printf("\033[%d;%dH%s%s%*s%s", startY+i, startX, hBg, hFg, modalWidth, "", reset)
			}
			fmt.Printf("\033[%d;%dH%s%s┌%s┐%s", startY, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)
			for i := 1; i < modalHeight-1; i++ {
				fmt.Printf("\033[%d;%dH%s%s│\033[%d;%dH%s%s│%s", startY+i, startX, hbBg, hbFg, startY+i, startX+modalWidth-1, hbBg, hbFg, reset)
			}
			fmt.Printf("\033[%d;%dH%s%s└%s┘%s", startY+modalHeight-1, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)

			msg := "FILE ALREADY EXISTS. OVERWRITE?"
			fmt.Printf("\033[%d;%dH%s%s%s", startY+2, startX+(modalWidth-len(msg))/2, hBg, hFg+boldOn, msg)
			fmt.Printf("\033[%d;%dH%s%s%s", startY+3, startX+(modalWidth-len(createInputName))/2, hBg, hFg, createInputName)
			fmt.Printf("\033[%d;%dH%s%s ENTER to overwrite | ESC to cancel %s", startY+5, startX+(modalWidth-34)/2, hBg, hFg, reset)
		}

		// 7.3 Folder Exists Modal
		if showFolderExistsError {
			modalWidth := 60
			modalHeight := 8
			startX := (width - modalWidth) / 2
			startY := (height - modalHeight) / 2

			hBg := "\033[41m" // Red background
			hFg := "\033[38;5;15m"
			hbBg := "\033[41m"
			hbFg := "\033[38;5;15m"

			for i := 0; i < modalHeight; i++ {
				fmt.Printf("\033[%d;%dH%s%s%*s%s", startY+i, startX, hBg, hFg, modalWidth, "", reset)
			}
			fmt.Printf("\033[%d;%dH%s%s┌%s┐%s", startY, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)
			for i := 1; i < modalHeight-1; i++ {
				fmt.Printf("\033[%d;%dH%s%s│\033[%d;%dH%s%s│%s", startY+i, startX, hbBg, hbFg, startY+i, startX+modalWidth-1, hbBg, hbFg, reset)
			}
			fmt.Printf("\033[%d;%dH%s%s└%s┘%s", startY+modalHeight-1, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)

			msg := fmt.Sprintf("FOLDER '%s' ALREADY EXISTS", createInputName)
			fmt.Printf("\033[%d;%dH%s%s%s", startY+3, startX+(modalWidth-len(msg))/2, hBg, hFg+boldOn, msg)
			fmt.Printf("\033[%d;%dH%s%s ESC / ENTER to dismiss %s", startY+5, startX+(modalWidth-24)/2, hBg, hFg, reset)
		}

		if showEditorError {
			modalWidth := 70
			modalHeight := 8
			startX := (width - modalWidth) / 2
			startY := (height - modalHeight) / 2

			hBg := "\033[41m" // Red background
			hFg := "\033[38;5;15m"
			hbBg := "\033[41m"
			hbFg := "\033[38;5;15m"

			for i := 0; i < modalHeight; i++ {
				fmt.Printf("\033[%d;%dH%s%s%*s%s", startY+i, startX, hBg, hFg, modalWidth, "", reset)
			}
			fmt.Printf("\033[%d;%dH%s%s┌%s┐%s", startY, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)
			for i := 1; i < modalHeight-1; i++ {
				fmt.Printf("\033[%d;%dH%s%s│\033[%d;%dH%s%s│%s", startY+i, startX, hbBg, hbFg, startY+i, startX+modalWidth-1, hbBg, hbFg, reset)
			}
			fmt.Printf("\033[%d;%dH%s%s└%s┘%s", startY+modalHeight-1, startX, hbBg, hbFg, strings.Repeat("─", modalWidth-2), reset)

			header := "EDITOR ERROR"
			fmt.Printf("\033[%d;%dH%s%s%s", startY+2, startX+(modalWidth-len(header))/2, hBg, hFg+boldOn, header)

			msg := lastEditorError
			if len(msg) > modalWidth-4 {
				msg = msg[:modalWidth-7] + "..."
			}
			fmt.Printf("\033[%d;%dH%s%s%s", startY+4, startX+(modalWidth-len(msg))/2, hBg, hFg, msg)
			fmt.Printf("\033[%d;%dH%s%s ESC / ENTER to dismiss %s", startY+6, startX+(modalWidth-24)/2, hBg, hFg, reset)
		}

		// 7.5 Create Days Modal (Calendar specific)
		if showCreateDays {
			modalWidth := 50
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

			fmt.Printf("\033[%d;%dH%s%s CALENDAR LENGTH (DAYS) %s", startY+1, startX+(modalWidth-24)/2, hBg, hFg+boldOn, reset)

			fmt.Printf("\033[%d;%dH%s%s > %s%s%s", startY+3, startX+4, hBg, hFg, reverseOn, createInputDays, reset+hBg+hFg)
			fmt.Printf("\033[%d;%dH%s%s ENTER to create (default 30) | ESC to cancel %s", startY+6, startX+(modalWidth-42)/2, hBg, hFg, reset)
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
			// Priority 1: Text Input Modals (Naming, Days)
			if showCreateName {
				if b[0] == '\r' || b[0] == '\n' {
					if createInputName != "" {
						// Prepare the creation logic
						targetDir := baseDir
						if len(entries) > 0 {
							selected := entries[selectedIndex]
							physPath := GetPhysicalPath(selected.RelPath, baseDir, currentResType)

							if selected.IsFile {
								targetDir = filepath.Join(baseDir, filepath.Dir(physPath))
							} else {
								targetDir = filepath.Join(baseDir, physPath)
							}

							// Special case for calendar: if it's a date-based virtual path,
							// ensure the physical day folder exists and is initialized.
							if currentResType == "calendar" {
								dayRelPath := selected.RelPath
								parts := strings.Split(dayRelPath, string(os.PathSeparator))
								if len(parts) > 0 {
									if len(parts[0]) == 4 && parts[0][0] >= '0' && parts[0][0] <= '9' {
										if len(parts) >= 2 && strings.Contains(parts[1], " ") {
											dayRelPath = filepath.Join(parts[0], parts[1])
										}
									} else if strings.Contains(parts[0], " ") {
										dayRelPath = parts[0]
									}
								}

								if t, err := GetDateFromRelPath(dayRelPath, baseDir); err == nil {
									dateFolder := t.Format("2006-01-02")
									dayDir := filepath.Join(baseDir, dateFolder)

									// Ensure date folder exists and has .nocti.json
									os.MkdirAll(dayDir, 0755)
									dfConfigPath := filepath.Join(dayDir, ".nocti.json")
									if _, err := os.Stat(dfConfigPath); os.IsNotExist(err) {
										dfConfig := map[string]string{
											"created_at": time.Now().Format(time.RFC3339),
											"type":       "event",
											"name":       filepath.Base(dayRelPath),
										}
										data, _ := json.MarshalIndent(dfConfig, "", "  ")
										os.WriteFile(dfConfigPath, data, 0644)
									}
								}
							}
						}

						if createTypeSelected >= len(currentCreateOptions) {
							showCreateName = false
							continue
						}
						selectedOpt := currentCreateOptions[createTypeSelected]

						newName := createInputName
						fullPath := ""
						switch selectedOpt.Type {
						case "file", "event", "template":
							ext := filepath.Ext(newName)
							if selectedOpt.Type == "file" || selectedOpt.Type == "template" {
								if ext != ".md" && ext != ".txt" {
									newName += ".md"
								}
							} else { // Event
								if ext != ".md" {
									newName += ".md"
								}
							}
							fullPath = filepath.Join(targetDir, newName)
						case "folder":
							fullPath = filepath.Join(targetDir, newName)
						case "notebook", "calendar", "todo":
							fullPath = filepath.Join(targetDir, newName)
						}

						var performCreation func()

						goToDaysModal := func() {
							showCreateName = false
							showCreateDays = true
							createInputDays = "30"
							pendingCreation = performCreation
						}

						performCreation = func() {
							if selectedOpt.Type == "calendar" && !showCreateDays {
								goToDaysModal()
								return
							}

							days := 0
							if selectedOpt.Type == "calendar" {
								fmt.Sscanf(createInputDays, "%d", &days)
								if days <= 0 {
									days = 30
								}
							}

							switch selectedOpt.Type {
							case "file", "template":
								os.MkdirAll(targetDir, 0755)

								content := ""
								metadata := make(map[string]string)
								if selectedOpt.Type == "template" {
									resRoot, _, _ := FindEnclosingResourceRoot(baseDir)
									templatePath := filepath.Join(resRoot, ".templates", selectedOpt.Metadata)
									if data, err := os.ReadFile(templatePath); err == nil {
										metadata, content = ParseTemplateFrontmatter(string(data))
									}
								} else if effectiveResType == "todo" {
									// Read template
									templatePath := "templates/todo_template.md"
									// Try to find it relative to project root if not in CWD
									if root, err := FindProjectRoot(); err == nil {
										templatePath = filepath.Join(root, "templates", "todo_template.md")
									}

									templateData, err := os.ReadFile(templatePath)
									if err == nil {
										content = string(templateData)
									} else {
										// Fallback if template file is missing
										content = fmt.Sprintf("# %s To Do List\n\n- [ ] Sample Task 1\n- [ ] Sample Task 2\n- [ ] Sample Task 3\n", strings.TrimSuffix(createInputName, filepath.Ext(createInputName)))
									}
								}

								// Replace macros in content
								macros := make(map[string]string)
								for k, v := range metadata {
									macros[k] = v
								}
								macros["NAME"] = strings.TrimSuffix(createInputName, filepath.Ext(createInputName))
								content = ReplaceMacros(content, macros)

								os.WriteFile(fullPath, []byte(content), 0644)
							case "folder":
								os.MkdirAll(fullPath, 0755)
							case "notebook", "calendar", "todo":
								os.MkdirAll(targetDir, 0755)
								var parentID, parentName string
								pConfig, err := FindEnclosingResourceIn(targetDir)
								if err == nil {
									parentID = pConfig.ID
									parentName = pConfig.Name
								}
								CreateResource(selectedOpt.Type, targetDir, newName, parentID, parentName, days)
							case "event": // Event (.md file)
								os.MkdirAll(targetDir, 0755)
								os.WriteFile(fullPath, []byte(""), 0644)
							}

							// Refresh
							var files []string
							if isProjectRoot {
								files, _ = ScanProjectResources(baseDir, showHidden)
							} else if currentResType == "calendar" {
								files, _ = ScanCalendarDays(baseDir, showHidden)
							} else if currentResType == "todo" {
								files, _ = ScanTodoItems(baseDir, showHidden)
							} else {
								files, _ = ScanNotebookFiles(baseDir, showHidden)
							}
							entries = BuildDisplayEntries(files, baseDir, true, currentResType == "calendar" || currentResType == "todo", currentResType)
							if selectedIndex >= len(entries) {
								selectedIndex = len(entries) - 1
							}
							if selectedIndex < 0 {
								selectedIndex = 0
							}
							showCreateName = false
							showCreateDays = false
						}

						// Check if path exists
						if info, err := os.Stat(fullPath); err == nil {
							if info.IsDir() {
								showFolderExistsError = true
								showCreateName = false
							} else {
								showOverwriteConfirm = true
								showCreateName = false
								pendingCreation = performCreation
							}
						} else {
							if selectedOpt.Type == "calendar" {
								goToDaysModal()
							} else {
								performCreation()
							}
						}
						continue
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
				continue
			}

			if showOverwriteConfirm {
				if b[0] == '\r' || b[0] == '\n' {
					pendingCreation()
					showOverwriteConfirm = false
					continue
				} else if b[0] == 27 { // ESC
					showOverwriteConfirm = false
					continue
				}
				continue
			}
			if showFolderExistsError {
				if b[0] == '\r' || b[0] == '\n' || b[0] == 27 {
					showFolderExistsError = false
				}
				continue
			}

			if showEditorError {
				if b[0] == '\r' || b[0] == '\n' || b[0] == 27 {
					showEditorError = false
				}
				continue
			}

			if showCreateDays {
				if b[0] == '\r' || b[0] == '\n' {
					if pendingCreation != nil {
						pendingCreation()
					}
					continue
				} else if b[0] == 27 { // ESC
					showCreateDays = false
					continue
				} else if b[0] == 127 || b[0] == 8 { // Backspace
					if len(createInputDays) > 0 {
						createInputDays = createInputDays[:len(createInputDays)-1]
					}
					continue
				} else if b[0] >= '0' && b[0] <= '9' {
					createInputDays += string(b[0])
					continue
				}
				continue
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

					// NEW: Pre-fill name from template if possible
					if createTypeSelected < len(currentCreateOptions) {
						opt := currentCreateOptions[createTypeSelected]
						if opt.Type == "template" {
							targetDir := baseDir
							if len(entries) > 0 {
								selected := entries[selectedIndex]
								physPath := GetPhysicalPath(selected.RelPath, baseDir, currentResType)
								if selected.IsFile {
									targetDir = filepath.Join(baseDir, filepath.Dir(physPath))
								} else {
									targetDir = filepath.Join(baseDir, physPath)
								}
							}

							resRoot, _, _ := FindEnclosingResourceRoot(baseDir)
							templatePath := filepath.Join(resRoot, ".templates", opt.Metadata)
							if data, err := os.ReadFile(templatePath); err == nil {
								metadata, _ := ParseTemplateFrontmatter(string(data))
								if fname, ok := metadata["filename"]; ok {
									// Support macros in filename
									fname = ReplaceMacros(fname, nil)
									// Support {{INC}}
									if strings.Contains(fname, "{{INC}}") {
										fname = ResolveIncrement(fname, targetDir)
									}
									createInputName = fname
								}
							}
						}
					}
					continue
				}
				// Arrow keys are handled in the n >= 3 block
				continue
			}

			// Priority 3: General Navigation
			if b[0] == 19 { // Ctrl+S
				showSettings = true
				configPath := filepath.Join(baseDir, ".nocti.json")
				if isProjectRoot {
					if root, err := FindProjectRoot(); err == nil {
						configPath = filepath.Join(root, ".nocti/nocti.json")
					}
				}

				sColors, sEditor := loadColorsAndEditor(baseDir)
				var colorsCopy ColorsConfig
				if sColors != nil {
					colorsCopy = *sColors
				}
				settingsState = &SettingsState{
					Tab:        TabColors,
					ConfigPath: configPath,
					Colors:     &colorsCopy,
					Editor:     sEditor,
				}
				continue
			}

			if b[0] == 20 { // Ctrl+T
				showHidden = !showHidden
				var files []string
				if isProjectRoot {
					files, _ = ScanProjectResources(baseDir, showHidden)
				} else if currentResType == "calendar" {
					files, _ = ScanCalendarDays(baseDir, showHidden)
				} else if currentResType == "todo" {
					files, _ = ScanTodoItems(baseDir, showHidden)
				} else {
					files, _ = ScanNotebookFiles(baseDir, showHidden)
				}
				entries = BuildDisplayEntries(files, baseDir, true, currentResType == "calendar" || currentResType == "todo", currentResType)
				if selectedIndex >= len(entries) {
					selectedIndex = len(entries) - 1
				}
				if selectedIndex < 0 {
					selectedIndex = 0
				}
				continue
			}
			if b[0] == 12 { // Ctrl+L
				showLineNumbers = !showLineNumbers
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
					currentResType = last.currentResType
					colors = last.colors
					editorCmd = last.editorCmd
					showLineNumbers = last.showLineNumbers
					showHidden = false // Reset hidden toggle when going back
					selectedIndex = 0
					listOffset = 0
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
					currentResType = last.currentResType
					colors = last.colors
					editorCmd = last.editorCmd
					showLineNumbers = last.showLineNumbers
					showHidden = false // Reset hidden toggle when going back
					selectedIndex = 0
					listOffset = 0
					previewOffset = 0
					focusList = true
					continue
				}
				break
			}

			if b[0] == 'n' || b[0] == 'N' {
				if len(entries) > 0 {
					selected := entries[selectedIndex]
					if selected.Name == ".nocti.json" || selected.Name == "nocti.json" {
						continue
					}
				}

				// Build options
				currentCreateOptions = []createOption{}
				if effectiveResType != "project" && effectiveResType != "calendar" {
					label := " File "
					if effectiveResType == "todo" {
						label = " Todo List "
					}
					currentCreateOptions = append(currentCreateOptions, createOption{Label: label, Type: "file"})
					if effectiveResType != "todo" {
						currentCreateOptions = append(currentCreateOptions, createOption{Label: " Folder ", Type: "folder"})
					}
				}
				currentCreateOptions = append(currentCreateOptions, createOption{Label: " Notebook ", Type: "notebook"})
				currentCreateOptions = append(currentCreateOptions, createOption{Label: " Calendar ", Type: "calendar"})
				currentCreateOptions = append(currentCreateOptions, createOption{Label: " Todo ", Type: "todo"})

				if effectiveResType == "calendar" {
					selectedIsDate := false
					if len(entries) > 0 {
						selected := entries[selectedIndex]
						if _, err := GetDateFromRelPath(selected.RelPath, baseDir); err == nil {
							selectedIsDate = true
						} else {
							parts := strings.Split(selected.RelPath, string(os.PathSeparator))
							if len(parts) > 1 {
								dayRelPath := parts[0]
								if len(parts) > 2 && !strings.Contains(parts[0], " ") && strings.Contains(parts[1], " ") {
									dayRelPath = filepath.Join(parts[0], parts[1])
								}
								if _, err := GetDateFromRelPath(dayRelPath, baseDir); err == nil {
									selectedIsDate = true
								}
							}
						}
					}
					if selectedIsDate {
						currentCreateOptions = append(currentCreateOptions, createOption{Label: " Event ", Type: "event"})
					}
				}

				// Add templates
				resRoot, _, err := FindEnclosingResourceRoot(baseDir)
				if err == nil {
					templateDir := filepath.Join(resRoot, ".templates")
					if entries, err := os.ReadDir(templateDir); err == nil {
						for _, entry := range entries {
							if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
								currentCreateOptions = append(currentCreateOptions, createOption{
									Label:    " " + entry.Name() + " ",
									Type:     "template",
									Metadata: entry.Name(),
								})
							}
						}
					}
				}

				showCreateType = true
				createTypeSelected = 0
				// If project or calendar, we might want to start at Notebook?
				if effectiveResType == "project" || effectiveResType == "calendar" {
					for i, opt := range currentCreateOptions {
						if opt.Type == "notebook" {
							createTypeSelected = i
							break
						}
					}
				}
				continue
			}

			if b[0] == '\t' {
				focusList = !focusList
				if focusList {
					selectedLinkIdx = -1
				}
			}
			if b[0] == '\r' || b[0] == '\n' {
				if !focusList && selectedLinkIdx >= 0 {
					// Open selected link
					if selectedLinkIdx < len(allLinks) {
						OpenURL(allLinks[selectedLinkIdx].link.URL)
					}
					continue
				}
				if len(entries) > 0 {
					selected := entries[selectedIndex]
					if !selected.IsFile && (selected.ResourceType == "notebook" || selected.ResourceType == "calendar" || selected.ResourceType == "todo") {
						// Push current state to stack
						navStack = append(navStack, navState{
							dir:             baseDir,
							entries:         entries,
							isProjectRoot:   isProjectRoot,
							currentResType:  currentResType,
							colors:          colors,
							editorCmd:       editorCmd,
							showLineNumbers: showLineNumbers,
						})

						// Jump to resource - ALWAYS USE PHYSICAL PATH
						physPath := GetPhysicalPath(selected.RelPath, baseDir, currentResType)
						newBaseDir := filepath.Join(baseDir, physPath)
						showHidden = false // Reset hidden toggle
						currentResType = selected.ResourceType

						var newFiles []string
						if currentResType == "calendar" {
							newFiles, _ = ScanCalendarDays(newBaseDir, showHidden)
						} else if currentResType == "todo" {
							newFiles, _ = ScanTodoItems(newBaseDir, showHidden)
						} else {
							newFiles, _ = ScanNotebookFiles(newBaseDir, showHidden)
						}
						newEntries := BuildDisplayEntries(newFiles, newBaseDir, true, currentResType == "calendar" || currentResType == "todo", currentResType)

						// Refresh colors and editor for the new resource
						colors, editorCmd = loadColorsAndEditor(newBaseDir)

						entries = newEntries
						baseDir = newBaseDir
						isProjectRoot = false
						selectedIndex = 0
						previewOffset = 0
						focusList = true
						continue
					} else if selected.IsFile {
						if currentResType == "calendar" && selected.Name != ".nocti.json" && selected.Name != "nocti.json" {
							// Check if it's a virtual day or a real file under a day
							parts := strings.Split(selected.RelPath, string(os.PathSeparator))
							if len(parts) == 1 || (currentResType == "calendar" && len(parts) == 2 && strings.Contains(parts[0], " ")) {
								// Likely a virtual day or year/day - check physical path
								physPath := GetPhysicalPath(selected.RelPath, baseDir, currentResType)
								if _, err := os.Stat(filepath.Join(baseDir, physPath)); err != nil {
									continue
								}
							}
						}

						term.Restore(int(os.Stdin.Fd()), oldState)
						fmt.Print(showCursor + exitAltScreen)

						physPath := GetPhysicalPath(selected.RelPath, baseDir, currentResType)

						filePath := filepath.Join(baseDir, physPath)
						cmd := exec.Command(editorCmd, filePath)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						err = cmd.Run()
						if err != nil {
							lastEditorError = fmt.Sprintf("could not launch editor %q: %v", editorCmd, err)
							showEditorError = true
						}

						// Refresh after returning from editor - ENSURE BASEDIR IS PHYSICAL
						// (It should already be if navigation was fixed, but let's be safe)
						if !isProjectRoot {
							conf, err := FindEnclosingResourceIn(baseDir)
							if err == nil {
								currentResType = conf.Type
							}
						}

						var newFiles []string
						if isProjectRoot {
							newFiles, err = ScanProjectResources(baseDir, showHidden)
						} else if currentResType == "calendar" {
							newFiles, err = ScanCalendarDays(baseDir, showHidden)
						} else if currentResType == "todo" {
							newFiles, err = ScanTodoItems(baseDir, showHidden)
						} else {
							newFiles, err = ScanNotebookFiles(baseDir, showHidden)
						}
						if err == nil {
							entries = BuildDisplayEntries(newFiles, baseDir, true, currentResType == "calendar" || currentResType == "todo", currentResType)
						}
						colors, editorCmd = loadColorsAndEditor(baseDir)

						if selectedIndex >= len(entries) {
							selectedIndex = len(entries) - 1
						}
						if selectedIndex < 0 {
							selectedIndex = 0
						}

						// Re-enter raw mode and alt screen
						oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
						fmt.Print(enterAltScreen + hideCursor)
					}
				}
			}
		} else if n >= 3 && b[0] == 27 && b[1] == 91 {
			// Handle extended escape sequences (Ctrl+Arrows, Home, End, etc.)
			// Ctrl+Up:   ESC [ 1 ; 5 A
			// Ctrl+Down: ESC [ 1 ; 5 B
			isCtrlUp := n >= 6 && b[2] == 49 && b[3] == 59 && b[4] == 53 && b[5] == 65
			isCtrlDown := n >= 6 && b[2] == 49 && b[3] == 59 && b[4] == 53 && b[5] == 66
			isHome := b[2] == 72 || (n >= 4 && b[2] == 49 && b[3] == 126)
			isEnd := b[2] == 70 || (n >= 4 && b[2] == 52 && b[3] == 126)
			isUp := b[2] == 65
			isDown := b[2] == 66
			isShiftTab := b[2] == 90

			if isShiftTab && !focusList {
				if len(allLinks) > 0 {
					selectedLinkIdx = (selectedLinkIdx + 1) % len(allLinks)
				}
				continue
			}

			if showCreateType {
				switch b[2] {
				case 65: // Up
					if createTypeSelected > 0 {
						createTypeSelected--
					}
				case 66: // Down
					if createTypeSelected < len(currentCreateOptions)-1 {
						createTypeSelected++
					}
				}
				continue
			}

			if !showHelp && !showCreateName {
				if focusList {
					if isCtrlUp {
						selectedIndex -= 7
						if selectedIndex < 0 {
							selectedIndex = 0
						}
						previewOffset = 0
						selectedLinkIdx = -1
						if selectedIndex < listOffset {
							listOffset = selectedIndex
						}
					} else if isCtrlDown {
						selectedIndex += 7
						if selectedIndex >= len(entries) {
							selectedIndex = len(entries) - 1
						}
						previewOffset = 0
						selectedLinkIdx = -1
						if selectedIndex >= listOffset+contentHeight {
							listOffset = selectedIndex - contentHeight + 1
						}
					} else if isUp {
						if selectedIndex > 0 {
							selectedIndex--
						}
						previewOffset = 0
						selectedLinkIdx = -1
						if selectedIndex < listOffset {
							listOffset = selectedIndex
						}
					} else if isDown {
						if selectedIndex < len(entries)-1 {
							selectedIndex++
						}
						previewOffset = 0
						selectedLinkIdx = -1
						if selectedIndex >= listOffset+contentHeight {
							listOffset = selectedIndex - contentHeight + 1
						}
					} else if isHome {
						selectedIndex = 0
						listOffset = 0
						previewOffset = 0
						selectedLinkIdx = -1
					} else if isEnd {
						selectedIndex = len(entries) - 1
						if selectedIndex < 0 {
							selectedIndex = 0
						}
						previewOffset = 0
						selectedLinkIdx = -1
						if selectedIndex >= contentHeight {
							listOffset = selectedIndex - contentHeight + 1
						}
					}
				} else {
					// Preview focus navigation
					switch b[2] {
					case 65: // Up
						if previewOffset > 0 {
							previewOffset--
						}
						selectedLinkIdx = -1
					case 66: // Down
						if previewOffset < len(allPreviewLines)-contentHeight {
							previewOffset++
						}
						selectedLinkIdx = -1
					case 53: // PgUp (ESC [ 5 ~)
						if n >= 4 && b[3] == 126 {
							previewOffset -= contentHeight
						}
						selectedLinkIdx = -1
					case 54: // PgDn (ESC [ 6 ~)
						if n >= 4 && b[3] == 126 {
							previewOffset += contentHeight
						}
						selectedLinkIdx = -1
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
