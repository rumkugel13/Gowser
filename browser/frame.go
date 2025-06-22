package browser

import (
	"bytes"
	"fmt"
	"gowser/rect"
	"gowser/task"
	u "gowser/url"
	"image"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	urllib "net/url"
)

type Frame struct {
	tab           *Tab
	parent_frame  *Frame
	frame_element *HtmlNode
	needs_style   bool
	needs_layout  bool
	needs_paint   bool

	Document                *LayoutNode
	scroll                  float64
	scroll_changed_in_frame bool
	needs_focus_scroll      bool
	zoom                    float64
	Nodes                   *HtmlNode
	rules                   []Rule
	url                     *u.URL
	js                      *JSContext
	Loaded                  bool
	allowed_origins         []string

	frame_width  float64
	frame_height float64
	window_id    int
}

func NewFrame(tab *Tab, parent_frame *Frame, frame_element *HtmlNode) *Frame {
	frame := &Frame{
		tab:           tab,
		parent_frame:  parent_frame,
		frame_element: frame_element,
		scroll:        0,
		zoom:          1.0,
	}
	frame.window_id = len(frame.tab.window_id_to_frame)
	frame.tab.window_id_to_frame[frame.window_id] = frame
	return frame
}

func (f *Frame) Load(url *u.URL, payload string) {
	f.Loaded = false
	f.zoom = 1.0
	f.scroll = 0
	f.scroll_changed_in_frame = true
	fmt.Println("Requesting URL:", url)
	start := time.Now()
	headers, body, err := url.Request(f.url, payload)
	if err != nil {
		fmt.Println("Request failed: " + err.Error())
		return
	}
	fmt.Println("Request took:", time.Since(start))

	f.url = url

	f.allowed_origins = nil
	if val, ok := headers["content-security-policy"]; ok {
		csp := strings.Fields(val)
		if len(csp) > 0 && csp[0] == "default-src" {
			f.allowed_origins = make([]string, 0)
			for _, origin := range csp[1:] {
				new_url, err := u.NewURL(origin)
				if err != nil {
					fmt.Println("Invalid URL: " + err.Error())
					continue
				}
				f.allowed_origins = append(f.allowed_origins, new_url.Origin())
			}
		}
	}

	start = time.Now()
	f.Nodes = NewHTMLParser(string(body)).Parse()
	if PRINT_HTML_TREE {
		f.Nodes.PrintTree(0)
	}
	fmt.Println("Parsing took:", time.Since(start))

	start = time.Now()
	if f.js != nil {
		f.js.Discarded = true
	}
	f.js = f.tab.get_js(url)
	f.js.AddWindow(f)
	scripts := f.scripts(f.Nodes)
	for _, script := range scripts {
		script_url, err := url.Resolve(script)
		if err != nil {
			fmt.Println("Resolving URL failed:", err.Error())
			continue
		}
		if !f.allowed_request(script_url) {
			fmt.Println("Blocked script", script_url, "due to CSP")
			continue
		}
		fmt.Println("Loading script:", script_url)
		_, code, err := script_url.Request(url, "")
		if err != nil {
			fmt.Println("Error loading script:", err)
		} else {
			task := task.NewTask(func(i ...interface{}) {
				start := time.Now()
				f.tab.browser.measure.Time("eval_" + script)
				f.js.Run(script, string(code), f.window_id)
				f.tab.browser.measure.Stop("eval_" + script)
				fmt.Println("Eval "+script+" took:", time.Since(start))
			}, script, code)
			f.tab.TaskRunner.ScheduleTask(task)
		}
	}
	fmt.Println("Loading scripts took:", time.Since(start))

	start = time.Now()
	f.rules = slices.Clone(DEFAULT_STYLE_SHEET)
	links := f.links(f.Nodes)
	for _, link := range links {
		style_url, err := url.Resolve(link)
		if err != nil {
			fmt.Println("Resolving URL failed:", err.Error())
			continue
		}
		if !f.allowed_request(style_url) {
			fmt.Println("Blocked stylesheet", style_url, "due to CSP")
			continue
		}
		fmt.Println("Loading stylesheet:", style_url)
		_, style_body, err := style_url.Request(url, "")
		if err != nil {
			fmt.Println("Error loading stylesheet:", err)
		} else {
			f.rules = append(f.rules, NewCSSParser(string(style_body)).Parse()...)
		}
	}
	fmt.Println("Loading stylesheets took:", time.Since(start))

	start = time.Now()
	images := f.images(f.Nodes)
	for _, img := range images {
		elt, _ := img.Token.(ElementToken)
		src := elt.Attributes["src"]
		image_url, err := url.Resolve(src)
		if err != nil {
			fmt.Println("Resolving URL failed:", err.Error())
			continue
		}
		if !f.allowed_request(image_url) {
			fmt.Println("Blocked image", image_url, "due to CSP")
			continue
		}
		fmt.Println("Loading image:", image_url)
		_, img_body, err := image_url.Request(url, "")
		if err != nil {
			fmt.Println("Error loading image:", err)
			img.Image = BROKEN_IMAGE
		} else {
			image, _, err := image.Decode(bytes.NewReader(img_body))
			if err != nil {
				fmt.Println("Error decoding image:", err)
				img.Image = BROKEN_IMAGE
			} else {
				img.Image = image
			}
		}
	}
	fmt.Println("Loading images took:", time.Since(start))

	start = time.Now()
	iframes := f.frames(f.Nodes)
	for _, iframe := range iframes {
		elt, _ := iframe.Token.(ElementToken)
		src := elt.Attributes["src"]
		iframe_url, err := url.Resolve(src)
		if err != nil {
			fmt.Println("Resolving URL failed:", err.Error())
			continue
		}
		if !f.allowed_request(iframe_url) {
			fmt.Println("Blocked iframe", iframe_url, "due to CSP")
			iframe.Frame = nil
			continue
		}
		iframe.Frame = NewFrame(f.tab, f, iframe)

		task := task.NewTask(func(i ...interface{}) {
			fmt.Println("Loading iframe:", iframe_url)
			iframe.Frame.Load(iframe_url, "")
		}, iframe_url)
		f.tab.TaskRunner.ScheduleTask(task)
	}
	fmt.Println("Loading iframes took:", time.Since(start))

	f.Document = NewLayoutNode(NewDocumentLayout(), f.Nodes, nil, nil, f)
	f.SetNeedsRender()
	f.Loaded = true
}

