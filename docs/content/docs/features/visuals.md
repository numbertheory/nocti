---
title: "Advanced Visuals"
icon: "palette"
weight: 3
---

Nocti supports several ways to add color and visual emphasis to your notes and tables.

## Inline Highlights

You can highlight specific words or phrases using the following syntax:

`[:bg_color: text]` or `[:fg_color:bg_color: text]`

**Examples:**
- `[:yellow: This text has a yellow background]`
- `[:white:blue: This is white text on a blue background]`

---

## Table Coloring

You can apply colors to entire rows, columns, or individual cells in Markdown tables using special markers. 

Color priority is: **Cell > Row > Column**.

### Column Coloring
Define column-wide colors in the table's separator row using the `[:col:color:]` marker.

```markdown
| Header 1 | Header 2 |
| :--- [:col:blue:] | :--- [:col:cyan:] |
| Data 1 | Data 2 |
```

### Row Coloring
Color an entire row by placing the `[:row:color:]` marker at the beginning of the first cell.

```markdown
| [:row:red:] Row 1 Col 1 | Row 1 Col 2 |
| Row 2 Col 1 | Row 2 Col 2 |
```

### Cell Coloring
Color a specific cell using the `[:cell:color:]` marker at the start of the cell.

```markdown
| [:cell:green:] Special Cell | Normal Cell |
```

### Using Foreground and Background
For all table markers, you can specify both foreground and background colors using the `fg:bg` format.

**Example:** `[:row:white:red:]` (White text on a red background for the whole row).

---

## Supported Colors

Nocti supports standard terminal colors and extended 256-color palette names:

- **Standard**: `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`
- **Extended**: `gray`, `darkgray`, `orange`, `pink`, `purple`, `teal`, `gold`, and many more.
- **Special**: `default` (resets to terminal default)
