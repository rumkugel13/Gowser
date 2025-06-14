package browser

import (
	"gowser/accessibility"
	"gowser/html"
	"gowser/url"
)

type CommitData struct {
	url                *url.URL
	scroll             *float64
	height             float64
	display_list       []html.Command
	composited_updates map[*html.HtmlNode]html.VisualEffectCommand
	accessibility_tree *accessibility.AccessibilityNode
	focus              *html.HtmlNode
}

func NewCommitData(url *url.URL, scroll *float64, height float64, display_list []html.Command,
	composited_updates map[*html.HtmlNode]html.VisualEffectCommand, accessibility_tree *accessibility.AccessibilityNode,
	focus *html.HtmlNode) *CommitData {
	return &CommitData{
		url:                url,
		scroll:             scroll,
		height:             height,
		display_list:       display_list,
		composited_updates: composited_updates,
		accessibility_tree: accessibility_tree,
		focus:              focus,
	}
}
