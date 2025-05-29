package browser

import (
	"gowser/url"
	"strings"

	"modernc.org/tk9.0"
)

const (
	DefaultWidth  = 800
	DefaultHeight = 600
	HSTEP         = 13
	VSTEP         = 18
	SCROLL_STEP = 100
)

type Browser struct {
	window       *tk9_0.Window
	canvas       *tk9_0.CanvasWidget
	display_list []LayoutItem
	scroll       int
}

type LayoutItem struct {
	Char string
	X    int
	Y    int
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
	body := url.Request()
	text := lex(body)
	b.display_list = layout(text)
	b.Draw()
}

func (b *Browser) Draw() {
	b.canvas.Delete("all")
	for _, item := range b.display_list {
		if item.Y > b.scroll + DefaultHeight {
			continue // Skip items that are outside the visible area
		}
		if item.Y + VSTEP < b.scroll {
			continue // Skip items that are above the visible area
		}
		b.canvas.CreateText(item.X, item.Y - b.scroll, tk9_0.Txt(item.Char))
	}
}

func lex(body string) string {
	text := strings.Builder{}
	inTag := false
	for _, char := range body {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			text.WriteRune(char)
		}
	}
	return text.String()
}

func layout(text string) []LayoutItem {
	layout := []LayoutItem{}
	cursor_x, cursor_y := HSTEP, VSTEP
	for _, char := range text {
		layout = append(layout, LayoutItem{string(char), cursor_x, cursor_y})
		cursor_x += HSTEP
		if cursor_x >= DefaultWidth-HSTEP {
			cursor_x = HSTEP
			cursor_y += VSTEP
		}
	}
	return layout
}
