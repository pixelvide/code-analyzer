package php

import (
	"testing"
)

func TestCommentedFunctionsRule_Apply(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedCount int
	}{
		{
			name:          "No comments",
			content:       "<?php echo 'hello'; ?>",
			expectedCount: 0,
		},
		{
			name: "Commented function single line",
			content: `
				// function oldMethod() {
				//    return false;
				// }
			`,
			expectedCount: 1,
		},
		{
			name: "Commented function multi line",
			content: `
				/*
				public function deprecatedMethod($arg) {
					$this->doSomething();
				}
				*/
			`,
			expectedCount: 1,
		},
		{
			name: "Regular comment",
			content: `
				// This is just a comment about a function
				// function is efficient
			`,
			expectedCount: 0,
		},
	}

	rule := &CommentedFunctionsRule{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Apply(tt.content)
			if result == nil {
				if tt.expectedCount > 0 {
					t.Errorf("expected %d commented functions, got nil", tt.expectedCount)
				}
				return
			}

			finding := result.(CommentedFunctionsFinding)
			if len(finding.CommentedList) != tt.expectedCount {
				t.Errorf("expected %d commented functions, got %d", tt.expectedCount, len(finding.CommentedList))
			}
		})
	}
}
