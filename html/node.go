package html

import (
	"fmt"
	"strings"
)

type Node struct {
	Token    Token
	Children *[]Node
	Parent   *Node
	Style    map[string]string
}

func NewNode(token Token, parent *Node) Node {
	return Node{
		Token:    token,
		Children: &[]Node{},
		Parent:   parent,
		Style:    make(map[string]string),
	}
}

func (n *Node) PrintTree(indent int) {
	fmt.Println(strings.Repeat(" ", indent) + n.Token.String())
	for _, child := range *n.Children {
		child.PrintTree(indent + 2)
	}
}

func TreeToList(tree *Node, list *[]Node) []Node {
	*list = append(*list, *tree)
	for _, child := range *tree.Children {
		TreeToList(&child, list)
	}
	return *list
}
