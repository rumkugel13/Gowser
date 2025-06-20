package browser

import (
	"fmt"
	"gowser/html"
	"gowser/rect"
	"strings"
)

type A11yNode interface {
	Node() *html.HtmlNode
	Parent() A11yNode
	Children() []A11yNode
	HitTest(float64, float64) A11yNode
	Build()
	Bounds() []*rect.Rect
	Role() string
	Text() string
	map_to_parent(*rect.Rect)
	String() string
}

type AccessibilityNode struct {
	node     *html.HtmlNode
	parent   A11yNode
	children []A11yNode
	role     string
	text     string
	bounds   []*rect.Rect
}

func NewAccessibilityNode(node *html.HtmlNode, parent A11yNode) *AccessibilityNode {
	a := &AccessibilityNode{
		node:     node,
		children: make([]A11yNode, 0),
		text:     "",
		parent:   parent,
	}
	a.bounds = a.compute_bounds()

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
		} else if elt.Tag == "img" {
			a.role = "image"
		} else if elt.Tag == "iframe" {
			a.role = "iframe"
		} else {
			a.role = "none"
		}
	}
	return a
}

func (a *AccessibilityNode) String() string {
	return fmt.Sprint("AccessibilityNode(role='", a.role, "', text='", a.text, "')")
}

func (a *AccessibilityNode) ContainsPoint(x, y float64) bool {
	for _, bound := range a.bounds {
		if bound.ContainsPoint(x, y) {
			return true
		}
	}
	return false
}

func (a *AccessibilityNode) HitTest(x, y float64) A11yNode {
	var node A11yNode
	if a.ContainsPoint(x, y) {
		node = a
	}
	for _, child := range a.children {
		res := child.HitTest(x, y)
		if res != nil {
			node = res
		}
	}
	return node
}

func (a *AccessibilityNode) Build() {
	for _, child_node := range a.node.Children {
		a.build_internal(child_node)
	}

	if a.role == "StaticText" {
		a.text = a.node.Token.String()
	} else if a.role == "focusable text" {
		a.text = "Focusable text: " + a.node.Token.(html.TextToken).Text
	} else if a.role == "focusable" {
		a.text = "Focusable element"
	} else if a.role == "textbox" {
		elt, _ := a.node.Token.(html.ElementToken)
		var value string
		if val, ok := elt.Attributes["value"]; ok {
			value = val
		} else if elt.Tag != "input" && len(a.node.Children) > 0 {
			if txt, isText := a.node.Children[0].Token.(html.TextToken); isText {
				value = txt.Text
			} else {
				value = ""
			}
		}
		a.text = "Input box: " + value
	} else if a.role == "button" {
		a.text = "Button"
	} else if a.role == "link" {
		a.text = "Link"
	} else if a.role == "alert" {
		a.text = "Alert"
	} else if a.role == "document" {
		a.text = "Document"
	} else if a.role == "image" {
		elt, _ := a.node.Token.(html.ElementToken)
		if val, ok := elt.Attributes["alt"]; ok {
			a.text = "Image: " + val
		} else {
			a.text = "Image"
		}
	}

	if elt, ok := a.node.Token.(html.ElementToken); ok && elt.IsFocused {
		a.text += " is focused"
	}
}

func (a *AccessibilityNode) build_internal(child_node *html.HtmlNode) {
	var child A11yNode
	if elt, ok := child_node.Token.(html.ElementToken); ok && elt.Tag == "iframe" && child_node.Frame != nil && child_node.Frame.(*Frame).Loaded {
		child = NewFrameAccessibilityNode(child_node, a)
	} else {
		child = NewAccessibilityNode(child_node, a)
	}

	if child.Role() != "none" {
		a.children = append(a.children, child)
		child.Build()
	} else {
		for _, grandchild_node := range child_node.Children {
			a.build_internal(grandchild_node)
		}
	}
}

