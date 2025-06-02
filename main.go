package main

import (
	"gowser/browser"
	u "gowser/url"
	"os"

	"modernc.org/tk9.0"
)

func main() {
	var url *u.URL
	if len(os.Args) > 1 {
		url = u.NewURL(os.Args[1])
	} else {
		url = u.NewURL("https://browser.engineering/")
	}
	browser := browser.NewBrowser()
	browser.NewTab(url)
	tk9_0.App.Center().Wait()
}
