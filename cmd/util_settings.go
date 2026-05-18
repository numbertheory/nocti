package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type SettingsTab int

const (
	TabColors SettingsTab = iota
	TabEditor
	TabSave
)

type SettingsState struct {
	Tab           SettingsTab
	SelectedIndex int
	ActiveField   bool
	InputValue    string
	ConfigPath    string
	Colors        *ColorsConfig
	Editor        string
	ScrollOffset  int
}

func (s *SettingsState) Save() error {
	data, err := os.ReadFile(s.ConfigPath)
	if err != nil {
		return err
	}

	var fullConfig map[string]interface{}
	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return err
	}

	fullConfig["colors"] = s.Colors
	fullConfig["editor"] = s.Editor

	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.ConfigPath, updatedData, 0644)
}

func (s *SettingsState) HandleInput(b []byte, n int) (bool, bool) {
	if s.ActiveField {
		if b[0] == '\r' || b[0] == '\n' {
			if s.Tab == TabColors {
				fields := GetColorFields()
				fieldName := fields[s.SelectedIndex]
				reflect.ValueOf(s.Colors).Elem().FieldByName(fieldName).SetString(s.InputValue)
			} else if s.Tab == TabEditor {
				s.Editor = s.InputValue
			}
			s.ActiveField = false
		} else if b[0] == 27 && n == 1 { // ESC (not arrow key)
			s.ActiveField = false
		} else if b[0] == 127 || b[0] == 8 { // Backspace
			if len(s.InputValue) > 0 {
				s.InputValue = s.InputValue[:len(s.InputValue)-1]
			}
		} else if b[0] >= 32 && b[0] <= 126 {
			s.InputValue += string(b[0])
		}
		return true, false
	}

	// Navigation between tabs
	if b[0] == '\t' {
		s.Tab = (s.Tab + 1) % 3
		s.SelectedIndex = 0
		return true, false
	}

	// Exit without saving
	if (b[0] == 27 && n == 1) || b[0] == 'q' || b[0] == 'Q' {
		return false, false
	}

	if s.Tab == TabSave {
		if n >= 3 && b[0] == 27 && b[1] == 91 {
			if b[2] == 'A' || b[2] == 'B' { // Up or Down
				s.SelectedIndex = 1 - s.SelectedIndex
				return true, false
			}
		}
		if b[0] == '\r' || b[0] == '\n' {
			if s.SelectedIndex == 0 {
				s.Save()
				return false, true // Save and exit
			} else {
				return false, false // Just exit
			}
		}
		return true, false
	}

	// Field Selection/Editing
	if b[0] == '\r' || b[0] == '\n' {
		s.ActiveField = true
		if s.Tab == TabColors {
			fields := GetColorFields()
			fieldName := fields[s.SelectedIndex]
			s.InputValue = reflect.ValueOf(s.Colors).Elem().FieldByName(fieldName).String()
		} else if s.Tab == TabEditor {
			s.InputValue = s.Editor
		}
		return true, false
	}

	// Navigation within fields
	if n >= 3 && b[0] == 27 && b[1] == 91 {
		if b[2] == 'A' { // Up
			s.SelectedIndex--
			return true, false
		} else if b[2] == 'B' { // Down
			s.SelectedIndex++
			return true, false
		}
	}

	return true, false
}

func GetColorFields() []string {
	v := reflect.ValueOf(ColorsConfig{})
	t := v.Type()
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		fields = append(fields, t.Field(i).Name)
	}
	return fields
}

func fieldToLabel(name string) string {
	var label strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			label.WriteRune(' ')
		}
		label.WriteRune(r)
	}
	return label.String()
}

