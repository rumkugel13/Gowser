package browser

import (
	"fmt"
	"gowser/css"
	"gowser/html"
	"gowser/layout"
	"gowser/rect"
	"gowser/task"
	"gowser/try"
	u "gowser/url"
	"math"
	urllib "net/url"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	WIDTH                 = 800.
	HEIGHT                = 600.
	SCROLL_STEP           = 100.
	PRINT_DISPLAY_LIST    = false
	PRINT_DOCUMENT_LAYOUT = false
	PRINT_HTML_TREE       = false
)

type Tab struct {
	display_list          []html.Command
	scroll                float64
	document              *layout.LayoutNode
	url                   *u.URL
	tab_height            float64
	history               []*u.URL
	Nodes                 *html.HtmlNode
	rules                 []css.Rule
	focus                 *html.HtmlNode
	js                    *JSContext
	allowed_origins       []string
	TaskRunner            *TaskRunner
	browser               *Browser
	scroll_changed_in_tab bool
	needs_style           bool
	needs_layout          bool
	needs_paint           bool
	composited_updates    []*html.HtmlNode
	zoom                  float64
	dark_mode             bool
}

func NewTab(browser *Browser, tab_height float64) *Tab {
	tab := &Tab{
		scroll:                0,
		tab_height:            tab_height,
		history:               make([]*u.URL, 0),
		browser:               browser,
		scroll_changed_in_tab: false,
		zoom:                  1.0,
		dark_mode:             browser.dark_mode,
	}
	tab.TaskRunner = NewTaskRunner(tab)
	tab.TaskRunner.StartThread()
	return tab
}

func (t *Tab) Load(url *u.URL, payload string) {
	t.focus = nil
	t.zoom = 1
	t.scroll = 0
	t.scroll_changed_in_tab = true
	t.TaskRunner.ClearPendingTasks()
	fmt.Println("Requesting URL:", url)
	start := time.Now()
	headers, body := url.Request(t.url, payload)
	fmt.Println("Request took:", time.Since(start))
	t.history = append(t.history, url)
	t.url = url

	t.allowed_origins = nil
	if val, ok := headers["content-security-policy"]; ok {
		csp := strings.Fields(val)
		if len(csp) > 0 && csp[0] == "default-src" {
			t.allowed_origins = make([]string, 0)
			for _, origin := range csp[1:] {
				t.allowed_origins = append(t.allowed_origins, u.NewURL(origin).Origin())
			}
		}
	}

	start = time.Now()
	t.Nodes = html.NewHTMLParser(body).Parse()
	if PRINT_HTML_TREE {
		t.Nodes.PrintTree(0)
	}
	fmt.Println("Parsing took:", time.Since(start))

	start = time.Now()
	if t.js != nil {
		t.js.Discarded = true
	}
	t.js = NewJSContext(t)
	scripts := t.scripts(t.Nodes)
	for _, script := range scripts {
		script_url := url.Resolve(script)
		if !t.allowed_request(script_url) {
			fmt.Println("Blocked script", script_url, "due to CSP")
			continue
		}
		fmt.Println("Loading script:", script_url)
		var code string
		err := try.Try(func() {
			_, code = script_url.Request(url, "")
		})
		if err != nil {
			fmt.Println("Error loading script:", err)
		} else {
			task := task.NewTask(func(i ...interface{}) {
				start := time.Now()
				t.browser.measure.Time("eval_" + script)
				t.js.Run(script, code)
				t.browser.measure.Stop("eval_" + script)
				fmt.Println("Eval "+script+" took:", time.Since(start))
			}, script, code)
			t.TaskRunner.ScheduleTask(task)
		}
	}
	fmt.Println("Loading scripts took:", time.Since(start))

	start = time.Now()
	t.rules = slices.Clone(DEFAULT_STYLE_SHEET)
	links := t.links(t.Nodes)
	for _, link := range links {
		style_url := url.Resolve(link)
		if !t.allowed_request(style_url) {
			fmt.Println("Blocked stylesheet", style_url, "due to CSP")
			continue
		}
		fmt.Println("Loading stylesheet:", style_url)
		var style_body string
		err := try.Try(func() {
			_, style_body = style_url.Request(url, "")
		})
		if err != nil {
			fmt.Println("Error loading stylesheet:", err)
		} else {
			t.rules = append(t.rules, css.NewCSSParser(style_body).Parse()...)
		}
	}
	fmt.Println("Loading stylesheets took:", time.Since(start))
	t.SetNeedsRender()
}

func (t *Tab) scripts(nodes *html.HtmlNode) []string {
	flatNodes := html.TreeToList(nodes)
	links := []string{}
	for _, node := range flatNodes {
		if element, ok := node.Token.(html.ElementToken); ok && element.Tag == "script" {
			if src, exists := element.Attributes["src"]; exists {
				links = append(links, src)
			}
		}
	}
	return links
}

