package browser

import (
	// "gowser/layout"
	"fmt"
	"gowser/css"
	"gowser/url"
	"image"
	"image/color"
	"os"
	"unsafe"

	"github.com/fogleman/gg"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	DEFAULT_STYLE_SHEET []css.Rule
)

type Browser struct {
	tabs         []*Tab
	active_tab   *Tab
	sdl_window   *sdl.Window
	root_surface *gg.Context
	chrome       *Chrome
	focus        string
	RED_MASK     uint32
	GREEN_MASK   uint32
	BLUE_MASK    uint32
	ALPHA_MASK   uint32
}

func NewBrowser() *Browser {
	browser := &Browser{
		tabs:       make([]*Tab, 0),
		active_tab: nil,
	}

	window, err := sdl.CreateWindow("Gowser", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		DefaultWidth, DefaultHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		panic("Could not create sdl window")
	}
	browser.sdl_window = window

	browser.root_surface = gg.NewContext(DefaultWidth, DefaultHeight)

	if sdl.BYTEORDER == sdl.BIG_ENDIAN {
		browser.RED_MASK = 0xff000000
		browser.GREEN_MASK = 0x00ff0000
		browser.BLUE_MASK = 0x0000ff00
		browser.ALPHA_MASK = 0x000000ff
	} else {
		browser.RED_MASK = 0x000000ff
		browser.GREEN_MASK = 0x0000ff00
		browser.BLUE_MASK = 0x00ff0000
		browser.ALPHA_MASK = 0xff000000
	}

	browser.chrome = NewChrome(browser)
	load_default_style_sheet()
	return browser
}

func (b *Browser) Draw() {
	canvas := b.root_surface
	canvas.SetColor(color.White)
	canvas.Clear()
	b.active_tab.Draw(canvas, b.chrome.bottom)

	cmds := b.chrome.paint()
	// layout.PrintCommands(cmds)
	for _, cmd := range cmds {
		cmd.Execute(0, canvas)
	}

	gg_img := b.root_surface.Image()
	gg_bytes, ok := gg_img.(*image.RGBA)
	if !ok {
		panic("Image is not RGBA")
	}

	depth := 32
	pitch := int(4 * DefaultWidth)
	sdl_surface, err := sdl.CreateRGBSurfaceFrom(
		unsafe.Pointer(&gg_bytes.Pix[0]),
		DefaultWidth, DefaultHeight, depth, pitch,
		b.RED_MASK, b.GREEN_MASK, b.BLUE_MASK, b.ALPHA_MASK,
	)
	if err != nil {
		panic("Cannot create rgb surface")
	}
	defer sdl_surface.Free()

	rect := &sdl.Rect{X: 0, Y: 0, W: DefaultWidth, H: DefaultHeight}
	window_surface, err := b.sdl_window.GetSurface()
	if err != nil {
		panic("Cannot get window surface")
	}
	sdl_surface.Blit(rect, window_surface, rect)
	b.sdl_window.UpdateSurface()
}

func (b *Browser) NewTab(url *url.URL) {
	new_tab := NewTab(DefaultHeight - b.chrome.bottom)
	new_tab.Load(url, "")
	b.active_tab = new_tab
	b.tabs = append(b.tabs, new_tab)
	b.Draw()
}

func (b *Browser) HandleQuit() {
	b.sdl_window.Destroy()
}

func (b *Browser) HandleDown() {
	b.active_tab.scrollDown()
	b.Draw()
}

func (b *Browser) HandleClick(e *sdl.MouseButtonEvent) {
	if float64(e.Y) < b.chrome.bottom {
		b.focus = ""
		b.chrome.click(float64(e.X), float64(e.Y))
	} else {
		b.focus = "content"
		b.chrome.blur()
		tab_y := float64(e.Y) - b.chrome.bottom
		b.active_tab.click(float64(e.X), tab_y)
	}

	b.Draw()
}

func (b *Browser) HandleKey(e *sdl.TextInputEvent) {
	char := e.GetText()[0]
	if !(0x20 <= char && char < 0x7f) {
		return
	}
	if b.chrome.keypress(rune(char)) {
		b.Draw()
	} else if b.focus == "content" {
		b.active_tab.keypress(rune(char))
		b.Draw()
	}
}

func (b *Browser) HandleEnter() {
	b.chrome.enter()
	b.Draw()
}

func load_default_style_sheet() {
	data, err := os.ReadFile("browser.css")
	if err != nil {
		fmt.Println("Error loading default style sheet:", err)
		return
	}

	fmt.Println("Loading default style sheet from browser.css")
	parser := css.NewCSSParser(string(data))
	DEFAULT_STYLE_SHEET = parser.Parse()
}
