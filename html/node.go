package html

import (
	"fmt"
	"gowser/animate"
	"image"
	"slices"
	"strconv"
	"strings"
)

type HtmlNode struct {
	Token        Token
	Children     []*HtmlNode
	Parent       *HtmlNode
	Style        map[string]string
	Animations   map[string]animate.Animation
	BlendOp      VisualEffectCommand
	LayoutObject any // layout.LayoutNode
	Image        image.Image
	Frame        any // browser.Frame
}

func NewNode(token Token, parent *HtmlNode) *HtmlNode {
	return &HtmlNode{
		Token:      token,
		Children:   []*HtmlNode{},
		Parent:     parent,
		Style:      make(map[string]string),
		Animations: make(map[string]animate.Animation),
	}
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
	if GetTabIndex(node) < 0 {
		return false
	} else if _, ok := node.Token.(ElementToken).Attributes["tabindex"]; ok {
		return true
	} else if _, ok := node.Token.(ElementToken).Attributes["contenteditable"]; ok {
		return true
	} else {
		return slices.Contains([]string{"input", "button", "a"}, node.Token.(ElementToken).Tag)
	}
}
