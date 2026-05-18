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
