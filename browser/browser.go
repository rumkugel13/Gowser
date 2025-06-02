package browser

import (
	"gowser/url"

	tk9_0 "modernc.org/tk9.0"
)

type Browser struct {
	tabs       []*Tab
	active_tab *Tab
	window     *tk9_0.Window
	canvas     *tk9_0.CanvasWidget
}

func NewBrowser() *Browser {
	browser := &Browser{
		tabs:       make([]*Tab, 0),
		active_tab: nil,
		window:     tk9_0.App.Center(),
		canvas:     tk9_0.Canvas(tk9_0.Width(DefaultWidth), tk9_0.Height(DefaultHeight), tk9_0.Background("white")),
	}
	tk9_0.Pack(browser.canvas)
	tk9_0.Bind(tk9_0.App, "<Down>", tk9_0.Command(browser.handle_down))
	tk9_0.Bind(tk9_0.App, "<Button-1>", tk9_0.Command(browser.handle_click))
	return browser
}

func (b *Browser) Draw() {
	b.canvas.Delete("all")
	b.active_tab.Draw(b.canvas)
}

func (b *Browser) NewTab(url *url.URL) {
	new_tab := NewTab()
	new_tab.Load(url)
	b.active_tab = new_tab
	b.tabs = append(b.tabs, new_tab)
	b.Draw()
}

func (b *Browser) handle_down(e *tk9_0.Event) {
	b.active_tab.scrollDown()
	b.Draw()
}

func (b *Browser) handle_click(e *tk9_0.Event) {
	b.active_tab.click(float32(e.X), float32(e.Y))
	b.Draw()
}
