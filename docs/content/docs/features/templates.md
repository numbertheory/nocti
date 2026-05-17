---
title: "Templates"
icon: "description"
weight: 2
---

Templates allow you to create new pages with pre-filled content and dynamic metadata. They are specific to each resource (notebook, todo, or calendar) and are stored within the resource itself.

## Setup

To use templates in a resource, create a hidden directory named `.templates` at the root of that resource.

```bash
cd my-notebook
mkdir .templates
```

Any markdown (`.md`) files you place in this folder will automatically appear as options in the "New" dialog within `nocti list`.

## Creating from a Template

1.  Run `nocti list` inside your notebook.
2.  Press **n** to open the creation menu.
3.  Select your template from the list.
4.  Enter a name for the new file (or use the pre-filled one).

## Dynamic Content

Templates support placeholders that are automatically replaced when a new file is created.

### Built-in Variables

| Variable | Description |
| :--- | :--- |
| `{{NAME}}` | The name you provide for the new file (without extension). |
| `{{DATE}}` | The current date in `YYYY-MM-DD` format. |
| `{{TIME}}` | The current time in `HH:MM:SS` format. |
| `{{DATETIME}}` | Full timestamp (`YYYY-MM-DD HH:MM:SS`). |
| `{{UUID}}` | Generates a standard version 4 UUID. |
| `{{SHORT_ID}}` | Generates a random 8-character alphanumeric ID. |

### Date & Time Macros

For more granular control, you can use these individual components:

| Variable | Description | Example |
| :--- | :--- | :--- |
| `{{DAY}}` | Day of the month (01-31) | `16` |
| `{{MONTH}}` | Month number (01-12) | `05` |
| `{{YEAR}}` | Full year | `2026` |
| `{{WEEKDAY}}` | Full name of the day | `Saturday` |
| `{{TOMORROW}}` | Tomorrow's date | `2026-05-17` |
| `{{YESTERDAY}}` | Yesterday's date | `2026-05-15` |

### String Modifiers

You can transform any variable by adding a modifier after a pipe `|` character.

| Modifier | Description | Example (`{{NAME}}` = "Project X") |
| :--- | :--- | :--- |
| `LOWER` | Converts to lowercase | `{{NAME\|LOWER}}` -> `project x` |
| `UPPER` | Converts to uppercase | `{{NAME\|UPPER}}` -> `PROJECT X` |
| `SLUG` | Converts to a URL-friendly slug | `{{NAME\|SLUG}}` -> `project-x` |

### Custom Date Formatting

Date-related macros support custom formatting using the `+` prefix and Linux-style format specifiers.

**Example:** `{{TIME\|+%-I:%M%P}}` -> `6:26pm`

| Specifier | Description |
| :--- | :--- |
| `%Y`, `%y` | Full year / short year |
| `%m`, `%-m` | Month (01-12) / (1-12) |
| `%B`, `%b` | Full / abbreviated month name |
| `%d`, `%-d` | Day (01-31) / (1-31) |
| `%H`, `%I` | 24-hour / 12-hour clock |
| `%M`, `%S` | Minute / Second |
| `%p`, `%P` | Upper / lowercase AM/PM |
| `%A`, `%a` | Full / abbreviated weekday name |

### Frontmatter Variables

You can define custom metadata at the top of your template. Any key-value pair defined in the frontmatter can be used as a variable in the body.

The frontmatter section must end with a line containing at least three dashes (`---`).

**Example Template (`.templates/Meeting.md`):**
```markdown
Project: Nocti CLI
Attendees: @user1, @user2
---
# Meeting: {{NAME}}
Date: {{DATE}}
Project: {{Project}}
Attendees: {{Attendees}}

## Agenda
...
```

The frontmatter section itself is stripped from the final file.

## Advanced Naming

Templates can suggest or enforce specific naming conventions using the `filename` key in their frontmatter.

### Automated Date Naming
```markdown
filename: Log-{{DATE}}
---
# Daily Log for {{DATE}}
```
Selecting this template will pre-fill the naming prompt with `Log-2026-05-12`.

### Automated Incrementers
The `{{INC}}` variable automatically finds the next available number in the current directory.

```markdown
filename: Task-{{INC}}
---
# Task {{INC}}
```
If `Task-1.md` and `Task-2.md` already exist in the folder, selecting this template will pre-fill the naming prompt with `Task-3`.
