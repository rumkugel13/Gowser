package html

import (
	"fmt"
	"testing"
)

func TestNodeReferenceConsistency(t *testing.T) {
	// Create a simple node structure
	root := &Node{
		Token: ElementToken{Tag: "div"},
	}

	// Get the node through different paths
	list := TreeToList(root)
	firstRef := list[0]

	fmt.Printf("Original: %p\n", root)
	fmt.Printf("Through list: %p\n", firstRef)

	if root != firstRef {
		t.Error("Node references don't match after TreeToList")
	}
}
