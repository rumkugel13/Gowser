package css

import (
	"gowser/html"
)

type Selector interface {
	Matches(node html.Node) bool
}

type TagSelector struct {
	Tag string
}

func NewTagSelector(tag string) *TagSelector {
	return &TagSelector{Tag: tag}
}

func (s *TagSelector) Matches(node html.Node) bool {
	if tag, ok := node.Token.(html.TagToken); ok {
		return tag.Tag == s.Tag
	}
	return false
}

type DescendantSelector struct {
	Ancestor Selector
	Descendant Selector
}

func NewDescendantSelector(ancestor Selector, descendant Selector) *DescendantSelector {
	return &DescendantSelector{
		Ancestor:   ancestor,
		Descendant: descendant,
	}
}

func (s *DescendantSelector) Matches(node html.Node) bool {
	if !s.Descendant.Matches(node) {
		return false
	}
	for node.Parent != nil {
		if s.Ancestor.Matches(*node.Parent) {
			return true
		}
		node = *node.Parent
	}
	return false
}