func DrawSettingsPanel(width, height int, state *SettingsState) {
	reset := "\033[0m"
	boldOn := "\033[1m"
	reverseOn := "\033[7m"
	clearScreen := "\033[2J"
	cursorHome := "\033[H"

	fmt.Print(reset + clearScreen + cursorHome)

	// Draw Header
	headerBg := "\033[48;5;236m"
	headerFg := "\033[38;5;15m"
	fmt.Printf("%s%s%-*s%s\n", headerBg, headerFg, width, " SETTINGS ", reset)

	// Draw Tabs
	tabX := 2
	tabY := 3
	fmt.Printf("\033[%d;%dH", tabY, tabX)
	tabs := []string{" Colors ", " Editor ", " Save "}
	for i, t := range tabs {
		style := reset
		if state.Tab == SettingsTab(i) {
			style = reverseOn + boldOn
		}
		fmt.Printf("%s%s%s  ", style, t, reset)
	}

	contentY := 5
	contentHeight := height - contentY - 2

	if state.Tab == TabColors {
		fields := GetColorFields()
		// Ensure SelectedIndex is within bounds
		if state.SelectedIndex >= len(fields) {
			state.SelectedIndex = len(fields) - 1
		}
		if state.SelectedIndex < 0 {
			state.SelectedIndex = 0
		}

		// Adjust scroll offset
		if state.SelectedIndex < state.ScrollOffset {
			state.ScrollOffset = state.SelectedIndex
		} else if state.SelectedIndex >= state.ScrollOffset+contentHeight {
			state.ScrollOffset = state.SelectedIndex - contentHeight + 1
		}

		for i := 0; i < contentHeight; i++ {
			fieldIdx := i + state.ScrollOffset
			if fieldIdx >= len(fields) {
				break
			}
			fieldName := fields[fieldIdx]
			label := fieldToLabel(fieldName)

			v := reflect.ValueOf(state.Colors).Elem()
			fieldVal := v.FieldByName(fieldName).String()

			row := contentY + i
			fmt.Printf("\033[%d;%dH", row, tabX)

			if fieldIdx == state.SelectedIndex {
				if state.ActiveField {
					fmt.Printf("%s%s: %s%s%s_ %s", boldOn, label, reset, reverseOn, state.InputValue, reset)
				} else {
					fmt.Printf("%s > %s: %s%s", reverseOn+boldOn, label, reset, fieldVal)
				}
			} else {
				fmt.Printf("  %s: %s", label, fieldVal)
			}
			fmt.Print("\033[K")
		}

		// Draw Color Reference Table
		colorX := width / 2
		if colorX < 40 {
			colorX = 40
		}
		colorY := contentY
		colorNames := GetColorNames()
		colWidth := 20
		maxRows := contentHeight

		// Draw Header with fixed width
		fmt.Printf("\033[%d;%dH%s %-*s %s", colorY-1, colorX, boldOn+reverseOn, colWidth-2, "AVAILABLE COLORS", reset)

		for i, name := range colorNames {
			col := i / maxRows
			row := i % maxRows
			if colorX+((col+1)*colWidth) > width {
				break
			}

			fmt.Printf("\033[%d;%dH", colorY+row, colorX+(col*colWidth))

			bgCode := GetColorCode(name, "")
			fgCode := "\033[37m" // White
			if IsLightColor(name) {
				fgCode = "\033[30m" // Black
			}

			// Draw entry with same fixed width as header
			fmt.Printf("%s%s %-*s %s", bgCode, fgCode, colWidth-2, name, reset)
		}
	} else if state.Tab == TabEditor {
		fmt.Printf("\033[%d;%dH", contentY, tabX)
		if state.ActiveField {
			fmt.Printf("%sEditor Command: %s%s%s_ %s", boldOn, reset, reverseOn, state.InputValue, reset)
		} else {
			fmt.Printf("%s > Editor Command: %s%s", reverseOn+boldOn, reset, state.Editor)
		}
		fmt.Print("\033[K")
	} else if state.Tab == TabSave {
		options := []string{"Save and Exit Settings", "Don't Save and Exit Settings"}
		for i, opt := range options {
			fmt.Printf("\033[%d;%dH", contentY+i, tabX)
			style := reset
			if i == state.SelectedIndex {
				style = reverseOn + boldOn
			}
			fmt.Printf("%s %s %s", style, opt, reset)
			fmt.Print("\033[K")
		}
	}

	// Draw Footer
	footerY := height
	footerBg := "\033[48;5;236m"
	footerFg := "\033[38;5;15m"
	helpText := " TAB: Next Tab | ↑/↓: Select | ENTER: Edit/Select | q/ESC: Exit (No Save) "
	fmt.Printf("\033[%d;1H%s%s%-*s%s", footerY, footerBg, footerFg, width, helpText, reset)
}
