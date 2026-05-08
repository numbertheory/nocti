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
		dayStr := fmt.Sprintf("%s %d", day.Month().String(), day.Day())
		dayPrefix := dayStr

		if multiYear {
			if day.Year() != currentYear {
				currentYear = day.Year()
				days = append(days, fmt.Sprintf("%d%c", currentYear, os.PathSeparator))
			}
			// Indent days under year folders
			dayPrefix = fmt.Sprintf("%d%c%s", day.Year(), os.PathSeparator, dayStr)
		}

		days = append(days, dayPrefix+string(os.PathSeparator))

		// Check for date-specific folder: YYYY-MM-DD
		dateFolder := day.Format("2006-01-02")
		dateFolderPath := filepath.Join(searchDir, dateFolder)
		if info, err := os.Stat(dateFolderPath); err == nil && info.IsDir() {
			// Read .nocti.json in date folder
			dfConfigPath := filepath.Join(dateFolderPath, ".nocti.json")
			if dfData, err := os.ReadFile(dfConfigPath); err == nil {
				var dfConfig struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(dfData, &dfConfig); err == nil && dfConfig.Type == "event" {
					// Scan contents of date folder
					dfEntries, err := os.ReadDir(dateFolderPath)
					if err == nil {
						for _, dfe := range dfEntries {
							if dfe.Name() == ".nocti.json" {
								continue
							}
							// Add as a child of the day
							// We use the day's display path (dayPrefix) as the parent path
							// But we need to make sure BuildDisplayEntries handles this.
							// BuildDisplayEntries builds based on path separators.
							// So if dayPrefix is "May 06", we return "May 06/filename"
							resultsPath := filepath.Join(dayPrefix, dfe.Name())
							if dfe.IsDir() {
								resultsPath += string(os.PathSeparator)
							}
							days = append(days, resultsPath)
						}
					}
				}
			}
		}
	}

	// Scan for nested resources (folders with .nocti.json)
	var subResources []string
	entries, err := os.ReadDir(searchDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				resConfigPath := filepath.Join(searchDir, entry.Name(), ".nocti.json")
				if data, err := os.ReadFile(resConfigPath); err == nil {
					var resConfig struct {
						Type string `json:"type"`
					}
					if err := json.Unmarshal(data, &resConfig); err == nil {
						// Exclude "event" type resources (they are processed within the days loop)
						if resConfig.Type != "event" {
							subResources = append(subResources, entry.Name()+string(os.PathSeparator))
						}
					}
				}
			}
		}
		sort.Strings(subResources)
	}

	if config.ResourcesFirst {
		results = append(results, subResources...)
		if showHidden {
			results = append(results, ".nocti.json")
		}
		results = append(results, days...)
	} else {
		if showHidden {
			results = append(results, ".nocti.json")
		}
		results = append(results, days...)
		results = append(results, subResources...)
	}

	return results, nil
}
