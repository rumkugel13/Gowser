package layout

type LayoutNode struct {
	Layout              Layout
	parent              *LayoutNode
	children            []*LayoutNode
	X, Y, Width, Height float32
}

func NewLayoutNode(layout Layout, parent *LayoutNode) *LayoutNode {
	node := &LayoutNode{
		Layout:   layout,
		parent:   parent,
		children: make([]*LayoutNode, 0),
	}
	layout.Wrap(node)
	return node
}
