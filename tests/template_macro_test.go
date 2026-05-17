package tests

import (
	"nocti/cmd"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReplaceMacros(t *testing.T) {
	macros := map[string]string{
		"NAME":    "My Meeting Notes",
		"Project": "Nocti CLI",
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04:05")

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Basic Name",
			content:  "Title: {{NAME}}",
			expected: "Title: My Meeting Notes",
		},
		{
			name:     "Basic Date",
			content:  "Date: {{DATE}}",
			expected: "Date: " + dateStr,
		},
		{
			name:     "Lower Modifier",
			content:  "Lower: {{NAME|LOWER}}",
			expected: "Lower: my meeting notes",
		},
		{
			name:     "Upper Modifier",
			content:  "Upper: {{NAME|UPPER}}",
			expected: "Upper: MY MEETING NOTES",
		},
		{
			name:     "Slug Modifier",
			content:  "Slug: {{NAME|SLUG}}",
			expected: "Slug: my-meeting-notes",
		},
		{
			name:     "Extended Date Time",
			content:  "Time: {{TIME}}, DT: {{DATETIME}}",
			expected: "Time: " + timeStr + ", DT: " + now.Format("2006-01-02 15:04:05"),
		},
		{
			name:     "Day Month Year",
			content:  "D: {{DAY}}, M: {{MONTH}}, Y: {{YEAR}}",
			expected: "D: " + now.Format("02") + ", M: " + now.Format("01") + ", Y: " + now.Format("2006"),
		},
		{
			name:     "Weekday",
			content:  "Weekday: {{WEEKDAY}}",
			expected: "Weekday: " + now.Weekday().String(),
		},
		{
			name:     "Tomorrow Yesterday",
			content:  "T: {{TOMORROW}}, Y: {{YESTERDAY}}",
			expected: "T: " + now.AddDate(0, 0, 1).Format("2006-01-02") + ", Y: " + now.AddDate(0, 0, -1).Format("2006-01-02"),
		},
		{
			name:     "Custom Macro",
			content:  "Project: {{Project}}",
			expected: "Project: Nocti CLI",
		},
		{
			name:     "Custom Macro with Slug",
			content:  "Project Slug: {{Project|SLUG}}",
			expected: "Project Slug: nocti-cli",
		},
		{
			name:     "Unknown Macro",
			content:  "Unknown: {{UNKNOWN}}",
			expected: "Unknown: {{UNKNOWN}}",
		},
		{
			name:     "Spaces in Macro",
			content:  "Spaces: {{ NAME | UPPER }}",
			expected: "Spaces: MY MEETING NOTES",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cmd.ReplaceMacros(tt.content, macros)
			if got != tt.expected {
				t.Errorf("ReplaceMacros() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseTemplateFrontmatter(t *testing.T) {
	input := `Project: Nocti CLI
Attendees: @user1, @user2
---
# Meeting: {{NAME}}
Date: {{DATE}}`

	metadata, body := cmd.ParseTemplateFrontmatter(input)

	if metadata["Project"] != "Nocti CLI" {
		t.Errorf("Expected Project 'Nocti CLI', got %q", metadata["Project"])
	}
	if metadata["Attendees"] != "@user1, @user2" {
		t.Errorf("Expected Attendees '@user1, @user2', got %q", metadata["Attendees"])
	}
	if !strings.Contains(body, "# Meeting: {{NAME}}") {
		t.Errorf("Body missing content, got %q", body)
	}
}

func TestResolveIncrement(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "Task-1.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "Task-2.md"), []byte(""), 0644)

	pattern := "Task-{{INC}}"
	got := cmd.ResolveIncrement(pattern, tmpDir)
	expected := "Task-3"

	if got != expected {
		t.Errorf("ResolveIncrement() = %q, want %q", got, expected)
	}
}
