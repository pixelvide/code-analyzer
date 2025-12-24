package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fKB", kb)
	}
	return fmt.Sprintf("%.2fMB", kb/1024)
}

// Truncate truncates a string to max length with ellipsis
func Truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return "..." + s[len(s)-maxLen+3:]
	}
	return s
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetTimestamp returns current timestamp or CI pipeline ID
func GetTimestamp() string {
	timestamp := time.Now().Format("2006-01-02T15:04:05Z07:00")
	if ciPipeline := os.Getenv("CI_PIPELINE_ID"); ciPipeline != "" {
		timestamp = ciPipeline
	}
	return timestamp
}

// ShouldSkip determines if a path should be skipped
func ShouldSkip(path string, customExcludes []string) bool {
	// Default excludes that apply to all analyzers
	defaultExcludes := []string{".git"}

	// Check default excludes
	for _, exclude := range defaultExcludes {
		if strings.Contains(path, exclude) {
			return true
		}
	}

	// Check custom excludes
	for _, exclude := range customExcludes {
		if strings.Contains(path, exclude) {
			return true
		}
	}

	return false

}

// WriteArtifact writes an artifact to JSON file
func WriteArtifact(outputPath string, report interface{}) error {
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode JSON: %v", err)
	}

	return nil
}
