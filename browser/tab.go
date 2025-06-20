package browser

import (
	"bytes"
	"cmp"
	"fmt"
	"gowser/html"
	u "gowser/url"
	"image"
	"math"
	"os"
	"time"
)

const (
	WIDTH                    = 800.
	HEIGHT                   = 600.
	SCROLL_STEP              = 100.
	PRINT_DISPLAY_LIST       = false
	PRINT_DOCUMENT_LAYOUT    = false
	PRINT_HTML_TREE          = false
	PRINT_ACCESSIBILITY_TREE = false
)

var (
	BROKEN_IMAGE image.Image
)

func init() {
	image_bytes, err1 := os.ReadFile("Broken_Image.png")
	image, _, err2 := image.Decode(bytes.NewReader(image_bytes))
	if err := cmp.Or(err1, err2); err != nil {
		panic("Could not load broken image: " + err.Error())
	}
	BROKEN_IMAGE = image
}

type Tab struct {
	display_list []html.Command

	url                   *u.URL
	tab_height            float64
	history               []*u.URL
	focus                 *html.HtmlNode
	focused_frame         *Frame
	needs_raf_callbacks   bool
	needs_accessibility   bool
	needs_paint           bool
	root_frame            *Frame
	dark_mode             bool
	scroll_changed_in_tab bool
	scroll                float64

	accessibility_is_on bool
	accessibility_tree  *AccessibilityNode
	has_spoken_document bool
	accessibility_focus bool
	loaded              bool

	TaskRunner *TaskRunner
	browser    *Browser

	composited_updates []*html.HtmlNode
	zoom               float64

	window_id_to_frame map[int]*Frame
	origin_to_js       map[string]*JSContext
}

func NewTab(browser *Browser, tab_height float64) *Tab {
	tab := &Tab{
		tab_height:         tab_height,
		history:            make([]*u.URL, 0),
		browser:            browser,
		dark_mode:          browser.dark_mode,
		window_id_to_frame: make(map[int]*Frame),
		zoom:               1.0,
		origin_to_js:       make(map[string]*JSContext),
	}
	tab.TaskRunner = NewTaskRunner(tab)
	tab.TaskRunner.StartThread()
	return tab
}

func (t *Tab) Load(url *u.URL, payload string) {
	t.loaded = false
	t.history = append(t.history, url)
	t.TaskRunner.ClearPendingTasks()
	t.root_frame = NewFrame(t, nil, nil)
	t.root_frame.Load(url, payload)
	t.root_frame.frame_width = WIDTH
	t.root_frame.frame_height = t.tab_height
	t.loaded = true
}

func (t *Tab) click(x, y float64) {
	t.Render()
	t.root_frame.click(x, y)
}

func (t *Tab) go_back() {
	if len(t.history) > 1 {
		t.history = t.history[:len(t.history)-1] // pop
		back := t.history[len(t.history)-1]
		t.history = t.history[:len(t.history)-1] // pop
		t.Load(back, "")
	}
}

func (t *Tab) Render() {
	t.browser.measure.Time("render")

	for _, frame := range t.window_id_to_frame {
		if frame.Loaded {
			frame.Render()
		}
	}

	if t.needs_accessibility {
		t.accessibility_tree = NewAccessibilityNode(t.root_frame.Nodes, nil)
		t.accessibility_tree.Build()
		if PRINT_ACCESSIBILITY_TREE {
			A11yPrintTree(t.accessibility_tree, 0)
		}
		t.needs_accessibility = false
		t.needs_paint = true
	}

	if t.needs_paint {
		start := time.Now()
		t.display_list = make([]html.Command, 0)
		paint_tree(t.root_frame.Document, &t.display_list)
		if PRINT_DISPLAY_LIST {
			html.PrintCommands(t.display_list, 0)
		}
		fmt.Println("Paint took:", time.Since(start))
		t.needs_paint = false
	}

	t.browser.measure.Stop("render")
}

