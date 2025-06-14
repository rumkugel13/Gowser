package accessibility

import (
	"fmt"
	"gowser/html"
	"strings"
)

type AccessibilityNode struct {
	node     *html.HtmlNode
	children []*AccessibilityNode
	role     string
}

func NewAccessibilityNode(node *html.HtmlNode) *AccessibilityNode {
	a := &AccessibilityNode{
		node:     node,
		children: make([]*AccessibilityNode, 0),
	}

	if _, isText := node.Token.(html.TextToken); isText {
		if html.IsFocusable(node.Parent) {
			a.role = "focusable text"
		} else {
			a.role = "StaticText"
		}
	} else {
		elt := node.Token.(html.ElementToken)
		if val, ok := elt.Attributes["role"]; ok {
			a.role = val
		} else if elt.Tag == "a" {
			a.role = "link"
		} else if elt.Tag == "input" {
			a.role = "textbox"
		} else if elt.Tag == "button" {
			a.role = "button"
		} else if elt.Tag == "html" {
			a.role = "document"
		} else if html.IsFocusable(node) {
			a.role = "focusable"
		} else {
			a.role = "none"
		}
	}
	return a
}

func (a *AccessibilityNode) String() string {
	return fmt.Sprint("AccessibilityNode(role='", a.role, "')")
}

func (a *AccessibilityNode) Build() {
	for _, child_node := range a.node.Children {
		a.build_internal(child_node)
	}
}

func (a *AccessibilityNode) build_internal(child_node *html.HtmlNode) {
	child := NewAccessibilityNode(child_node)
	if child.role != "none" {
		a.children = append(a.children, child)
		child.Build()
	} else {
		for _, grandchild_node := range child_node.Children {
			a.build_internal(grandchild_node)
		}
	}
}

func (n *AccessibilityNode) PrintTree(indent int) {
	fmt.Println(strings.Repeat(" ", indent) + n.String())
	for _, child := range n.children {
		child.PrintTree(indent + 2)
	}
}
