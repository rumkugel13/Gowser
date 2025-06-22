package browser

import (
	"testing"
)

func TestTagSelector(t *testing.T) {
	node := &HtmlNode{Token: ElementToken{Tag: "div"}}
	sel := NewTagSelector("div")
	if !sel.Matches(node) {
		t.Error("TagSelector should match node with correct tag")
	}
	sel2 := NewTagSelector("span")
	if sel2.Matches(node) {
		t.Error("TagSelector should not match node with different tag")
	}
}

func TestClassSelector(t *testing.T) {
	node := &HtmlNode{Token: ElementToken{Tag: "div", Attributes: map[string]string{"class": "foo bar"}}}
	sel := NewClassSelector(".foo")
	if !sel.Matches(node) {
		t.Error("ClassSelector should match node with correct class")
	}
	sel2 := NewClassSelector(".baz")
	if sel2.Matches(node) {
		t.Error("ClassSelector should not match node with missing class")
	}
}

func TestDescendantSelector(t *testing.T) {
	ancestor := &HtmlNode{Token: ElementToken{Tag: "section"}}
	child := &HtmlNode{Token: ElementToken{Tag: "div"}, Parent: ancestor}
	tagSel := NewTagSelector("section")
	childSel := NewTagSelector("div")
	descSel := NewDescendantSelector(tagSel, childSel)
	if !descSel.Matches(child) {
		t.Error("DescendantSelector should match when ancestor matches")
	}
	otherAncestor := &HtmlNode{Token: ElementToken{Tag: "article"}}
	child2 := &HtmlNode{Token: ElementToken{Tag: "div"}, Parent: otherAncestor}
	if descSel.Matches(child2) {
		t.Error("DescendantSelector should not match when ancestor does not match")
	}
}

func TestPseudoclassSelector(t *testing.T) {
	node := &HtmlNode{Token: ElementToken{Tag: "input", IsFocused: true}}
	baseSel := NewTagSelector("input")
	pseudoSel := NewPseudoclassSelector("focus", baseSel)
	if !pseudoSel.Matches(node) {
		t.Error("PseudoclassSelector should match when base matches and IsFocused is true")
	}
	node2 := &HtmlNode{Token: ElementToken{Tag: "input", IsFocused: false}}
	if pseudoSel.Matches(node2) {
		t.Error("PseudoclassSelector should not match when IsFocused is false")
	}
	node3 := &HtmlNode{Token: ElementToken{Tag: "button", IsFocused: true}}
	if pseudoSel.Matches(node3) {
		t.Error("PseudoclassSelector should not match when base does not match")
	}
}