func (t *Tab) run_animation_frame(scroll *float64) {
	if !t.root_frame.scroll_changed_in_frame {
		t.root_frame.scroll = *scroll
	}

	needs_composite := false
	for _, frame := range t.window_id_to_frame {
		if !frame.Loaded {
			continue
		}

		t.browser.measure.Time("eval_run_raf_handlers")
		frame.js.DispatchRAF(frame.window_id)
		t.browser.measure.Stop("eval_run_raf_handlers")

		for _, node := range html.TreeToList(frame.Nodes) {
			for property_name, animation := range node.Animations {
				value := animation.Animate()
				if value != "" {
					node.Style[property_name] = value
					t.composited_updates = append(t.composited_updates, node)
					t.SetNeedsPaint()
				}
			}
		}
		if frame.needs_style || frame.needs_layout {
			needs_composite = true
		}
	}

	t.Render()

	if t.focus != nil && t.focused_frame.needs_focus_scroll {
		t.focused_frame.scroll_to(t.focus)
		t.focused_frame.needs_focus_scroll = false
	}

	for _, frame := range t.window_id_to_frame {
		if frame == t.root_frame {
			continue
		}
		if frame.scroll_changed_in_frame {
			needs_composite = true
			frame.scroll_changed_in_frame = false
		}
	}

	scroll = nil
	if t.root_frame.scroll_changed_in_frame {
		scroll = &t.root_frame.scroll
	}

	var composited_updates map[*html.HtmlNode]html.VisualEffectCommand
	if !needs_composite {
		composited_updates = map[*html.HtmlNode]html.VisualEffectCommand{}
		for _, node := range t.composited_updates {
			composited_updates[node] = node.BlendOp
		}
	}
	t.composited_updates = make([]*html.HtmlNode, 0)

	root_frame_focused := t.focused_frame == nil || t.focused_frame == t.root_frame
	commit_data := NewCommitData(t.root_frame.url, scroll, math.Ceil(t.root_frame.Document.Height + 2*VSTEP), t.display_list, composited_updates, t.accessibility_tree, t.focus, root_frame_focused)
	t.display_list = make([]html.Command, 0)
	t.root_frame.scroll_changed_in_frame = false

	t.browser.Commit(t, commit_data)
}

func (t *Tab) SetNeedsRenderAllFrames() {
	for _, frame := range t.window_id_to_frame {
		// note: might need sort based on insertion order
		frame.SetNeedsRender()
	}
}

func (t *Tab) SetNeedsAccessibility() {
	if !t.accessibility_is_on {
		return
	}
	t.needs_accessibility = true
	t.browser.SetNeedsAnimationFrame(t)
}

func (t *Tab) SetNeedsPaint() {
	t.needs_paint = true
	t.browser.SetNeedsAnimationFrame(t)
}

func (t *Tab) keypress(char rune) {
	frame := t.root_frame
	if t.focused_frame != nil {
		frame = t.focused_frame
	}
	frame.keypress(char)
}

func (t *Tab) backspace() {
	frame := t.root_frame
	if t.focused_frame != nil {
		frame = t.focused_frame
	}
	frame.backspace()
}

func (t *Tab) advance_tab() {
	frame := t.root_frame
	if t.focused_frame != nil {
		frame = t.focused_frame
	}
	frame.advance_tab()
}

func (t *Tab) enter() {
	if t.focus != nil {
		frame := t.root_frame
		if t.focused_frame != nil {
			frame = t.focused_frame
		}
		frame.activate_element(t.focus)
	}
}

func (t *Tab) ZoomBy(increment bool) {
	if increment {
		t.zoom *= 1.1
		t.scroll *= 1.1
	} else {
		t.zoom *= 1 / 1.1
		t.scroll *= 1 / 1.1
	}
	t.scroll_changed_in_tab = true
	t.SetNeedsRenderAllFrames()
}

func (t *Tab) ResetZoom() {
	t.scroll /= t.zoom
	t.zoom = 1.0
	t.scroll_changed_in_tab = true
	t.SetNeedsRenderAllFrames()
}

func (t *Tab) set_dark_mode(val bool) {
	t.dark_mode = val
	t.SetNeedsRenderAllFrames()
}

func (t *Tab) ScrollUp() {
	frame := t.root_frame
	if t.focused_frame != nil {
		frame = t.focused_frame
	}
	frame.scroll_up()
	t.SetNeedsPaint()
}

func (t *Tab) ScrollDown() {
	frame := t.root_frame
	if t.focused_frame != nil {
		frame = t.focused_frame
	}
	frame.scroll_down()
	t.SetNeedsPaint()
}

func paint_tree(layout_object *LayoutNode, displayList *[]html.Command) {
	cmds := layout_object.Layout.Paint()

	if _, ok := layout_object.Layout.(*IframeLayout); ok && layout_object.Node.Frame != nil && layout_object.Node.Frame.(*Frame).Loaded {
		paint_tree(layout_object.Node.Frame.(*Frame).Document, &cmds)
	} else {
		for _, child := range layout_object.Children.Get() {
			paint_tree(child, &cmds)
		}
	}

	cmds = layout_object.Layout.PaintEffects(cmds)
	*displayList = append(*displayList, cmds...)
}

func (t *Tab) get_js(url *u.URL) *JSContext {
	origin := url.Origin()
	if _, found := t.origin_to_js[origin]; !found {
		t.origin_to_js[origin] = NewJSContext(t, origin)
	}
	return t.origin_to_js[origin]
}

func (t *Tab) post_message(message string, target_window_id int) {
	frame := t.window_id_to_frame[target_window_id]
	frame.js.dispatch_post_message(message, target_window_id)
}
