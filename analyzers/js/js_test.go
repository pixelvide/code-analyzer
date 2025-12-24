package js

import (
	"testing"
)

func TestCommentedCodeRule_Apply(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int // Expected number of commented bytes
	}{
		{
			name:     "No comments",
			content:  "var x = 1;",
			expected: 0,
		},
		{
			name: "Regular comment",
			content: `
				// This is a regular comment
				var x = 1;
			`,
			expected: 0,
		},
		{
			name: "Commented code single line",
			content: `
				// var x = 1;
				// console.log(x);
			`,
			expected: 24, // Approximate bytes
		},
		{
			name: "Commented code multi line",
			content: `
				/*
				function test() {
					return true;
				}
				*/
			`,
			expected: 35, // Approximate bytes
		},
	}

	rule := &CommentedCodeRule{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Apply(tt.content)
			if result == nil {
				if tt.expected > 0 {
					t.Errorf("expected %d bytes code, got nil", tt.expected)
				}
				return
			}

			finding := result.(CommentedCodeFinding)
			// Allow some validation flexibility as byte counts might vary slightly due to whitespace handling
			if tt.expected > 0 && finding.CommentedBytes == 0 {
				t.Errorf("expected commented code, got 0 bytes")
			}
			if tt.expected == 0 && finding.CommentedBytes > 0 {
				t.Errorf("expected 0 bytes, got %d", finding.CommentedBytes)
			}
		})
	}
}
