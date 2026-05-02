# `nocti list`

The `list` command displays the content of a notebook resource.

## Usage

```bash
nocti list [resource-name]
```

## Description

The `list` command recursively scans a directory and identifies all `.md` and `.txt` files. It is specifically designed to work with **notebook** resources.

### Constraints and Behavior

*   **Markdown and Text Only**: Only files with `.md` or `.txt` extensions are listed.
*   **Resource Boundaries**: If the scan encounters a subdirectory that is its own Nocti resource (contains a `.nocti.json` file), it will **not** recurse into that directory.
*   **Git Ignored**: The `.git` directory is automatically skipped.
*   **Context Aware**: 
    *   If run without arguments inside a notebook resource, it lists files in the current directory and its subdirectories.
    *   If run with a `resource-name` argument (from a parent directory), it lists files within that specific resource.

## Customization

The interactive TUI can be customized via the `colors` key in your `.nocti/nocti.json` or a local `.nocti.json` file.

### Available Color Keys

*   `file_list`: Background color for the file list header.
*   `preview_pane`: Background color for the preview pane header.
*   `help_bg`: Background color of the help modal.
*   `help_fg`: Foreground (text) color of the help modal.
*   `help_border_bg`: Background color for the help modal border.
*   `help_border_fg`: Foreground (line) color for the help modal border.

### Example Configuration

```json
{
  "colors": {
    "file_list": "blue",
    "preview_pane": "orange",
    "help_bg": "darkgray",
    "help_fg": "white",
    "help_border_bg": "black",
    "help_border_fg": "gray"
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
