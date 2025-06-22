package browser

import (
	"gowser/animate"
	"maps"
	"strconv"
	"strings"
)

var (
	INHERITED_PROPERTIES = map[string]string{
		"font-family": "Arial",
		"font-size":   "16px",
		"font-style":  "normal",
		"font-weight": "normal",
		"color":       "black",
	}
)

func Style(node *HtmlNode, rules []Rule, tab *Tab) {
	if node.Style == nil {
		init_style(node)
	}

	var needs_style bool
	for _, field := range node.Style {
		if field.Dirty {
			needs_style = true
			break
		}
	}

	if needs_style {
		old_style := map[string]string{}
		for prop, val := range node.Style {
			old_style[prop] = val.Value
		}
		new_style := maps.Clone(CSS_PROPERTIES)

		for property, default_value := range INHERITED_PROPERTIES {
			if node.Parent != nil {
				parent_field := node.Parent.Style[property]
				parent_value := parent_field.Read(node.Style[property])
				new_style[property] = parent_value
			} else {
				new_style[property] = default_value
			}
		}

		for _, rule := range rules {
			if rule.Media != "" {
				if (rule.Media == "dark") != tab.dark_mode {
					continue
				}
			}
			selector := rule.Selector
			body := rule.Body
			if !selector.Matches(node) {
				continue
			}
			maps.Copy(new_style, body)
		}

		if element, ok := node.Token.(ElementToken); ok {
			if style, exists := element.Attributes["style"]; exists {
				parser := NewCSSParser(style)
				pairs := parser.Body()
				maps.Copy(new_style, pairs)
			}
		}

		if strings.HasSuffix(new_style["font-size"], "%") {
			var parent_font_size string
			if node.Parent != nil {
				parent_field := node.Parent.Style["font-size"]
				parent_font_size = parent_field.Read(node.Style["font-size"])
			} else {
				parent_font_size = INHERITED_PROPERTIES["font-size"]
			}
			node_pct, err := strconv.ParseFloat(strings.TrimSuffix(new_style["font-size"], "%"), 32)
			if err != nil {
				node_pct = 1.0 // Default to 100% if parsing fails
			} else {
				node_pct /= 100.0 // Convert percentage to a fraction
			}
			parent_px, err := strconv.ParseFloat(strings.TrimSuffix(parent_font_size, "px"), 32)
			if err != nil {
				parent_px = 16.0 // Default to 16px if parsing fails
			}
			new_style["font-size"] = strconv.FormatFloat(node_pct*parent_px, 'f', -1, 32) + "px"
		}

		if len(old_style) != 0 {
			transitions := diff_styles(old_style, new_style)
			for property, transition := range transitions {
				if property == "opacity" {
					tab.SetNeedsRenderAllFrames()
					oldfVal, _ := strconv.ParseFloat(transition.old_value, 32)
					newfVal, _ := strconv.ParseFloat(transition.new_value, 32)

					animation := animate.NewNumericAnimation(oldfVal, newfVal, transition.num_frames)
					node.Animations[property] = animation
					new_style[property] = animation.Animate()
				}
			}
		}

		for prop, field := range node.Style {
			field.Set(new_style[prop])
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
	for property, num_frames := range ParseTransition(new_style["transition"]) {
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
