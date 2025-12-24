package conflicts

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code-analyzer/analyzers"
	"code-analyzer/models"
	"code-analyzer/utils"
)

// ConflictsAnalyzer detects unresolved merge conflicts in files
type ConflictsAnalyzer struct {
	rules []analyzers.Rule
}

// NewConflictsAnalyzer creates a new conflicts analyzer
func NewConflictsAnalyzer() *ConflictsAnalyzer {
	return &ConflictsAnalyzer{
		rules: []analyzers.Rule{
			&ConflictMarkersRule{},
		},
	}
}

// Name returns the analyzer name
func (a *ConflictsAnalyzer) Name() string {
	return "Conflicts Analyzer"
}

// Description returns what this analyzer does
func (a *ConflictsAnalyzer) Description() string {
	return "Detects unresolved Git merge conflict markers in files"
}

// Run executes the conflicts analysis
func (a *ConflictsAnalyzer) Run(config analyzers.Config) ([]models.Issue, error) {
	results := []models.ConflictFileAnalysis{}
	var allIssues []models.Issue

	err := filepath.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip binary files and very large files
		if info.Size() > 10*1024*1024 { // Skip files > 10MB
			return nil
		}

		if utils.ShouldSkip(path, config.ExcludePaths) {
			return nil
		}

		analysis := a.analyzeFile(path)
		if analysis != nil && len(analysis.ConflictLines) >= config.MinValue {
			results = append(results, *analysis)
			allIssues = append(allIssues, analysis.Issues...)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by number of conflicts
	sort.Slice(results, func(i, j int) bool {
		return len(results[i].ConflictLines) > len(results[j].ConflictLines)
	})

	// Limit to top N
	if len(results) > config.TopN {
		results = results[:config.TopN]
	}

	// Generate artifact if requested
	if config.OutputFile != "" {
		if err := a.generateArtifact(results, config); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to generate artifact: %v\n", err)
		} else {
			fmt.Printf("✅ Artifact generated: %s\n\n", config.OutputFile)
		}
	}

	// Print results
	a.printResults(results)
	return allIssues, nil
}

func (a *ConflictsAnalyzer) analyzeFile(path string) *models.ConflictFileAnalysis {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var conflictLines []int
	var conflictSnippets []string
	lineNum := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if len(trimmed) == 0 {
			continue
		}

		// Git conflict markers have VERY specific format:
		// <<<<<<< HEAD (or branch) - exactly 7 '<', space, then text, NO other characters after
		// ======= - EXACTLY and ONLY 7 '=' characters, nothing before or after
		// >>>>>>> branch - exactly 7 '>', space, then text, NO other characters after

		isConflictMarker := false

		// Start marker: <<<<<<< (must have space after 7th '<')
		if len(trimmed) >= 8 && trimmed[:7] == "<<<<<<<" && trimmed[7] == ' ' {
			// Must NOT be in a comment (no /*, */)
			if !strings.Contains(line, "/*") && !strings.Contains(line, "*/") {
				isConflictMarker = true
			}
		}

		// Separator: EXACTLY "=======" and nothing else
		// This is key - CSS comments have more ='s or have */ at the end
		if trimmed == "=======" {
			isConflictMarker = true
		}

		// End marker: >>>>>>> (must have space after 7th '>')
		if len(trimmed) >= 8 && trimmed[:7] == ">>>>>>>" && trimmed[7] == ' ' {
			// Must NOT be in a comment
			if !strings.Contains(line, "/*") && !strings.Contains(line, "*/") {
				isConflictMarker = true
			}
		}

		if isConflictMarker {
			conflictLines = append(conflictLines, lineNum)
			if len(conflictSnippets) < 5 {
				conflictSnippets = append(conflictSnippets, trimmed)
			}
		}
	}

	if len(conflictLines) == 0 {
		return nil
	}

	// Count conflict blocks (each block has <<<, ===, >>>)
	conflictBlocks := len(conflictLines) / 3
	if conflictBlocks == 0 {
		conflictBlocks = 1
	}

	var issues []models.Issue
	// Map conflict lines to issues
	// We only create an issue for the start of the block (<<<<<<<) to avoid spamming
	// But conflictLines contains ALL lines (start, mid, end)
	// Actually, conflictLines in the current implementation stores EVERY line that is a marker
	// So we can just iterate them. Or grouping them.
	// Let's create an issue for every marker for visibility, or just the Start marker.
	// The scanner logic adds lines with Markers.
	// Let's create an issue for each marker found.
	for i, line := range conflictLines {
		desc := "Merge conflict marker"
		if i < len(conflictSnippets) {
			desc = fmt.Sprintf("Merge conflict marker: %s", conflictSnippets[i])
		}
		issues = append(issues, models.Issue{
			Path:        path,
			Description: desc,
			Line:        line,
			Severity:    "critical",
		})
	}

	return &models.ConflictFileAnalysis{
		Path:             path,
		ConflictLines:    conflictLines,
		ConflictBlocks:   conflictBlocks,
		ConflictSnippets: conflictSnippets,
		Issues:           issues,
	}
}

