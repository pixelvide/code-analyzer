package php

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

// PHPAnalyzer analyzes PHP files for various code quality issues
type PHPAnalyzer struct {
	rules []analyzers.Rule
}

// NewPHPAnalyzer creates a new PHP analyzer with default rules
func NewPHPAnalyzer() *PHPAnalyzer {
	return &PHPAnalyzer{
		rules: []analyzers.Rule{
			&CommentedFunctionsRule{},
			&LaravelCatchBlockRule{},
		},
	}
}

// Name returns the analyzer name
func (a *PHPAnalyzer) Name() string {
	return "PHP Analyzer"
}

// Description returns what this analyzer does
func (a *PHPAnalyzer) Description() string {
	return "Analyzes PHP files for commented functions and other issues"
}

// Run executes the PHP analysis
func (a *PHPAnalyzer) Run(config analyzers.Config) ([]models.Issue, error) {
	results := []models.PHPFileAnalysis{}
	totalFunctions := 0
	totalCommented := 0
	var allIssues []models.Issue

	err := filepath.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".php") {
			return nil
		}
		if utils.ShouldSkip(path, config.ExcludePaths) {
			return nil
		}

		analysis := a.analyzeFile(path)
		if analysis != nil {
			// Skip if below threshold AND no other issues
			if analysis.CommentedFunctions < config.MinValue && len(analysis.Issues) == 0 {
				return nil
			}
			if config.MinRatio > 0 && analysis.CommentRatio < config.MinRatio && len(analysis.Issues) == 0 {
				return nil
			}

			results = append(results, *analysis)
			totalFunctions += analysis.TotalFunctions
			totalCommented += analysis.CommentedFunctions
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
			return results[i].CommentedFunctions > results[j].CommentedFunctions
		})
	}

	// Limit to top N
	if len(results) > config.TopN {
		results = results[:config.TopN]
	}

	// Generate artifact if requested
	if config.OutputFile != "" {
		if err := a.generateArtifact(results, config, totalFunctions, totalCommented); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to generate artifact: %v\n", err)
		} else {
			fmt.Printf("✅ Artifact generated: %s\n\n", config.OutputFile)
		}
	}

	// Print results
	a.printResults(results, totalFunctions, totalCommented)
	return allIssues, nil
}

func (a *PHPAnalyzer) analyzeFile(path string) *models.PHPFileAnalysis {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	contentStr := string(content)

	var analysis *models.PHPFileAnalysis
	var allIssues []models.Issue

	// Apply commented functions rule
	cfRule := &CommentedFunctionsRule{}
	if finding := cfRule.Apply(contentStr); finding != nil {
		result := finding.(CommentedFunctionsFinding)

		totalBytes := len(content)
		commentedBytes := len(result.CommentedList) * 20 // rough estimate
		ratio := 0.0
		if len(result.AllFunctions) > 0 {
			ratio = float64(len(result.CommentedList)) / float64(len(result.AllFunctions)) * 100
		}

		// Set path for issues
		for i := range result.Issues {
			result.Issues[i].Path = path
		}
		allIssues = append(allIssues, result.Issues...)

		analysis = &models.PHPFileAnalysis{
			Path:               path,
			TotalFunctions:     len(result.AllFunctions),
			CommentedFunctions: len(result.CommentedList),
			FunctionList:       result.AllFunctions,
			CommentedList:      result.CommentedList,
			CommentRatio:       ratio,
			TotalBytes:         totalBytes,
			CommentedBytes:     commentedBytes,
		}
	}

	// Apply Laravel Catch Block Rule
	var catchMissing, catchMisplaced int
	if strings.Contains(path, "app/") {
		lcbRule := &LaravelCatchBlockRule{}
		if finding := lcbRule.Apply(contentStr); finding != nil {
			result := finding.(LaravelCatchBlockFinding)
			catchMissing = result.MissingReport
			catchMisplaced = result.MisplacedReport
			for i := range result.Issues {
				result.Issues[i].Path = path
			}
			allIssues = append(allIssues, result.Issues...)
		}
	}

	if analysis == nil && len(allIssues) == 0 {
		return nil
	}

	if analysis == nil {
		// Create a basic analysis object if we only have other issues
		analysis = &models.PHPFileAnalysis{
			Path:       path,
			TotalBytes: len(content),
		}
	}

	analysis.CatchBlocksMissingReport = catchMissing
	analysis.CatchBlocksMisplacedReport = catchMisplaced

	analysis.Issues = allIssues
	return analysis
}