func (t *Tab) links(nodes *html.HtmlNode) []string {
	flatNodes := html.TreeToList(nodes)
	links := []string{}
	for _, node := range flatNodes {
		if element, ok := node.Token.(html.ElementToken); ok && element.Tag == "link" {
			if rel, exists := element.Attributes["rel"]; exists && rel == "stylesheet" {
				if href, exists := element.Attributes["href"]; exists {
					links = append(links, href)
				}
			}
		}
	}
	return links
}

func (t *Tab) click(x, y float64) {
	t.Render()
	t.focus_element(nil)

	y += t.scroll
	loc_rect := rect.NewRect(x, y, x+1, y+1)
	objs := []*layout.LayoutNode{}
	for _, obj := range layout.TreeToList(t.document) {
		if layout.AbsoluteBoundsForObj(obj).Intersects(loc_rect) {
			objs = append(objs, obj)
		}
	}

	if len(objs) == 0 {
		return
	}

	elt := objs[len(objs)-1].Node
	if elt != nil && t.js.DispatchEvent("click", elt) {
		return
	}
	for elt != nil {
		_, ok := elt.Token.(html.ElementToken)
		if !ok {
			// pass, text token
		} else if is_focusable(elt) {
			t.focus_element(elt)
			t.activate_element(elt)
			return
		}
		elt = elt.Parent
	}
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

	if t.needs_style {
		if t.dark_mode {
			INHERITED_PROPERTIES["color"] = "white"
		} else {
			INHERITED_PROPERTIES["color"] = "black"
		}
		start := time.Now()
		sort.SliceStable(t.rules, func(i, j int) bool {
			return css.CascadePriority(t.rules[i]) < css.CascadePriority(t.rules[j])
		})
		Style(t.Nodes, t.rules, t)
		fmt.Println("Styling took:", time.Since(start))
		t.needs_layout = true
		t.needs_style = false
	}

	if t.needs_layout {
		start := time.Now()
		t.document = layout.NewLayoutNode(layout.NewDocumentLayout(), t.Nodes, nil)
		t.document.Layout.(*layout.DocumentLayout).LayoutWithZoom(t.zoom)
		if PRINT_DOCUMENT_LAYOUT {
			layout.PrintTree(t.document, 0)
		}
		fmt.Println("Layout took:", time.Since(start))
		t.needs_paint = true
		t.needs_layout = false
	}

	if t.needs_paint {
		start := time.Now()
		t.display_list = make([]html.Command, 0)
		layout.PaintTree(t.document, &t.display_list)
		if PRINT_DISPLAY_LIST {
			html.PrintCommands(t.display_list, 0)
		}
		fmt.Println("Paint took:", time.Since(start))
		t.needs_paint = false
	}

	clamped_scroll := t.clamp_scroll(t.scroll)
	if clamped_scroll != t.scroll {
		t.scroll_changed_in_tab = true
	}
	t.scroll = clamped_scroll

	t.browser.measure.Stop("render")
}

func (t *Tab) clamp_scroll(scroll float64) float64 {
	height := math.Ceil(t.document.Height + 2*layout.VSTEP)
	maxscroll := height - t.tab_height
	return max(0, min(scroll, maxscroll))
}

func (t *Tab) run_animation_frame(scroll *float64) {
	if !t.scroll_changed_in_tab {
		t.scroll = *scroll
	}
	t.browser.measure.Time("eval_run_raf_handlers")
	t.js.ctx.PevalString("__runRAFHandlers()")
	t.browser.measure.Stop("eval_run_raf_handlers")

	for _, node := range html.TreeToList(t.Nodes) {
		for property_name, animation := range node.Animations {
			value := animation.Animate()
			if value != "" {
				node.Style[property_name] = value
				t.composited_updates = append(t.composited_updates, node)
				t.SetNeedsPaint()
			}
		}
	}

	needs_composite := t.needs_style || t.needs_layout

	t.Render()

	scroll = nil
	if t.scroll_changed_in_tab {
		scroll = &t.scroll
	}

	var composited_updates map[*html.HtmlNode]html.VisualEffectCommand
	if !needs_composite {
		composited_updates = map[*html.HtmlNode]html.VisualEffectCommand{}
		for _, node := range t.composited_updates {
			composited_updates[node] = node.BlendOp
		}
	}
	t.composited_updates = make([]*html.HtmlNode, 0)

	document_height := math.Ceil(t.document.Height + 2*layout.VSTEP)
	commit_data := NewCommitData(t.url, scroll, document_height, t.display_list, composited_updates)
	t.display_list = make([]html.Command, 0)
	t.browser.Commit(t, commit_data)
	t.scroll_changed_in_tab = false
}

func (t *Tab) SetNeedsRender() {
	t.needs_style = true
	t.browser.SetNeedsAnimationFrame(t)
}

func (t *Tab) SetNeedsLayout() {
	t.needs_layout = true
	t.browser.SetNeedsAnimationFrame(t)
}

func (t *Tab) SetNeedsPaint() {
	t.needs_paint = true
	t.browser.SetNeedsAnimationFrame(t)
}

