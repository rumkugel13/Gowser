package layout

import (
	"gowser/css"
	"gowser/html"
	"gowser/rect"

	"golang.org/x/image/font"
)

type LayoutNode struct {
	Node                *html.HtmlNode
	Layout              Layout
	Parent              *LayoutNode
	Children            []*LayoutNode
	X, Y, Width, Height float64
	Zoom                float64
	Font                font.Face
	Ascent              float64
	Descent             float64
}

func NewLayoutNode(layout Layout, htmlNode *html.HtmlNode, parent *LayoutNode) *LayoutNode {
	node := &LayoutNode{
		Node:     htmlNode,
		Layout:   layout,
		Parent:   parent,
		Children: make([]*LayoutNode, 0),
	}
	layout.Wrap(node)
	htmlNode.LayoutObject = node
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

func (l *LayoutNode) self_rect() *rect.Rect {
	return rect.NewRect(l.X, l.Y, l.X+l.Width, l.Y+l.Height)
}
