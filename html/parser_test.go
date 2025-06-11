package html

import (
	"fmt"
	"strings"
	"testing"
)

func ValidateParentChildRelationships(node *HtmlNode) []string {
	issues := []string{}

	// Check this node's children
	for i, child := range node.Children {
		if child.Parent != node {
			issues = append(issues, fmt.Sprintf(
				"Child %d (%v) has incorrect parent pointer (expected: %v, got: %v)",
				i, child.Token, node.Token, child.Parent.Token))
		}
		// Recursively check children
		issues = append(issues, ValidateParentChildRelationships(child)...)
	}

	return issues
}

func TestParentChildRelationships(t *testing.T) {
	html := `<html><body><div>text</div></body></html>`
	parser := NewHTMLParser(html)
	root := parser.Parse()

	issues := ValidateParentChildRelationships(root)
	if len(issues) > 0 {
		t.Errorf("Found parent-child relationship issues:\n%s",
			strings.Join(issues, "\n"))
	}
}
