package cmd

import (
	"strings"
)

// FormatTables takes raw lines from a markdown file, identifies table blocks,
// formats them with aligned columns and ASCII borders, and returns the modified lines.
// Lines that are not part of a table are returned unchanged.
func FormatTables(lines []string) []string {
	var out []string

	type tableRow struct {
		isSeparator bool
		cols        []string
	}

	var currentTable []tableRow

	flushTable := func() {
		if len(currentTable) == 0 {
			return
		}

		// Find max columns
		maxCols := 0
		for _, row := range currentTable {
			if len(row.cols) > maxCols {
				maxCols = len(row.cols)
			}
		}

		// Calculate max width per column
		colWidths := make([]int, maxCols)
		for _, row := range currentTable {
			if row.isSeparator {
				continue
			}
			for i, col := range row.cols {
				w := VisibleLenWithLinks(strings.TrimSpace(col))
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}

		// Render Top Border
		var topParts []string
		for i := 0; i < maxCols; i++ {
			w := colWidths[i]
			if w < 1 {
				w = 1
			}
			topParts = append(topParts, strings.Repeat("─", w+2))
		}
		out = append(out, "┌"+strings.Join(topParts, "┬")+"┐")

		// Render Rows
		for rIdx, row := range currentTable {
			if row.isSeparator {
				// Render separator
				var parts []string
				for i := 0; i < maxCols; i++ {
					w := colWidths[i]
					if w < 1 {
						w = 1
					}
					parts = append(parts, strings.Repeat("─", w+2))
				}
				out = append(out, "├"+strings.Join(parts, "┼")+"┤")
			} else {
				// Render data row
				var parts []string
				for i := 0; i < maxCols; i++ {
					col := ""
					if i < len(row.cols) {
						col = strings.TrimSpace(row.cols[i])
					}
					// pad
					w := VisibleLenWithLinks(col)
					pad := ""
					targetW := colWidths[i]
					if targetW < 1 {
						targetW = 1
					}
					if w < targetW {
						pad = strings.Repeat(" ", targetW-w)
					}
					parts = append(parts, " "+col+pad+" ")
				}
				out = append(out, "│"+strings.Join(parts, "│")+"│")
			}

			// If last row, add bottom border
			if rIdx == len(currentTable)-1 {
				var botParts []string
				for i := 0; i < maxCols; i++ {
					w := colWidths[i]
					if w < 1 {
						w = 1
					}
					botParts = append(botParts, strings.Repeat("─", w+2))
				}
				out = append(out, "└"+strings.Join(botParts, "┴")+"┘")
			}
		}

		currentTable = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			// Parse columns
			content := trimmed[1 : len(trimmed)-1]
			cols := strings.Split(content, "|")

			// Check if it's a separator
			isSep := true
			for _, col := range cols {
				cTrim := strings.TrimSpace(col)
				cTrim = strings.Trim(cTrim, ":-")
				if cTrim != "" {
					isSep = false
					break
				}
			}

			currentTable = append(currentTable, tableRow{
				isSeparator: isSep,
				cols:        cols,
			})
		} else {
			flushTable()
			out = append(out, line)
		}
	}
	flushTable()

	return out
}
