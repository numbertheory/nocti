package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

func GetDateFromRelPath(relPath string, baseDir string) (time.Time, error) {
	// Try to get year from .nocti.json
	defaultYear := time.Now().Year()
	configPath := filepath.Join(baseDir, ".nocti.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var config struct {
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal(data, &config); err == nil && config.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, config.CreatedAt); err == nil {
				defaultYear = t.Year()
			}
		}
	}

	var t time.Time
	var err error

	// Handle "Year/Month Day" or "Month Day"
	parts := strings.Split(relPath, string(os.PathSeparator))
	if len(parts) == 2 {
		t, err = time.Parse("2006/January 2", relPath)
	} else {
		t, err = time.Parse("January 2", relPath)
		if err == nil {
			t = time.Date(defaultYear, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
	}
	return t, err
}

func GetPhysicalPath(relPath string, baseDir string, currentResType string) string {
	if currentResType != "calendar" {
		return relPath
	}
	if relPath == ".nocti.json" || relPath == "nocti.json" || relPath == "." {
		return relPath
	}

	parts := strings.Split(strings.TrimSuffix(relPath, string(os.PathSeparator)), string(os.PathSeparator))
	if len(parts) == 0 {
		return relPath
	}

	dayRelPath := ""
	fileStartIndex := -1

	// Check if parts[0] is a year
	isYear := len(parts[0]) == 4 && parts[0][0] >= '0' && parts[0][0] <= '9'

	if isYear {
		if len(parts) >= 2 && strings.Contains(parts[1], " ") {
			dayRelPath = filepath.Join(parts[0], parts[1])
			fileStartIndex = 2
		} else {
			// Just a year folder
			return relPath
		}
	} else if strings.Contains(parts[0], " ") {
		dayRelPath = parts[0]
		fileStartIndex = 1
	} else {
		// Sub-resource at root level
		return relPath
	}

	if t, err := GetDateFromRelPath(dayRelPath, baseDir); err == nil {
		dateFolder := t.Format("2006-01-02")
		if fileStartIndex != -1 && fileStartIndex < len(parts) {
			phys := filepath.Join(dateFolder, filepath.Join(parts[fileStartIndex:]...))
			if strings.HasSuffix(relPath, string(os.PathSeparator)) {
				phys += string(os.PathSeparator)
			}
			return phys
		}
		phys := dateFolder
		if strings.HasSuffix(relPath, string(os.PathSeparator)) {
			phys += string(os.PathSeparator)
		}
		return phys
	}

	return relPath
}

func IsHoliday(relPath string, baseDir string) bool {
	t, err := GetDateFromRelPath(relPath, baseDir)
	if err != nil {
		return false
	}

	c := cal.NewBusinessCalendar()
	c.AddHoliday(us.Holidays...)
	actual, observed, _ := c.IsHoliday(t)
	return actual || observed
}
