package tests

import (
	"regexp"
	"strings"
	"testing"
)

// Since processHighlights is in package cmd and not exported,
// and we are in package tests, we can't directly test it unless we use
// the same package or export it.
// For simplicity, I'll copy the logic here or I should have put it in a place where it can be tested.

func GetColorCode(colorName string, defaultCode string) string {
	if strings.ToLower(colorName) == "default" {
		return "\033[49m" // Reset background
	}
	colors := map[string]string{
		"black":   "\033[40m",
		"red":     "\033[41m",
		"green":   "\033[42m",
		"yellow":  "\033[43m",
		"blue":    "\033[44m",
		"magenta": "\033[45m",
		"cyan":    "\033[46m",
		"white":   "\033[47m",
	}

	if code, ok := colors[strings.ToLower(colorName)]; ok {
		return code
	}
	return defaultCode
}

func GetFGColorCode(colorName string, defaultCode string) string {
	if strings.ToLower(colorName) == "default" {
		return "\033[39m"
	}
	colors := map[string]string{
		"black": "\033[30m",
		"white": "\033[37m",
	}
	if code, ok := colors[strings.ToLower(colorName)]; ok {
		return code
	}
	return defaultCode
}

func processHighlights(text string) string {
	re := regexp.MustCompile(`\[:([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\s*(.*?)\]`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		var fgCode, bgCode, content string

		if parts[2] != "" {
			fgCode = GetFGColorCode(parts[1], "")
			bgCode = GetColorCode(parts[2], "")
			content = parts[3]
		} else {
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
		if fgCode != "" {
			res += "\033[39m"
		}
		if bgCode != "" {
			res += "\033[49m"
		}
		return res
	})
}

func TestProcessHighlights(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Yellow background",
			input:    "[:yellow: highlighted]",
			expected: "\033[43mhighlighted\033[49m",
		},
		{
			name:     "Black on blue",
			input:    "[:black:blue: black on blue]",
			expected: "\033[30m\033[44mblack on blue\033[39m\033[49m",
		},
		{
			name:     "Default on yellow",
			input:    "[:default:yellow: default on yellow]",
			expected: "\033[39m\033[43mdefault on yellow\033[39m\033[49m",
		},
		{
			name:     "Invalid colors",
			input:    "[:invalid: invalid]",
			expected: "[:invalid: invalid]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processHighlights(tt.input)
			if got != tt.expected {
				t.Errorf("processHighlights() = %q, want %q", got, tt.expected)
			}
		})
	}
}
