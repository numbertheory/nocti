---
title: "Settings"
icon: "settings"
weight: 4
---

Nocti provides a full-screen settings panel to customize your experience directly from the interface.

## Accessing Settings

Press `Ctrl+S` while in the resource list to open the settings panel. Settings are context-aware:
- If you are in a **Resource** (Notebook, Todo, Calendar), settings will modify that resource's `.nocti.json`.
- If you are at the **Project Root**, settings will modify the global `.nocti/nocti.json`.

---

## Navigation

The settings panel is organized into three tabs. Use **TAB** to cycle through them.

### 1. Colors Tab
Customize the visual theme of the Nocti CLI.
- **Fields**: You can change background and foreground colors for the file list, preview pane, help modal, and more.
- **Reference Table**: A live preview of all [Supported Colors](../visuals/#supported-colors) is displayed on the right side for easy reference.
- **Usage**: Use `↑/↓` to select a field, press `ENTER` to edit, type the color name, and press `ENTER` again to confirm.

### 2. Editor Tab
Configure which terminal editor Nocti uses to open files (e.g., `nvim`, `vim`, `nano`, `code`).
- If no editor is set in your config, Nocti defaults to `nvim`.
- Saving a change here will explicitly add the `editor` key to your configuration file.

### 3. Save Tab
Explicitly manage your changes.
- **Save and Exit Settings**: Writes all modifications to the configuration file and returns to the list.
- **Don't Save and Exit Settings**: Discards all modifications made during this session and returns to the list.

---

## Shortcuts Summary

| Key | Action |
| :--- | :--- |
| `TAB` | Switch to the next tab (Colors → Editor → Save) |
| `↑ / ↓` | Navigate through fields or options |
| `ENTER` | Start editing a field or confirm an option |
| `q` or `ESC` | Exit settings immediately **without saving** |
| `Ctrl+S` | Open settings (from main list) |

---

## Configuration Files

Settings modified in this panel are persisted as JSON in your workspace. You can also edit these files manually:

**Resource Level (`.nocti.json`):**
```json
{
  "colors": {
    "file_list_bg": "blue",
    "preview_pane_fg": "white"
  },
  "editor": "vim"
}
```

**Project Level (`.nocti/nocti.json`):**
This file handles defaults for all resources within the project.
