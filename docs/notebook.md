# Notebook Customization

You can customize the appearance of the `nocti list` command by defining colors in your main `.nocti/nocti.json` file.

## Color Configuration

Add an optional `colors` key at the top level of your `nocti.json` file:

```json
{
  "name": "my-project",
  "version": "development",
  "colors": {
    "file_list": "blue",
    "preview_pane": "orange"
  },
  ...
}
```

### Supported Colors

The following color names are supported (assuming your terminal has 256-color support):

| | | | |
| :--- | :--- | :--- | :--- |
| `black` | `red` | `green` | `yellow` |
| `blue` | `magenta` | `cyan` | `white` |
| `gray` | `darkgray` | `lightgray` | `silver` |
| `brightred` | `brightgreen` | `brightyellow` | `brightblue` |
| `brightmagenta` | `brightcyan` | `brightwhite` | `orange` |
| `darkorange` | `pink` | `hotpink` | `purple` |
| `violet` | `brown` | `navy` | `teal` |
| `olive` | `maroon` | `aqua` | `fuchsia` |
| `lime` | `skyblue` | `gold` | `indigo` |
| `coral` | `turquoise` | `plum` | `orchid` |
| `salmon` | | | |

### Default Colors

If no colors are defined, the following defaults are used:
*   `file_list`: `blue`
*   `preview_pane`: `orange`

## Previewing Files

When using `nocti list` inside a notebook, you can:
1.  Navigate the file list using **Up/Down arrow keys**.
2.  Switch focus to the preview pane using **TAB**.
3.  Scroll the preview content using **Up/Down arrow keys**, **PgUp**, or **PgDn** when the preview pane is focused.
4.  Exit the interactive mode by pressing **q**.
