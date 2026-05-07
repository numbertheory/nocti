package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		DaysLength     int    `json:"daysLength"`
		CreatedAt      string `json:"created_at"`
		ResourcesFirst bool   `json:"resources_first"`
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

	var days []string
	currentYear := -1
	for i := -config.DaysLength; i <= config.DaysLength; i++ {
		day := centerDate.AddDate(0, 0, i)

		if multiYear {
			if day.Year() != currentYear {
				currentYear = day.Year()
				days = append(days, fmt.Sprintf("%d%c", currentYear, os.PathSeparator))
			}
			// Indent days under year folders
			days = append(days, fmt.Sprintf("%d%c%s %d", day.Year(), os.PathSeparator, day.Month().String(), day.Day()))
		} else {
			days = append(days, fmt.Sprintf("%s %d", day.Month().String(), day.Day()))
		}
	}

	// Scan for nested resources (folders with .nocti.json)
	var subResources []string
	entries, err := os.ReadDir(searchDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				if _, err := os.Stat(filepath.Join(searchDir, entry.Name(), ".nocti.json")); err == nil {
					subResources = append(subResources, entry.Name()+string(os.PathSeparator))
				}
			}
		}
		sort.Strings(subResources)
	}

	if config.ResourcesFirst {
		results = append(results, subResources...)
		results = append(results, days...)
	} else {
		results = append(results, days...)
		results = append(results, subResources...)
	}

	if showHidden {
		results = append([]string{".nocti.json"}, results...)
	}

	return results, nil
}
