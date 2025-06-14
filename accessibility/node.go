package accessibility

import (
	"fmt"
	"gowser/html"
	"gowser/layout"
	"gowser/rect"
	"strings"
)

type AccessibilityNode struct {
	Node     *html.HtmlNode
	Children []*AccessibilityNode
	Role     string
	Text     string
	Bounds   []*rect.Rect
}

func NewAccessibilityNode(node *html.HtmlNode) *AccessibilityNode {
	a := &AccessibilityNode{
		Node:     node,
		Children: make([]*AccessibilityNode, 0),
		Text:     "",
	}
	a.Bounds = a.compute_bounds()

	if _, isText := node.Token.(html.TextToken); isText {
		if html.IsFocusable(node.Parent) {
			a.Role = "focusable text"
		} else {
			a.Role = "StaticText"
		}
	} else {
		elt := node.Token.(html.ElementToken)
		if val, ok := elt.Attributes["role"]; ok {
			a.Role = val
		} else if elt.Tag == "a" {
			a.Role = "link"
		} else if elt.Tag == "input" {
			a.Role = "textbox"
		} else if elt.Tag == "button" {
			a.Role = "button"
		} else if elt.Tag == "html" {
			a.Role = "document"
		} else if html.IsFocusable(node) {
			a.Role = "focusable"
		} else {
			a.Role = "none"
		}
	}
	return a
}

func (a *AccessibilityNode) String() string {
	return fmt.Sprint("AccessibilityNode(role='", a.Role, "', text='", a.Text, "')")
}

func (a *AccessibilityNode) ContainsPoint(x, y float64) bool {
	for _, bound := range a.Bounds {
		if bound.ContainsPoint(x, y) {
			return true
		}
	}
	return false
}

func (a *AccessibilityNode) HitTest(x, y float64) *AccessibilityNode {
	var node *AccessibilityNode
	if a.ContainsPoint(x, y) {
		node = a
	}
	for _, child := range a.Children {
		res := child.HitTest(x, y)
		if res != nil {
			node = res
		}
	}
	return node
}

func (a *AccessibilityNode) Build() {
	for _, child_node := range a.Node.Children {
		a.build_internal(child_node)
	}

	if a.Role == "StaticText" {
		a.Text = a.Node.Token.String()
	} else if a.Role == "focusable text" {
		a.Text = "Focusable text: " + a.Node.Token.(html.TextToken).Text
	} else if a.Role == "focusable" {
		a.Text = "Focusable element"
	} else if a.Role == "textbox" {
		elt, _ := a.Node.Token.(html.ElementToken)
		var value string
		if val, ok := elt.Attributes["value"]; ok {
			value = val
		} else if elt.Tag != "input" && len(a.Node.Children) > 0 {
			if txt, isText := a.Node.Children[0].Token.(html.TextToken); isText {
				value = txt.Text
			} else {
				value = ""
			}
		}
		a.Text = "Input box: " + value
	} else if a.Role == "button" {
		a.Text = "Button"
	} else if a.Role == "link" {
		a.Text = "Link"
	} else if a.Role == "alert" {
		a.Text = "Alert"
	} else if a.Role == "document" {
		a.Text = "Document"
	}

	if elt, ok := a.Node.Token.(html.ElementToken); ok && elt.IsFocused {
		a.Text += " is focused"
	}
}

func (a *AccessibilityNode) build_internal(child_node *html.HtmlNode) {
	child := NewAccessibilityNode(child_node)
	if child.Role != "none" {
		a.Children = append(a.Children, child)
		child.Build()
	} else {
		for _, grandchild_node := range child_node.Children {
			a.build_internal(grandchild_node)
		}
	}
}

func (a *AccessibilityNode) compute_bounds() []*rect.Rect {
	if a.Node.LayoutObject != nil {
		return []*rect.Rect{layout.AbsoluteBoundsForObj(a.Node.LayoutObject.(*layout.LayoutNode))}
	}

	if _, ok := a.Node.Token.(html.TextToken); ok {
		return []*rect.Rect{}
	}

	inline := a.Node.Parent
	bounds := []*rect.Rect{}
	for inline.LayoutObject == nil {
		inline = inline.Parent
	}

	for _, line := range inline.LayoutObject.(*layout.LayoutNode).Children {
		line_bounds := rect.NewRectEmpty()
		for _, child := range line.Children {
			if child.Node.Parent == a.Node {
				line_bounds = line_bounds.Union(rect.NewRect(
					child.X, child.Y, child.X+child.Width, child.Y+child.Height,
				))
			}
		}
		bounds = append(bounds, line_bounds)
	}
	return bounds
}

func (n *AccessibilityNode) PrintTree(indent int) {
	fmt.Println(strings.Repeat(" ", indent) + n.String())
	for _, child := range n.Children {
		child.PrintTree(indent + 2)
	}
}

func TreeToList(tree *AccessibilityNode) []*AccessibilityNode {
	list := []*AccessibilityNode{tree}
	for _, child := range tree.Children {
		list = append(list, TreeToList(child)...)
	}
	return list
}
