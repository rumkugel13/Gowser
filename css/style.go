package css

import (
	"gowser/html"
	"maps"
)

func Style(node *html.Node, rules map[Selector]map[string]string) {
	for selector, styles := range rules {
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
