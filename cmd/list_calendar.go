package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func ScanCalendarDays(searchDir string, showHidden bool) ([]string, error) {
	var results []string

	// Read config from .nocti.json
	configPath := filepath.Join(searchDir, ".nocti.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read calendar config: %w", err)
	}

	var config struct {
		DaysLength int    `json:"daysLength"`
		CreatedAt  string `json:"created_at"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse calendar config: %w", err)
	}

	if config.DaysLength <= 0 {
		config.DaysLength = 30
	}

	centerDate := time.Now()
	if config.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, config.CreatedAt); err == nil {
			centerDate = t
		}
	}

	startDate := centerDate.AddDate(0, 0, -config.DaysLength)
	endDate := centerDate.AddDate(0, 0, config.DaysLength)

	multiYear := startDate.Year() != endDate.Year()

	currentYear := -1
	for i := -config.DaysLength; i <= config.DaysLength; i++ {
		day := centerDate.AddDate(0, 0, i)

		if multiYear {
			if day.Year() != currentYear {
				currentYear = day.Year()
				results = append(results, fmt.Sprintf("%d%c", currentYear, os.PathSeparator))
			}
			// Indent days under year folders
			results = append(results, fmt.Sprintf("%d%c%s %d", day.Year(), os.PathSeparator, day.Month().String(), day.Day()))
		} else {
			results = append(results, fmt.Sprintf("%s %d", day.Month().String(), day.Day()))
		}
	}

	if showHidden {
		results = append(results, ".nocti.json")
	}

	return results, nil
}
