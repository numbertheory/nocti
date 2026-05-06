package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ScanCalendarDays(searchDir string, showHidden bool) ([]string, error) {
	var results []string

	// Read daysLength from .nocti.json
	configPath := filepath.Join(searchDir, ".nocti.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read calendar config: %w", err)
	}

	var config struct {
		DaysLength int `json:"daysLength"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse calendar config: %w", err)
	}

	if config.DaysLength <= 0 {
		config.DaysLength = 30
	}

	for i := 1; i <= config.DaysLength; i++ {
		results = append(results, fmt.Sprintf("Day %d", i))
	}

	if showHidden {
		results = append(results, ".nocti.json")
	}

	return results, nil
}
