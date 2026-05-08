package cmd

import (
	"fmt"

	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

func GetCalendarDayPreview(relPath string, baseDir string) []string {
	t, err := GetDateFromRelPath(relPath, baseDir)
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

	// Week of year
	_, week := t.ISOWeek()

	// Holidays
	c := cal.NewBusinessCalendar()
	c.AddHoliday(us.Holidays...)

	actual, observed, holiday := c.IsHoliday(t)

	holidayStr := ""
	if actual || observed {
		holidayStr = holiday.Name
	}

	lines := []string{
		fmt.Sprintf("Date:         %s", t.Format("2006-01-02")),
		fmt.Sprintf("Day of Year:  %d", yDay),
		fmt.Sprintf("Week of Year: %d", week),
		fmt.Sprintf("Days Left:    %d", daysLeft),
		fmt.Sprintf("Quarter:      Q%d", quarter),
		fmt.Sprintf("Day of Week:  %s", t.Weekday().String()),
	}

	if holidayStr != "" {
		lines = append(lines, fmt.Sprintf("Holidays:    %s", holidayStr))
	}

	return lines
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
