package main

import (
	"gowser/browser"
	"gowser/url"
	"os"

	"modernc.org/tk9.0"
)

func main() {
	url := url.NewURL(os.Args[1])
	browser := browser.NewBrowser()
	browser.NewTab(url)
	tk9_0.App.Center().Wait()
}