func (a *AccessibilityNode) compute_bounds() []*rect.Rect {
	if a.node.LayoutObject != nil {
		return []*rect.Rect{AbsoluteBoundsForObj(a.node.LayoutObject.(*LayoutNode))}
	}

	if _, ok := a.node.Token.(html.TextToken); ok {
		return []*rect.Rect{}
	}

	inline := a.node.Parent
	bounds := []*rect.Rect{}
	for inline.LayoutObject == nil {
		inline = inline.Parent
	}

	for _, line := range inline.LayoutObject.(*LayoutNode).Children.Get() {
		line_bounds := rect.NewRectEmpty()
		for _, child := range line.Children.Get() {
			if child.Node.Parent == a.node {
				line_bounds = line_bounds.Union(rect.NewRect(
					child.X, child.Y, child.X+child.Width, child.Y+child.Height,
				))
			}
		}
		bounds = append(bounds, line_bounds)
	}
	return bounds
}

func (a *AccessibilityNode) map_to_parent(abs_bound *rect.Rect) {
	return
}

func absolute_bounds(a A11yNode) []*rect.Rect {
	abs_bounds := make([]*rect.Rect, 0)
	for _, bounds := range a.Bounds() {
		abs_bound := bounds.Clone()
		var obj A11yNode
		if _, ok := a.(*FrameAccessibilityNode); ok {
			obj = a.Parent()
		} else {
			obj = a
		}
		for obj != nil {
			obj.map_to_parent(abs_bound)
			obj = obj.Parent()
		}
		abs_bounds = append(abs_bounds, abs_bound)
	}
	return abs_bounds
}

func (a *AccessibilityNode) Node() *html.HtmlNode {
	return a.node
}

func (a *AccessibilityNode) Parent() A11yNode {
	return a.parent
}

func (a *AccessibilityNode) Children() []A11yNode {
	return a.children
}

func (a *AccessibilityNode) Bounds() []*rect.Rect {
	return a.bounds
}

func (a *AccessibilityNode) Role() string {
	return a.role
}

func (a *AccessibilityNode) Text() string {
	return a.text
}

type FrameAccessibilityNode struct {
	AccessibilityNode
	scroll float64
	zoom   float64
}

func NewFrameAccessibilityNode(node *html.HtmlNode, parent *AccessibilityNode) *FrameAccessibilityNode {
	return &FrameAccessibilityNode{
		AccessibilityNode: *NewAccessibilityNode(node, parent),
		scroll:            node.Frame.(*Frame).scroll,
		zoom:              node.LayoutObject.(*LayoutNode).Zoom.Get(),
	}
}

func (n *FrameAccessibilityNode) Build() {
	n.build_internal(n.node.Frame.(*Frame).Nodes)
}

func (n *FrameAccessibilityNode) HitTest(x, y float64) A11yNode {
	bounds := n.bounds[0]
	if !bounds.ContainsPoint(x, y) {
		return nil
	}
	new_x := x - bounds.Left - dpx(1, n.zoom)
	new_y := y - bounds.Top - dpx(1, n.zoom)
	var node A11yNode = &n.AccessibilityNode
	for _, child := range n.children {
		res := child.HitTest(new_x, new_y)
		if res != nil {
			node = res
		}
	}
	return node
}

func (n *FrameAccessibilityNode) map_to_parent(rect *rect.Rect) {
	bounds := n.bounds[0]
	rect.Offset(bounds.Left, bounds.Top-n.scroll)
	rect = rect.Union(bounds)
}

func (a *FrameAccessibilityNode) Node() *html.HtmlNode {
	return a.node
}

func (a *FrameAccessibilityNode) Parent() A11yNode {
	return a.parent
}

func (a *FrameAccessibilityNode) Children() []A11yNode {
	return a.children
}

func (a *FrameAccessibilityNode) Bounds() []*rect.Rect {
	return a.bounds
}

func (a *FrameAccessibilityNode) Role() string {
	return a.role
}

func (a *FrameAccessibilityNode) Text() string {
	return a.text
}

func A11yPrintTree(n A11yNode, indent int) {
	fmt.Println(strings.Repeat(" ", indent) + n.String())
	for _, child := range n.Children() {
		A11yPrintTree(child, indent+2)
	}
}

func A11yTreeToList(tree A11yNode) []A11yNode {
	list := []A11yNode{tree}
	for _, child := range tree.Children() {
		list = append(list, A11yTreeToList(child)...)
	}
	return list
}
