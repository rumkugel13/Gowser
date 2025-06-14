package css

import (
	"gowser/html"
)

type Selector interface {
	Matches(node *html.HtmlNode) bool
	Priority() int
}

type TagSelector struct {
	Tag      string
	priority int
}

func NewTagSelector(tag string) *TagSelector {
	return &TagSelector{Tag: tag, priority: 1}
}

func (s *TagSelector) Matches(node *html.HtmlNode) bool {
	if element, ok := node.Token.(html.ElementToken); ok {
		return element.Tag == s.Tag
	}
	return false
}

func (s *TagSelector) Priority() int {
	return s.priority
}

type DescendantSelector struct {
	Ancestor   Selector
	Descendant Selector
	priority   int
}

func NewDescendantSelector(ancestor Selector, descendant Selector) *DescendantSelector {
	return &DescendantSelector{
		Ancestor:   ancestor,
		Descendant: descendant,
		priority:   ancestor.Priority() + descendant.Priority(),
	}
}

func (s *DescendantSelector) Matches(node *html.HtmlNode) bool {
	if !s.Descendant.Matches(node) {
		return false
	}
	for node.Parent != nil {
		if s.Ancestor.Matches(node.Parent) {
			return true
		}
		node = node.Parent
	}
	return false
}

func (s *DescendantSelector) Priority() int {
	return s.priority
}

type PseudoclassSelector struct {
	pseudoclass string
	base        Selector
	priority    int
}

func NewPseudoclassSelector(pseudoclass string, base Selector) *PseudoclassSelector {
	return &PseudoclassSelector{
		pseudoclass: pseudoclass,
		base:        base,
		priority:    base.Priority(),
	}
}

func (s *PseudoclassSelector) Matches(node *html.HtmlNode) bool {
	if !s.base.Matches(node) {
		return false
	}
	if s.pseudoclass == "focus" {
		return node.Token.(html.ElementToken).IsFocused
	} else {
		return false
	}
}

func (s *PseudoclassSelector) Priority() int {
	return s.priority
}
