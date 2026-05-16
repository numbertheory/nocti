package tests

import (
	"nocti/cmd"
	"strings"
	"testing"
)

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
			got := cmd.ProcessHighlights(tt.input)
			if got != tt.expected {
				t.Errorf("ProcessHighlights() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrepareLineForDisplayPreservesHighlights(t *testing.T) {
	// 1. Original text with highlighting syntax
	input := "[:yellow: highlighted] and [link](http://example.com)"

	// 2. Process highlights (adds ANSI codes)
	highlighted := cmd.ProcessHighlights(input)

	// Verify it has ANSI codes
	if !strings.Contains(highlighted, "\033[43m") {
		t.Fatalf("Expected highlighted string to contain yellow background ANSI code, got: %q", highlighted)
	}

	// 3. Prepare for display (should preserve highlights and add OSC 8)
	rendered := cmd.PrepareLineForDisplay(highlighted, false, -1, 0)

	// Check if it still contains the yellow background code
	if !strings.Contains(rendered.Display, "\033[43m") {
		t.Errorf("PrepareLineForDisplay stripped highlights!\nInput: %q\nOutput: %q", highlighted, rendered.Display)
	}

	// Check if OSC 8 is present
	if !strings.Contains(rendered.Display, "\033]8;;http://example.com\033\\") {
		t.Errorf("OSC 8 link start sequence missing or incorrect: %q", rendered.Display)
	}
	if !strings.Contains(rendered.Display, "\033]8;;\033\\") {
		t.Errorf("OSC 8 link end sequence missing: %q", rendered.Display)
	}

	// Also check if link text is processed
	if !strings.Contains(rendered.Display, "link") || strings.Contains(rendered.Display, "http://example.com") {
		// Note: The URL is in the escape sequence, but should NOT be in the visible text
		// This check is a bit naive if it just looks for the string anywhere,
		// but since it's in the escape sequence, it WILL be there.
		// We should check that it's NOT there as plain text.
		stripped := cmd.StripANSI(rendered.Display)
		if strings.Contains(stripped, "http://example.com") {
			t.Errorf("URL should not be visible in stripped display: %q", stripped)
		}
	}
	// Verify the URL and IsMarkdown in adjustedLinks
	if len(rendered.Links) != 1 {
		t.Fatalf("Expected 1 adjusted link, got %d", len(rendered.Links))
	}
	if rendered.Links[0].URL != "http://example.com" {
		t.Errorf("Expected adjusted link URL http://example.com, got %q", rendered.Links[0].URL)
	}
	if !rendered.Links[0].IsMarkdown {
		t.Errorf("Expected IsMarkdown to be true for adjusted link")
	}
}
