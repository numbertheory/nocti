package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var newestFirst bool
var jsonOutput bool

type Match struct {
	LineNo int    `json:"line_no"`
	Text   string `json:"text"`
}

type SearchResult struct {
	Path    string      `json:"path"`
	Score   int         `json:"score"`
	ModTime os.FileInfo `json:"-"`
	Matches []Match     `json:"matches"`
}

type JSONSearchResult struct {
	Path      string    `json:"path"`
	Score     int       `json:"score"`
	UpdatedAt time.Time `json:"updated_at"`
	Matches   []Match   `json:"matches"`
}

type JSONSearchResponse struct {
	TotalHits int                `json:"total_hits"`
	FileCount int                `json:"file_count"`
	Results   []JSONSearchResult `json:"results"`
}

func ScanAllFilesRecursive(searchDir string, currentResType string, isProjectRoot bool) ([]string, error) {
	var results []string

	err := filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == searchDir {
			return nil
		}

		name := d.Name()
		if d.IsDir() {
			// Skip hidden directories except .nocti if we are at project root (though we don't usually search inside .nocti)
			if strings.HasPrefix(name, ".") {
				if name != ".nocti" || !isProjectRoot {
					return filepath.SkipDir
				}
			}
			// Don't skip sub-resources, we want to search everything down the hierarchy.
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".md" || ext == ".txt" {
			if strings.HasPrefix(name, ".") {
				return nil
			}
			relPath, err := filepath.Rel(searchDir, path)
			if err == nil {
				results = append(results, relPath)
			}
		}
		return nil
	})

	return results, err
}

func PerformSearch(searchDir string, currentResType string, isProjectRoot bool, keywords []string, newestFirst bool) ([]SearchResult, error) {
	files, err := ScanAllFilesRecursive(searchDir, currentResType, isProjectRoot)
	if err != nil {
		return nil, err
	}

	// Prepare regex patterns for whole-word matching
	type keywordPattern struct {
		re *regexp.Regexp
	}
	var patterns []keywordPattern
	for _, kw := range keywords {
		// Smart-case: if keyword has uppercase, it's case-sensitive
		isCaseSensitive := false
		for _, r := range kw {
			if r >= 'A' && r <= 'Z' {
				isCaseSensitive = true
				break
			}
		}

		prefix := "(?i)"
		if isCaseSensitive {
			prefix = ""
		}
		// \b matches word boundaries
		pattern := prefix + `\b` + regexp.QuoteMeta(kw) + `\b`
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid keyword '%s': %w", kw, err)
		}
		patterns = append(patterns, keywordPattern{re: re})
	}

	var results []SearchResult

	for _, f := range files {
		// Skip directories in results (Scan functions return folders with trailing /)
		if strings.HasSuffix(f, string(os.PathSeparator)) {
			continue
		}

		// For calendar/todo, we might need GetPhysicalPath if f is virtual
		physPath := f
		if !isProjectRoot {
			physPath = GetPhysicalPath(f, searchDir, currentResType)
		}

		fullPath := filepath.Join(searchDir, physPath)
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			absPath = fullPath
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		// Read file
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		contentStr := string(content)
		score := 0
		matchesAll := true
		var fileMatches []Match

		for _, p := range patterns {
			matches := p.re.FindAllStringIndex(contentStr, -1)
			if len(matches) == 0 {
				matchesAll = false
				break
			}
			score += len(matches)
		}

		if matchesAll {
			// Find all matching lines
			scanner := bufio.NewScanner(strings.NewReader(contentStr))
			lineNo := 1
			for scanner.Scan() {
				line := scanner.Text()
				foundAny := false
				for _, p := range patterns {
					if p.re.MatchString(line) {
						foundAny = true
						break
					}
				}
				if foundAny {
					fileMatches = append(fileMatches, Match{
						LineNo: lineNo,
						Text:   strings.TrimSpace(line),
					})
				}
				lineNo++
			}

			results = append(results, SearchResult{
				Path:    absPath,
				Score:   score,
				ModTime: info,
				Matches: fileMatches,
			})
		}
	}

	// Ranking
	if newestFirst {
		sort.Slice(results, func(i, j int) bool {
			return results[i].ModTime.ModTime().After(results[j].ModTime.ModTime())
		})
	} else {
		sort.Slice(results, func(i, j int) bool {
			if results[i].Score == results[j].Score {
				return results[i].Path < results[j].Path
			}
			return results[i].Score > results[j].Score
		})
	}

	return results, nil
}

var SearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for keywords in the current nocti resource",
	Long: `Search for keywords across markdown and text files in the current context.
By default, results are ranked by term frequency (score).
Use the --newest flag to sort by the most recently modified files first.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		keywords := strings.Fields(query)

		var searchDir string
		var currentResType string
		var isProjectRoot bool

		// Detect context (copied/adapted from ListCmd)
		root, err := FindProjectRoot()
		wd, _ := os.Getwd()
		if err == nil && wd == root {
			isProjectRoot = true
			searchDir = root
		} else {
			resRoot, resType, err := FindEnclosingResource()
			if err != nil {
				return fmt.Errorf("not inside a nocti resource: %w", err)
			}
			searchDir = resRoot
			currentResType = resType
		}

		// Re-fetch resType if it's not project root to be sure
		if !isProjectRoot {
			config, err := FindEnclosingResourceIn(searchDir)
			if err == nil {
				currentResType = config.Type
			}
		}

		results, err := PerformSearch(searchDir, currentResType, isProjectRoot, keywords, newestFirst)
		if err != nil {
			return err
		}

		totalHits := 0
		for _, res := range results {
			totalHits += res.Score
		}

		// JSON Output
		if jsonOutput {
			jsonResults := make([]JSONSearchResult, len(results))
			for i, res := range results {
				jsonResults[i] = JSONSearchResult{
					Path:      res.Path,
					Score:     res.Score,
					UpdatedAt: res.ModTime.ModTime(),
					Matches:   res.Matches,
				}
			}
			response := JSONSearchResponse{
				TotalHits: totalHits,
				FileCount: len(results),
				Results:   jsonResults,
			}
			data, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		// Output
		if len(results) == 0 {
			fmt.Println("No matches found.")
			return nil
		}

		colors, _ := loadColorsAndEditor(searchDir)

		// Set up colors
		highlightFg := "\033[1;33m" // Bold Yellow
		highlightBg := ""           // Default
		lnFg := "\033[34m"          // Blue
		lnBg := ""                  // Default
		reset := "\033[0m"
		fileNameFg := "\033[32m" // Green
		fileNameBg := ""         // Default
		scoreFg := "\033[35m"    // Magenta
		scoreBg := ""            // Default

		// Support NO_COLOR environment variable
		if os.Getenv("NO_COLOR") == "1" {
			highlightFg = ""
			highlightBg = ""
			lnFg = ""
			lnBg = ""
			reset = ""
			fileNameFg = ""
			fileNameBg = ""
			scoreFg = ""
			scoreBg = ""
		} else if colors != nil {
			if colors.SearchHighlightFg != "" {
				highlightFg = GetFGColorCode(colors.SearchHighlightFg, highlightFg)
			}
			if colors.SearchHighlightBg != "" {
				highlightBg = GetColorCode(colors.SearchHighlightBg, highlightBg)
			}
			if colors.SearchLnFg != "" {
				lnFg = GetFGColorCode(colors.SearchLnFg, lnFg)
			}
			if colors.SearchLnBg != "" {
				lnBg = GetColorCode(colors.SearchLnBg, lnBg)
			}
			if colors.SearchFileFg != "" {
				fileNameFg = GetFGColorCode(colors.SearchFileFg, fileNameFg)
			}
			if colors.SearchFileBg != "" {
				fileNameBg = GetColorCode(colors.SearchFileBg, fileNameBg)
			}
			if colors.SearchScoreFg != "" {
				scoreFg = GetFGColorCode(colors.SearchScoreFg, scoreFg)
			}
			if colors.SearchScoreBg != "" {
				scoreBg = GetColorCode(colors.SearchScoreBg, scoreBg)
			}
		}

		// Re-prepare patterns for highlighting
		type keywordPattern struct {
			re *regexp.Regexp
		}
		var patterns []keywordPattern
		for _, kw := range keywords {
			isCaseSensitive := false
			for _, r := range kw {
				if r >= 'A' && r <= 'Z' {
					isCaseSensitive = true
					break
				}
			}
			prefix := "(?i)"
			if isCaseSensitive {
				prefix = ""
			}
			pattern := prefix + `\b` + regexp.QuoteMeta(kw) + `\b`
			re, _ := regexp.Compile(pattern)
			patterns = append(patterns, keywordPattern{re: re})
		}

		fmt.Printf("Found %d matches across %d files:\n", totalHits, len(results))
		for _, res := range results {
			if newestFirst {
				fmt.Printf("%s%s[%s]%s %s%s%s%s\n", scoreBg, scoreFg, res.ModTime.ModTime().Format("2006-01-02 15:04"), reset, fileNameBg, fileNameFg, res.Path, reset)
			} else {
				fmt.Printf("%s%s(score: %d)%s %s%s%s%s\n", scoreBg, scoreFg, res.Score, reset, fileNameBg, fileNameFg, res.Path, reset)
			}
			for _, m := range res.Matches {
				// Highlight matches in line
				line := m.Text
				highlightedLine := line

				// Sort patterns by length descending to avoid partial highlights of longer keywords?
				// Actually, they don't overlap if we use word boundaries.
				// We need to be careful with overlapping matches if we had any.

				// Apply highlights
				for _, p := range patterns {
					highlightedLine = p.re.ReplaceAllStringFunc(highlightedLine, func(match string) string {
						return highlightFg + highlightBg + match + reset
					})
				}

				// Truncate match text if too long (accounting for ANSI codes is hard, so we do it before highlighting)
				displayText := highlightedLine
				if len(m.Text) > 80 {
					// Re-calculate highlightedLine on truncated text
					truncated := m.Text[:77] + "..."
					displayText = truncated
					for _, p := range patterns {
						displayText = p.re.ReplaceAllStringFunc(displayText, func(match string) string {
							return highlightFg + highlightBg + match + reset
						})
					}
				}
				fmt.Printf("  %s%s%d%s: %s\n", lnBg, lnFg, m.LineNo, reset, displayText)
			}
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(SearchCmd)
	SearchCmd.Flags().BoolVarP(&newestFirst, "newest", "n", false, "Sort by newest matches first")
	SearchCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
}
