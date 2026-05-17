package cmd

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseTemplateFrontmatter separates frontmatter metadata from the template body.
func ParseTemplateFrontmatter(raw string) (map[string]string, string) {
	metadata := make(map[string]string)
	lines := strings.Split(raw, "\n")
	separatorIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) >= 3 && strings.Count(trimmed, "-") == len(trimmed) {
			separatorIdx = i
			break
		}
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			metadata[key] = val
		}
	}

	if separatorIdx == -1 {
		return metadata, raw
	}

	body := strings.Join(lines[separatorIdx+1:], "\n")
	return metadata, body
}

// ResolveIncrement finds the next available number for the {{INC}} macro.
func ResolveIncrement(pattern string, targetDir string) string {
	rePattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\{\\{INC\\}\\}", "(\\d+)")
	re := regexp.MustCompile("^" + rePattern + "(\\..+)?$")

	maxInc := -1
	files, _ := os.ReadDir(targetDir)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		matches := re.FindStringSubmatch(f.Name())
		if len(matches) > 1 {
			inc, err := strconv.Atoi(matches[1])
			if err == nil {
				if inc > maxInc {
					maxInc = inc
				}
			}
		}
	}

	nextInc := maxInc + 1
	if nextInc < 1 {
		nextInc = 1
	}

	return strings.ReplaceAll(pattern, "{{INC}}", strconv.Itoa(nextInc))
}

// ReplaceMacros handles all template variable replacements including modifiers.
func ReplaceMacros(content string, macros map[string]string) string {
	now := time.Now()

	// Add built-in date macros to the map
	allMacros := make(map[string]string)
	for k, v := range macros {
		allMacros[k] = v
	}
	allMacros["DATE"] = now.Format("2006-01-02")
	allMacros["TIME"] = now.Format("15:04:05")
	allMacros["DATETIME"] = now.Format("2006-01-02 15:04:05")
	allMacros["DAY"] = now.Format("02")
	allMacros["MONTH"] = now.Format("01")
	allMacros["YEAR"] = now.Format("2006")
	allMacros["WEEKDAY"] = now.Weekday().String()
	allMacros["TOMORROW"] = now.AddDate(0, 0, 1).Format("2006-01-02")
	allMacros["YESTERDAY"] = now.AddDate(0, 0, -1).Format("2006-01-02")

	// Match {{KEY}} or {{KEY|MOD}}
	re := regexp.MustCompile(`\{\{([^}|]+)(?:\|([^}]+))?\}\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		key := strings.TrimSpace(parts[1])
		modifier := ""
		if len(parts) > 2 {
			modifier = strings.TrimSpace(parts[2])
		}

		val, ok := allMacros[key]
		if !ok {
			return match
		}

		if modifier != "" {
			if strings.HasPrefix(modifier, "+") {
				formatStr := modifier[1:]
				// Only apply date formatting to macros that represent dates/times
				dateMacros := map[string]bool{
					"DATE":      true,
					"TIME":      true,
					"DATETIME":  true,
					"DAY":       true,
					"MONTH":     true,
					"YEAR":      true,
					"WEEKDAY":   true,
					"TOMORROW":  true,
					"YESTERDAY": true,
				}

				if dateMacros[key] {
					// We need the actual time object for relative dates
					targetTime := now
					switch key {
					case "TOMORROW":
						targetTime = now.AddDate(0, 0, 1)
					case "YESTERDAY":
						targetTime = now.AddDate(0, 0, -1)
					}
					return linuxDateToFormat(targetTime, formatStr)
				}
			}

			switch strings.ToUpper(modifier) {
			case "LOWER":
				val = strings.ToLower(val)
			case "UPPER":
				val = strings.ToUpper(val)
			case "SLUG":
				val = slugify(val)
			}
		}
		return val
	})
}

func linuxDateToFormat(t time.Time, layout string) string {
	// Re-implementing with a robust approach:
	// 1. Identify all % tokens
	// 2. Map them to their Go values
	// 3. Join everything back

	var result strings.Builder
	for i := 0; i < len(layout); i++ {
		if layout[i] == '%' && i+1 < len(layout) {
			token := ""
			offset := 0
			if layout[i+1] == '-' && i+2 < len(layout) {
				token = layout[i : i+3]
				offset = 2
			} else {
				token = layout[i : i+2]
				offset = 1
			}

			switch token {
			case "%Y":
				result.WriteString(t.Format("2006"))
			case "%y":
				result.WriteString(t.Format("06"))
			case "%m":
				result.WriteString(t.Format("01"))
			case "%-m":
				result.WriteString(t.Format("1"))
			case "%B":
				result.WriteString(t.Format("January"))
			case "%b", "%h":
				result.WriteString(t.Format("Jan"))
			case "%d":
				result.WriteString(t.Format("02"))
			case "%-d":
				result.WriteString(t.Format("2"))
			case "%H":
				result.WriteString(t.Format("15"))
			case "%I":
				result.WriteString(t.Format("03"))
			case "%-I":
				result.WriteString(t.Format("3"))
			case "%M":
				result.WriteString(t.Format("04"))
			case "%S":
				result.WriteString(t.Format("05"))
			case "%p":
				result.WriteString(t.Format("PM"))
			case "%P":
				result.WriteString(strings.ToLower(t.Format("PM")))
			case "%A":
				result.WriteString(t.Format("Monday"))
			case "%a":
				result.WriteString(t.Format("Mon"))
			case "%z":
				result.WriteString(t.Format("-0700"))
			case "%Z":
				result.WriteString(t.Format("MST"))
			case "%%":
				result.WriteByte('%')
			default:
				result.WriteString(token)
			}
			i += offset
		} else {
			result.WriteByte(layout[i])
		}
	}
	return result.String()
}

func slugify(s string) string {
	s = strings.ToLower(s)
	// Replace non-alphanumeric with -
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	// Trim -
	return strings.Trim(s, "-")
}
