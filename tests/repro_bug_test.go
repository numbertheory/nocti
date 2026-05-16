package tests

import (
	"nocti/cmd"
	"strings"
	"testing"
)

func TestPrepareLineForDisplayPreservesANSI(t *testing.T) {
	// Simulate what happens in GetFilePreview:
	// 1. Original text with highlighting syntax
	input := "[:yellow: highlighted] and [link](http://example.com)"

	// 2. Process highlights (adds ANSI codes)
	highlighted := cmd.ProcessHighlights(input)

	// Verify it has ANSI codes
	if !strings.Contains(highlighted, "\033[43m") {
		t.Fatalf("Expected highlighted string to contain yellow background ANSI code, got: %q", highlighted)
	}

	// 3. Prepare for display (should preserve highlights)
	rendered := cmd.PrepareLineForDisplay(highlighted, false, -1, 0)

	// Check if it still contains the yellow background code
	if !strings.Contains(rendered.Display, "\033[43m") {
		t.Errorf("PrepareLineForDisplay stripped highlights!\nInput: %q\nOutput: %q", highlighted, rendered.Display)
	}

	// Also check if link is processed
	if !strings.Contains(rendered.Display, "link") || strings.Contains(rendered.Display, "http://example.com") {
		t.Errorf("Link was not correctly processed in rendered output: %q", rendered.Display)
	}
}
