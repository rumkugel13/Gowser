package css

import (
	"gowser/html"
)

type Selector interface {
	Matches(node *html.Node) bool
	Priority() int
}

type TagSelector struct {
	Tag string
	priority int
}

func NewTagSelector(tag string) *TagSelector {
	return &TagSelector{Tag: tag, priority: 1}
}

func (s *TagSelector) Matches(node *html.Node) bool {
	if tag, ok := node.Token.(html.TagToken); ok {
		return tag.Tag == s.Tag
	}
	return false
}

func (s *TagSelector) Priority() int {
	return s.priority
}

type DescendantSelector struct {
	Ancestor Selector
	Descendant Selector
	priority int
}

func NewDescendantSelector(ancestor Selector, descendant Selector) *DescendantSelector {
	return &DescendantSelector{
		Ancestor:   ancestor,
		Descendant: descendant,
		priority:   ancestor.Priority() + descendant.Priority(),
	}
}

func (s *DescendantSelector) Matches(node *html.Node) bool {
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