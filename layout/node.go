package layout

import (
	"gowser/css"
	"gowser/html"
	"gowser/rect"
)

type LayoutNode struct {
	Node                *html.HtmlNode
	Layout              Layout
	parent              *LayoutNode
	children            []*LayoutNode
	X, Y, Width, Height float64
	Zoom                float64
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

func AbsoluteBoundsForObj(obj *LayoutNode) *rect.Rect {
	rect := rect.NewRect(obj.X, obj.Y, obj.X+obj.Width, obj.Y+obj.Height)
	cur := obj.Node
	for cur != nil {
		// note: on err map returns default value, which is ""
		dx, dy := css.ParseTransform(cur.Style["transform"])
		rect = html.MapTranslation(rect, dx, dy, false)
		cur = cur.Parent
	}
	return rect
}
