package css

import (
	"gowser/html"
	"maps"
)

func Style(node *html.Node) {
	if tag, ok := node.Token.(html.TagToken); ok {
		if style, exists := tag.Attributes["style"]; exists {
			parser := NewCSSParser(style)
			pairs := parser.body()
			maps.Copy(node.Style, pairs)
		}
	}

	for _, child := range *node.Children {
		Style(&child)
	}
}
