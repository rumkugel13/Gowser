package layout

import (
    "gowser/html"
    "testing"
)

func TestBasicLayout(t *testing.T) {
    // Create a simple HTML structure
    htmlNode := &html.Node{
        Token: html.TagToken{Tag: "html"},
        Children: []*html.Node{
            {
                Token: html.TagToken{Tag: "body"},
                Children: []*html.Node{
                    {
                        Token: html.TextToken{Text: "Hello World"},
                        Style: map[string]string{
                            "font-size":   "16",
                            "font-weight": "normal",
                            "font-style":  "roman",
                            "color":       "black",
                        },
                    },
                },
                Style: map[string]string{},
            },
        },
        Style: map[string]string{},
    }

    // Create document layout
    doc := NewDocumentLayout(htmlNode)
    doc.Layout()

    // Basic tests
    if doc.Width() != DefaultWidth-2*HSTEP {
        t.Errorf("Document width incorrect, got: %f, want: %f", 
            doc.Width(), DefaultWidth-2*HSTEP)
    }

    if doc.X() != HSTEP {
        t.Errorf("Document X position incorrect, got: %f, want: %f", 
            doc.X(), HSTEP)
    }

    if doc.Y() != VSTEP {
        t.Errorf("Document Y position incorrect, got: %f, want: %f", 
            doc.Y(), VSTEP)
    }

    // Test children
    if len(doc.Children()) != 1 {
        t.Fatalf("Expected 1 child, got %d", len(doc.Children()))
    }

    // Test paint commands
    var commands []Command
    PaintTree(doc, &commands)
    if len(commands) == 0 {
        t.Error("Expected paint commands, got none")
    }
}