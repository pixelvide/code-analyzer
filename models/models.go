package models

// Issue represents a specific finding in a file
type Issue struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
}

// CodeQualityIssue represents a GitLab Code Quality report issue
type CodeQualityIssue struct {
	Description string   `json:"description"`
	CheckName   string   `json:"check_name"`
	Fingerprint string   `json:"fingerprint"`
	Severity    string   `json:"severity"`
	Location    Location `json:"location"`
}

type Location struct {
	Path  string `json:"path"`
	Lines Lines  `json:"lines"`
}

type Lines struct {
	Begin int `json:"begin"`
}

// HTMLFileAnalysis represents analysis results for an HTML file
type HTMLFileAnalysis struct {
	Path           string  `json:"path"`
	TotalLines     int     `json:"total_lines"`
	CommentedLines int     `json:"commented_lines"`
	CommentedBytes int     `json:"commented_bytes"`
	TotalBytes     int     `json:"total_bytes"`
	CommentRatio   float64 `json:"comment_ratio"`
	LargestBlock   int     `json:"largest_block"`
	Issues         []Issue `json:"issues"`
}

// HTMLAnalysisReport represents the complete HTML analysis report
type HTMLAnalysisReport struct {
	Timestamp      string             `json:"timestamp"`
	ScanDirectory  string             `json:"scan_directory"`
	TotalFiles     int                `json:"total_files"`
	TotalCommented int                `json:"total_commented_bytes"`
	SortMode       string             `json:"sort_mode"`
	MinComments    int                `json:"min_comments"`
	Results        []HTMLFileAnalysis `json:"results"`
}

// PHPFileAnalysis represents analysis results for a PHP file
type PHPFileAnalysis struct {
	Path               string   `json:"path"`
	TotalFunctions     int      `json:"total_functions"`
	CommentedFunctions int      `json:"commented_functions"`
	FunctionList       []string `json:"function_list"`
	CommentedList      []string `json:"commented_list"`
	CommentRatio       float64  `json:"comment_ratio"`
	TotalBytes         int      `json:"total_bytes"`
	CommentedBytes     int      `json:"commented_bytes"`
	Issues             []Issue  `json:"issues"`
	// Laravel Catch Block metrics
	CatchBlocksMissingReport   int `json:"catch_blocks_missing_report,omitempty"`
	CatchBlocksMisplacedReport int `json:"catch_blocks_misplaced_report,omitempty"`
}

// PHPAnalysisReport represents the complete PHP analysis report
type PHPAnalysisReport struct {
	Timestamp          string            `json:"timestamp"`
	ScanDirectory      string            `json:"scan_directory"`
	TotalFiles         int               `json:"total_files"`
	TotalFunctions     int               `json:"total_functions"`
	CommentedFunctions int               `json:"commented_functions"`
	Results            []PHPFileAnalysis `json:"results"`
}

// ConflictFileAnalysis represents analysis results for a file with conflicts
type ConflictFileAnalysis struct {
	Path             string   `json:"path"`
	ConflictLines    []int    `json:"conflict_lines"`
	ConflictBlocks   int      `json:"conflict_blocks"`
	ConflictSnippets []string `json:"conflict_snippets"`
	Issues           []Issue  `json:"issues"`
}

// ConflictAnalysisReport represents the complete conflict analysis report
type ConflictAnalysisReport struct {
	Timestamp      string                 `json:"timestamp"`
	ScanDirectory  string                 `json:"scan_directory"`
	TotalFiles     int                    `json:"total_files"`
	TotalConflicts int                    `json:"total_conflicts"`
	Results        []ConflictFileAnalysis `json:"results"`
}

// JSFileAnalysis represents analysis results for a JS/TS file
type JSFileAnalysis struct {
	Path           string  `json:"path"`
	TotalLines     int     `json:"total_lines"`
	CommentedLines int     `json:"commented_lines"`
	CommentedBytes int     `json:"commented_bytes"`
	TotalBytes     int     `json:"total_bytes"`
	CommentRatio   float64 `json:"comment_ratio"`
	LargestBlock   int     `json:"largest_block"`
	Issues         []Issue `json:"issues"`
}

// JSAnalysisReport represents the complete JS analysis report
type JSAnalysisReport struct {
	Timestamp      string           `json:"timestamp"`
	ScanDirectory  string           `json:"scan_directory"`
	TotalFiles     int              `json:"total_files"`
	TotalCommented int              `json:"total_commented_bytes"`
	SortMode       string           `json:"sort_mode"`
	MinComments    int              `json:"min_comments"`
	Results        []JSFileAnalysis `json:"results"`
}
