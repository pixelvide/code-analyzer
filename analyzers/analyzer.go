package analyzers

import "code-analyzer/models"

// Analyzer is the interface that all code analyzers must implement
type Analyzer interface {
	// Run executes the analysis and returns issues found
	Run(config Config) ([]models.Issue, error)

	// Name returns the analyzer name
	Name() string

	// Description returns what this analyzer does
	Description() string
}

// Config holds configuration for running an analyzer
type Config struct {
	RootDir      string
	TopN         int
	MinValue     int
	MinRatio     float64 // Minimum ratio (0-100) to include
	SortBy       string
	OutputFile   string
	ExcludePaths []string // Paths to exclude from analysis
}

// Rule represents a single analysis rule that can be applied
type Rule interface {
	// Name returns the rule name
	Name() string

	// Apply applies the rule to content and returns findings
	Apply(content string) interface{}
}
