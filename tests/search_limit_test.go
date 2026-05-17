package tests

import (
	"nocti/cmd"
	"os"
	"path/filepath"
	"testing"
)

func TestPerformSearch_Limit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-search-limit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), []byte(`{"type":"notebook"}`), 0644)

	// File with many matches on multiple lines
	content := "apple\napple\napple\napple\napple\n"
	os.WriteFile(filepath.Join(tmpDir, "many_apples.md"), []byte(content), 0644)

	// File with few matches
	os.WriteFile(filepath.Join(tmpDir, "few_apples.md"), []byte("apple\nbanana\n"), 0644)

	// 1. Search with no limit
	results, err := cmd.PerformSearch(tmpDir, "notebook", false, []string{"apple"}, false, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	for _, res := range results {
		if filepath.Base(res.Path) == "many_apples.md" {
			if len(res.Matches) != 5 {
				t.Errorf("many_apples.md: expected 5 matches, got %d", len(res.Matches))
			}
			if res.Score != 5 {
				t.Errorf("many_apples.md: expected score 5, got %d", res.Score)
			}
			if res.HiddenMatches != 0 {
				t.Errorf("many_apples.md: expected 0 hidden matches, got %d", res.HiddenMatches)
			}
		}
	}

	// 2. Search with limit = 2
	results, err = cmd.PerformSearch(tmpDir, "notebook", false, []string{"apple"}, false, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results (limit should not limit files), got %d", len(results))
	}
	for _, res := range results {
		if filepath.Base(res.Path) == "many_apples.md" {
			if len(res.Matches) != 2 {
				t.Errorf("many_apples.md: expected 2 matches (limited), got %d", len(res.Matches))
			}
			if res.Score != 2 {
				t.Errorf("many_apples.md: expected score 2 (limited), got %d", res.Score)
			}
			if res.HiddenMatches != 3 {
				t.Errorf("many_apples.md: expected 3 hidden matches, got %d", res.HiddenMatches)
			}
		} else if filepath.Base(res.Path) == "few_apples.md" {
			if len(res.Matches) != 1 {
				t.Errorf("few_apples.md: expected 1 match, got %d", len(res.Matches))
			}
			if res.Score != 1 {
				t.Errorf("few_apples.md: expected score 1, got %d", res.Score)
			}
			if res.HiddenMatches != 0 {
				t.Errorf("few_apples.md: expected 0 hidden matches, got %d", res.HiddenMatches)
			}
		}
	}
}
