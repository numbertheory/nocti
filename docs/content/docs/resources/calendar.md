---
title: "Calendar"
icon: "calendar_month"
weight: 2
---

The Calendar resource in Nocti provides a chronological timeline for managing events and notes tied to specific dates.

## Overview

Unlike notebooks, which are primarily hierarchical folder structures, calendars are centered around a specific creation date and provide a linear view of time.

## Features

### Date-Based Listing
- **Center Date:** Calendars are automatically centered around their creation date.
- **Configurable Length:** By default, a calendar shows 30 days before and 30 days after its creation date (61 days total). This can be customized during creation.
- **Chronological Order:** Days are listed from earliest to latest, making it easy to see past and future contexts.
- **Year Grouping:** If a calendar spans multiple years, dates are automatically organized into year-based "folders" (e.g., `2023/`, `2024/`).

### Information Panel (Preview)
When you select a day in the calendar, the preview pane displays a detailed information panel:
- **Full Date:** Displays the date in `YYYY-MM-DD` format.
- **Day of Year:** Shows the day number (1–366).
- **Days Left:** Calculates the remaining days in the year.
- **Quarter:** Identifies which quarter of the year the day falls in (Q1–Q4).
- **Day of Week:** Shows the full name of the weekday.
- **Holidays:** Automatically identifies US Federal Holidays.

### Holiday Highlighting
Dates that are recognized as holidays are automatically highlighted in the list view to make them stand out.
- **Default Style:** Holidays are displayed with a **gold** foreground.
- **Dynamic Preview:** The "Holidays" field only appears in the information panel when a holiday is actually present for the selected day.

### Customization
You can customize the appearance of holidays and the calendar range by editing the `.nocti.json` file within the calendar resource.

#### Other Configuration Options
- `daysLength`: The number of days to show before and after the creation date (default is 30).
- `resources_first`: If set to `true`, nested nocti resources will be listed before the calendar heading and its days. By default, the calendar heading and days appear first.

Example `.nocti.json` configuration:
```json
{
  "name": "My Calendar",
  "type": "calendar",
  "daysLength": 30,
  "resources_first": true,
  "colors": {
    "calendar_holiday_fg": "hotpink",
    "calendar_holiday_bg": "default"
  }
}
```

### Navigation Shortcuts
Nocti provides specialized shortcuts for navigating long calendar lists:
- **`Home` / `End`**: Jump immediately to the beginning or end of the calendar.
- **`Ctrl + Up` / `Ctrl + Down`**: Advance or retreat the selector by exactly **7 days**, allowing for quick weekly navigation.
- **`Ctrl + T`**: Toggle the visibility of the hidden `.nocti.json` configuration file at the top of the list.

## Usage

### Creating a Calendar
To create a new calendar, run:
```bash
nocti new
```
And select **calendar** from the list. You will be prompted for a name and the number of days to include in the range.

### Viewing a Calendar
You can view a calendar by navigating into its directory and running:
```bash
nocti list
```
Or by selecting it from the project root list.

*Note: Pressing ENTER on a calendar day is currently a placeholder for future event management features.*
