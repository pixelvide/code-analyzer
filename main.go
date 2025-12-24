package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code-analyzer/analyzers"
	"code-analyzer/analyzers/conflicts"
	"code-analyzer/analyzers/html"
	"code-analyzer/analyzers/js"
	"code-analyzer/analyzers/php"
	"code-analyzer/config"
	"code-analyzer/models"
)

func main() {
	// CLI flags
	configFile := flag.String("config", "analysis-config.yaml", "Path to YAML configuration file")
	flag.Parse()

	// Load config file
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config file: %v\n", err)
		os.Exit(1)
	}

	// Build analyzer list
	var analyzersToRun []struct {
		Name      string
		Analyzer  analyzers.Analyzer
		Extension string
	}
	allAnalyzers := map[string]analyzers.Analyzer{
		"html":      html.NewHTMLAnalyzer(),
		"php":       php.NewPHPAnalyzer(),
		"js":        js.NewJSAnalyzer(),
		"conflicts": conflicts.NewConflictsAnalyzer(),
	}

	analyzersConfig := make(map[string]config.AnalyzerConfig)

	// Determine which analyzers to run based on config
	for name, analyzerCfg := range cfg.Analyzers {
		if analyzerCfg.Enabled {
			if analyzer, exists := allAnalyzers[name]; exists {
				analyzersToRun = append(analyzersToRun, struct {
					Name      string
					Analyzer  analyzers.Analyzer
					Extension string
				}{
					Name:      strings.ToUpper(name),
					Analyzer:  analyzer,
					Extension: name,
				})
				analyzersConfig[name] = analyzerCfg
			} else {
				fmt.Fprintf(os.Stderr, "⚠️  Unknown analyzer in config: %s\n", name)
			}
		}
	}

	if len(analyzersToRun) == 0 {
		fmt.Fprintf(os.Stderr, "No enabled analyzers found in config\n")
		os.Exit(1)
	}

	fmt.Printf("🔍 Code Analysis Tool (ALL ANALYZERS)\n")
	fmt.Println(strings.Repeat("=", 61))
	fmt.Printf("Config File: %s\n", *configFile)
	fmt.Printf("Scanning: %s\n", cfg.Dir)
	fmt.Printf("Running: %d analyzers\n", len(analyzersToRun))
	fmt.Println()

	successCount := 0
	var allIssues []struct {
		Analyzer string
		Issue    models.Issue
	}

	// Run all updated analyzers
	for i, item := range analyzersToRun {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("📊 Running Analyzer %d/%d: %s\n", i+1, len(analyzersToRun), item.Name)
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()

		// Get specific config for this analyzer from YAML
		analyzerYamlCfg := analyzersConfig[item.Extension]

		// Map YAML config to run config
		runConfig := analyzers.Config{
			RootDir:      cfg.Dir,
			TopN:         analyzerYamlCfg.TopN,
			MinValue:     analyzerYamlCfg.Min,
			MinRatio:     analyzerYamlCfg.MinRatio,
			SortBy:       analyzerYamlCfg.Sort,
			ExcludePaths: analyzerYamlCfg.Exclude,
		}

		// Set default values if not present
		if runConfig.SortBy == "" {
			runConfig.SortBy = "ratio"
		}
		if runConfig.MinValue == 0 {
			runConfig.MinValue = 1
		}
		if runConfig.TopN == 0 {
			runConfig.TopN = 100
		}

		// Set output file
		if cfg.Output != "" {
			runConfig.OutputFile = filepath.Join(cfg.Output, fmt.Sprintf("%s-analysis.json", item.Extension))
		}

		issues, err := item.Analyzer.Run(runConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Analyzer %s failed: %v\n", item.Name, err)
		} else {
			successCount++
			for _, issue := range issues {
				allIssues = append(allIssues, struct {
					Analyzer string
					Issue    models.Issue
				}{
					Analyzer: item.Extension,
					Issue:    issue,
				})
			}
		}
	}

	// Generate GitLab Code Quality Report if configured
	if cfg.GitLabReport != "" {
		// If configured with artifacts directory, put it there
		reportPath := cfg.GitLabReport
		// We do NOT automatically join with cfg.Output anymore, as that forces it into artifacts/
		// Users should specify full relative path in config if they want it in artifacts/

		if err := generateGitLabReport(reportPath, allIssues); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to generate GitLab report: %v\n", err)
		} else {
			fmt.Printf("\n✅ GitLab Code Quality Report generated: %s\n", reportPath)
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	if successCount == len(analyzersToRun) {
		fmt.Printf("✅ Analysis Complete: %d/%d analyzers succeeded\n", successCount, len(analyzersToRun))
	} else {
		fmt.Printf("⚠️  Analysis Complete: %d/%d analyzers succeeded\n", successCount, len(analyzersToRun))
		os.Exit(1)
	}
	fmt.Println(strings.Repeat("=", 60))
}

func generateGitLabReport(outputPath string, findings []struct {
	Analyzer string
	Issue    models.Issue
}) error {
	var report []models.CodeQualityIssue

	for _, finding := range findings {
		// Create fingerprint
		hashContent := fmt.Sprintf("%s:%d:%s", finding.Issue.Description, finding.Issue.Line, finding.Issue.Path)
		hasher := md5.New()
		hasher.Write([]byte(hashContent))
		fingerprint := hex.EncodeToString(hasher.Sum(nil))

		// Ensure path is relative to project root if possible
		// finding.Issue.Path should already be relative or absolute depending on how it was found.

		report = append(report, models.CodeQualityIssue{
			Description: finding.Issue.Description,
			CheckName:   fmt.Sprintf("%s-check", finding.Analyzer),
			Fingerprint: fingerprint,
			Severity:    finding.Issue.Severity,
			Location: models.Location{
				Path: finding.Issue.Path,
				Lines: models.Lines{
					Begin: finding.Issue.Line,
				},
			},
		})
	}

	// Write to file
	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
