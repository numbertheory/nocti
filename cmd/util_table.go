package cmd

import (
	"regexp"
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
		rowColor    string
		cellColors  []string
	}

	var currentTable []tableRow
	var colColors []string
	var tableColor string

	tableColorRe := regexp.MustCompile(`^\[:table:([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\]`)
	rowColorRe := regexp.MustCompile(`^\[:row:([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\]`)
	cellColorRe := regexp.MustCompile(`^\[:cell:([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\]`)
	colColorRe := regexp.MustCompile(`\[:col:([a-zA-Z0-9]+):(?:([a-zA-Z0-9]+):)?\]`)

	applyColors := func(fg, bg, content string) string {
		fgCode := ""
		if fg != "" {
			fgCode = GetFGColorCode(fg, "")
		}
		bgCode := ""
		if bg != "" {
			bgCode = GetColorCode(bg, "")
		}

		if fgCode == "" && bgCode == "" {
			return content
		}

		res := ""
		if fgCode != "" {
			res += fgCode
		}
		if bgCode != "" {
			res += bgCode
		}
		res += content
		if fgCode != "" {
			res += "\033[39m"
		}
		if bgCode != "" {
			res += "\033[49m"
		}
		return res
	}

	flushTable := func() {
		if len(currentTable) == 0 {
			return
		}

		// 1. First pass to parse column colors from any separator row
		// AND parse table-wide color if present in the first cell of any data row
		for _, row := range currentTable {
			if row.isSeparator {
				if colColors == nil {
					colColors = make([]string, len(row.cols))
				}
				for i, col := range row.cols {
					if m := colColorRe.FindStringSubmatch(col); m != nil {
						if m[2] != "" {
							colColors[i] = m[1] + ":" + m[2]
						} else {
							colColors[i] = m[1]
						}
					}
				}
			}
		}

		// Define border colors if tableColor is set
		var borderFG, borderBG string
		if tableColor != "" {
			cParts := strings.Split(tableColor, ":")
			if len(cParts) > 1 {
				borderFG, borderBG = cParts[0], cParts[1]
			} else {
				borderBG = cParts[0]
			}
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
		topLine := "┌" + strings.Join(topParts, "┬") + "┐"
		out = append(out, applyColors(borderFG, borderBG, topLine))

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
				sepLine := "├" + strings.Join(parts, "┼") + "┤"
				out = append(out, applyColors(borderFG, borderBG, sepLine))
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

					cellContent := " " + col + pad + " "

					// Apply Colors
					var fg, bg string
					// Priority: Cell > Row > Column > Table
					if i < len(row.cellColors) && row.cellColors[i] != "" {
						cParts := strings.Split(row.cellColors[i], ":")
						if len(cParts) > 1 {
							fg, bg = cParts[0], cParts[1]
						} else {
							bg = cParts[0]
						}
					} else if row.rowColor != "" {
						cParts := strings.Split(row.rowColor, ":")
						if len(cParts) > 1 {
							fg, bg = cParts[0], cParts[1]
						} else {
							bg = cParts[0]
						}
					} else if i < len(colColors) && colColors[i] != "" {
						cParts := strings.Split(colColors[i], ":")
						if len(cParts) > 1 {
							fg, bg = cParts[0], cParts[1]
						} else {
							bg = cParts[0]
						}
					} else if tableColor != "" {
						fg, bg = borderFG, borderBG
					}

					parts = append(parts, applyColors(fg, bg, cellContent))
				}
				rowText := applyColors(borderFG, borderBG, "│") + strings.Join(parts, applyColors(borderFG, borderBG, "│")) + applyColors(borderFG, borderBG, "│")
				out = append(out, rowText)
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
				botLine := "└" + strings.Join(botParts, "┴") + "┘"
				out = append(out, applyColors(borderFG, borderBG, botLine))
			}
		}

		currentTable = nil
		colColors = nil
		tableColor = ""
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
				// Ignore col markers when checking if separator
				cTrim = colColorRe.ReplaceAllString(cTrim, "")
				if strings.TrimSpace(cTrim) != "" {
					isSep = false
					break
				}
			}

			if isSep {
				currentTable = append(currentTable, tableRow{isSeparator: true, cols: cols})
			} else {
				// Parse table/row/cell colors
				var rowColor string
				cellColors := make([]string, len(cols))

				for i, col := range cols {
					cTrim := strings.TrimSpace(col)
					if i == 0 {
						if m := tableColorRe.FindStringSubmatch(cTrim); m != nil {
							if m[2] != "" {
								tableColor = m[1] + ":" + m[2]
							} else {
								tableColor = m[1]
							}
							cols[i] = strings.TrimSpace(tableColorRe.ReplaceAllString(cTrim, ""))
							cTrim = cols[i]
						}
						if m := rowColorRe.FindStringSubmatch(cTrim); m != nil {
							if m[2] != "" {
								rowColor = m[1] + ":" + m[2]
							} else {
								rowColor = m[1]
							}
							cols[i] = strings.TrimSpace(rowColorRe.ReplaceAllString(cTrim, ""))
							cTrim = cols[i]
						}
					}
					if m := cellColorRe.FindStringSubmatch(cTrim); m != nil {
						if m[2] != "" {
							cellColors[i] = m[1] + ":" + m[2]
						} else {
							cellColors[i] = m[1]
						}
						cols[i] = strings.TrimSpace(cellColorRe.ReplaceAllString(cTrim, ""))
					}
				}

				currentTable = append(currentTable, tableRow{
					isSeparator: false,
					cols:        cols,
					rowColor:    rowColor,
					cellColors:  cellColors,
				})
			}
		} else {
			flushTable()
			out = append(out, line)
		}
	}
	flushTable()

	return out
}
