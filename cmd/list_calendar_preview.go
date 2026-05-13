package cmd

import (
	"fmt"

	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

func GetCalendarDayPreview(relPath string, baseDir string) []PreviewLine {
	t, err := GetDateFromRelPath(relPath, baseDir)
	if err != nil {
		return []PreviewLine{
			{Text: "[ Calendar Day ]", LineNo: 1},
			{Text: relPath, LineNo: 2},
			{Text: err.Error(), LineNo: 3},
		}
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

	lines := []PreviewLine{
		{Text: fmt.Sprintf("Date:         %s", t.Format("2006-01-02")), LineNo: 1},
		{Text: fmt.Sprintf("Day of Year:  %d", yDay), LineNo: 2},
		{Text: fmt.Sprintf("Week of Year: %d", week), LineNo: 3},
		{Text: fmt.Sprintf("Days Left:    %d", daysLeft), LineNo: 4},
		{Text: fmt.Sprintf("Quarter:      Q%d", quarter), LineNo: 5},
		{Text: fmt.Sprintf("Day of Week:  %s", t.Weekday().String()), LineNo: 6},
	}

	if holidayStr != "" {
		lines = append(lines, PreviewLine{Text: fmt.Sprintf("Holidays:    %s", holidayStr), LineNo: 7})
	}

	return lines
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
