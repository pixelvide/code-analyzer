package html

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
			content:  "<div>Hello</div>",
			expected: 0,
		},
		{
			name: "Regular comment",
			content: `
				<!-- This is a regular comment -->
				<div>Hello</div>
			`,
			expected: 0,
		},
		{
			name: "Commented code",
			content: `
				<!--
				<div class="hidden">
					<span>Old content</span>
				</div>
				-->
			`,
			expected: 50, // Approximate
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
			if tt.expected > 0 && finding.CommentedBytes == 0 {
				t.Errorf("expected commented code, got 0 bytes")
			}
			if tt.expected == 0 && finding.CommentedBytes > 0 {
				t.Errorf("expected 0 bytes, got %d", finding.CommentedBytes)
			}
		})
	}
}
