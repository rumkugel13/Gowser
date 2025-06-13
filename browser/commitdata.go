package browser

import (
	"gowser/html"
	"gowser/url"
)

type CommitData struct {
	url                *url.URL
	scroll             *float64
	height             float64
	display_list       []html.Command
	composited_updates map[*html.HtmlNode]html.VisualEffectCommand
}

func NewCommitData(url *url.URL, scroll *float64, height float64, display_list []html.Command, composited_updates map[*html.HtmlNode]html.VisualEffectCommand) *CommitData {
	return &CommitData{
		url:                url,
		scroll:             scroll,
		height:             height,
		display_list:       display_list,
		composited_updates: composited_updates,
	}
}