func (a *ConflictsAnalyzer) printResults(results []models.ConflictFileAnalysis) {
	if len(results) == 0 {
		fmt.Println("✅ No files with unresolved merge conflicts found!")
		return
	}

	totalConflicts := 0
	for _, r := range results {
		totalConflicts += r.ConflictBlocks
	}

	fmt.Printf("🚨 Found %d files with unresolved merge conflicts!\n", len(results))
	fmt.Printf("📊 Total Conflict Blocks: %d\n\n", totalConflicts)

	fmt.Printf("%-5s %-70s %10s %15s\n",
		"Rank", "File", "Blocks", "Lines")
	fmt.Println(strings.Repeat("-", 105))

	for i, result := range results {
		relPath := utils.Truncate(result.Path, 70)
		fmt.Printf("%-5d %-70s %10d %15d\n",
			i+1, relPath,
			result.ConflictBlocks,
			len(result.ConflictLines))
	}

	fmt.Println()
	a.printTop10(results)
	fmt.Println("✅ Analysis complete!")
}

func (a *ConflictsAnalyzer) printTop10(results []models.ConflictFileAnalysis) {
	fmt.Printf("📋 Top 10 Files with Conflicts:\n")
	fmt.Println(strings.Repeat("-", 80))

	topCount := utils.Min(10, len(results))
	for i := 0; i < topCount; i++ {
		r := results[i]
		fmt.Printf("%2d. %s\n", i+1, r.Path)
		fmt.Printf("    🚨 %d conflict blocks | 📍 Lines: %v\n",
			r.ConflictBlocks, formatLineNumbers(r.ConflictLines[:utils.Min(6, len(r.ConflictLines))]))
		if len(r.ConflictSnippets) > 0 {
			fmt.Printf("    💬 Preview: %s\n", r.ConflictSnippets[0])
		}
	}
	fmt.Println()
}

func (a *ConflictsAnalyzer) generateArtifact(results []models.ConflictFileAnalysis, config analyzers.Config) error {
	totalBlocks := 0
	for _, r := range results {
		totalBlocks += r.ConflictBlocks
	}

	report := models.ConflictAnalysisReport{
		Timestamp:      utils.GetTimestamp(),
		ScanDirectory:  config.RootDir,
		TotalFiles:     len(results),
		TotalConflicts: totalBlocks,
		Results:        results,
	}

	return utils.WriteArtifact(config.OutputFile, report)
}

func formatLineNumbers(lines []int) string {
	if len(lines) == 0 {
		return "[]"
	}

	strs := make([]string, len(lines))
	for i, line := range lines {
		strs[i] = fmt.Sprintf("%d", line)
	}

	result := strings.Join(strs, ", ")
	if len(lines) > 5 {
		result += "..."
	}
	return result
}

// ConflictMarkersRule detects Git conflict markers
type ConflictMarkersRule struct{}

func (r *ConflictMarkersRule) Name() string {
	return "Conflict Markers Detector"
}

func (r *ConflictMarkersRule) Apply(content string) interface{} {
	// Not used in this implementation - we scan line by line in analyzeFile
	return nil
}
