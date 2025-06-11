package layout

import "gowser/html"

type LayoutNode struct {
	Node                *html.HtmlNode
	Layout              Layout
	parent              *LayoutNode
	children            []*LayoutNode
	X, Y, Width, Height float64
}

func NewLayoutNode(layout Layout, htmlNode *html.HtmlNode, parent *LayoutNode) *LayoutNode {
	node := &LayoutNode{
		Node:     htmlNode,
		Layout:   layout,
		parent:   parent,
		children: make([]*LayoutNode, 0),
	}
	layout.Wrap(node)
	return node
}
