package cmd

import (
	"regexp"
	"strings"
)

// ColorizeJSON applies ANSI color codes to a string containing JSON data.
// It uses a robust tokenization approach to avoid corrupting ANSI sequences.
func ColorizeJSON(text string) string {
	// Colors
	keyColor := "\033[34m"        // Blue
	stringColor := "\033[32m"     // Green
	numberColor := "\033[33m"     // Yellow
	boolColor := "\033[35m"       // Magenta
	nullColor := "\033[31m"       // Red
	dateColor := "\033[38;5;129m" // Violet
	reset := "\033[0m"

	// Combined regex to match all JSON tokens in order of precedence:
	// 1. Strings (including escaped quotes)
	// 2. Numbers (integers, floats, exponents)
	// 3. Booleans/Null
	// 4. Structural characters (colons, braces, etc. - we'll keep these uncolored or handle them separately)

	tokenRe := regexp.MustCompile(`"([^"\\]|\\.)*"|(-?\d+(\.\d+)?([eE][+-]?\d+)?)|(true|false|null)`)

	// We also need to know if a string is a KEY (followed by a colon)
	// We'll process the string and keep track of colons.

	var result strings.Builder
	lastEnd := 0

	// Find all matches
	matches := tokenRe.FindAllStringSubmatchIndex(text, -1)

	for _, m := range matches {
		start, end := m[0], m[1]

		// Add structural/unmatched text before the token
		result.WriteString(text[lastEnd:start])

		token := text[start:end]

		if strings.HasPrefix(token, `"`) {
			// It's a string. Check if it's a key by looking ahead for a colon.
			isKey := false
			remaining := text[end:]
			for i := 0; i < len(remaining); i++ {
				if remaining[i] == ' ' || remaining[i] == '\t' || remaining[i] == '\r' || remaining[i] == '\n' {
					continue
				}
				if remaining[i] == ':' {
					isKey = true
				}
				break
			}

			if isKey {
				result.WriteString(keyColor + token + reset)
			} else {
				// Check if it's an ISO date
				dateRe := regexp.MustCompile(`^"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*"$`)
				if dateRe.MatchString(token) {
					result.WriteString(dateColor + token + reset)
				} else {
					result.WriteString(stringColor + token + reset)
				}
			}
		} else if (token[0] >= '0' && token[0] <= '9') || token[0] == '-' {
			// It's a number
			result.WriteString(numberColor + token + reset)
		} else if token == "true" || token == "false" {
			// It's a boolean
			result.WriteString(boolColor + token + reset)
		} else if token == "null" {
			// It's null
			result.WriteString(nullColor + token + reset)
		} else {
			// Fallback for anything else (shouldn't happen with our regex)
			result.WriteString(token)
		}

		lastEnd = end
	}

	// Add remaining text after last match
	result.WriteString(text[lastEnd:])

	return result.String()
}