func (a *PHPAnalyzer) printResults(results []models.PHPFileAnalysis, totalFunctions, totalCommented int) {
	if len(results) == 0 {
		fmt.Println("✅ No PHP files with commented functions found!")
		return
	}

	fmt.Printf("Found %d files with commented functions\n", len(results))
	fmt.Printf("📊 Total Functions: %d | Commented: %d (%.1f%%)\n\n",
		totalFunctions, totalCommented,
		float64(totalCommented)/float64(totalFunctions)*100)

	fmt.Printf("%-5s %-60s %10s %10s %10s\n",
		"Rank", "File", "Total", "Commented", "Ratio")
	fmt.Println(strings.Repeat("-", 100))

	for i, result := range results {
		relPath := utils.Truncate(result.Path, 60)
		fmt.Printf("%-5d %-60s %10d %10d %9.1f%%\n",
			i+1, relPath,
			result.TotalFunctions,
			result.CommentedFunctions,
			result.CommentRatio)

		// Optional: Print catch block warnings if present
		if result.CatchBlocksMissingReport > 0 || result.CatchBlocksMisplacedReport > 0 {
			fmt.Printf("      ⚠️  Catch Blocks: %d missing report(), %d misplaced\n",
				result.CatchBlocksMissingReport, result.CatchBlocksMisplacedReport)
		}
	}

	fmt.Println()
	a.printTop10(results)
	fmt.Println("✅ Analysis complete!")
}

func (a *PHPAnalyzer) printTop10(results []models.PHPFileAnalysis) {
	fmt.Printf("📋 Top 10 Files with Commented Functions:\n")
	fmt.Println(strings.Repeat("-", 80))

	topCount := utils.Min(10, len(results))
	for i := 0; i < topCount; i++ {
		r := results[i]
		fmt.Printf("%2d. %s\n", i+1, r.Path)
		fmt.Printf("    📊 %d/%d functions commented (%.1f%%)\n",
			r.CommentedFunctions, r.TotalFunctions, r.CommentRatio)
		if len(r.CommentedList) > 0 {
			fmt.Printf("    💀 Commented: %s\n",
				strings.Join(r.CommentedList[:utils.Min(5, len(r.CommentedList))], ", "))
		}
	}
	fmt.Println()
}

func (a *PHPAnalyzer) generateArtifact(results []models.PHPFileAnalysis, config analyzers.Config, totalFunctions, totalCommented int) error {
	report := models.PHPAnalysisReport{
		Timestamp:          utils.GetTimestamp(),
		ScanDirectory:      config.RootDir,
		TotalFiles:         len(results),
		TotalFunctions:     totalFunctions,
		CommentedFunctions: totalCommented,
		Results:            results,
	}

	return utils.WriteArtifact(config.OutputFile, report)
}

// CommentedFunctionsRule detects commented-out PHP functions
type CommentedFunctionsRule struct{}

type CommentedFunctionsFinding struct {
	AllFunctions  []string
	CommentedList []string
	Issues        []models.Issue
}

func (r *CommentedFunctionsRule) Name() string {
	return "Commented Functions Detector"
}

func (r *CommentedFunctionsRule) Apply(content string) interface{} {
	cleanCode := removePHPComments(content)
	allFunctions := findPHPFunctions(content)
	activeFunctions := findPHPFunctions(cleanCode)
	commentedFunctions := difference(allFunctions, activeFunctions)

	if len(commentedFunctions) == 0 {
		return nil
	}

	var issues []models.Issue
	for _, funcName := range commentedFunctions {
		// Find line number of commented function
		// We use a regex specific to this function name
		funcRegex := regexp.MustCompile(`(?m)(?:^|[\s/]+|[*]+)\s*(?:public|private|protected|static)?\s*function\s+` + regexp.QuoteMeta(funcName) + `\s*\(`)
		loc := funcRegex.FindStringIndex(content)

		line := 0
		if loc != nil {
			line = strings.Count(content[:loc[0]], "\n") + 1
		}

		issues = append(issues, models.Issue{
			Description: fmt.Sprintf("Commented out PHP function: %s", funcName),
			Line:        line,
			Severity:    "major",
		})
	}

	return CommentedFunctionsFinding{
		AllFunctions:  allFunctions,
		CommentedList: commentedFunctions,
		Issues:        issues,
	}
}

func removePHPComments(code string) string {
	code = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(code, "")
	lines := strings.Split(code, "\n")
	var cleanLines []string
	for _, line := range lines {
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		cleanLines = append(cleanLines, line)
	}
	return strings.Join(cleanLines, "\n")
}

func findPHPFunctions(code string) []string {
	functions := []string{}
	functionRegex := regexp.MustCompile(`(?m)(?:^|[\s/]+|[*]+)\s*(?:public|private|protected|static)?\s*function\s+(\w+)\s*\(`)
	matches := functionRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			funcName := match[1]
			if funcName != "__construct" && funcName != "__destruct" {
				functions = append(functions, funcName)
			}
		}
	}
	return functions
}

func difference(a, b []string) []string {
	mb := make(map[string]bool, len(b))
	for _, x := range b {
		mb[x] = true
	}
	var diff []string
	for _, x := range a {
		if !mb[x] {
			diff = append(diff, x)
		}
	}
	return diff
}
