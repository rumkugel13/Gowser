package html

import "strings"
import "fmt"

type Node struct {
	Token Token
	Children *[]Node
	Parent *Node
	Attributes map[string]string
}

func NewNode(token Token, attributes map[string]string, parent *Node) Node {
	return Node{
		Token: token,
		Children: &[]Node{},
		Parent: parent,
		Attributes: attributes,
	}
}

func (n *Node) PrintTree(indent int) {
	fmt.Println(strings.Repeat(" ", indent) + n.Token.String())
	for _, child := range *n.Children {
		child.PrintTree(indent + 2)
	}
}