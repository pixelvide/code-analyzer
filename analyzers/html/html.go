package html

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"code-analyzer/analyzers"
	"code-analyzer/models"
	"code-analyzer/utils"
)

// HTMLAnalyzer analyzes HTML files for various code quality issues
type HTMLAnalyzer struct {
	rules []analyzers.Rule
}

// NewHTMLAnalyzer creates a new HTML analyzer with default rules
func NewHTMLAnalyzer() *HTMLAnalyzer {
	return &HTMLAnalyzer{
		rules: []analyzers.Rule{
			&CommentedCodeRule{},
		},
	}
}

// Name returns the analyzer name
func (a *HTMLAnalyzer) Name() string {
	return "HTML Analyzer"
}

// Description returns what this analyzer does
func (a *HTMLAnalyzer) Description() string {
	return "Analyzes HTML files for commented code blocks and other issues"
}

// Run executes the HTML analysis
func (a *HTMLAnalyzer) Run(config analyzers.Config) ([]models.Issue, error) {
	results := []models.HTMLFileAnalysis{}
	var allIssues []models.Issue

	err := filepath.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".html") {
			return nil
		}
		if utils.ShouldSkip(path, config.ExcludePaths) {
			return nil
		}

		analysis := a.analyzeFile(path)
		if analysis != nil {
			if analysis.CommentedBytes < config.MinValue {
				return nil
			}
			if config.MinRatio > 0 && analysis.CommentRatio < config.MinRatio {
				return nil
			}
			results = append(results, *analysis)
			allIssues = append(allIssues, analysis.Issues...)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort results
	if config.SortBy == "ratio" {
		sort.Slice(results, func(i, j int) bool {
			return results[i].CommentRatio > results[j].CommentRatio
		})
	} else {
		sort.Slice(results, func(i, j int) bool {
			return results[i].CommentedBytes > results[j].CommentedBytes
		})
	}

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

func (a *HTMLAnalyzer) analyzeFile(path string) *models.HTMLFileAnalysis {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Apply commented code rule
	rule := &CommentedCodeRule{}
	finding := rule.Apply(string(content))

	if finding == nil {
		return nil
	}

	result := finding.(CommentedCodeFinding)
	if result.CommentedBytes == 0 {
		return nil
	}

	// Set path for issues
	for i := range result.Issues {
		result.Issues[i].Path = path
	}

	totalBytes := len(content)
	totalLines := strings.Count(string(content), "\n") + 1
	ratio := float64(result.CommentedBytes) / float64(totalBytes) * 100

	return &models.HTMLFileAnalysis{
		Path:           path,
		TotalLines:     totalLines,
		CommentedLines: result.CommentedLines,
		CommentedBytes: result.CommentedBytes,
		TotalBytes:     totalBytes,
		CommentRatio:   ratio,
		LargestBlock:   result.LargestBlock,
		Issues:         result.Issues,
	}
}

func (a *HTMLAnalyzer) printResults(results []models.HTMLFileAnalysis) {
	if len(results) == 0 {
		fmt.Println("✅ No HTML files with significant commented code found!")
		return
	}

	totalCommented := 0
	for _, r := range results {
		totalCommented += r.CommentedBytes
	}

	fmt.Printf("Found %d files with commented code\n", len(results))
	fmt.Printf("📊 Total Commented Code: %s (%.2f KB)\n\n",
		utils.FormatBytes(totalCommented), float64(totalCommented)/1024)

	fmt.Printf("%-5s %-60s %12s %10s %8s %10s\n",
		"Rank", "File", "Commented", "Total", "Ratio", "Largest")
	fmt.Println(strings.Repeat("-", 115))

	for i, result := range results {
		relPath := utils.Truncate(result.Path, 60)
		fmt.Printf("%-5d %-60s %12s %10s %7.1f%% %10s\n",
			i+1, relPath,
			utils.FormatBytes(result.CommentedBytes),
			utils.FormatBytes(result.TotalBytes),
			result.CommentRatio,
			utils.FormatBytes(result.LargestBlock))
	}

	fmt.Println()
	a.printTop10(results)
	fmt.Println("✅ Analysis complete!")
}

func (a *HTMLAnalyzer) printTop10(results []models.HTMLFileAnalysis) {
	fmt.Printf("📋 Top 10 High-Impact Files:\n")
	fmt.Println(strings.Repeat("-", 80))

	topCount := utils.Min(10, len(results))
	for i := 0; i < topCount; i++ {
		r := results[i]
		fmt.Printf("%2d. %s\n", i+1, r.Path)
		fmt.Printf("    💾 Size: %s | 💬 Comments: %s (%.1f%%) | 📦 Largest: %s\n",
			utils.FormatBytes(r.TotalBytes),
			utils.FormatBytes(r.CommentedBytes),
			r.CommentRatio,
			utils.FormatBytes(r.LargestBlock))
	}
	fmt.Println()
}

func (a *HTMLAnalyzer) generateArtifact(results []models.HTMLFileAnalysis, config analyzers.Config) error {
	totalCommented := 0
	for _, r := range results {
		totalCommented += r.CommentedBytes
	}

	report := models.HTMLAnalysisReport{
		Timestamp:      utils.GetTimestamp(),
		ScanDirectory:  config.RootDir,
		TotalFiles:     len(results),
		TotalCommented: totalCommented,
		SortMode:       config.SortBy,
		MinComments:    config.MinValue,
		Results:        results,
	}

	return utils.WriteArtifact(config.OutputFile, report)
}

// CommentedCodeRule detects commented-out HTML code
type CommentedCodeRule struct{}

type CommentedCodeFinding struct {
	CommentedBytes int
	CommentedLines int
	LargestBlock   int
	Issues         []models.Issue
}

func (r *CommentedCodeRule) Name() string {
	return "Commented Code Detector"
}

func (r *CommentedCodeRule) Apply(content string) interface{} {
	commentRegex := regexp.MustCompile(`(?s)<!--.*?-->`)
	matches := commentRegex.FindAllStringIndex(content, -1)

	commentedBytes := 0
	commentedLines := 0
	largestBlock := 0
	var issues []models.Issue

	tagRegex := regexp.MustCompile(`<[/a-zA-Z][^>]*>`)

	for _, loc := range matches {
		start, end := loc[0], loc[1]
		match := content[start:end]

		// Heuristic: It's likely commented code if it contains HTML tags
		// We strip the comment markers first to avoid matching them (though standard regex handles that)
		inner := match
		if len(match) >= 7 {
			inner = match[4 : len(match)-3]
		}

		if !tagRegex.MatchString(inner) {
			continue
		}

		matchLen := len(match)
		matchLines := strings.Count(match, "\n") + 1
		commentedBytes += matchLen
		commentedLines += matchLines
		if matchLen > largestBlock {
			largestBlock = matchLen
		}

		// Calculate line number
		lineNumber := strings.Count(content[:start], "\n") + 1

		issues = append(issues, models.Issue{
			Description: fmt.Sprintf("Commented out HTML code block (%d bytes)", matchLen),
			Line:        lineNumber,
			Severity:    "minor",
			Path:        "", // Will be populated by analyzeFile
		})
	}

	if commentedBytes == 0 {
		return nil
	}

	return CommentedCodeFinding{
		CommentedBytes: commentedBytes,
		CommentedLines: commentedLines,
		LargestBlock:   largestBlock,
		Issues:         issues,
	}
}
