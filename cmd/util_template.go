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

func slugify(s string) string {
	s = strings.ToLower(s)
	// Replace non-alphanumeric with -
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	// Trim -
	return strings.Trim(s, "-")
}
