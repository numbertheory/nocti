package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
)

// StripANSI removes ANSI escape codes from a string to get its visible length.
func StripANSI(text string) string {
	re := regexp.MustCompile(`\033\[[0-9;]*[mK]`)
	return re.ReplaceAllString(text, "")
}

// VisibleLen returns the length of the string without ANSI escape codes.
func VisibleLen(text string) int {
	return len(StripANSI(text))
}

// Link represents a detected URL in the text.
type Link struct {
	URL   string
	Start int // Visible start index
	End   int // Visible end index
}

// DetectLinks finds all HTTP/HTTPS URLs in the text (ignoring ANSI codes for positioning).
func DetectLinks(text string) []Link {
	// Simple regex for URLs
	re := regexp.MustCompile(`https?://[^\s)\]]+`)

	// We need to map positions in the stripped string back to something useful,
	// but since we render the string WITH ANSI codes, it's easier to just strip
	// and find links, then when we render, we can highlight them.
	stripped := StripANSI(text)
	matches := re.FindAllStringIndex(stripped, -1)

	var links []Link
	for _, m := range matches {
		links = append(links, Link{
			URL:   stripped[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
		})
	}
	return links
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
