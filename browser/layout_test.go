package browser

import (
	"testing"
)

func TestBasicLayout(t *testing.T) {
	// Create a simple HTML structure
	htmlNode := &HtmlNode{
		Token: ElementToken{Tag: "html"},
		Children: []*HtmlNode{
			{
				Token: ElementToken{Tag: "body"},
				Children: []*HtmlNode{
					{
						Token: TextToken{Text: "Hello World"},
						Style: map[string]*ProtectedField[string]{
							"font-size":   {Value: "16"},
							"font-weight": {Value: "normal"},
							"font-style":  {Value: "roman"},
							"color":       {Value: "black"},
						},
					},
				},
				Style: map[string]*ProtectedField[string]{},
			},
		},
		Style: map[string]*ProtectedField[string]{},
	}

	// Create document layout
	doc := NewLayoutNode(NewDocumentLayout(), htmlNode, nil, nil, nil)
	doc.Layout.(*DocumentLayout).LayoutWithZoom(1.0)

	// Basic tests
	if doc.Width.Get() != WIDTH-2*HSTEP {
		t.Errorf("Document width incorrect, got: %f, want: %f",
			doc.Width.Get(), WIDTH-2*HSTEP)
	}

	if doc.X.Get() != HSTEP {
		t.Errorf("Document X position incorrect, got: %f, want: %f",
			doc.X.Get(), HSTEP)
	}

	if doc.Y.Get() != VSTEP {
		t.Errorf("Document Y position incorrect, got: %f, want: %f",
			doc.Y.Get(), VSTEP)
	}

	// Test children
	if len(doc.Children.Get()) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(doc.Children.Get()))
	}

	// Test paint commands
	var commands []Command
	PaintTree(doc, &commands)
	if len(commands) == 0 {
		t.Error("Expected paint commands, got none")
	}
}

func TestFontSizeAndStyleApplied(t *testing.T) {
	// Create a sample HtmlNode with inline style
	node := &HtmlNode{
		Token: ElementToken{Tag: "span", Attributes: map[string]string{
			"style": "font-size: 24px; font-style: italic;",
		}},
		Style: nil, // Will be initialized by init_style
	}

	// Simulate style application (you may need to adjust this call)
	// init_style(node)
	// Define empty rules for testing
	var rules []Rule
	// If you have a function to apply inline styles, call it here. Otherwise, assume init_style handles it.
	Style(node, rules, nil)

	// Check font-size
	fontSize := node.Style["font-size"].Get()
	if fontSize != "24px" {
		t.Errorf("Expected font-size '24px', got '%s'", fontSize)
	}

	// Check font-style
	fontStyle := node.Style["font-style"].Get()
	if fontStyle != "italic" {
		t.Errorf("Expected font-style 'italic', got '%s'", fontStyle)
	}
}
