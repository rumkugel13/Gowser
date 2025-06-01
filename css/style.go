package css

import (
	"gowser/html"
	"maps"
)

func Style(node *html.Node, rules []Rule) {
	for _, rule := range rules {
		selector := rule.Selector
		styles := rule.Body
		if !selector.Matches(*node) {
			continue
		}
		maps.Copy(node.Style, styles)
	}

	if tag, ok := node.Token.(html.TagToken); ok {
		if style, exists := tag.Attributes["style"]; exists {
			parser := NewCSSParser(style)
			pairs := parser.body()
			maps.Copy(node.Style, pairs)
		}
	}

	for _, child := range *node.Children {
		Style(&child, rules)
	}
}