func (f *Frame) allowed_request(url *u.URL) bool {
	return f.allowed_origins == nil || slices.Contains(f.allowed_origins, url.Origin())
}

func (f *Frame) scripts(nodes *HtmlNode) []string {
	flatNodes := TreeToList(nodes)
	links := []string{}
	for _, node := range flatNodes {
		if element, ok := node.Token.(ElementToken); ok && element.Tag == "script" {
			if src, exists := element.Attributes["src"]; exists {
				links = append(links, src)
			}
		}
	}
	return links
}

func (f *Frame) links(nodes *HtmlNode) []string {
	flatNodes := TreeToList(nodes)
	links := []string{}
	for _, node := range flatNodes {
		if element, ok := node.Token.(ElementToken); ok && element.Tag == "link" {
			if rel, exists := element.Attributes["rel"]; exists && rel == "stylesheet" {
				if href, exists := element.Attributes["href"]; exists {
					links = append(links, href)
				}
			}
		}
	}
	return links
}

func (f *Frame) images(nodes *HtmlNode) []*HtmlNode {
	flatNodes := TreeToList(nodes)
	images := []*HtmlNode{}
	for _, node := range flatNodes {
		if element, ok := node.Token.(ElementToken); ok && element.Tag == "img" {
			images = append(images, node)
		}
	}
	return images
}

func (f *Frame) frames(nodes *HtmlNode) []*HtmlNode {
	flatNodes := TreeToList(nodes)
	iframes := []*HtmlNode{}
	for _, node := range flatNodes {
		if element, ok := node.Token.(ElementToken); ok && element.Tag == "iframe" {
			if _, exists := element.Attributes["src"]; exists {
				iframes = append(iframes, node)
			}
		}
	}
	return iframes
}

func (f *Frame) SetNeedsRender() {
	f.needs_style = true
	f.tab.SetNeedsAccessibility()
	f.tab.SetNeedsPaint()
}

func (f *Frame) SetNeedsLayout() {
	f.needs_layout = true
	f.tab.SetNeedsAccessibility()
	f.tab.SetNeedsPaint()
}

func (f *Frame) Render() {
	if f.needs_style {
		if f.tab.dark_mode {
			INHERITED_PROPERTIES["color"] = "white"
		} else {
			INHERITED_PROPERTIES["color"] = "black"
		}
		start := time.Now()
		sort.SliceStable(f.rules, func(i, j int) bool {
			return CascadePriority(f.rules[i]) < CascadePriority(f.rules[j])
		})
		Style(f.Nodes, f.rules, f.tab)
		fmt.Println("Styling took:", time.Since(start))
		f.needs_layout = true
		f.needs_style = false
	}

	if f.needs_layout {
		start := time.Now()
		f.Document.Layout.(*DocumentLayout).LayoutWithZoom(f.tab.zoom)
		if PRINT_DOCUMENT_LAYOUT {
			PrintTree(f.Document, 0)
		}
		fmt.Println("Layout took:", time.Since(start))
		f.tab.needs_accessibility = true
		f.needs_paint = true
		f.needs_layout = false
	}

	clamped_scroll := f.clamp_scroll(f.scroll)
	if clamped_scroll != f.scroll {
		f.scroll_changed_in_frame = true
	}
	f.scroll = clamped_scroll
}

