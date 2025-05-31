package browser

import (
	"fmt"
	"gowser/html"
	"gowser/layout"
	"gowser/url"
	"time"

	tk9_0 "modernc.org/tk9.0"
)

const (
	DefaultWidth  = 800.
	DefaultHeight = 600.
	SCROLL_STEP   = 100.
)

type Browser struct {
	window       *tk9_0.Window
	canvas       *tk9_0.CanvasWidget
	display_list []layout.LayoutItem
	scroll       float32
}

func NewBrowser() *Browser {
	browser := &Browser{}
	browser.canvas = tk9_0.Canvas(tk9_0.Width(DefaultWidth), tk9_0.Height(DefaultHeight))
	browser.window = tk9_0.App.Center()
	tk9_0.Pack(browser.canvas)
	browser.scroll = 0
	tk9_0.Bind(tk9_0.App, "<Down>", tk9_0.Command(func() {
		browser.scroll += SCROLL_STEP
		browser.Draw()
	}))
	return browser
}

func (b *Browser) Load(url *url.URL) {
	fmt.Println("Requesting URL:", url)
	start := time.Now()
	body := url.Request()
	fmt.Println("Request took:", time.Since(start))

	start = time.Now()
	nodes := html.NewHTMLParser(body).Parse()
	fmt.Println("Parsing took:", time.Since(start))

	// nodes.PrintTree(0)
	start = time.Now()
	b.display_list = layout.NewLayout(&nodes).Display_list
	fmt.Println("Layout took:", time.Since(start))

	start = time.Now()
	b.Draw()
	fmt.Println("Drawing took:", time.Since(start))
}

func (b *Browser) Draw() {
	b.canvas.Delete("all")
	for _, item := range b.display_list {
		if item.Y > b.scroll+DefaultHeight {
			continue // Skip items that are outside the visible area
		}
		if item.Y+layout.VSTEP < b.scroll {
			continue // Skip items that are above the visible area
		}
		b.canvas.CreateText(item.X, item.Y-b.scroll, tk9_0.Txt(item.Word), tk9_0.Anchor("nw"), tk9_0.Font(item.Font))
	}
}
