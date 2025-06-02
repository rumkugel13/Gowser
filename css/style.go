package css

import (
	"gowser/html"
	"maps"
	"strconv"
	"strings"
)

var (
	INHERITED_PROPERTIES = map[string]string{
		"font-size":   "16px",
		"font-style":  "normal",
		"font-weight": "normal",
		"color":       "black",
	}
)

func Style(node *html.Node, rules []Rule) {
	node.Style = make(map[string]string)
	for property, default_value := range INHERITED_PROPERTIES {
		if node.Parent != nil {
			node.Style[property] = node.Parent.Style[property]
		} else {
			node.Style[property] = default_value
		}
	}

	for _, rule := range rules {
		selector := rule.Selector
		styles := rule.Body
		if !selector.Matches(node) {
			continue
		}
		maps.Copy(node.Style, styles)
	}

	if element, ok := node.Token.(html.ElementToken); ok {
		if style, exists := element.Attributes["style"]; exists {
			parser := NewCSSParser(style)
			pairs := parser.body()
			maps.Copy(node.Style, pairs)
		}
	}

	if strings.HasSuffix(node.Style["font-size"], "%") {
		var parent_font_size string
		if node.Parent != nil {
			parent_font_size = node.Parent.Style["font-size"]
		} else {
			parent_font_size = INHERITED_PROPERTIES["font-size"]
		}
		node_pct, err := strconv.ParseFloat(strings.TrimSuffix(node.Style["font-size"], "%"), 32)
		if err != nil {
			node_pct = 1.0 // Default to 100% if parsing fails
		} else {
			node_pct /= 100.0 // Convert percentage to a fraction
		}
		parent_px, err := strconv.ParseFloat(strings.TrimSuffix(parent_font_size, "px"), 32)
		if err != nil {
			parent_px = 16.0 // Default to 16px if parsing fails
		}
		node.Style["font-size"] = strconv.FormatFloat(node_pct*parent_px, 'f', -1, 32) + "px"
	}

	for _, child := range node.Children {
		Style(child, rules)
	}
}
