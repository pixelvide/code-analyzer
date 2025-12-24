package js

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

// JSAnalyzer analyzes JavaScript/TypeScript files for commented code
type JSAnalyzer struct {
	rules []analyzers.Rule
}

// NewJSAnalyzer creates a new JS analyzer
func NewJSAnalyzer() *JSAnalyzer {
	return &JSAnalyzer{
		rules: []analyzers.Rule{
			&CommentedCodeRule{},
		},
	}
}

// Name returns the analyzer name
func (a *JSAnalyzer) Name() string {
	return "JS Analyzer"
}

// Description returns what this analyzer does
func (a *JSAnalyzer) Description() string {
	return "Analyzes JS/TS files for commented code blocks"
}

// Run executes the JS analysis
func (a *JSAnalyzer) Run(config analyzers.Config) ([]models.Issue, error) {
	results := []models.JSFileAnalysis{}
	var allIssues []models.Issue

	err := filepath.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".js" && ext != ".jsx" && ext != ".ts" && ext != ".tsx" {
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

func (a *JSAnalyzer) analyzeFile(path string) *models.JSFileAnalysis {
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

	return &models.JSFileAnalysis{
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

func (a *JSAnalyzer) printResults(results []models.JSFileAnalysis) {
	if len(results) == 0 {
		fmt.Println("✅ No JS/TS files with significant commented code found!")
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

func (a *JSAnalyzer) printTop10(results []models.JSFileAnalysis) {
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

func (a *JSAnalyzer) generateArtifact(results []models.JSFileAnalysis, config analyzers.Config) error {
	totalCommented := 0
	for _, r := range results {
		totalCommented += r.CommentedBytes
	}

	report := models.JSAnalysisReport{
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

// CommentedCodeRule detects commented-out JS code
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
	commentedBytes := 0
	commentedLines := 0
	largestBlock := 0
	var issues []models.Issue

	// 1. Detect multi-line comments /* ... */
	multiLineRegex := regexp.MustCompile(`(?s)/\*(.*?)\*/`)
	multiLineMatches := multiLineRegex.FindAllStringSubmatchIndex(content, -1)

	for _, loc := range multiLineMatches {
		// loc[0], loc[1] is the whole match
		// loc[2], loc[3] is the first group (.*?)
		if len(loc) >= 4 {
			commentStart, commentEnd := loc[2], loc[3]
			commentContent := content[commentStart:commentEnd]

			if isCode(commentContent) {
				fullMatch := content[loc[0]:loc[1]]
				matchLen := len(fullMatch)
				matchLines := strings.Count(fullMatch, "\n") + 1
				commentedBytes += matchLen
				commentedLines += matchLines
				if matchLen > largestBlock {
					largestBlock = matchLen
				}

				// Calculate line number
				lineNumber := strings.Count(content[:loc[0]], "\n") + 1
				issues = append(issues, models.Issue{
					Description: fmt.Sprintf("Commented out JS code block (%d bytes)", matchLen),
					Line:        lineNumber,
					Severity:    "minor",
				})
			}
		}
	}

	// 2. Detect single-line comments // ...
	lines := strings.Split(content, "\n")
	var currentBlock strings.Builder
	inBlock := false
	blockStartLine := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check for single line comment
		if strings.HasPrefix(trimmed, "//") {
			commentContent := strings.TrimPrefix(trimmed, "//")
			if inBlock {
				currentBlock.WriteString("\n" + commentContent)
			} else {
				inBlock = true
				blockStartLine = i + 1
				currentBlock.Reset()
				currentBlock.WriteString(commentContent)
			}
		} else {
			if inBlock {
				// End of block, analyze it
				blockContent := currentBlock.String()
				if isCode(blockContent) {
					linesInBlock := strings.Count(blockContent, "\n") + 1
					// Approx bytes
					blockOriginalBytes := len(blockContent) + (linesInBlock * 2)

					commentedBytes += blockOriginalBytes
					commentedLines += linesInBlock
					if blockOriginalBytes > largestBlock {
						largestBlock = blockOriginalBytes
					}

					issues = append(issues, models.Issue{
						Description: fmt.Sprintf("Commented out JS code block (%d bytes)", blockOriginalBytes),
						Line:        blockStartLine,
						Severity:    "minor",
					})
				}
				inBlock = false
			}
		}
	}
	// Check last block
	if inBlock {
		blockContent := currentBlock.String()
		if isCode(blockContent) {
			linesInBlock := strings.Count(blockContent, "\n") + 1
			blockOriginalBytes := len(blockContent) + (linesInBlock * 2)
			commentedBytes += blockOriginalBytes
			commentedLines += linesInBlock
			if blockOriginalBytes > largestBlock {
				largestBlock = blockOriginalBytes
			}
			issues = append(issues, models.Issue{
				Description: fmt.Sprintf("Commented out JS code block (%d bytes)", blockOriginalBytes),
				Line:        blockStartLine,
				Severity:    "minor",
			})
		}
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

// isCode uses heuristics to determine if text looks like code
func isCode(text string) bool {
	// Simple heuristics: code often contains these symbols
	// We want to avoid flagging normal text comments
	indicators := []string{
		";", "{", "}", "function", "const ", "var ", "let ", "=>", "return", "import ", "export ",
		"class ", "if (", "for (", "while (", "console.log",
	}

	score := 0
	for _, ind := range indicators {
		if strings.Contains(text, ind) {
			score++
		}
	}

	// Negative heuristics for text
	textIndicators := []string{
		"TODO:", "FIXME:", "NOTE:", "http://", "https://", " This ", " The ", " To ",
	}
	for _, ind := range textIndicators {
		if strings.Contains(text, ind) {
			score--
		}
	}

	return score >= 1
}
