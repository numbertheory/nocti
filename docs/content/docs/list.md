# `nocti list`

The `list` command displays the content of a notebook resource.

## Usage

```bash
nocti list [resource-name]
```

## Description

The `list` command recursively scans a directory and identifies all `.md` and `.txt` files. It is specifically designed to work with **notebook** resources.

### Interactive Creation

While in interactive mode, you can press **n** to create new files, folders, or sub-resources. If you have defined **templates** for the current resource, they will also appear in this menu. See the [Templates]({{< ref "templates.md" >}}) page for more details.

### Constraints and Behavior

*   **Markdown and Text Only**: Only files with `.md` or `.txt` extensions are listed.
*   **Highlighting**: You can add custom background and foreground colors to text in your files using a special syntax that the preview pane interprets.
    *   **Background only**: `[:background-color: Your text here]`
    *   **Foreground and Background**: `[:foreground-color:background-color: Your text here]`
    *   Example: `[:black:yellow: This is important!]`
    *   Colors match the names available for [Customization](#available-color-keys).
*   **Resource Boundaries**: If the scan encounters a subdirectory that is its own Nocti resource (contains a `.nocti.json` file), it will **not** recurse into that directory.
*   **Git Ignored**: The `.git` directory is automatically skipped.
*   **Toggling Line Numbers**: In the interactive view, you can press **Ctrl+L** to toggle line numbers in the preview pane.
*   **Link Navigation**: When the preview pane is in focus (press **Tab** to switch focus), you can:
    *   **Shift+Tab**: Cycle through all detected web links (`http://` or `https://`) in the preview content. The selected link will be highlighted.
    *   **Enter**: Open the currently selected link in your system's default web browser.
*   **Context Aware**: 
    *   If run without arguments inside a notebook resource, it lists files in the current directory and its subdirectories.
    *   If run with a `resource-name` argument (from a parent directory), it lists files within that specific resource.

## Customization

The interactive TUI can be customized via the `colors` key in your `.nocti/nocti.json` or a local `.nocti.json` file.

### Available Color Keys

*   `file_list_bg`: Background color for the file list header.
*   `file_list_fg`: Foreground color for the file list header.
*   `preview_pane_bg`: Background color for the preview pane header.
*   `preview_pane_fg`: Foreground color for the preview pane header.
*   `help_bg`: Background color of the help modal.
*   `help_fg`: Foreground (text) color of the help modal.
*   `help_border_bg`: Background color for the help modal border.
*   `help_border_fg`: Foreground (line) color for the help modal border.
*   `nocti_notebook_fg`: Background color for the nocti notebooks in the file view.
*   `nocti_notebook_bg`: Foreground color for the nocti notebooks in the file view.
*   `nocti_calendar_fg`: Background color for the nocti calendars in the file view.
*   `nocti_calendar_bg`: Foreground color for the nocti calendars in the file view.
*   `nocti_todo_fg`: Background color for the nocti todos in the file view.
*   `nocti_todo_bg`: Foreground color for the nocti todos in the file view.

### Example Configuration

```json
{
  "colors": {
    "file_list_bg": "blue",
    "file_list_fg": "white",
    "preview_pane_bg": "orange",
    "preview_pane_fg": "white"
    "help_bg": "darkgray",
    "help_fg": "white",
    "help_border_bg": "black",
    "help_border_fg": "gray",
    "nocti_notebook_fg": "cyan",
    "nocti_notebook_bg": "default",
    "nocti_calendar_fg": "magenta",
    "nocti_calendar_bg": "default",
    "nocti_todo_fg": "green",
    "nocti_todo_bg": "default",
  }
}
```

Supported color names include standard terminal colors like `red`, `green`, `blue`, `yellow`, `magenta`, `cyan`, `white`, `black`, and extended ones like `darkgray`, `lightgray`, `orange`, `purple`, etc.

## Examples

List files in the current notebook:
```bash
cd my-notebook
nocti list
```

List files in a specific notebook from the project root:
```bash
nocti list "Personal Journal"
```
