package tests

import (
	"nocti/cmd"
	"strings"
	"testing"
)

func TestTableColoring(t *testing.T) {
	input := []string{
		"| Header 1 | Header 2 |",
		"| :--- [:col:blue:] | :--- [:col:cyan:] |",
		"| [:row:red:] Row 1 Col 1 | Row 1 Col 2 |",
		"| [:cell:green:] Cell Col 1 | Cell Col 2 |",
		"| Normal 1 | Normal 2 |",
	}

	output := cmd.FormatTables(input)

	// Print output for debugging
	for i, line := range output {
		t.Logf("%d: %q", i, line)
	}

	if len(output) < 7 {
		t.Fatalf("Expected at least 7 lines, got %d", len(output))
	}

	// Row 1 (Header) should be blue in Col 1, cyan in Col 2
	if !strings.Contains(output[1], "\033[44m Header 1") {
		t.Errorf("Header 1 (output[1]) should be blue, got %q", output[1])
	}
	if !strings.Contains(output[1], "\033[46m Header 2") {
		t.Errorf("Header 2 (output[1]) should be cyan, got %q", output[1])
	}

	// Row 3 (Row 1) should be red in both cols (overrides col colors)
	if !strings.Contains(output[3], "\033[41m Row 1 Col 1") {
		t.Errorf("Row 1 Col 1 (output[3]) should be red, got %q", output[3])
	}
	if !strings.Contains(output[3], "\033[41m Row 1 Col 2") {
		t.Errorf("Row 1 Col 2 (output[3]) should be red, got %q", output[3])
	}

	// Row 4 (Row 2) Cell 1 should be green (overrides col blue)
	if !strings.Contains(output[4], "\033[42m Cell Col 1") {
		t.Errorf("Cell Col 1 (output[4]) should be green, got %q", output[4])
	}
	// Row 4 (Row 2) Cell 2 should be cyan (col color)
	if !strings.Contains(output[4], "\033[46m Cell Col 2") {
		t.Errorf("Cell Col 2 (output[4]) should be cyan, got %q", output[4])
	}
}

func TestTableWideColoring(t *testing.T) {
	input := []string{
		"| [:table:yellow:] Header 1 | Header 2 |",
		"| :--- | :--- |",
		"| Data 1 | Data 2 |",
	}

	output := cmd.FormatTables(input)

	// Print output for debugging
	for i, line := range output {
		t.Logf("%d: %q", i, line)
	}

	// All borders and cells should have yellow background (\033[43m)
	for i, line := range output {
		if !strings.Contains(line, "\033[43m") {
			t.Errorf("Line %d should contain yellow background, got %q", i, line)
		}
	}

	// Border chars should be colored
	if !strings.Contains(output[0], "\033[43m┌") {
		t.Errorf("Top border should be yellow, got %q", output[0])
	}
	if !strings.Contains(output[1], "\033[43m│") {
		t.Errorf("Row border should be yellow, got %q", output[1])
	}
}
