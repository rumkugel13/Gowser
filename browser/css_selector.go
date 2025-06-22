package browser

import (
	"slices"
	"strings"
)

type Selector interface {
	Matches(node *HtmlNode) bool
	Priority() int
}

type TagSelector struct {
	Tag      string
	priority int
}

func NewTagSelector(tag string) *TagSelector {
	return &TagSelector{Tag: tag, priority: 1}
}

func (s *TagSelector) Matches(node *HtmlNode) bool {
	if element, ok := node.Token.(ElementToken); ok {
		return element.Tag == s.Tag
	}
	return false
}

func (s *TagSelector) Priority() int {
	return s.priority
}

type ClassSelector struct {
	Class    string
	priority int
}

func NewClassSelector(class string) *ClassSelector {
	return &ClassSelector{Class: class, priority: 10}
}

func (s *ClassSelector) Matches(node *HtmlNode) bool {
	if element, ok := node.Token.(ElementToken); ok {
		return slices.Contains(strings.Fields(element.Attributes["class"]), s.Class[1:])
	}
	return false
}

func (s *ClassSelector) Priority() int {
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

func (s *DescendantSelector) Matches(node *HtmlNode) bool {
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

func (s *PseudoclassSelector) Matches(node *HtmlNode) bool {
	if !s.base.Matches(node) {
		return false
	}
	if s.pseudoclass == "focus" {
		return node.Token.(ElementToken).IsFocused
	} else {
		return false
	}
}

func (s *PseudoclassSelector) Priority() int {
	return s.priority
}
