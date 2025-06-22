package browser

import (
	"gowser/url"
)

type CommitData struct {
	url                *url.URL
	scroll             *float64
	height             float64
	display_list       []Command
	composited_updates map[*HtmlNode]VisualEffectCommand
	accessibility_tree *AccessibilityNode
	focus              *HtmlNode
	root_frame_focused bool
}

func NewCommitData(url *url.URL, scroll *float64, height float64, display_list []Command,
	composited_updates map[*HtmlNode]VisualEffectCommand, accessibility_tree *AccessibilityNode,
	focus *HtmlNode, root_frame_focused bool) *CommitData {
	return &CommitData{
		url:                url,
		scroll:             scroll,
		height:             height,
		display_list:       display_list,
		composited_updates: composited_updates,
		accessibility_tree: accessibility_tree,
		focus:              focus,
		root_frame_focused: root_frame_focused,
	}
}
