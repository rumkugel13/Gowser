package browser

import (
	"fmt"
	"gowser/css"
	"gowser/layout"
	"gowser/task"
	"gowser/url"
	"image"
	"image/color"
	"image/draw"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/fogleman/gg"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	REFRESH_RATE_SEC = .033
)

var (
	DEFAULT_STYLE_SHEET []css.Rule
)

type Browser struct {
	tabs                    []*Tab
	ActiveTab               *Tab
	sdl_window              *sdl.Window
	root_surface            *gg.Context
	chrome                  *Chrome
	focus                   string
	RED_MASK                uint32
	GREEN_MASK              uint32
	BLUE_MASK               uint32
	ALPHA_MASK              uint32
	chrome_surface          *gg.Context
	tab_surface             *gg.Context
	animation_timer         *time.Timer
	needs_raster_and_draw   bool
	needs_animation_frame   bool
	measure                 *MeasureTime
	lock                    *sync.Mutex
	active_tab_url          *url.URL
	active_tab_scroll       float64
	active_tab_height       float64
	active_tab_display_list []layout.Command
}

func NewBrowser() *Browser {
	// browser thread
	browser := &Browser{
		tabs:                  make([]*Tab, 0),
		ActiveTab:             nil,
		needs_raster_and_draw: false,
		needs_animation_frame: false,
		measure:               NewMeasureTime(),
		lock:                  &sync.Mutex{},
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
		srcRect := image.Rect(0, int(b.ActiveTab.scroll), WIDTH, b.tab_surface.Height())
		dstRect := image.Rect(0, int(b.chrome.bottom), WIDTH, max(b.root_surface.Height()-int(b.chrome.bottom), b.tab_surface.Height()))
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
	b.lock.Lock()
	b.new_tab_internal(url)
	b.lock.Unlock()
}

func (b *Browser) new_tab_internal(url *url.URL) {
	new_tab := NewTab(b, HEIGHT-b.chrome.bottom)
	b.tabs = append(b.tabs, new_tab)
	b.set_active_tab(new_tab)
	b.ScheduleLoad(url, "")
}

func (b *Browser) HandleQuit() {
	b.measure.Finish()
	for _, tab := range b.tabs {
		tab.TaskRunner.SetNeedsQuit()
	}
	b.sdl_window.Destroy()
}

func (b *Browser) HandleDown() {
	b.lock.Lock()
	if b.active_tab_height == 0 {
		b.lock.Unlock()
		return
	}
	b.active_tab_scroll = b.clamp_scroll(b.active_tab_scroll + SCROLL_STEP)
	b.SetNeedsRasterAndDraw()
	b.needs_animation_frame = true
	b.lock.Unlock()
}

func (b *Browser) clamp_scroll(scroll float64) float64 {
	height := b.active_tab_height
	max_scroll := height - (HEIGHT - b.chrome.bottom)
	return max(0, min(scroll, max_scroll))
}

func (b *Browser) HandleClick(e *sdl.MouseButtonEvent) {
	b.lock.Lock()
	if float64(e.Y) < b.chrome.bottom {
		b.focus = ""
		b.chrome.click(float64(e.X), float64(e.Y))
		b.SetNeedsRasterAndDraw()
	} else {
		if b.focus != "content" {
			b.focus = "content"
			b.chrome.focus = ""
			b.SetNeedsRasterAndDraw()
		}
		b.chrome.blur()
		tab_y := float64(e.Y) - b.chrome.bottom
		tab_x := float64(e.X)
		task := task.NewTask(func(i ...interface{}) {
			b.ActiveTab.click(tab_x, tab_y)
		}, tab_x, tab_y)
		b.ActiveTab.TaskRunner.ScheduleTask(task)
	}
	b.lock.Unlock()
}

func (b *Browser) HandleKey(e *sdl.TextInputEvent) {
	b.lock.Lock()
	char := e.GetText()[0]
	if !(0x20 <= char && char < 0x7f) {
		return
	}
	if b.chrome.keypress(rune(char)) {
		b.SetNeedsRasterAndDraw()
	} else if b.focus == "content" {
		task := task.NewTask(func(i ...interface{}) {
			b.ActiveTab.keypress(rune(char))
		}, rune(char))
		b.ActiveTab.TaskRunner.ScheduleTask(task)
	}
	b.lock.Unlock()
}

func (b *Browser) HandleEnter() {
	b.lock.Lock()
	if b.chrome.enter() {
		b.SetNeedsRasterAndDraw()
	}
	b.lock.Unlock()
}

func (b *Browser) Commit(tab *Tab, data *CommitData) {
	b.lock.Lock()
	if tab == b.ActiveTab {
		b.active_tab_url = data.url
		if data.scroll != nil {
			b.active_tab_scroll = *data.scroll
		}
		b.active_tab_height = data.height
		if len(data.display_list) > 0 {
			b.active_tab_display_list = data.display_list
		}
		b.animation_timer = nil
		b.SetNeedsRasterAndDraw()
	}
	b.lock.Unlock()
}

func (b *Browser) RasterAndDraw() {
	b.lock.Lock()
	if !b.needs_raster_and_draw {
		b.lock.Unlock()
		return
	}
	b.measure.Time("raster_and_draw")

	b.raster_chrome()
	b.raster_tab()
	b.Draw()

	b.measure.Stop("raster_and_draw")
	b.needs_raster_and_draw = false
	b.lock.Unlock()
}

func (b *Browser) ScheduleAnimationFrame() {
	callback := func() {
		b.lock.Lock()
		scroll := b.active_tab_scroll
		b.needs_animation_frame = false
		task := task.NewTask(func(i ...interface{}) {
			b.ActiveTab.run_animation_frame(&scroll)
		}, scroll)
		b.ActiveTab.TaskRunner.ScheduleTask(task)
		b.lock.Unlock()
	}
	b.lock.Lock()
	if b.needs_animation_frame && b.animation_timer == nil {
		ms := int(REFRESH_RATE_SEC * 1000)
		duration := time.Duration(ms)
		b.animation_timer = time.AfterFunc(duration, callback)
	}
	b.lock.Unlock()
}

func (b *Browser) SetNeedsRasterAndDraw() {
	b.needs_raster_and_draw = true
}

func (b *Browser) SetNeedsAnimationFrame(tab *Tab) {
	b.lock.Lock()
	if tab == b.ActiveTab {
		b.needs_animation_frame = true
	}
	b.lock.Unlock()
}

func (b *Browser) ScheduleLoad(url *url.URL, body string) {
	b.ActiveTab.TaskRunner.ClearPendingTasks()
	task := task.NewTask(func(i ...interface{}) {
		b.ActiveTab.Load(url, body)
	}, url, body)
	b.ActiveTab.TaskRunner.ScheduleTask(task)
}

func (b *Browser) set_active_tab(new_tab *Tab) {
	b.ActiveTab = new_tab
	b.active_tab_scroll = 0
	b.active_tab_url = nil
	b.needs_animation_frame = true
	b.animation_timer = nil
}

func (b *Browser) raster_tab() {
	if b.active_tab_height == 0 {
		return
	}
	start := time.Now()
	tab_height := b.active_tab_height

	if b.tab_surface == nil || tab_height != float64(b.tab_surface.Height()) {
		b.tab_surface = gg.NewContext(WIDTH, int(tab_height))
	}

	canvas := b.tab_surface
	canvas.SetColor(color.White)
	canvas.Clear()
	// layout.PrintCommands(b.active_tab_display_list, 0)
	for _, cmd := range b.active_tab_display_list {
		cmd.Execute(canvas)
	}
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
