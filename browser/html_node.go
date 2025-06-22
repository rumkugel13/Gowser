package browser

import (
	"fmt"
	"gowser/animate"
	"image"
	"slices"
	"strconv"
	"strings"
)

var (
	CSS_PROPERTIES = map[string]string{
		"font-size": "inherit", "font-weight": "inherit",
		"font-style": "inherit", "color": "inherit",
		"opacity": "1.0", "transition": "",
		"transform": "none", "mix-blend-mode": "",
		"border-radius": "0px", "overflow": "visible",
		"outline": "none", "background-color": "transparent",
		"image-rendering": "auto",
	}
)

type HtmlNode struct {
	Token        Token
	Children     []*HtmlNode
	Parent       *HtmlNode
	Style        map[string]*ProtectedField[string]
	Animations   map[string]animate.Animation
	BlendOp      VisualEffectCommand
	LayoutObject *LayoutNode
	Image        image.Image
	Frame        *Frame
}

func NewNode(token Token, parent *HtmlNode) *HtmlNode {
	node := &HtmlNode{
		Token:      token,
		Children:   []*HtmlNode{},
		Parent:     parent,
		Animations: make(map[string]animate.Animation),
	}
	node.Style = nil
	return node
}

func (n *HtmlNode) PrintTree(indent int) {
	fmt.Println(strings.Repeat(" ", indent) + n.Token.String())
	for _, child := range n.Children {
		child.PrintTree(indent + 2)
	}
}

func TreeToList(tree *HtmlNode) []*HtmlNode {
	list := []*HtmlNode{tree}
	for _, child := range tree.Children {
		list = append(list, TreeToList(child)...)
	}
	return list
}

func GetTabIndex(node *HtmlNode) int {
	var tabIndex int
	if element, ok := node.Token.(ElementToken); ok {
		if val, ok := element.Attributes["tabindex"]; ok {
			iVal, err := strconv.Atoi(val)
			if err != nil {
				iVal = 9999999
			}
			tabIndex = iVal
		} else {
			tabIndex = 9999999
		}
	}
	if tabIndex == 0 {
		return 9999999
	} else {
		return tabIndex
	}
}

func IsFocusable(node *HtmlNode) bool {
	if GetTabIndex(node) <= 0 {
		return false
	} else if _, ok := node.Token.(ElementToken).Attributes["tabindex"]; ok {
		return true
	} else if _, ok := node.Token.(ElementToken).Attributes["contenteditable"]; ok {
		return true
	} else {
		return slices.Contains([]string{"input", "button", "a"}, node.Token.(ElementToken).Tag)
	}
}

func dirty_style(node *HtmlNode) {
	for _, val := range node.Style {
		val.Mark()
	}
}

func init_style(node *HtmlNode) {
	style := map[string]*ProtectedField[string]{}
	for prop := range CSS_PROPERTIES {
		var dependencies []ProtectedMarker
		if node.Parent != nil && INHERITED_PROPERTIES[prop] != "" {
			if parentStyleValue, exists := node.Parent.Style[prop]; exists {
				dependencies = append(dependencies, parentStyleValue)
			}
		}
		style[prop] = NewProtectedField[string](node, prop, node.Parent, &dependencies)
	}
	node.Style = style
}
