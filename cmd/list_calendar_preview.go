package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

func GetCalendarDayPreview(relPath string, baseDir string) []string {
	// relPath can be "Month Day" or "Year/Month Day"

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
		// "2024/May 1"
		t, err = time.Parse("2006/January 2", relPath)
	} else {
		// "May 1"
		t, err = time.Parse("January 2", relPath)
		if err == nil {
			t = time.Date(defaultYear, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
	}

	if err != nil {
		return []string{"[ Calendar Day ]", relPath, err.Error()}
	}

	year := t.Year()
	yDay := t.YearDay()

	// Days left in year
	daysInYear := 365
	if isLeap(year) {
		daysInYear = 366
	}
	daysLeft := daysInYear - yDay

	// Quarter
	quarter := (int(t.Month())-1)/3 + 1

	// Holidays
	c := cal.NewBusinessCalendar()
	c.AddHoliday(us.Holidays...)

	actual, observed, holiday := c.IsHoliday(t)

	holidayStr := "None"
	if actual || observed {
		holidayStr = holiday.Name
	}

	return []string{
		fmt.Sprintf("Date:        %s", t.Format("2006-01-02")),
		fmt.Sprintf("Day of Year: %d", yDay),
		fmt.Sprintf("Days Left:   %d", daysLeft),
		fmt.Sprintf("Quarter:     Q%d", quarter),
		fmt.Sprintf("Day of Week: %s", t.Weekday().String()),
		fmt.Sprintf("Holidays:    %s", holidayStr),
	}
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
