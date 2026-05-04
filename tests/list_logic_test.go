package tests

import (
	"encoding/json"
	"nocti/cmd"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDisplayEntriesEmpty(t *testing.T) {
	files := []string{}
	entries := cmd.BuildDisplayEntries(files, ".")
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries for empty file list, got %d", len(entries))
	}
}

func TestBuildDisplayEntries(t *testing.T) {
	files := []string{
		"note1.md",
		"folder1/note2.txt",
		"folder1/subfolder/note3.md",
		"empty-folder" + string(os.PathSeparator),
	}

	entries := cmd.BuildDisplayEntries(files, ".")

	if len(entries) != 6 {
		t.Fatalf("Expected 6 entries, got %d", len(entries))
	}

	expectedOrder := []string{"empty-folder", "folder1", "note2.txt", "subfolder", "note3.md", "note1.md"}

	for i, name := range expectedOrder {
		if entries[i].Name != name {
			t.Errorf("Entry %d: expected name %s, got %s", i, name, entries[i].Name)
		}
	}
}

func TestScanNotebookFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-scan-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "note1.md"), []byte(""), 0644)
	os.Mkdir(filepath.Join(tmpDir, "empty-dir"), 0755)
	os.Chdir(tmpDir) // WalkDir behavior might depend on path relative to searchDir

	os.Mkdir(filepath.Join(tmpDir, "dir-with-txt"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "dir-with-txt", "note2.txt"), []byte(""), 0644)
	os.Mkdir(filepath.Join(tmpDir, "dir-with-other"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "dir-with-other", "other.pdf"), []byte(""), 0644)
	os.Mkdir(filepath.Join(tmpDir, "sub-resource"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub-resource", ".nocti.json"), []byte(`{"type":"notebook"}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sub-resource", "visible.md"), []byte(""), 0644)

	os.Mkdir(filepath.Join(tmpDir, "todo-resource"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "todo-resource", ".nocti.json"), []byte(`{"type":"todo"}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "todo-resource", "hidden-task.md"), []byte(""), 0644)

	os.Mkdir(filepath.Join(tmpDir, ".hidden-dir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".hidden-dir", "secret.md"), []byte(""), 0644)

	// Settings and templates
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), []byte("{}"), 0644)
	os.Mkdir(filepath.Join(tmpDir, ".templates"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".templates", "template.md"), []byte(""), 0644)

	files, err := cmd.ScanNotebookFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("ScanNotebookFiles failed: %v", err)
	}

	foundNote1 := false
	foundEmptyDir := false
	foundNote2 := false
	foundOtherDir := false
	foundSubResourceFile := false
	foundTodoResource := false
	foundTodoTask := false
	foundHiddenDir := false
	foundSettings := false
	foundTemplate := false

	for _, f := range files {
		if f == "note1.md" {
			foundNote1 = true
		} else if f == "empty-dir"+string(os.PathSeparator) {
			foundEmptyDir = true
		} else if f == "dir-with-txt/note2.txt" || f == "dir-with-txt\\note2.txt" {
			foundNote2 = true
		} else if filepath.Base(f) == "other.pdf" || strings.Contains(f, "dir-with-other") {
			foundOtherDir = true
		} else if f == "sub-resource/visible.md" || f == "sub-resource\\visible.md" {
			foundSubResourceFile = true
		} else if f == "todo-resource"+string(os.PathSeparator) {
			foundTodoResource = true
		} else if strings.Contains(f, "hidden-task.md") {
			foundTodoTask = true
		} else if strings.Contains(f, ".hidden-dir") {
			foundHiddenDir = true
		} else if f == ".nocti.json" {
			foundSettings = true
		} else if strings.Contains(f, ".templates") {
			foundTemplate = true
		}
	}

	if !foundNote1 {
		t.Error("note1.md not found")
	}
	if !foundEmptyDir {
		t.Error("empty-dir not found")
	}
	if !foundNote2 {
		t.Error("note2.txt not found")
	}
	if foundOtherDir {
		t.Error("dir-with-other should have been ignored")
	}
	if !foundSubResourceFile {
		t.Error("sub-resource/visible.md should have been found (notebook recursion)")
	}
	if !foundTodoResource {
		t.Error("todo-resource folder not found")
	}
	if foundTodoTask {
		t.Error("todo-resource content should have been ignored (non-notebook resource)")
	}
	if foundHiddenDir {
		t.Error(".hidden-dir should have been ignored")
	}
	if foundSettings {
		t.Error(".nocti.json should have been hidden")
	}
	if foundTemplate {
		t.Error(".templates should have been hidden")
	}

	// Test with showHidden = true
	filesHidden, _ := cmd.ScanNotebookFiles(tmpDir, true)
	foundSettingsHidden := false
	foundTemplateHidden := false
	foundHiddenDirStillHidden := false

	for _, f := range filesHidden {
		if f == ".nocti.json" {
			foundSettingsHidden = true
		} else if f == ".templates/template.md" || f == ".templates\\template.md" {
			foundTemplateHidden = true
		} else if strings.Contains(f, ".hidden-dir") {
			foundHiddenDirStillHidden = true
		}
	}

	if !foundSettingsHidden {
		t.Error(".nocti.json should be visible when showHidden is true")
	}
	if !foundTemplateHidden {
		t.Error(".templates/template.md should be visible when showHidden is true")
	}
	if foundHiddenDirStillHidden {
		t.Error(".hidden-dir should still be ignored even when showHidden is true")
	}
}

func TestFindEnclosingResource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-enclosing-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(parentDir, "child")
	os.MkdirAll(childDir, 0755)

	config := struct {
		Type string `json:"type"`
	}{Type: "notebook"}
	data, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(parentDir, ".nocti.json"), data, 0644)

	os.Chdir(childDir)

	foundPath, resType, err := cmd.FindEnclosingResource()
	if err != nil {
		t.Fatalf("FindEnclosingResource failed: %v", err)
	}

	if filepath.Base(foundPath) != "parent" {
		t.Errorf("Expected foundPath to be 'parent', got %s", foundPath)
	}
	if resType != "notebook" {
		t.Errorf("Expected resType 'notebook', got %s", resType)
	}
}

func TestColorHelpers(t *testing.T) {
	tests := []struct {
		name     string
		helper   func(string, string) string
		expected string
	}{
		{"blue", cmd.GetColorCode, "\033[48;5;4m"},
		{"BLUE", cmd.GetColorCode, "\033[48;5;4m"},
		{"unknown", cmd.GetColorCode, "default"},
		{"red", cmd.GetFGColorCode, "\033[38;5;1m"},
		{"RED", cmd.GetFGColorCode, "\033[38;5;1m"},
		{"unknown", cmd.GetFGColorCode, "default"},
	}

	for _, tt := range tests {
		result := tt.helper(tt.name, "default")
		if result != tt.expected {
			t.Errorf("%s: expected %s, got %s", tt.name, tt.expected, result)
		}
	}
}

func TestGetFilePreview(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-preview-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := "Line 1\nLine 2 is long\nLine 3"
	path := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(path, []byte(content), 0644)

	lines := cmd.GetFilePreview(path, 20)
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}
	if lines[1] != "Line 2 is long" {
		t.Errorf("Expected 'Line 2 is long', got '%s'", lines[1])
	}

	lines = cmd.GetFilePreview(path, 10)
	if lines[1] != "Line 2 is " {
		t.Errorf("Expected 'Line 2 is ', got '%s'", lines[1])
	}
}

func TestFindEnclosingResourceNegative(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-enclosing-neg-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	_, _, err = cmd.FindEnclosingResource()
	if err == nil {
		t.Error("Expected error when no enclosing resource found, got nil")
	}
}
