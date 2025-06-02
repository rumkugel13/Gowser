package browser

import (
	// "gowser/layout"
	"gowser/url"

	tk9_0 "modernc.org/tk9.0"
)

type Browser struct {
	tabs       []*Tab
	active_tab *Tab
	window     *tk9_0.Window
	canvas     *tk9_0.CanvasWidget
	chrome     *Chrome
}

func NewBrowser() *Browser {
	browser := &Browser{
		tabs:       make([]*Tab, 0),
		active_tab: nil,
		window:     tk9_0.App.Center(),
		canvas:     tk9_0.Canvas(tk9_0.Width(DefaultWidth), tk9_0.Height(DefaultHeight), tk9_0.Background("white")),
	}
	browser.chrome = NewChrome(browser)
	tk9_0.Pack(browser.canvas)
	tk9_0.Bind(tk9_0.App, "<Down>", tk9_0.Command(browser.handle_down))
	tk9_0.Bind(tk9_0.App, "<Button-1>", tk9_0.Command(browser.handle_click))
	tk9_0.Bind(tk9_0.App, "<Key>", tk9_0.Command(browser.handle_key))
	tk9_0.Bind(tk9_0.App, "<Return>", tk9_0.Command(browser.handle_enter))
	return browser
}

func (b *Browser) Draw() {
	b.canvas.Delete("all")
	b.active_tab.Draw(b.canvas, b.chrome.bottom)

	cmds := b.chrome.paint()
	// layout.PrintCommands(cmds)
	for _, cmd := range cmds {
		cmd.Execute(0, *b.canvas)
	}
}

func (b *Browser) NewTab(url *url.URL) {
	new_tab := NewTab(DefaultHeight - b.chrome.bottom)
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
	if float32(e.Y) < b.chrome.bottom {
		b.chrome.click(float32(e.X), float32(e.Y))
	} else {
		tab_y := float32(e.Y) - b.chrome.bottom
		b.active_tab.click(float32(e.X), tab_y)
	}

	b.Draw()
}

func (b *Browser) handle_key(e *tk9_0.Event) {
	// note: second check gets rid of "space" or "ShiftL"
	if len(e.Keysym) == 0 || len(e.Keysym) > 1 {
		return
	}
	if !(0x20 <= int(e.Keysym[0]) && int(e.Keysym[0]) < 0x7f) {
		return
	}
	b.chrome.keypress(rune(e.Keysym[0]))
	b.Draw()
}

func (b *Browser) handle_enter(e *tk9_0.Event) {
	b.chrome.enter()
	b.Draw()
}