func (f *Frame) click(x, y float64) {
	f.focus_element(nil)

	y += f.scroll
	loc_rect := rect.NewRect(x, y, x+1, y+1)
	objs := []*LayoutNode{}
	for _, obj := range LayoutTreeToList(f.Document) {
		if AbsoluteBoundsForObj(obj).Intersects(loc_rect) {
			objs = append(objs, obj)
		}
	}

	if len(objs) == 0 {
		return
	}

	obj := objs[len(objs)-1].Node
	if obj != nil && f.js.DispatchEvent("click", obj, f.window_id) {
		return
	}
	for obj != nil {
		elt, ok := obj.Token.(ElementToken)
		if !ok {
			// pass, text token
		} else if elt.Tag == "iframe" {
			abs_bounds := AbsoluteBoundsForObj(obj.LayoutObject)
			border := dpx(1.0, obj.LayoutObject.Zoom.Get())
			new_x := x - abs_bounds.Left - border
			new_y := y - abs_bounds.Top - border
			obj.Frame.click(new_x, new_y)
			return
		} else if IsFocusable(obj) {
			f.focus_element(obj)
			f.activate_element(obj)
			f.SetNeedsRender()
			return
		}
		obj = obj.Parent
	}
}

func (f *Frame) focus_element(node *HtmlNode) {
	if node != nil && node != f.tab.focus {
		f.needs_focus_scroll = true
	}
	if f.tab.focus != nil {
		tok := f.tab.focus.Token.(ElementToken)
		tok.IsFocused = false
		f.tab.focus.Token = tok
		dirty_style(f.tab.focus)
	}
	if f.tab.focused_frame != nil && f.tab.focused_frame != f {
		f.tab.focused_frame.SetNeedsRender()
	}
	f.tab.focus = node
	f.tab.focused_frame = f
	if node != nil {
		tok := node.Token.(ElementToken)
		tok.IsFocused = true
		node.Token = tok
		dirty_style(node)
	}
	f.SetNeedsRender()
}
func (f *Frame) advance_tab() {
	focusable_nodes := []*HtmlNode{}
	for _, node := range TreeToList(f.Nodes) {
		if _, ok := node.Token.(ElementToken); ok && IsFocusable(node) && GetTabIndex(node) >= 0 {
			focusable_nodes = append(focusable_nodes, node)
		}
	}
	sort.SliceStable(focusable_nodes, func(i, j int) bool {
		return GetTabIndex(focusable_nodes[i]) < GetTabIndex(focusable_nodes[j])
	})

	idx := 0
	if slices.Contains(focusable_nodes, f.tab.focus) {
		idx = slices.Index(focusable_nodes, f.tab.focus) + 1
	}

	if idx < len(focusable_nodes) {
		f.focus_element(focusable_nodes[idx])
		f.tab.browser.FocusContent()
	} else {
		f.focus_element(nil)
		f.tab.browser.FocusAddressbar()
	}
	f.SetNeedsRender()
}

func (f *Frame) activate_element(node *HtmlNode) {
	elt, _ := node.Token.(ElementToken)
	if elt.Tag == "input" {
		elt.Attributes["value"] = ""
		f.SetNeedsRender()
	} else if elt.Tag == "a" && elt.Attributes["href"] != "" {
		url, err := f.url.Resolve(elt.Attributes["href"])
		if err != nil {
			fmt.Println("Resolving URL failed:", err.Error())
		} else {
			f.Load(url, "")
		}
	} else if elt.Tag == "button" {
		for node != nil {
			elt, _ := node.Token.(ElementToken)
			if elt.Tag == "form" && elt.Attributes["action"] != "" {
				f.submit_form(node)
			}
			node = node.Parent
		}
	}
}

