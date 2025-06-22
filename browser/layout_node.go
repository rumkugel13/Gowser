package browser

import (
	"gowser/rect"

	"golang.org/x/image/font"
)

type LayoutNode struct {
	Node                *HtmlNode
	Layout              Layout
	Parent              *LayoutNode
	Previous            *LayoutNode
	Children            *ProtectedField[[]*LayoutNode]
	X, Y, Width, Height *ProtectedField[float64]
	Zoom                *ProtectedField[float64]
	Font                *ProtectedField[font.Face]
	Ascent, Descent     *ProtectedField[float64]
	Frame               *Frame

	has_dirty_descendants bool
}

func NewLayoutNode(layout Layout, htmlNode *HtmlNode, parent, previous *LayoutNode, frame *Frame) *LayoutNode {
	node := &LayoutNode{
		Node:     htmlNode,
		Layout:   layout,
		Parent:   parent,
		Previous: previous,
		Frame:    frame,
	}
	switch layout.(type) {
	case *DocumentLayout:
		node.Children = NewProtectedField[[]*LayoutNode](node, "children", parent, nil)
		node.Zoom = NewProtectedField[float64](node, "zoom", parent, &[]ProtectedMarker{})
		node.Width = NewProtectedField[float64](node, "width", parent, &[]ProtectedMarker{})
		node.X = NewProtectedField[float64](node, "x", parent, &[]ProtectedMarker{})
		node.Y = NewProtectedField[float64](node, "y", parent, &[]ProtectedMarker{})
		node.Height = NewProtectedField[float64](node, "height", parent, nil)
		node.has_dirty_descendants = true
	case *BlockLayout:
		node.Children = NewProtectedField[[]*LayoutNode](node, "children", parent, nil)
		node.Zoom = NewProtectedField[float64](node, "zoom", parent, &[]ProtectedMarker{node.Parent.Zoom})
		node.Width = NewProtectedField[float64](node, "width", parent, &[]ProtectedMarker{node.Parent.Width})
		node.X = NewProtectedField[float64](node, "x", parent, &[]ProtectedMarker{node.Parent.X})
		if previous != nil {
			node.Y = NewProtectedField[float64](node, "y", parent, &[]ProtectedMarker{previous.Y, previous.Height})
		} else {
			node.Y = NewProtectedField[float64](node, "y", parent, &[]ProtectedMarker{parent.Y})
		}
		node.Height = NewProtectedField[float64](node, "height", parent, nil)
		node.has_dirty_descendants = true
	case *LineLayout:
		node.Children = &ProtectedField[[]*LayoutNode]{Dirty: false}
		node.Zoom = NewProtectedField[float64](node, "zoom", parent, &[]ProtectedMarker{node.Parent.Zoom})
		node.X = NewProtectedField[float64](node, "x", parent, &[]ProtectedMarker{node.Parent.X})
		if previous != nil {
			node.Y = NewProtectedField[float64](node, "y", parent, &[]ProtectedMarker{node.Previous.Y, node.Previous.Height})
		} else {
			node.Y = NewProtectedField[float64](node, "y", parent, &[]ProtectedMarker{node.Parent.Y})
		}
		node.Layout.(*LineLayout).initialized_fields = false
		node.Width = NewProtectedField[float64](node, "width", parent, &[]ProtectedMarker{node.Parent.Width})
		node.Ascent = NewProtectedField[float64](node, "ascent", parent, nil)
		node.Descent = NewProtectedField[float64](node, "descent", parent, nil)
		node.Height = NewProtectedField[float64](node, "height", parent, &[]ProtectedMarker{node.Ascent, node.Descent})
		node.has_dirty_descendants = true
	case *TextLayout:
		node.Children = &ProtectedField[[]*LayoutNode]{Dirty: false}
		node.Zoom = NewProtectedField[float64](node, "zoom", parent, &[]ProtectedMarker{node.Parent.Zoom})
		node.Font = NewProtectedField[font.Face](node, "font", parent, &[]ProtectedMarker{
			node.Zoom,
			node.Node.Style["font-weight"],
			node.Node.Style["font-style"],
			node.Node.Style["font-size"],
		})
		node.Width = NewProtectedField[float64](node, "width", parent, &[]ProtectedMarker{node.Font})
		node.Height = NewProtectedField[float64](node, "height", parent, &[]ProtectedMarker{node.Font})
		node.Ascent = NewProtectedField[float64](node, "ascent", parent, &[]ProtectedMarker{node.Font})
		node.Descent = NewProtectedField[float64](node, "descent", parent, &[]ProtectedMarker{node.Font})
		if node.Previous != nil {
			node.X = NewProtectedField[float64](node, "x", parent, &[]ProtectedMarker{node.Previous.X, node.Previous.Font, node.Previous.Width})
		} else {
			node.X = NewProtectedField[float64](node, "x", parent, &[]ProtectedMarker{node.Parent.X})
		}
		node.Y = NewProtectedField[float64](node, "y", parent, &[]ProtectedMarker{node.Ascent, node.Parent.Y, node.Parent.Ascent})
		node.has_dirty_descendants = true
	case *EmbedLayout, *InputLayout, *ImageLayout, *IframeLayout:
		node.Children = &ProtectedField[[]*LayoutNode]{Dirty: false}
		node.Zoom = NewProtectedField[float64](node, "zoom", parent, &[]ProtectedMarker{node.Parent.Zoom})
		node.Font = NewProtectedField[font.Face](node, "font", parent, &[]ProtectedMarker{
			node.Zoom,
			node.Node.Style["font-weight"],
			node.Node.Style["font-style"],
			node.Node.Style["font-size"],
		})
		node.Width = NewProtectedField[float64](node, "width", parent, &[]ProtectedMarker{node.Zoom})
		node.Height = NewProtectedField[float64](node, "height", parent, &[]ProtectedMarker{node.Zoom, node.Font, node.Width})
		node.Ascent = NewProtectedField[float64](node, "ascent", parent, &[]ProtectedMarker{node.Height})
		node.Descent = NewProtectedField[float64](node, "descent", parent, &[]ProtectedMarker{})
		if previous != nil {
			node.X = NewProtectedField[float64](node, "x", parent, &[]ProtectedMarker{node.Previous.X, node.Previous.Font, node.Previous.Width})
		} else {
			node.X = NewProtectedField[float64](node, "x", parent, &[]ProtectedMarker{node.Parent.X})
		}
		node.Y = NewProtectedField[float64](node, "y", parent, &[]ProtectedMarker{node.Ascent, node.Parent.Y, node.Parent.Ascent})
	}

	// note: only invalid for documentlayout
	if parent != nil {
		parent.Zoom.invalidations[node.Zoom] = true
	}
	layout.Wrap(node)
	htmlNode.LayoutObject = node
	return node
}

