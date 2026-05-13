package cmd

import (
	"regexp"
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
