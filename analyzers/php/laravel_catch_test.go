package php

import (
	"testing"
)

func TestLaravelCatchBlockRule_Apply(t *testing.T) {
	rule := &LaravelCatchBlockRule{}

	tests := []struct {
		name          string
		content       string
		wantIssues    int
		wantSeverity  string
		wantLine      int
	}{
		{
			name: "Critical: No report call",
			content: `<?php
namespace App\Http\Controllers;

class TestController {
    public function index() {
        try {
            // something
        } catch (\Exception $e) {
            // silent fail
        }
    }
}
`,
			wantIssues:   1,
			wantSeverity: "critical",
			wantLine:     8,
		},
		{
			name: "Medium: report call not first",
			content: `<?php
namespace App\Http\Controllers;

class TestController {
    public function index() {
        try {
            // something
        } catch (\Exception $e) {
            \Log::error($e);
            report($e);
        }
    }
}
`,
			wantIssues:   1,
			wantSeverity: "medium",
			wantLine:     8,
		},
		{
			name: "Valid: report call is first",
			content: `<?php
namespace App\Http\Controllers;

class TestController {
    public function index() {
        try {
            // something
        } catch (\Exception $e) {
            report($e);
            return response()->json(['error' => 'fail']);
        }
    }
}
`,
			wantIssues: 0,
		},
		{
			name: "Multiple catch blocks",
			content: `<?php
class Test {
    function test() {
        try {}
        catch (A $e) { report($e); }
        catch (B $e) {}
    }
}
`,
			wantIssues:   1,
			wantSeverity: "critical",
			wantLine:     6, // The line of catch (B $e)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Apply(tt.content)

			if tt.wantIssues == 0 {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected issues, got nil")
				return
			}

			finding, ok := result.(LaravelCatchBlockFinding)
			if !ok {
				t.Errorf("Expected LaravelCatchBlockFinding, got %T", result)
				return
			}

			if len(finding.Issues) != tt.wantIssues {
				t.Errorf("Expected %d issues, got %d", tt.wantIssues, len(finding.Issues))
			}

			if len(finding.Issues) > 0 {
				issue := finding.Issues[0]
				if issue.Severity != tt.wantSeverity {
					t.Errorf("Expected severity %s, got %s", tt.wantSeverity, issue.Severity)
				}
				if issue.Line != tt.wantLine {
					t.Errorf("Expected line %d, got %d", tt.wantLine, issue.Line)
				}
			}
		})
	}
}
