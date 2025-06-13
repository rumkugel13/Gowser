package browser

import (
	"fmt"
	"gowser/css"
	"gowser/html"
	"gowser/task"
	"gowser/trace"
	"gowser/url"
	"image"
	"image/color"
	"image/draw"
	"os"
	"slices"
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

func init() {
	data, err := os.ReadFile("browser.css")
	if err != nil {
		fmt.Println("Error loading default style sheet:", err)
		return
	}

	fmt.Println("Loading default style sheet from browser.css")
	parser := css.NewCSSParser(string(data))
	DEFAULT_STYLE_SHEET = parser.Parse()
}

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
	needs_animation_frame   bool
	measure                 *trace.MeasureTime
	lock                    *sync.Mutex
	active_tab_url          *url.URL
	active_tab_scroll       float64
	active_tab_height       float64
	active_tab_display_list []html.Command
	composited_layers       []*html.CompositedLayer
	draw_list               []html.Command
	needs_composite         bool
	needs_raster            bool
	needs_draw              bool
	composited_updates      map[*html.HtmlNode]html.VisualEffectCommand
	dark_mode               bool
}

func NewBrowser() *Browser {
	// browser thread
	browser := &Browser{
		tabs:                  make([]*Tab, 0),
		ActiveTab:             nil,
		needs_animation_frame: false,
		measure:               trace.NewMeasureTime(),
		lock:                  &sync.Mutex{},
		needs_composite:       false,
		needs_raster:          false,
		needs_draw:            false,
		composited_updates:    make(map[*html.HtmlNode]html.VisualEffectCommand),
		dark_mode:             false,
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
	return browser
}

func (b *Browser) Draw() {
	start := time.Now()
	canvas := b.root_surface
	background_color := color.White
	if b.dark_mode {
		background_color = color.Black
	}
	canvas.SetColor(background_color)
	canvas.Clear()

	// fast:
	{
		canvas.Push()
		canvas.Translate(0, b.chrome.bottom-b.active_tab_scroll)
		for _, item := range b.draw_list {
			item.Execute(canvas)
		}
		canvas.Pop()
		// srcRect := image.Rect(0, int(b.ActiveTab.scroll), WIDTH, b.tab_surface.Height())
		// dstRect := image.Rect(0, int(b.chrome.bottom), WIDTH, max(b.root_surface.Height()-int(b.chrome.bottom), b.tab_surface.Height()))
		// draw.Draw(canvas.Image().(*image.RGBA), dstRect, b.tab_surface.Image(), srcRect.Min, draw.Src)

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

func (b *Browser) HandleUp() {
	b.lock.Lock()
	if b.active_tab_height == 0 {
		b.lock.Unlock()
		return
	}
	b.active_tab_scroll = b.clamp_scroll(b.active_tab_scroll - SCROLL_STEP)
	b.SetNeedsDraw()
	b.needs_animation_frame = true
	b.lock.Unlock()
}

func (b *Browser) HandleDown() {
	b.lock.Lock()
	if b.active_tab_height == 0 {
		b.lock.Unlock()
		return
	}
	b.active_tab_scroll = b.clamp_scroll(b.active_tab_scroll + SCROLL_STEP)
	b.SetNeedsDraw()
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
		b.SetNeedsRaster()
	} else {
		if b.focus != "content" {
			b.focus = "content"
			b.chrome.focus = ""
			b.SetNeedsRaster()
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
		b.SetNeedsRaster()
	} else if b.focus == "content" {
		task := task.NewTask(func(i ...interface{}) {
			b.ActiveTab.keypress(rune(char))
		}, rune(char))
		b.ActiveTab.TaskRunner.ScheduleTask(task)
	}
	b.lock.Unlock()
}

func (b *Browser) HandleBackspace() {
	b.lock.Lock()
	if b.chrome.backspace() {
		b.SetNeedsRaster()
	} else if b.focus == "content" {
		task := task.NewTask(func(i ...interface{}) {
			b.ActiveTab.backspace()
		})
		b.ActiveTab.TaskRunner.ScheduleTask(task)
	}
	b.lock.Unlock()
}

func (b *Browser) HandleEnter() {
	b.lock.Lock()
	if b.chrome.enter() {
		b.SetNeedsRaster()
	} else if b.focus == "content" {
		task := task.NewTask(func(i ...interface{}) {
			b.ActiveTab.enter()
		})
		b.ActiveTab.TaskRunner.ScheduleTask(task)
	}
	b.lock.Unlock()
}

func (b *Browser) HandleTab() {
	b.focus = "content"
	task := task.NewTask(func(i ...interface{}) {
		b.ActiveTab.advance_tab()
	})
	b.ActiveTab.TaskRunner.ScheduleTask(task)
}

func (b *Browser) FocusContent() {
	b.lock.Lock()
	b.chrome.blur()
	b.focus = "content"
	b.lock.Unlock()
}

func (b *Browser) FocusAddressbar() {
	b.lock.Lock()
	b.chrome.focus_addressbar()
	b.SetNeedsRaster()
	b.lock.Unlock()
}

func (b *Browser) CycleTabs() {
	b.lock.Lock()
	active_idx := slices.Index(b.tabs, b.ActiveTab)
	new_active_idx := (active_idx + 1) % len(b.tabs)
	b.set_active_tab(b.tabs[new_active_idx])
	b.lock.Unlock()
}

func (b *Browser) GoBack() {
	task := task.NewTask(func(i ...interface{}) {
		b.ActiveTab.go_back()
	})
	b.ActiveTab.TaskRunner.ScheduleTask(task)
	b.clear_data()
}

func (b *Browser) IncrementZoom(increment bool) {
	task := task.NewTask(func(i ...interface{}) {
		b.ActiveTab.ZoomBy(increment)
	}, increment)
	b.ActiveTab.TaskRunner.ScheduleTask(task)
}

func (b *Browser) ResetZoom() {
	task := task.NewTask(func(i ...interface{}) {
		b.ActiveTab.ResetZoom()
	})
	b.ActiveTab.TaskRunner.ScheduleTask(task)
}

func (b *Browser) ToggleDarkMode() {
	b.dark_mode = !b.dark_mode
	task := task.NewTask(func(i ...interface{}) {
		b.ActiveTab.set_dark_mode(b.dark_mode)
	}, b.dark_mode)
	b.ActiveTab.TaskRunner.ScheduleTask(task)
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
		b.composited_updates = data.composited_updates
		if b.composited_updates == nil {
			b.composited_updates = make(map[*html.HtmlNode]html.VisualEffectCommand)
			b.SetNeedsComposite()
		} else {
			b.SetNeedsDraw()
		}
	}
	b.lock.Unlock()
}

func (b *Browser) CompositeRasterAndDraw() {
	b.lock.Lock()
	if !b.needs_composite && !b.needs_raster && !b.needs_draw {
		b.lock.Unlock()
		return
	}
	b.measure.Time("raster_and_draw")

	if b.needs_composite {
		b.composite()
	}
	if b.needs_raster {
		b.raster_chrome()
		b.raster_tab()
	}
	if b.needs_draw {
		b.paint_draw_list()
		b.Draw()
	}

	b.measure.Stop("raster_and_draw")
	b.needs_composite = false
	b.needs_raster = false
	b.needs_draw = false
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

func (b *Browser) SetNeedsComposite() {
	b.needs_composite = true
	b.needs_raster = true
	b.needs_draw = true
}

func (b *Browser) SetNeedsRaster() {
	b.needs_raster = true
	b.needs_draw = true
}

func (b *Browser) SetNeedsDraw() {
	b.needs_draw = true
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
	b.clear_data()
	b.needs_animation_frame = true
	b.animation_timer = nil
	task := task.NewTask(func(i ...interface{}) {
		b.ActiveTab.set_dark_mode(b.dark_mode)
	}, b.dark_mode)
	b.ActiveTab.TaskRunner.ScheduleTask(task)
}

func (b *Browser) clear_data() {
	b.active_tab_scroll = 0
	b.active_tab_url = nil
	b.active_tab_display_list = make([]html.Command, 0)
	b.composited_layers = make([]*html.CompositedLayer, 0)
	b.composited_updates = make(map[*html.HtmlNode]html.VisualEffectCommand)
}

func (b *Browser) raster_tab() {
	start := time.Now()
	for _, composited_layer := range b.composited_layers {
		composited_layer.Raster()
	}
	fmt.Println("Tab raster took:", time.Since(start))
}

func (b *Browser) raster_chrome() {
	start := time.Now()
	canvas := b.chrome_surface
	background_color := color.White
	if b.dark_mode {
		background_color = color.Black
	}
	canvas.SetColor(background_color)
	canvas.Clear()

	cmds := b.chrome.paint()
	// layout.PrintCommands(cmds)
	for _, cmd := range cmds {
		cmd.Execute(canvas)
	}
	fmt.Println("Chrome raster took:", time.Since(start))
}

func (b *Browser) composite() {
	add_parent_pointers(b.active_tab_display_list, nil)
	b.composited_layers = make([]*html.CompositedLayer, 0)

	var all_commands []html.Command
	for _, cmd := range b.active_tab_display_list {
		all_commands = append(all_commands, html.CommandTreeToList(cmd)...)
	}

	var non_composited_commands []html.Command
	for _, cmd := range all_commands {
		// note: need to check for other types of visualeffect as well
		if v, ok := cmd.(html.VisualEffectCommand); (ok && !v.NeedsCompositing()) || html.IsPaintCommand(cmd) {
			if cmd.GetParent() == nil {
				non_composited_commands = append(non_composited_commands, cmd)
			} else if v, ok := cmd.GetParent().(html.VisualEffectCommand); ok && v.NeedsCompositing() {
				non_composited_commands = append(non_composited_commands, cmd)
			}
		}
	}

	for _, cmd := range non_composited_commands {
		merged := false

		for i := len(b.composited_layers) - 1; i >= 0; i-- {
			layer := b.composited_layers[i]
			if layer.CanMerge(cmd) {
				layer.Add(cmd)
				merged = true
				break
			} else if layer.AbsoluteBounds().Intersects(html.LocalToAbsolute(cmd, cmd.Rect())) {
				layer := html.NewCompositedLayer(cmd)
				b.composited_layers = append(b.composited_layers, layer)
			}
		}

		if !merged {
			layer := html.NewCompositedLayer(cmd)
			b.composited_layers = append(b.composited_layers, layer)
		}
	}
}

func add_parent_pointers(nodes []html.Command, parent html.Command) {
	for _, node := range nodes {
		node.SetParent(parent)
		add_parent_pointers(*node.Children(), node)
	}
}

func (b *Browser) paint_draw_list() {
	b.draw_list = make([]html.Command, 0)
	new_effects := make(map[html.Command]html.Command)
	for _, composited_layer := range b.composited_layers {
		var current_effect html.Command = html.NewDrawCompositedLayer(composited_layer)
		if len(composited_layer.DisplayItems) == 0 {
			continue
		}
		parent := composited_layer.DisplayItems[0].GetParent()
		for parent != nil {
			new_parent := b.get_latest(parent.(html.VisualEffectCommand))
			if parent_effect, ok := new_effects[new_parent]; ok {
				children := parent_effect.Children()
				// note: do we need addchild method?
				*children = append(*children, current_effect)
				break
			} else {
				current_effect = new_parent.(html.VisualEffectCommand).Clone(current_effect)
				new_effects[new_parent] = current_effect
				parent = parent.GetParent()
			}
		}
		if parent == nil {
			b.draw_list = append(b.draw_list, current_effect)
		}
	}
}

func (b *Browser) get_latest(effect html.VisualEffectCommand) html.Command {
	node := effect.GetNode()
	if _, ok := b.composited_updates[node]; !ok {
		return effect
	}
	if _, ok := effect.(*html.DrawBlend); !ok {
		return effect
	}
	return b.composited_updates[node]
}