func AbsoluteBoundsForObj(obj *LayoutNode) *rect.Rect {
	rect := rect.NewRect(obj.X.Get(), obj.Y.Get(), obj.X.Get()+obj.Width.Get(), obj.Y.Get()+obj.Height.Get())
	cur := obj.Node
	for cur != nil {
		// note: on err map returns default value, which is ""
		// another note: using get instead of value crashes
		dx, dy := ParseTransform(cur.Style["transform"].Value)
		rect = MapTranslation(rect, dx, dy, false)
		cur = cur.Parent
	}
	return rect
}

// todo: move this to layout, seperate function for seperate fields
func (l *LayoutNode) layout_needed() bool {
	if l.Children != nil && l.Children.Dirty {
		return true
	}
	if l.Zoom.Dirty {
		return true
	}
	if l.X.Dirty {
		return true
	}
	if l.Y.Dirty {
		return true
	}
	if l.Width.Dirty {
		return true
	}
	if l.Height.Dirty {
		return true
	}
	if l.Font != nil && l.Font.Dirty {
		return true
	}
	if l.Ascent != nil && l.Ascent.Dirty {
		return true
	}
	if l.Descent != nil && l.Descent.Dirty {
		return true
	}
	if l.has_dirty_descendants {
		return true
	}
	return false
}

func (l *LayoutNode) self_rect() *rect.Rect {
	return rect.NewRect(l.X.Get(), l.Y.Get(), l.X.Get()+l.Width.Get(), l.Y.Get()+l.Height.Get())
}
