package tests

import (
	"nocti/cmd"
	"os"
	"path/filepath"
	"testing"
)

func TestScanCalendarDays(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nocti-calendar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := `{"type":"calendar", "daysLength": 5}`
	os.WriteFile(filepath.Join(tmpDir, ".nocti.json"), []byte(config), 0644)

	// Test default scan
	days, err := cmd.ScanCalendarDays(tmpDir, false)
	if err != nil {
		t.Fatalf("ScanCalendarDays failed: %v", err)
	}

	if len(days) != 5 {
		t.Fatalf("Expected 5 days, got %d", len(days))
	}
	if days[0] != "Day 1" || days[4] != "Day 5" {
		t.Errorf("Days incorrect: %v", days)
	}

	// Test with showHidden
	daysHidden, err := cmd.ScanCalendarDays(tmpDir, true)
	if err != nil {
		t.Fatalf("ScanCalendarDays failed with showHidden: %v", err)
	}

	if len(daysHidden) != 6 {
		t.Fatalf("Expected 6 entries with hidden, got %d", len(daysHidden))
	}
	foundJson := false
	for _, d := range daysHidden {
		if d == ".nocti.json" {
			foundJson = true
		}
	}
	if !foundJson {
		t.Error(".nocti.json not found in hidden scan")
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

	if len(days) != 30 {
		t.Fatalf("Expected default 30 days, got %d", len(days))
	}
}
