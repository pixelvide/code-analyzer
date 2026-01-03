package php

import (
	"code-analyzer/models"

	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/node/stmt"
	"github.com/z7zmey/php-parser/php7"
	"github.com/z7zmey/php-parser/walker"
)

// LaravelCatchBlockRule checks for proper error reporting in try-catch blocks
type LaravelCatchBlockRule struct{}

func (r *LaravelCatchBlockRule) Name() string {
	return "Laravel Catch Block Rule"
}

// LaravelCatchBlockFinding holds the issues found by the rule
type LaravelCatchBlockFinding struct {
	Issues         []models.Issue
	MissingReport  int
	MisplacedReport int
}

func (r *LaravelCatchBlockRule) Apply(content string) interface{} {
	parser := php7.NewParser([]byte(content), "7.4")
	parser.Parse()

	root := parser.GetRootNode()
	if root == nil {
		return nil
	}

	v := &catchVisitor{
		issues: []models.Issue{},
	}
	root.Walk(v)

	if len(v.issues) == 0 {
		return nil
	}

	return LaravelCatchBlockFinding{
		Issues:          v.issues,
		MissingReport:   v.missingReport,
		MisplacedReport: v.misplacedReport,
	}
}

type catchVisitor struct {
	issues          []models.Issue
	missingReport   int
	misplacedReport int
}

// Ensure catchVisitor implements walker.Visitor
var _ walker.Visitor = (*catchVisitor)(nil)

func (v *catchVisitor) EnterNode(w walker.Walkable) bool {
	if n, ok := w.(node.Node); ok {
		if catchNode, ok := n.(*stmt.Catch); ok {
			v.analyzeCatch(catchNode)
		}
	}
	return true
}

func (v *catchVisitor) LeaveNode(w walker.Walkable) {
	// no-op
}

func (v *catchVisitor) EnterChildNode(key string, w walker.Walkable) {
	// no-op
}

func (v *catchVisitor) LeaveChildNode(key string, w walker.Walkable) {
	// no-op
}

func (v *catchVisitor) EnterChildList(key string, w walker.Walkable) {
	// no-op
}

func (v *catchVisitor) LeaveChildList(key string, w walker.Walkable) {
	// no-op
}

func (v *catchVisitor) analyzeCatch(n *stmt.Catch) {
	// Check statements in the catch block
	stmts := n.Stmts

	foundReport := false
	isFirst := false

	for i, s := range stmts {
		// Look for report(...) call
		if isReportCall(s) {
			foundReport = true
			if i == 0 {
				isFirst = true
			}
			break
		}
	}

	// Default line number 0 if position not available, but usually it is.
	startLine := 0
	if n.GetPosition() != nil {
		startLine = n.GetPosition().StartLine
	}

	if !foundReport {
		v.missingReport++
		v.issues = append(v.issues, models.Issue{
			Description: "Critical: Catch block missing report() call in Laravel app file",
			Line:        startLine,
			Severity:    "critical",
		})
	} else if !isFirst {
		v.misplacedReport++
		v.issues = append(v.issues, models.Issue{
			Description: "Medium Risk: report() call is not the first statement in catch block",
			Line:        startLine,
			Severity:    "medium",
		})
	}
}

func isReportCall(n node.Node) bool {
	// We expect an expression statement containing a function call
	if exprStmt, ok := n.(*stmt.Expression); ok {
		if funcCall, ok := exprStmt.Expr.(*expr.FunctionCall); ok {
			// Check function name
			if nameNode, ok := funcCall.Function.(*name.Name); ok {
				// name.Name parts are parts of the namespace/name
				// For "report", it should be a single part "report"
				parts := nameNode.Parts
				if len(parts) == 1 {
					if s, ok := parts[0].(*name.NamePart); ok {
						return s.Value == "report"
					}
				}
			}
		}
	}
	return false
}
