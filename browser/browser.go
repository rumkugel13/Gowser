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
	display_list []layout.Command
	scroll       float32
	document     *layout.DocumentLayout
}

func NewBrowser() *Browser {
	browser := &Browser{}
	browser.canvas = tk9_0.Canvas(tk9_0.Width(DefaultWidth), tk9_0.Height(DefaultHeight))
	browser.window = tk9_0.App.Center()
	tk9_0.Pack(browser.canvas)
	browser.scroll = 0
	tk9_0.Bind(tk9_0.App, "<Down>", tk9_0.Command(func() {
		max_y := max(browser.document.Height()+2*layout.VSTEP-DefaultHeight, 0)
		browser.scroll = min(browser.scroll+SCROLL_STEP, max_y)
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
	// nodes.PrintTree(0)
	fmt.Println("Parsing took:", time.Since(start))

	start = time.Now()
	b.document = layout.NewDocumentLayout(&nodes)
	b.document.Layout()
	b.display_list = make([]layout.Command, 0)
	layout.PaintTree(b.document, &b.display_list)
	// layout.PrintTree(b.document, 0)
	// for _, cmd := range b.display_list {
	// 	fmt.Println("Command:", cmd)
	// }
	fmt.Println("Layout took:", time.Since(start))

	start = time.Now()
	b.Draw()
	fmt.Println("Drawing took:", time.Since(start))
}

func (b *Browser) Draw() {
	b.canvas.Delete("all")
	for _, cmd := range b.display_list {
		if cmd.Top() > b.scroll+DefaultHeight {
			continue // Skip items that are outside the visible area
		}
		if cmd.Bottom() < b.scroll {
			continue // Skip items that are above the visible area
		}
		cmd.Execute(b.scroll, *b.canvas)
	}
}
