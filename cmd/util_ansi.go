package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"
)

// StripANSI removes ANSI escape codes from a string to get its visible length.
func StripANSI(text string) string {
	// Remove standard ANSI escape codes: \033[...m and \033[...K
	re := regexp.MustCompile(`\033\[[0-9;]*[mK]`)
	text = re.ReplaceAllString(text, "")
	// Remove OSC 8 sequences: \033]8;;...\033\ and \033]8;;\033\
	osc8 := regexp.MustCompile(`\033\]8;;.*?\033\\`)
	return osc8.ReplaceAllString(text, "")
}

// StripANSIWithMapping removes ANSI escape codes and returns the stripped string
// along with a mapping from stripped RUNE indices to original BYTE indices.
func StripANSIWithMapping(text string) (string, []int) {
	// Combined regex for standard ANSI and OSC 8
	re := regexp.MustCompile(`\033\[[0-9;]*[mK]|\033\]8;;.*?\033\\`)
	matches := re.FindAllStringIndex(text, -1)

	var stripped strings.Builder
	mapping := make([]int, 0, utf8.RuneCountInString(text))

	lastEnd := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		// Text before the ANSI code
		before := text[lastEnd:start]
		for i, r := range before {
			mapping = append(mapping, lastEnd+i)
			stripped.WriteRune(r)
		}
		lastEnd = end
	}
	// Remaining text
	remaining := text[lastEnd:]
	for i, r := range remaining {
		mapping = append(mapping, lastEnd+i)
		stripped.WriteRune(r)
	}
	// Map the end of the string as well
	mapping = append(mapping, len(text))

	return stripped.String(), mapping
}

// VisibleLen returns the number of visual columns occupied by the string (ignoring ANSI).
func VisibleLen(text string) int {
	return utf8.RuneCountInString(StripANSI(text))
}

// VisibleLenWithLinks returns the length of the string as it will appear in the preview,
// accounting for both ANSI codes and hidden URL parts of Markdown links.
func VisibleLenWithLinks(text string) int {
	stripped := StripANSI(text)

	// 1. Detect Markdown links and subtract the length of the bracket/URL parts
	markdownRe := regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s)]+?)(?:\s+"[^"]*")?\)`)
	mdMatches := markdownRe.FindAllStringSubmatch(stripped, -1)

	totalVisible := utf8.RuneCountInString(stripped)
	for _, m := range mdMatches {
		fullMatchLen := utf8.RuneCountInString(m[0])
		textPartLen := utf8.RuneCountInString(m[1])
		// We hide the brackets and the (url) part
		totalVisible -= (fullMatchLen - textPartLen)
	}

	// 2. Detect highlight syntax [:fg:bg: text] or [:bg: text] and subtract the extra parts
	highlightRe := regexp.MustCompile(`\[:([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\s*(.*?)\]`)
	hMatches := highlightRe.FindAllStringSubmatch(stripped, -1)
	for _, m := range hMatches {
		fullMatchLen := utf8.RuneCountInString(m[0])
		contentLen := utf8.RuneCountInString(m[3])
		// We hide the [:color:] or [:fg:bg:] part and the closing bracket
		totalVisible -= (fullMatchLen - contentLen)
	}

	// 3. Detect special table markers [:table:color:], [:row:color:], [:cell:color:], [:col:color:]
	// These are removed entirely from the visible length
	tableMarkerRe := regexp.MustCompile(`\[:(table|row|cell|col):([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\]`)
	tmMatches := tableMarkerRe.FindAllString(stripped, -1)
	for _, m := range tmMatches {
		totalVisible -= utf8.RuneCountInString(m)
	}

	return totalVisible
}

// Link represents a detected URL in the text.
type Link struct {
	URL         string
	DisplayText string
	Start       int // Visible start index
	End         int // Visible end index
	IsMarkdown  bool
}

// DetectLinks finds all HTTP/HTTPS URLs and Markdown links in the text (ignoring ANSI codes for positioning).
func DetectLinks(text string) []Link {
	stripped := StripANSI(text)
	var links []Link

	// 1. Detect Markdown links: [text](url)
	markdownRe := regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s)]+?)(?:\s+"[^"]*")?\)`)
	mdMatches := markdownRe.FindAllStringSubmatchIndex(stripped, -1)

	// We'll keep track of which parts of the stripped string are "consumed" by markdown links
	// to avoid double-detecting bare URLs inside them.
	consumed := make([]bool, len(stripped)+1)

	for _, m := range mdMatches {
		fullStart, fullEnd := m[0], m[1]
		textStart, textEnd := m[2], m[3]
		urlStart, urlEnd := m[4], m[5]

		links = append(links, Link{
			URL:         stripped[urlStart:urlEnd],
			DisplayText: stripped[textStart:textEnd],
			Start:       fullStart,
			End:         fullEnd,
			IsMarkdown:  true,
		})
		for i := fullStart; i < fullEnd; i++ {
			consumed[i] = true
		}
	}

	// 2. Detect Bare URLs: https?://...
	bareRe := regexp.MustCompile(`https?://[^\s)\]"']+`)
	bareMatches := bareRe.FindAllStringIndex(stripped, -1)

	for _, m := range bareMatches {
		// Check if this bare URL was already part of a markdown link
		if consumed[m[0]] {
			continue
		}

		links = append(links, Link{
			URL:         stripped[m[0]:m[1]],
			DisplayText: stripped[m[0]:m[1]],
			Start:       m[0],
			End:         m[1],
			IsMarkdown:  false,
		})
	}

	// Sort links by start position
	sort.Slice(links, func(i, j int) bool {
		return links[i].Start < links[j].Start
	})

	return links
}

