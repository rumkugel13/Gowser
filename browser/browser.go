package browser

import (
	"fmt"
	"gowser/css"
	"gowser/layout"
	"gowser/url"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"time"
	"unsafe"

	"github.com/fogleman/gg"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	DEFAULT_STYLE_SHEET []css.Rule
)

type Browser struct {
	tabs           []*Tab
	active_tab     *Tab
	sdl_window     *sdl.Window
	root_surface   *gg.Context
	chrome         *Chrome
	focus          string
	RED_MASK       uint32
	GREEN_MASK     uint32
	BLUE_MASK      uint32
	ALPHA_MASK     uint32
	chrome_surface *gg.Context
	tab_surface    *gg.Context
}

func NewBrowser() *Browser {
	browser := &Browser{
		tabs:       make([]*Tab, 0),
		active_tab: nil,
	}

	window, err := sdl.CreateWindow("Gowser", sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		WIDTH, HEIGHT, sdl.WINDOW_SHOWN)
	if err != nil {
		panic("Could not create sdl window")
	}
	browser.sdl_window = window

	browser.root_surface = gg.NewContext(WIDTH, HEIGHT)

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
	browser.chrome_surface = gg.NewContext(WIDTH, int(browser.chrome.bottom))
	browser.tab_surface = nil
	load_default_style_sheet()
	return browser
}

func (b *Browser) Draw() {
	start := time.Now()
	canvas := b.root_surface
	canvas.SetColor(color.White)
	canvas.Clear()

	// fast:
	{
		srcRect := image.Rect(0, int(b.active_tab.scroll), WIDTH, b.tab_surface.Height())
		dstRect := image.Rect(0, int(b.chrome.bottom), WIDTH, b.tab_surface.Height())
		draw.Draw(canvas.Image().(*image.RGBA), dstRect, b.tab_surface.Image(), srcRect.Min, draw.Src)

		chromeRect := image.Rect(0, 0, WIDTH, int(b.chrome.bottom))
		draw.Draw(canvas.Image().(*image.RGBA), chromeRect, b.chrome_surface.Image(), chromeRect.Min, draw.Src)
	}

	// slow:
	// {
	// 	canvas.ResetClip()
	// 	tab_rect := layout.NewRect(0, b.chrome.bottom, WIDTH, HEIGHT)
	// 	tab_offset := b.chrome.bottom - b.active_tab.scroll
	// 	canvas.Push()
	// 	canvas.DrawRectangle(tab_rect.Left, tab_rect.Top, tab_rect.Right-tab_rect.Left, tab_rect.Bottom-tab_rect.Top)
	// 	canvas.Clip()
	// 	canvas.Translate(0, tab_offset)
	// 	canvas.DrawImage(b.tab_surface.Image(), 0, 0)
	// 	canvas.Pop()

	// 	canvas.ResetClip()
	// 	chrome_rect := layout.NewRect(0, 0, WIDTH, b.chrome.bottom)
	// 	canvas.Push()
	// 	canvas.DrawRectangle(chrome_rect.Left, chrome_rect.Top, chrome_rect.Right-chrome_rect.Left, chrome_rect.Bottom-chrome_rect.Top)
	// 	canvas.Clip()
	// 	canvas.DrawImage(b.chrome_surface.Image(), 0, 0)
	// 	canvas.Pop()
	// }

	gg_img := b.root_surface.Image()
	gg_bytes, ok := gg_img.(*image.RGBA)
	if !ok {
		panic("Image is not RGBA")
	}

	depth := 32
	pitch := int(4 * WIDTH)
	sdl_surface, err := sdl.CreateRGBSurfaceFrom(
		unsafe.Pointer(&gg_bytes.Pix[0]),
		WIDTH, HEIGHT, depth, pitch,
		b.RED_MASK, b.GREEN_MASK, b.BLUE_MASK, b.ALPHA_MASK,
	)
	if err != nil {
		panic("Cannot create rgb surface")
	}
	defer sdl_surface.Free()

	rect := &sdl.Rect{X: 0, Y: 0, W: WIDTH, H: HEIGHT}
	window_surface, err := b.sdl_window.GetSurface()
	if err != nil {
		panic("Cannot get window surface")
	}
	sdl_surface.Blit(rect, window_surface, rect)
	b.sdl_window.UpdateSurface()
	fmt.Println("Draw took:", time.Since(start))
}

func (b *Browser) NewTab(url *url.URL) {
	new_tab := NewTab(HEIGHT - b.chrome.bottom)
	new_tab.Load(url, "")
	b.active_tab = new_tab
	b.tabs = append(b.tabs, new_tab)
	b.raster_chrome()
	b.raster_tab()
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
		b.raster_chrome()
	} else {
		b.focus = "content"
		b.chrome.blur()
		url := b.active_tab.url
		tab_y := float64(e.Y) - b.chrome.bottom
		b.active_tab.click(float64(e.X), tab_y)
		if b.active_tab.url != url {
			b.raster_chrome()
		}
		b.raster_tab()
	}

	b.Draw()
}

func (b *Browser) HandleKey(e *sdl.TextInputEvent) {
	char := e.GetText()[0]
	if !(0x20 <= char && char < 0x7f) {
		return
	}
	if b.chrome.keypress(rune(char)) {
		b.raster_chrome()
		b.Draw()
	} else if b.focus == "content" {
		b.active_tab.keypress(rune(char))
		b.raster_tab()
		b.Draw()
	}
}

func (b *Browser) HandleEnter() {
	b.chrome.enter()
	b.raster_chrome()
	b.raster_tab()
	b.Draw()
}

func (b *Browser) raster_tab() {
	start := time.Now()
	tab_height := math.Ceil(b.active_tab.document.Height + 2*layout.VSTEP)

	if b.tab_surface == nil || tab_height != float64(b.tab_surface.Height()) {
		b.tab_surface = gg.NewContext(WIDTH, int(tab_height))
	}

	canvas := b.tab_surface
	canvas.SetColor(color.White)
	canvas.Clear()
	b.active_tab.Raster(canvas)
	fmt.Println("Tab raster took:", time.Since(start))
}

func (b *Browser) raster_chrome() {
	start := time.Now()
	canvas := b.chrome_surface
	canvas.SetColor(color.White)
	canvas.Clear()

	cmds := b.chrome.paint()
	// layout.PrintCommands(cmds)
	for _, cmd := range cmds {
		cmd.Execute(canvas)
	}
	fmt.Println("Chrome raster took:", time.Since(start))
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