func (t *Tab) keypress(char rune) {
	if t.focus != nil && t.focus.Token.(html.ElementToken).Tag == "input" {
		if t.focus.Token.(html.ElementToken).Attributes["value"] == "" {
			t.activate_element(t.focus)
		}
		if t.js.DispatchEvent("keydown", t.focus) {
			return
		}
		t.focus.Token.(html.ElementToken).Attributes["value"] += string(char)
		t.SetNeedsRender()
	}
}

func (t *Tab) backspace() {
	if t.focus != nil && t.focus.Token.(html.ElementToken).Tag == "input" {
		if t.focus.Token.(html.ElementToken).Attributes["value"] == "" {
			t.activate_element(t.focus)
		}
		if t.js.DispatchEvent("keydown", t.focus) {
			return
		}
		if len(t.focus.Token.(html.ElementToken).Attributes["value"]) > 0 {
			t.focus.Token.(html.ElementToken).Attributes["value"] = t.focus.Token.(html.ElementToken).Attributes["value"][:len(t.focus.Token.(html.ElementToken).Attributes["value"])-1]
		}
		t.SetNeedsRender()
	}
}

func (t *Tab) submit_form(elt *html.HtmlNode) {
	if t.js.DispatchEvent("submit", elt) {
		return
	}
	var inputs []*html.ElementToken
	for _, node := range html.TreeToList(elt) {
		if element, ok := node.Token.(html.ElementToken); ok && element.Tag == "input" && element.Attributes["name"] != "" {
			inputs = append(inputs, &element)
		}
	}

	var body string
	for _, input := range inputs {
		name := input.Attributes["name"]
		value := input.Attributes["value"]
		name = urllib.QueryEscape(name)
		value = urllib.QueryEscape(value)
		body += "&" + name + "=" + value
	}
	body = body[1:]

	url := t.url.Resolve(elt.Token.(html.ElementToken).Attributes["action"])
	t.Load(url, body)
}

func (t *Tab) advance_tab() {
	focusable_nodes := []*html.HtmlNode{}
	for _, node := range html.TreeToList(t.Nodes) {
		if _, ok := node.Token.(html.ElementToken); ok && is_focusable(node) {
			focusable_nodes = append(focusable_nodes, node)
		}
	}
	sort.SliceStable(focusable_nodes, func(i, j int) bool {
		return html.GetTabIndex(focusable_nodes[i]) < html.GetTabIndex(focusable_nodes[j])
	})

	idx := 0
	if slices.Contains(focusable_nodes, t.focus) {
		idx = slices.Index(focusable_nodes, t.focus) + 1
	}

	if idx < len(focusable_nodes) {
		t.focus_element(focusable_nodes[idx])
		t.browser.FocusContent()
	} else {
		t.focus_element(nil)
		t.browser.FocusAddressbar()
	}
	t.SetNeedsRender()
}

func is_focusable(node *html.HtmlNode) bool {
	if html.GetTabIndex(node) < 0 {
		return false
	} else if _, ok := node.Token.(html.ElementToken).Attributes["tabindex"]; ok {
		return true
	} else {
		return slices.Contains([]string{"input", "button", "a"}, node.Token.(html.ElementToken).Tag)
	}
}

func (t *Tab) enter() {
	if t.focus == nil {
		return
	}
	if t.js.DispatchEvent("click", t.focus) {
		return
	}
	t.activate_element(t.focus)
}

func (t *Tab) focus_element(node *html.HtmlNode) {
	if t.focus != nil {
		tok := t.focus.Token.(html.ElementToken)
		tok.IsFocused = false
		t.focus.Token = tok
	}
	t.focus = node
	if node != nil {
		tok := node.Token.(html.ElementToken)
		tok.IsFocused = true
		node.Token = tok
	}
	t.SetNeedsRender()
}

func (t *Tab) activate_element(node *html.HtmlNode) {
	elt, _ := node.Token.(html.ElementToken)
	if elt.Tag == "input" {
		elt.Attributes["value"] = ""
		t.SetNeedsRender()
	} else if elt.Tag == "a" && elt.Attributes["href"] != "" {
		url := t.url.Resolve(elt.Attributes["href"])
		t.Load(url, "")
	} else if elt.Tag == "button" {
		for node != nil {
			elt, _ := node.Token.(html.ElementToken)
			if elt.Tag == "form" && elt.Attributes["action"] != "" {
				t.submit_form(node)
			}
			node = node.Parent
		}
	}
}

func (t *Tab) allowed_request(url *u.URL) bool {
	return t.allowed_origins == nil || slices.Contains(t.allowed_origins, url.Origin())
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
	t.SetNeedsRender()
}

func (t *Tab) ResetZoom() {
	t.scroll /= t.zoom
	t.zoom = 1.0
	t.scroll_changed_in_tab = true
	t.SetNeedsRender()
}

func (t *Tab) set_dark_mode(val bool) {
	t.dark_mode = val
	t.SetNeedsRender()
}
