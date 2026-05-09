package tests

import (
	"encoding/json"
	"nocti/cmd"
	"os"
	"path/filepath"
	"testing"
)

func TestScanTodoItems(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-todo-scan-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "task2.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "task1.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte(""), 0644) // Should be ignored unless .md

	os.Mkdir(filepath.Join(tmpDir, "sub-notebook"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub-notebook", ".nocti.json"), []byte(`{"type":"notebook"}`), 0644)

	os.Mkdir(filepath.Join(tmpDir, "regular-folder"), 0755)

	files, err := cmd.ScanTodoItems(tmpDir, false)
	if err != nil {
		t.Fatalf("ScanTodoItems failed: %v", err)
	}

	// Expected order: task1.md, task2.md, sub-notebook/
	if len(files) != 3 {
		t.Fatalf("Expected 3 items, got %d: %v", len(files), files)
	}

	if files[0] != "task1.md" {
		t.Errorf("Expected first item task1.md, got %s", files[0])
	}
	if files[1] != "task2.md" {
		t.Errorf("Expected second item task2.md, got %s", files[1])
	}
	if files[2] != "sub-notebook"+string(os.PathSeparator) {
		t.Errorf("Expected third item sub-notebook/, got %s", files[2])
	}
}

func TestBuildDisplayEntriesForTodo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-todo-build-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}{Name: "My Todo", Type: "todo"}
	data, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), data, 0644)

	os.Mkdir(filepath.Join(tmpDir, "sub-notebook"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub-notebook", ".nocti.json"), []byte(`{"type":"notebook", "name":"Sub Notebook"}`), 0644)

	files := []string{
		"task1.md",
		"sub-notebook" + string(os.PathSeparator),
	}

	entries := cmd.BuildDisplayEntries(files, tmpDir, true, true, "todo")

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries (root + task + sub-notebook), got %d", len(entries))
	}

	// Root
	if entries[0].Name != "My Todo" || entries[0].Depth != 0 || entries[0].ResourceType != "todo" {
		t.Errorf("Root entry incorrect: %+v", entries[0])
	}

	// task1.md
	if entries[1].Name != "task1.md" || entries[1].Depth != 1 {
		t.Errorf("Task entry incorrect: %+v", entries[1])
	}

	// sub-notebook/
	if entries[2].Name != "Sub Notebook" || entries[2].Depth != 0 || entries[2].ResourceType != "notebook" {
		t.Errorf("Sub-notebook entry incorrect: %+v", entries[2])
	}
}

func TestGetTaskStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-todo-status-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `
# My Tasks
- [ ] Task 1
- [x] Task 2
- [X] Task 3
- [?] Task 4
- [ ] Task 5
Not a task
  - [ ] Indented task
`
	path := filepath.Join(tmpDir, "tasks.md")
	os.WriteFile(path, []byte(content), 0644)

	done, total := cmd.GetTaskStatus(path)

	// Total should be 6 (Task 1, 2, 3, 4, 5, and Indented task)
	if total != 6 {
		t.Errorf("Expected 6 total tasks, got %d", total)
	}

	// Done should be 3 (Task 2, 3, 4 - any non-space is done)
	if done != 3 {
		t.Errorf("Expected 3 done tasks, got %d", done)
	}
}