func (f *Frame) submit_form(elt *HtmlNode) {
	if f.js.DispatchEvent("submit", elt, f.window_id) {
		return
	}
	var inputs []*ElementToken
	for _, node := range TreeToList(elt) {
		if element, ok := node.Token.(ElementToken); ok && element.Tag == "input" && element.Attributes["name"] != "" {
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

	url, err := f.url.Resolve(elt.Token.(ElementToken).Attributes["action"])
	if err != nil {
		fmt.Println("Resolving URL failed:", err.Error())
	} else {
		f.Load(url, body)
	}
}

func (f *Frame) keypress(char rune) {
	if f.tab.focus != nil && f.tab.focus.Token.(ElementToken).Tag == "input" {
		if _, ok := f.tab.focus.Token.(ElementToken).Attributes["value"]; !ok {
			f.activate_element(f.tab.focus)
		}
		if f.js.DispatchEvent("keydown", f.tab.focus, f.window_id) {
			return
		}
		f.tab.focus.Token.(ElementToken).Attributes["value"] += string(char)
		f.SetNeedsRender()
	} else if f.tab.focus != nil && f.tab.focus.Token.(ElementToken).Attributes["contenteditable"] != "" {
		text_nodes := []*HtmlNode{}
		for _, t := range TreeToList(f.tab.focus) {
			if _, text := t.Token.(TextToken); text {
				text_nodes = append(text_nodes, t)
			}
		}
		var last_text *HtmlNode
		if len(text_nodes) > 0 {
			last_text = text_nodes[len(text_nodes)-1]
		} else {
			last_text = NewNode(NewTextToken(""), f.tab.focus)
			f.tab.focus.Children = append(f.tab.focus.Children, last_text)
		}
		txt := last_text.Token.(TextToken)
		txt.Text += string(char)
		last_text.Token = txt
		obj := f.tab.focus.LayoutObject
		_, isBlock := obj.Layout.(*BlockLayout)
		for !isBlock {
			obj = obj.Parent
			_, isBlock = obj.Layout.(*BlockLayout)
		}
		obj.Children.Mark()
		f.SetNeedsRender()
	}
}

func (f *Frame) backspace() {
	if f.tab.focus != nil && f.tab.focus.Token.(ElementToken).Tag == "input" {
		if _, ok := f.tab.focus.Token.(ElementToken).Attributes["value"]; !ok {
			f.activate_element(f.tab.focus)
		}
		if f.js.DispatchEvent("keydown", f.tab.focus, f.window_id) {
			return
		}
		if len(f.tab.focus.Token.(ElementToken).Attributes["value"]) > 0 {
			f.tab.focus.Token.(ElementToken).Attributes["value"] = f.tab.focus.Token.(ElementToken).Attributes["value"][:len(f.tab.focus.Token.(ElementToken).Attributes["value"])-1]
		}
		f.SetNeedsRender()
	} else if f.tab.focus != nil && f.tab.focus.Token.(ElementToken).Attributes["contenteditable"] != "" {
		text_nodes := []*HtmlNode{}
		for _, t := range TreeToList(f.tab.focus) {
			if _, text := t.Token.(TextToken); text {
				text_nodes = append(text_nodes, t)
			}
		}
		var last_text *HtmlNode
		if len(text_nodes) > 0 {
			last_text = text_nodes[len(text_nodes)-1]
		} else {
			last_text = NewNode(NewTextToken(""), f.tab.focus)
			f.tab.focus.Children = append(f.tab.focus.Children, last_text)
		}
		txt := last_text.Token.(TextToken)
		if len(txt.Text) > 0 {
			txt.Text = txt.Text[:len(txt.Text)-1]
		}
		last_text.Token = txt
		obj := f.tab.focus.LayoutObject
		_, isBlock := obj.Layout.(*BlockLayout)
		for !isBlock {
			obj = obj.Parent
			_, isBlock = obj.Layout.(*BlockLayout)
		}
		obj.Children.Mark()
		f.SetNeedsRender()
	}
}

func (f *Frame) clamp_scroll(scroll float64) float64 {
	height := math.Ceil(f.Document.Height.Get() + 2*VSTEP)
	maxscroll := height - f.frame_height
	return max(0, min(scroll, maxscroll))
}

func (f *Frame) scroll_up() {
	f.scroll = f.clamp_scroll(f.scroll - SCROLL_STEP)
	f.scroll_changed_in_frame = true
}

func (f *Frame) scroll_down() {
	f.scroll = f.clamp_scroll(f.scroll + SCROLL_STEP)
	f.scroll_changed_in_frame = true
}

func (f *Frame) scroll_to(elt *HtmlNode) {
	if f.needs_style || f.needs_layout {
		panic("scroll_to called without needs_style or needs_layout")
	}
	layoutNodes := LayoutTreeToList(f.Document)
	objIdx := slices.IndexFunc(layoutNodes, func(obj *LayoutNode) bool {
		// note: use elt here?
		return obj.Node == f.tab.focus
	})
	if objIdx == -1 {
		return
	}

	obj := layoutNodes[objIdx]
	if f.scroll < obj.Y.Get() && obj.Y.Get() < f.scroll+f.frame_height {
		return
	}

	new_scroll := obj.Y.Get() - SCROLL_STEP
	f.scroll = f.clamp_scroll(new_scroll)
	f.scroll_changed_in_frame = true
	f.tab.SetNeedsPaint()
}
