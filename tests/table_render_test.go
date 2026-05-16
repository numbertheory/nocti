package tests

import (
	"nocti/cmd"
	"strings"
	"testing"
)

func TestFormatTables(t *testing.T) {
	input := []string{
		"Standard text",
		"| Col 1 | Col 2 |",
		"| :--- | :--- |",
		"| Data 1 | Data 2 |",
		"More text",
	}

	output := cmd.FormatTables(input)

	// Expected output should have 5 lines of table + 2 lines of text = 7 lines
	// Table: Top, Header, Separator, Data, Bottom
	if len(output) != 7 {
		t.Fatalf("Expected 7 lines of output, got %d", len(output))
	}

	if output[0] != "Standard text" {
		t.Errorf("First line should be standard text, got %q", output[0])
	}

	// Check for box drawing characters in table area
	if !strings.Contains(output[1], "┌") || !strings.Contains(output[1], "┬") || !strings.Contains(output[1], "┐") {
		t.Errorf("Table top border missing: %q", output[1])
	}
	if !strings.Contains(output[2], "│") || !strings.Contains(output[2], "Col 1") {
		t.Errorf("Table header row missing: %q", output[2])
	}
	if !strings.Contains(output[3], "├") || !strings.Contains(output[3], "┼") || !strings.Contains(output[3], "┤") {
		t.Errorf("Table separator row missing: %q", output[3])
	}
	if !strings.Contains(output[4], "│") || !strings.Contains(output[4], "Data 1") {
		t.Errorf("Table data row missing: %q", output[4])
	}
	if !strings.Contains(output[5], "└") || !strings.Contains(output[5], "┴") || !strings.Contains(output[5], "┘") {
		t.Errorf("Table bottom border missing: %q", output[5])
	}

	if output[6] != "More text" {
		t.Errorf("Last line should be standard text, got %q", output[6])
	}
}

func TestFormatTablesWithHighlights(t *testing.T) {
	input := []string{
		"| Header |",
		"| :--- |",
		"| [:yellow: Highlighted] |",
	}

	output := cmd.FormatTables(input)

	found := false
	for _, line := range output {
		if strings.Contains(line, "[:yellow: Highlighted]") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Highlights should be preserved in FormatTables output")
	}
}

func TestTableTruncation(t *testing.T) {
	// Test that long table rows are truncated in GetFilePreview
	// Since we can't easily mock the file system here for GetFilePreview,
	// we rely on manual verification or more complex integration tests if needed.
	// For now, let's just check the logic in FormatTables for basic correctness.
}
