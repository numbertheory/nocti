---
title: "Search"
icon: "search"
weight: 1
---

The `nocti search` command allows you to perform powerful, context-aware keyword searches across your notes, calendars, and todo lists.

## Usage

```bash
nocti search <query> [flags]
```

### Context-Aware Scoping

`nocti search` automatically detects your current context and scopes the search accordingly:

- **Project Root:** If run from the root of your `nocti` project, it searches across all resources (notebooks, calendars, todos).
- **Resource Level:** If run from within a specific notebook, calendar, or todo list, it only searches files within that specific resource.
- **Subdirectories:** If run from a subfolder within a resource, it still searches the entire parent resource.

### Smart Matching

- **Whole Words:** Search only matches whole words. Searching for "Mina" will find "Mina" but will ignore "determination".
- **Smart Case-Sensitivity:**
    - If your query contains **any uppercase letters**, the search is **case-sensitive**.
    - If your query is **all lowercase**, the search is **case-insensitive**.
- **Multiple Keywords:** If you provide multiple words, `nocti` finds files that contain **all** of them.

## Flags

| Flag | Shorthand | Description |
| :--- | :--- | :--- |
| `--newest` | `-n` | Sort results by modification time (newest first) instead of relevance score. |
| `--json` | | Output results in raw JSON format for machine integration. |
| `--help` | `-h` | Display help for the search command. |

## Search Results

By default, results are ranked by a **relevance score** based on the frequency of the keywords in the file. Each result includes:

1. **Relevance Score or Date:** Depending on your flags.
2. **Absolute Path:** The full path to the file on your system.
3. **Matched Lines:** Every line containing your keywords, prefixed with its line number.

## Customization

### Colors

You can customize the search output colors in your `.nocti.json` file under the `colors` section:

| Setting | Default | Description |
| :--- | :--- | :--- |
| `search_highlight_fg` | Bold Yellow | Foreground color for the matched keyword. |
| `search_highlight_bg` | Default | Background color for the matched keyword. |
| `search_ln_fg` | Blue | Foreground color for the line numbers. |
| `search_ln_bg` | Default | Background color for the line numbers. |
| `search_file_fg` | Green | Foreground color for the filename. |
| `search_file_bg` | Default | Background color for the filename. |
| `search_score_fg` | Magenta | Foreground color for the score/date. |
| `search_score_bg` | Default | Background color for the score/date. |

### NO_COLOR Support

If you wish to disable all ANSI colors in the search output (e.g., for logging), set the `NO_COLOR` environment variable:

```bash
NO_COLOR=1 nocti search "my query"
```

## JSON Output

For developers building tools on top of `nocti`, the `--json` flag provides a structured representation of search hits:

```json
{
  "total_hits": 12,
  "file_count": 3,
  "results": [
    {
      "path": "/absolute/path/to/note.md",
      "score": 5,
      "updated_at": "2026-05-16T10:30:00Z",
      "matches": [
        { "line_no": 1, "text": "Match on first line" }
      ]
    }
  ]
}
```