// RenderedLine contains the final display string and the adjusted link positions.
type RenderedLine struct {
	Display string
	Links   []Link
}

// PrepareLineForDisplay processes a line to handle Markdown links and ANSI codes,
// returning the display string and adjusted link positions for interaction.
func PrepareLineForDisplay(text string, isSelected bool, selectedLinkIdx int, globalLinkStartIdx int) RenderedLine {
	links := DetectLinks(text)
	_, mapping := StripANSIWithMapping(text)

	display := ""
	var adjustedLinks []Link
	lastEndInOrig := 0

	for i, l := range links {
		// Add text before link, preserving ANSI codes
		display += text[lastEndInOrig:mapping[l.Start]]

		newStart := len(display)
		globalIdx := globalLinkStartIdx + i

		linkDisplay := l.DisplayText

		// Apply highlighting and OSC 8 hyperlink
		linkStart := fmt.Sprintf("\033]8;;%s\033\\", l.URL)
		linkEnd := "\033]8;;\033\\"

		if globalIdx == selectedLinkIdx && isSelected {
			display += linkStart + "\033[4;34;7m" + linkDisplay + "\033[24;39;27m" + linkEnd
		} else {
			display += linkStart + "\033[4;34m" + linkDisplay + "\033[24;39m" + linkEnd
		}

		newEnd := len(display)

		adjustedLinks = append(adjustedLinks, Link{
			URL:         l.URL,
			DisplayText: l.DisplayText,
			Start:       newStart,
			End:         newEnd,
			IsMarkdown:  l.IsMarkdown,
		})

		lastEndInOrig = mapping[l.End]
	}
	// Add remaining text, preserving ANSI codes
	display += text[lastEndInOrig:]

	return RenderedLine{
		Display: display,
		Links:   adjustedLinks,
	}
}

// OpenURL opens the given URL in the default system browser.
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

var colorMap = map[string]string{
	"black":         "0",
	"red":           "1",
	"green":         "2",
	"yellow":        "3",
	"blue":          "4",
	"magenta":       "5",
	"cyan":          "6",
	"white":         "7",
	"gray":          "38;5;244",
	"darkgray":      "38;5;236",
	"lightgray":     "38;5;250",
	"silver":        "38;5;7",
	"brightred":     "91",
	"brightgreen":   "92",
	"brightyellow":  "93",
	"brightblue":    "94",
	"brightmagenta": "95",
	"brightcyan":    "96",
	"brightwhite":   "97",
	"orange":        "38;5;208",
	"darkorange":    "38;5;166",
	"pink":          "38;5;205",
	"hotpink":       "38;5;198",
	"purple":        "38;5;93",
	"violet":        "38;5;129",
	"brown":         "38;5;94",
	"navy":          "38;5;18",
	"teal":          "38;5;30",
	"olive":         "38;5;58",
	"maroon":        "38;5;88",
	"aqua":          "38;5;51",
	"fuchsia":       "38;5;201",
	"lime":          "38;5;46",
	"skyblue":       "38;5;117",
	"gold":          "38;5;214",
	"indigo":        "38;5;54",
	"coral":         "38;5;209",
	"turquoise":     "38;5;45",
	"plum":          "38;5;96",
	"orchid":        "38;5;170",
	"salmon":        "38;5;210",
}

func GetColorNames() []string {
	var names []string
	for name := range colorMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func IsLightColor(name string) bool {
	lightColors := map[string]bool{
		"yellow":        true,
		"cyan":          true,
		"white":         true,
		"lightgray":     true,
		"silver":        true,
		"brightgreen":   true,
		"brightyellow":  true,
		"brightcyan":    true,
		"brightwhite":   true,
		"brightmagenta": true,
		"aqua":          true,
		"lime":          true,
		"skyblue":       true,
		"gold":          true,
		"salmon":        true,
		"pink":          true,
	}
	return lightColors[strings.ToLower(name)]
}

func GetFGColorCode(colorName string, defaultCode string) string {
	name := strings.ToLower(colorName)
	if name == "default" {
		return "\033[39m"
	}
	if code, ok := colorMap[name]; ok {
		if strings.Contains(code, ";") {
			return "\033[" + code + "m"
		}
		// Standard/Bright colors (30-37, 90-97)
		// Our map has 0-7 or 91-97.
		// If it's 0-7, add 30.
		if len(code) == 1 {
			return fmt.Sprintf("\033[3%sm", code)
		}
		return "\033[" + code + "m"
	}
	return defaultCode
}

func GetColorCode(colorName string, defaultCode string) string {
	name := strings.ToLower(colorName)
	if name == "default" {
		return "\033[49m"
	}
	if code, ok := colorMap[name]; ok {
		if strings.Contains(code, ";") {
			// Replace 38 with 48 for background
			bgCode := strings.Replace(code, "38", "48", 1)
			return "\033[" + bgCode + "m"
		}
		if len(code) == 1 {
			return fmt.Sprintf("\033[4%sm", code)
		}
		// Bright background codes are 100-107
		// Our map has 91-97 for foreground.
		// 91 (fg red) -> 101 (bg red)
		if len(code) == 2 && code[0] == '9' {
			return "\033[10" + code[1:2] + "m"
		}
		return "\033[" + code + "m"
	}
	return defaultCode
}
