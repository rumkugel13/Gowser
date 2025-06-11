package browser

import (
	"gowser/animate"
	"gowser/css"
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

func Style(node *html.HtmlNode, rules []css.Rule, tab *Tab) {
	old_style := node.Style
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
			parser := css.NewCSSParser(style)
			pairs := parser.Body()
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

	if len(old_style) != 0 {
		transitions := diff_styles(old_style, node.Style)
		for property, transition := range transitions {
			if property == "opacity" {
				tab.SetNeedsRender()
				oldfVal, _ := strconv.ParseFloat(transition.old_value, 32)
				newfVal, _ := strconv.ParseFloat(transition.new_value, 32)

				animation := animate.NewNumericAnimation(oldfVal, newfVal, transition.num_frames)
				node.Animations[property] = animation
				node.Style[property] = animation.Animate()
			}
		}
	}

	for _, child := range node.Children {
		Style(child, rules, tab)
	}
}

type Transition struct {
	old_value, new_value string
	num_frames           int
}

func diff_styles(old_style, new_style map[string]string) map[string]Transition {
	transitions := make(map[string]Transition)
	for property, num_frames := range css.ParseTransition(new_style["transition"]) {
		if _, ok := old_style[property]; !ok {
			continue
		}
		if _, ok := new_style[property]; !ok {
			continue
		}
		old_value, new_value := old_style[property], new_style[property]
		if old_value == new_value {
			continue
		}
		transitions[property] = Transition{old_value, new_value, num_frames}
	}
	return transitions
}
