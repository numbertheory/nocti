package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

type FlatSearchMatch struct {
	IsHeader  bool
	Path      string
	LineNo    int
	Text      string
	ResultIdx int
}

type SearchUIState struct {
	Query         string
	IsSearching   bool
	Results       []SearchResult
	FlatMatches   []FlatSearchMatch
	SelectedIndex int
	ScrollOffset  int
	SearchDir     string
	ResType       string
	IsProjectRoot bool
	Colors        *ColorsConfig
	Error         string
}

func (s *SearchUIState) PerformSearch() {
	if s.Query == "" {
		s.Results = nil
		s.FlatMatches = nil
		return
	}

	keywords := strings.Fields(s.Query)
	results, err := PerformSearch(s.SearchDir, s.ResType, s.IsProjectRoot, keywords, false, 0)
	if err != nil {
		s.Error = err.Error()
		return
	}

	s.Results = results
	s.FlattenResults()
	s.SelectedIndex = 0
	s.ScrollOffset = 0
	s.IsSearching = false
	s.Error = ""
}

func (s *SearchUIState) FlattenResults() {
	var flat []FlatSearchMatch
	for i, res := range s.Results {
		// Header for the file
		flat = append(flat, FlatSearchMatch{
			IsHeader:  true,
			Path:      res.Path,
			ResultIdx: i,
		})
		for _, m := range res.Matches {
			flat = append(flat, FlatSearchMatch{
				IsHeader:  false,
				Path:      res.Path,
				LineNo:    m.LineNo,
				Text:      m.Text,
				ResultIdx: i,
			})
		}
	}
	s.FlatMatches = flat
}

func (s *SearchUIState) HandleInput(b []byte, n int) (bool, *FlatSearchMatch, bool) {
	if s.IsSearching {
		if b[0] == '\r' || b[0] == '\n' {
			s.PerformSearch()
			return true, nil, false
		} else if b[0] == 27 && n == 1 { // ESC
			return false, nil, false // Close
		} else if b[0] == 127 || b[0] == 8 { // Backspace
			if len(s.Query) > 0 {
				s.Query = s.Query[:len(s.Query)-1]
			}
		} else if b[0] == '\t' {
			if len(s.FlatMatches) > 0 {
				s.IsSearching = false
			}
		} else if b[0] >= 32 && b[0] <= 126 {
			s.Query += string(b[0])
		}
		return true, nil, false
	}

	// Result navigation
	if b[0] == '\t' {
		s.IsSearching = true
		return true, nil, false
	}
	if (b[0] == 27 && n == 1) || b[0] == 'q' || b[0] == 'Q' || b[0] == 3 {
		return false, nil, false
	}
	if b[0] == '/' {
		s.IsSearching = true
		return true, nil, false
	}

	if b[0] == '\r' || b[0] == '\n' {
		if s.SelectedIndex < len(s.FlatMatches) {
			match := s.FlatMatches[s.SelectedIndex]
			return false, &match, true // Signal action
		}
	}

	if n >= 3 && b[0] == 27 && b[1] == 91 {
		if b[2] == 'A' { // Up
			if s.SelectedIndex > 0 {
				s.SelectedIndex--
			}
			return true, nil, false
		} else if b[2] == 'B' { // Down
			if s.SelectedIndex < len(s.FlatMatches)-1 {
				s.SelectedIndex++
			}
			return true, nil, false
		}
	}

	return true, nil, false
}

func DrawSearchUI(width, height int, state *SearchUIState) {
	reset := "\033[0m"
	reverseOn := "\033[7m"
	clearScreen := "\033[2J"
	cursorHome := "\033[H"

	fmt.Print(reset + clearScreen + cursorHome)

	// Colors
	highlightFg := "\033[1;33m" // Bold Yellow
	highlightBg := ""
	lnFg := "\033[34m"       // Blue
	fileNameFg := "\033[32m" // Green
	scoreFg := "\033[35m"    // Magenta

	if state.Colors != nil {
		highlightFg = GetFGColorCode(state.Colors.SearchHighlightFg, highlightFg)
		highlightBg = GetColorCode(state.Colors.SearchHighlightBg, highlightBg)
		lnFg = GetFGColorCode(state.Colors.SearchLnFg, lnFg)
		fileNameFg = GetFGColorCode(state.Colors.SearchFileFg, fileNameFg)
		scoreFg = GetFGColorCode(state.Colors.SearchScoreFg, scoreFg)
	}

	// Header
	headerBg := "\033[48;5;236m"
	headerFg := "\033[38;5;15m"
	fmt.Printf("%s%s%-*s%s\n", headerBg, headerFg, width, " SEARCH ", reset)

	// Query box
	fmt.Printf("\033[3;2HSearch: ")
	if state.IsSearching {
		fmt.Printf("%s%s%s_ %s", reverseOn, state.Query, reset, reset)
	} else {
		fmt.Printf("%s (Press / to edit)", state.Query)
	}

	if state.Error != "" {
		fmt.Printf("\033[4;2H\033[31mError: %s%s", state.Error, reset)
	}

	contentY := 6
	contentHeight := height - contentY - 1

	if len(state.FlatMatches) > 0 {
		// Adjust scroll
		if state.SelectedIndex < state.ScrollOffset {
			state.ScrollOffset = state.SelectedIndex
		} else if state.SelectedIndex >= state.ScrollOffset+contentHeight {
			state.ScrollOffset = state.SelectedIndex - contentHeight + 1
		}

		keywords := strings.Fields(state.Query)
		var patterns []*regexp.Regexp
		for _, kw := range keywords {
			pattern := "(?i)" + regexp.QuoteMeta(kw)
			re, _ := regexp.Compile(pattern)
			patterns = append(patterns, re)
		}

		for i := 0; i < contentHeight; i++ {
			idx := i + state.ScrollOffset
			if idx >= len(state.FlatMatches) {
				break
			}
			match := state.FlatMatches[idx]
			row := contentY + i
			fmt.Printf("\033[%d;2H", row)

			style := ""
			if idx == state.SelectedIndex {
				style = reverseOn
			}

			if match.IsHeader {
				fmt.Printf("%s%s%s %s%s", style, fileNameFg, match.Path, reset, reset)
				// Show score if header
				origRes := state.Results[match.ResultIdx]
				fmt.Printf(" %s(score: %d)%s", scoreFg, origRes.Score, reset)
			} else {
				displayText := match.Text
				for _, p := range patterns {
					displayText = p.ReplaceAllStringFunc(displayText, func(m string) string {
						return highlightFg + highlightBg + m + reset + style
					})
				}
				fmt.Printf("%s  %s%d%s: %s%s", style, lnFg, match.LineNo, reset+style, displayText, reset)
			}
			fmt.Print("\033[K")
		}
	} else if state.Query != "" && !state.IsSearching {
		fmt.Printf("\033[%d;2HNo matches found.", contentY)
	}

	// Footer
	footerY := height
	helpText := " ↑/↓: Navigate | ENTER: Open | TAB//: New Search | q/ESC: Exit "
	fmt.Printf("\033[%d;1H%s%s%-*s%s", footerY, headerBg, headerFg, width, helpText, reset)
}
