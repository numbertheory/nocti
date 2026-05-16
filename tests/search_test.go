package tests

import (
	"encoding/json"
	"nocti/cmd"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPerformSearch_Notebook(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-search-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a notebook
	config := struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}{Name: "My Notebook", Type: "notebook"}
	data, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), data, 0644)

	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "note1.md"), []byte("apple banana apple"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "note2.md"), []byte("banana cherry"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "note3.md"), []byte("apple cherry cherry cherry"), 0644)

	// Test case 1: Search for "apple"
	results, err := cmd.PerformSearch(tmpDir, "notebook", false, []string{"apple"}, false)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results for 'apple', got %d", len(results))
	}
	// results[0].Path is now absolute
	if filepath.Base(results[0].Path) != "note1.md" || results[0].Score != 2 {
		t.Errorf("First result should be note1.md with score 2, got %s (score %d)", results[0].Path, results[0].Score)
	}

	// Test case 2: Search for "apple cherry"
	results, err = cmd.PerformSearch(tmpDir, "notebook", false, []string{"apple", "cherry"}, false)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result for 'apple cherry', got %d", len(results))
	}
	if filepath.Base(results[0].Path) != "note3.md" {
		t.Errorf("Expected note3.md, got %s", results[0].Path)
	}
}

func TestPerformSearch_Newest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-search-newest-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a notebook
	config := struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}{Name: "My Notebook", Type: "notebook"}
	data, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), data, 0644)

	// Create files with different modification times
	f1 := filepath.Join(tmpDir, "old.md")
	f2 := filepath.Join(tmpDir, "new.md")

	os.WriteFile(f1, []byte("test content"), 0644)
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	os.WriteFile(f2, []byte("test content"), 0644)

	// Search with newestFirst = true
	results, err := cmd.PerformSearch(tmpDir, "notebook", false, []string{"test"}, true)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if filepath.Base(results[0].Path) != "new.md" {
		t.Errorf("Expected newest first (new.md), got %s", results[0].Path)
	}
}

func TestPerformSearch_Scoping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-search-scope-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Project Structure:
	// / (Project Root)
	//   note_root.md (contains "target")
	//   /notebook1
	//     note_nb1.md (contains "target")

	os.Mkdir(filepath.Join(tmpDir, ".nocti"), 0755)
	projConfig := struct{ Name string }{Name: "Project"}
	projData, _ := json.Marshal(projConfig)
	os.WriteFile(filepath.Join(tmpDir, ".nocti", "nocti.json"), projData, 0644)

	os.WriteFile(filepath.Join(tmpDir, "note_root.md"), []byte("target keyword"), 0644)

	nb1Dir := filepath.Join(tmpDir, "notebook1")
	os.Mkdir(nb1Dir, 0755)
	nbConfig := struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}{Name: "NB1", Type: "notebook"}
	nbData, _ := json.Marshal(nbConfig)
	os.WriteFile(filepath.Join(nb1Dir, ".nocti.json"), nbData, 0644)
	os.WriteFile(filepath.Join(nb1Dir, "note_nb1.md"), []byte("target keyword"), 0644)

	// 1. Search from Project Root
	results, err := cmd.PerformSearch(tmpDir, "", true, []string{"target"}, false)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Project root search: expected 2 results, got %d", len(results))
	}

	// 2. Search from Notebook 1
	results, err = cmd.PerformSearch(nb1Dir, "notebook", false, []string{"target"}, false)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Notebook search: expected 1 result, got %d", len(results))
	}
	if filepath.Base(results[0].Path) != "note_nb1.md" {
		t.Errorf("Notebook search: expected note_nb1.md, got %s", results[0].Path)
	}
}

func TestPerformSearch_WholeWordAndSmartCase(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-search-regex-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), []byte(`{"type":"notebook"}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "content.md"), []byte("Mina was here. Determination is key. mina is small."), 0644)

	// 1. Search "Mina" (Case-sensitive because of 'M')
	results, _ := cmd.PerformSearch(tmpDir, "notebook", false, []string{"Mina"}, false)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result for 'Mina', got %d", len(results))
	}
	if len(results[0].Matches) != 1 {
		t.Errorf("Expected 1 match for 'Mina', got %d", len(results[0].Matches))
	}
	if !strings.Contains(results[0].Matches[0].Text, "Mina") {
		t.Errorf("Match should contain 'Mina', got %s", results[0].Matches[0].Text)
	}

	// 2. Search "mina" (Case-insensitive)
	results, _ = cmd.PerformSearch(tmpDir, "notebook", false, []string{"mina"}, false)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result for 'mina', got %d", len(results))
	}
	// Should match both "Mina" and "mina" as whole words
	if len(results[0].Matches) != 1 { // They are on the same line in this test
		t.Logf("Matches: %+v", results[0].Matches)
	}
	// "mina" is matched 2 times in the score, but we see 1 line
	if results[0].Score != 2 {
		t.Errorf("Expected score 2 for 'mina', got %d", results[0].Score)
	}

	// 3. Search "nation" (Should NOT match "determination" because of whole-word)
	results, _ = cmd.PerformSearch(tmpDir, "notebook", false, []string{"nation"}, false)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'nation' (substring of determination), got %d", len(results))
	}
}
