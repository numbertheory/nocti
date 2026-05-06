package tests

import (
	"encoding/json"
	"nocti/cmd"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanCalendarDaysChronological(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-calendar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	createdAt := time.Date(2024, time.May, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	config := map[string]interface{}{
		"type":       "calendar",
		"daysLength": 1,
		"created_at": createdAt,
	}
	data, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), data, 0644)

	// daysLength 1 should give -1, 0, 1 -> 3 days
	days, err := cmd.ScanCalendarDays(tmpDir, false)
	if err != nil {
		t.Fatalf("ScanCalendarDays failed: %v", err)
	}

	if len(days) != 3 {
		t.Fatalf("Expected 3 days, got %d", len(days))
	}

	expectedDays := []string{
		"April 30",
		"May 1",
		"May 2",
	}

	for i, d := range days {
		if d != expectedDays[i] {
			t.Errorf("Expected %s, got %s", expectedDays[i], d)
		}
	}
}

func TestScanCalendarDaysMultiYear(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-calendar-multiyear-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	createdAt := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	config := map[string]interface{}{
		"type":       "calendar",
		"daysLength": 1,
		"created_at": createdAt,
	}
	data, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), data, 0644)

	// Jan 1, 2024 centered with length 1: Dec 31, 2023 | Jan 1, 2024 | Jan 2, 2024
	// Should produce folders: 2023/, 2024/
	days, err := cmd.ScanCalendarDays(tmpDir, false)
	if err != nil {
		t.Fatalf("ScanCalendarDays failed: %v", err)
	}

	// 2023 folder, Dec 31, 2024 folder, Jan 1, Jan 2 -> 5 entries
	if len(days) != 5 {
		t.Fatalf("Expected 5 entries (2 folders + 3 days), got %d: %v", len(days), days)
	}

	expected := []string{
		"2023" + string(os.PathSeparator),
		"2023" + string(os.PathSeparator) + "December 31",
		"2024" + string(os.PathSeparator),
		"2024" + string(os.PathSeparator) + "January 1",
		"2024" + string(os.PathSeparator) + "January 2",
	}

	for i, d := range days {
		if d != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], d)
		}
	}

	// Verify BuildDisplayEntries preserves order and creates structure
	entries := cmd.BuildDisplayEntries(days, tmpDir, false, true)
	if len(entries) != 5 {
		t.Fatalf("Expected 5 entries from BuildDisplayEntries, got %d", len(entries))
	}

	if entries[0].Name != "2023" || entries[0].IsFile {
		t.Errorf("Expected 2023 folder, got %+v", entries[0])
	}
	if entries[1].Name != "December 31" || !entries[1].IsFile {
		t.Errorf("Expected December 31 file, got %+v", entries[1])
	}
}

func TestScanCalendarDaysDefaultLength(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-calendar-default-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := `{"type":"calendar"}` // daysLength missing
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), []byte(config), 0644)

	days, err := cmd.ScanCalendarDays(tmpDir, false)
	if err != nil {
		t.Fatalf("ScanCalendarDays failed: %v", err)
	}

	// Default 30 daysLength means 30 before, 30 after, and center -> 61 days
	// Since today is 2026, and default length is small, it won't be multiyear unless we are at year boundary.
	// But let's just check length.
	if len(days) < 61 {
		t.Fatalf("Expected at least 61 entries, got %d", len(days))
	}
}
