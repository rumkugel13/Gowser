package layout

import "gowser/html"

type LayoutNode struct {
	Node                *html.Node
	Layout              Layout
	parent              *LayoutNode
	children            []*LayoutNode
	X, Y, Width, Height float32
}

func NewLayoutNode(layout Layout, htmlNode *html.Node, parent *LayoutNode) *LayoutNode {
	node := &LayoutNode{
		Node:     htmlNode,
		Layout:   layout,
		parent:   parent,
		children: make([]*LayoutNode, 0),
	}
	layout.Wrap(node)
	return node
}
