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

func IsHoliday(relPath string, baseDir string) bool {
	// Try to get year from .nocti.json if not in relPath
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

	if err != nil {
		return false
	}

	c := cal.NewBusinessCalendar()
	c.AddHoliday(us.Holidays...)
	actual, observed, _ := c.IsHoliday(t)
	return actual || observed
}
