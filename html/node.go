package html

import (
	"fmt"
	"gowser/animate"
	"strings"
)

type HtmlNode struct {
	Token      Token
	Children   []*HtmlNode
	Parent     *HtmlNode
	Style      map[string]string
	Animations map[string]animate.Animation
	BlendOp    VisualEffectCommand
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
