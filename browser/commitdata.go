package browser

import (
	"gowser/layout"
	"gowser/url"
)

type CommitData struct {
	url          *url.URL
	scroll       *float64
	height       float64
	display_list []layout.Command
}

func NewCommitData(url *url.URL, scroll *float64, height float64, display_list []layout.Command) *CommitData {
	return &CommitData{
		url: url,
		scroll: scroll,
		height: height,
		display_list: display_list,
	}
}
