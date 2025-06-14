package accessibility

import (
	"fmt"
	"gowser/html"
	"strings"
)

type AccessibilityNode struct {
	Node     *html.HtmlNode
	Children []*AccessibilityNode
	Role     string
	Text     string
}

func NewAccessibilityNode(node *html.HtmlNode) *AccessibilityNode {
	a := &AccessibilityNode{
		Node:     node,
		Children: make([]*AccessibilityNode, 0),
		Text:     "",
	}

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
