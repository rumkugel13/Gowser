package browser

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

func TestGetAttributes(t *testing.T) {
	parser := NewHTMLParser("")
	testCases := []struct {
		input    string
		wantTag  string
		wantAttr map[string]string
	}{
		{
			"div",
			"div",
			map[string]string{},
		},
		{
			"div class=\"main\"",
			"div",
			map[string]string{"class": "main"},
		},
		{
			"IMG SRC='test.jpg' ALT=\"Test Image\" data-test",
			"img",
			map[string]string{
				"src":       "test.jpg",
				"alt":       "Test Image",
				"data-test": "",
			},
		},
		{
			"input type=text required",
			"input",
			map[string]string{
				"type":     "text",
				"required": "",
			},
		},
		{
			`div class="sourceCode" id="cb18" data-replace="self.height/new_height" data-expected="False"`,
			"div",
			map[string]string{
				"class":         "sourceCode",
				"id":            "cb18",
				"data-replace":  "self.height/new_height",
				"data-expected": "False",
			},
		},
		{
			`div class="sourceCode" id="cb17" data-replace="children_dirty%20%3d%20True/children.mark()"`,
			"div",
			map[string]string{
				"class":        "sourceCode",
				"id":           "cb17",
				"data-replace": "children_dirty%20%3d%20True/children.mark()",
			},
		},
		{
			`pre class="sourceCode python"`,
			"pre",
			map[string]string{
				"class": "sourceCode python",
			},
		},
		{
			`pre 
			class="sourceCode python"`,
			"pre",
			map[string]string{
				"class": "sourceCode python",
			},
		},
		{
			"pre\nclass=\"sourceCode python\"",
			"pre",
			map[string]string{
				"class": "sourceCode python",
			},
		},
	}

	for _, tc := range testCases {
		gotTag, gotAttr := parser.get_attributes(tc.input)
		if gotTag != tc.wantTag {
			t.Errorf("get_attributes(%q) tag = %q, want %q", tc.input, gotTag, tc.wantTag)
		}
		if len(gotAttr) != len(tc.wantAttr) {
			t.Errorf("get_attributes(%q) attr count = %d, want %d", tc.input, len(gotAttr), len(tc.wantAttr))
		}
		for k, v := range tc.wantAttr {
			if gotAttr[k] != v {
				t.Errorf("get_attributes(%q) attr[%q] = %q, want %q", tc.input, k, gotAttr[k], v)
			}
		}
	}
}

func TestPreTagPreservesWhitespace(t *testing.T) {
	html := `<pre>
  line 1
    line 2
line   3
</pre>`
	parser := NewHTMLParser(html)
	root := parser.Parse()

	// Find the <pre> node
	var pre *HtmlNode
	for _, child := range TreeToList(root) {
		if el, ok := child.Token.(ElementToken); ok && el.Tag == "pre" {
			pre = child
			break
		}
	}
	if pre == nil {
		t.Fatal("<pre> node not found")
	}

	// Find the text node inside <pre>
	var text string
	for _, child := range pre.Children {
		if txt, ok := child.Token.(TextToken); ok {
			text = txt.Text
			break
		}
	}
	expected := `
  line 1
    line 2
line   3
`
	if text != expected {
		t.Errorf("expected preserved whitespace, got:\n%q\nwant:\n%q", text, expected)
	}
}
