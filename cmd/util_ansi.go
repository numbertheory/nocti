package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// StripANSI removes ANSI escape codes from a string to get its visible length.
func StripANSI(text string) string {
	re := regexp.MustCompile(`\033\[[0-9;]*[mK]`)
	return re.ReplaceAllString(text, "")
}

// StripANSIWithMapping removes ANSI escape codes and returns the stripped string
// along with a mapping from stripped indices to original indices.
func StripANSIWithMapping(text string) (string, []int) {
	re := regexp.MustCompile(`\033\[[0-9;]*[mK]`)
	matches := re.FindAllStringIndex(text, -1)

	var stripped strings.Builder
	mapping := make([]int, 0, len(text))

	lastEnd := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		// Text before the ANSI code
		for i := lastEnd; i < start; i++ {
			mapping = append(mapping, i)
			stripped.WriteByte(text[i])
		}
		lastEnd = end
	}
	// Remaining text
	for i := lastEnd; i < len(text); i++ {
		mapping = append(mapping, i)
		stripped.WriteByte(text[i])
	}
	// Map the end of the string as well
	mapping = append(mapping, len(text))

	return stripped.String(), mapping
}

// VisibleLen returns the length of the string without ANSI escape codes.
func VisibleLen(text string) int {
	return len(StripANSI(text))
}

// VisibleLenWithLinks returns the length of the string as it will appear in the preview,
// accounting for both ANSI codes and hidden URL parts of Markdown links.
func VisibleLenWithLinks(text string) int {
	stripped := StripANSI(text)

	// Detect Markdown links and subtract the length of the bracket/URL parts
	markdownRe := regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s)\]]+)\)`)
	matches := markdownRe.FindAllStringSubmatch(stripped, -1)

	totalVisible := len(stripped)
	for _, m := range matches {
		fullMatchLen := len(m[0])
		textPartLen := len(m[1])
		// We hide the brackets and the (url) part
		totalVisible -= (fullMatchLen - textPartLen)
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
	markdownRe := regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s)\]]+)\)`)
	mdMatches := markdownRe.FindAllStringSubmatchIndex(stripped, -1)

	// We'll keep track of which parts of the stripped string are "consumed" by markdown links
	// to avoid double-detecting bare URLs inside them.
	consumed := make([]bool, len(stripped))

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
	bareRe := regexp.MustCompile(`https?://[^\s)\]]+`)
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

		// Apply highlighting for the link itself
		if globalIdx == selectedLinkIdx && isSelected {
			display += "\033[4;34;7m" + linkDisplay + "\033[24;39;27m"
		} else {
			display += "\033[4;34m" + linkDisplay + "\033[24;39m"
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
