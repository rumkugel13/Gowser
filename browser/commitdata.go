package browser

import (
	"gowser/display"
	"gowser/url"
)

type CommitData struct {
	url          *url.URL
	scroll       *float64
	height       float64
	display_list []display.Command
}

func NewCommitData(url *url.URL, scroll *float64, height float64, display_list []display.Command) *CommitData {
	return &CommitData{
		url: url,
		scroll: scroll,
		height: height,
		display_list: display_list,
	}
}
