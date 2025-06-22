package browser

import (
	"fmt"
	"gowser/task"
	"os"
	"time"

	duk "gopkg.in/olebedev/go-duktape.v3"
)

var (
	RUNTIME_JS string
)

func init() {
	os.Chdir(os.Getenv("WORKSPACE_DIR"))
	data, err := os.ReadFile("runtime.js")
	if err != nil {
		fmt.Println("Error loading js runtime:", err)
		return
	}

	fmt.Println("Loading js runtime from runtime.js")
	RUNTIME_JS = string(data)
}

type JSContext struct {
	ctx            *duk.Context
	tab            *Tab
	node_to_handle map[*HtmlNode]int
	handle_to_node map[int]*HtmlNode
	Discarded      bool
	origin         string
}

func NewJSContext(tab *Tab, origin string) *JSContext {
	js := &JSContext{
		ctx:            duk.New(),
		tab:            tab,
		node_to_handle: make(map[*HtmlNode]int),
		handle_to_node: make(map[int]*HtmlNode),
		Discarded:      false,
		origin:         origin,
	}
	js.ctx.PushGlobalGoFunction("_log", log)
	_, err := js.ctx.PushGlobalGoFunction("_querySelectorAll", func(ctx *duk.Context) int {
		selector_text := ctx.SafeToString(0)
		window_id := ctx.GetInt(1)
		nodes := js.querySelectorAll(selector_text, window_id)
		arr := ctx.PushArray()
		for i, node := range nodes {
			handle := js.get_handle(node)
			ctx.PushInt(handle)
			ctx.PutPropIndex(arr, uint(i))
		}
		return 1 // 1 array
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_getAttribute", func(ctx *duk.Context) int {
		handle := ctx.GetInt(0)
		attr := ctx.GetString(1)
		res := js.get_attribute(handle, attr)
		ctx.PushString(res)
		return 1
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_innerHTML_set", func(ctx *duk.Context) int {
		handle := ctx.GetInt(0)
		s := ctx.GetString(1)
		window_id := ctx.GetInt(2)
		js.innerHTML_set(handle, s, window_id)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_style_set", func(ctx *duk.Context) int {
		handle := ctx.GetInt(0)
		s := ctx.GetString(1)
		window_id := ctx.GetInt(2)
		js.style_set(handle, s, window_id)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_XMLHttpRequest_send", func(ctx *duk.Context) int {
		method := ctx.GetString(0) // 0 is bottom of stack [0, .., -1]
		url := ctx.GetString(-5)   // -2 is second to top, in this case [0, -2, -1]
		body := ctx.GetString(-4)  // -1 is top of stack is stack is laid out correctly
		is_async := ctx.GetBoolean(-3)
		handle := ctx.GetInt(-2)
		window_id := ctx.GetInt(-1)
		out := js.xmlHttpRequest_send(method, url, body, is_async, handle, window_id)
		ctx.PushString(out)
		return 1
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_setTimeout", func(ctx *duk.Context) int {
		handle := ctx.GetInt(-3)
		time := ctx.GetInt(-2)
		window_id := ctx.GetInt(-1)
		js.setTimeout(handle, time, window_id)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_requestAnimationFrame", func(ctx *duk.Context) int {
		js.requestAnimationFrame()
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_setAttribute", func(ctx *duk.Context) int {
		handle := ctx.GetInt(-4)
		attr := ctx.GetString(-3)
		value := ctx.GetString(-2)
		window_id := ctx.GetInt(-1)
		js.setAttribute(handle, attr, value, window_id)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}

	_, err = js.ctx.PushGlobalGoFunction("_parent", func(ctx *duk.Context) int {
		window_id := ctx.GetInt(-1)
		val := js.parent(window_id)
		if val != -1 {
			ctx.PushInt(val)
		} else {
			ctx.PushUndefined()
		}
		return 1
	})
	if err != nil {
		fmt.Println(err)
	}

	_, err = js.ctx.PushGlobalGoFunction("_postMessage", func(ctx *duk.Context) int {
		window_id := ctx.GetInt(0)
		message := ctx.GetString(1)
		origin := ctx.GetString(2)
		js.postMessage(window_id, message, origin)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}

	err = js.ctx.PevalString("function Window(id) { this._id = id };")
	if err != nil {
		fmt.Println(err)
	}
	err = js.ctx.PevalString("WINDOWS = {}")
	if err != nil {
		fmt.Println(err)
	}
	return js
}

func (j *JSContext) AddWindow(frame *Frame) {
	code := fmt.Sprintf("var window_%d = new Window(%d);", frame.window_id, frame.window_id)
	err := j.ctx.PevalString(code)
	if err != nil {
		fmt.Println(err)
	}

	j.tab.browser.measure.Time("eval_runtime_js")
	err = j.ctx.PevalString(j.wrap(RUNTIME_JS, frame.window_id))
	if err != nil {
		fmt.Println(err)
	}
	j.tab.browser.measure.Stop("eval_runtime_js")

	code = fmt.Sprintf("WINDOWS[%d] = window_%d;", frame.window_id, frame.window_id)
	err = j.ctx.PevalString(code)
	if err != nil {
		fmt.Println(err)
	}
}

func (j *JSContext) wrap(script string, window_id int) string {
	return fmt.Sprintf("window = window_%d; %s", window_id, script)
}

func (j *JSContext) Run(script, code string, window_id int) (string, error) {
	err := j.ctx.PevalString(j.wrap(code, window_id))
	if err != nil {
		fmt.Println("Script", script, "crashed", err)
		return "", err
	}
	val := j.ctx.SafeToString(-1)
	j.ctx.Pop()
	return val, nil
}

func (j *JSContext) DispatchEvent(eventType string, elt *HtmlNode, window_id int) bool {
	handle := -1
	if val, ok := j.node_to_handle[elt]; ok {
		handle = val
	}

	j.tab.browser.measure.Time("eval_dispatch_event")
	event_dispatch_js := fmt.Sprintf("new window.Node(%d).dispatchEvent(new window.Event(\"%s\"));", handle, eventType)
	err := j.ctx.PevalString(j.wrap(event_dispatch_js, window_id))
	if err != nil {
		fmt.Println("Error executing dispatchEvent:", err)
	}
	j.tab.browser.measure.Stop("eval_dispatch_event")

	do_default := j.ctx.GetBoolean(-1)

	j.ctx.Pop() // pop Node
	return !do_default
}

func (j *JSContext) dispatch_settimeout(handle int, window_id int) {
	if j.Discarded {
		return
	}
	j.tab.browser.measure.Time("eval_set_timeout")
	j.ctx.PevalString(j.wrap(fmt.Sprintf("window.__runSetTimeout(%d)", handle), window_id))
	j.tab.browser.measure.Stop("eval_set_timeout")
}

func log(ctx *duk.Context) int {
	numArgs := ctx.GetTop()
	for i := range numArgs {
		fmt.Print(ctx.SafeToString(i), " ")
	}
	fmt.Println()
	return 0
}

func (j *JSContext) querySelectorAll(selector_text string, window_id int) []*HtmlNode {
	frame := j.tab.window_id_to_frame[window_id]
	j.throw_if_cross_origin(frame)
	selector := NewCSSParser(selector_text).Selector()
	var nodes []*HtmlNode
	for _, node := range TreeToList(j.tab.root_frame.Nodes) {
		if selector.Matches(node) {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (j *JSContext) get_handle(elt *HtmlNode) int {
	var handle int
	if node, ok := j.node_to_handle[elt]; !ok {
		handle = len(j.node_to_handle)
		j.node_to_handle[elt] = handle
		j.handle_to_node[handle] = elt
	} else {
		handle = node
	}
	return handle
}

func (j *JSContext) get_attribute(handle int, attribute string) string {
	elt := j.handle_to_node[handle]
	attr := elt.Token.(ElementToken).Attributes[attribute]
	return attr
}

func (j *JSContext) setAttribute(handle int, attr, value string, window_id int) {
	frame := j.tab.window_id_to_frame[window_id]
	j.throw_if_cross_origin(frame)
	elt := j.handle_to_node[handle]
	elt.Token.(ElementToken).Attributes[attr] = value
	obj := elt.LayoutObject
	_, iframe := obj.Layout.(*IframeLayout)
	_, image := obj.Layout.(*ImageLayout)
	if iframe || image {
		if attr == "width" || attr == "height" {
			obj.Width.Mark()
			obj.Height.Mark()
		}
	}
	j.tab.SetNeedsRenderAllFrames()
}

func (j *JSContext) innerHTML_set(handle int, s string, window_id int) {
	frame := j.tab.window_id_to_frame[window_id]
	j.throw_if_cross_origin(frame)
	doc := NewHTMLParser("<html><body>" + s + "</body></html>").Parse()
	new_nodes := doc.Children[0].Children
	elt := j.handle_to_node[handle]
	elt.Children = new_nodes
	for _, child := range elt.Children {
		child.Parent = elt
	}
	obj := elt.LayoutObject
	_, isBlock := obj.Layout.(*BlockLayout)
	for !isBlock {
		obj = obj.Parent
		_, isBlock = obj.Layout.(*BlockLayout)
	}
	obj.Children.Mark()
	frame.SetNeedsRender()
}

func (j *JSContext) style_set(handle int, s string, window_id int) {
	frame := j.tab.window_id_to_frame[window_id]
	j.throw_if_cross_origin(frame)
	elt := j.handle_to_node[handle]
	elt.Token.(ElementToken).Attributes["style"] = s
	dirty_style(elt)
	frame.SetNeedsRender()
}

func (j *JSContext) xmlHttpRequest_send(method string, url string, body string, is_async bool, handle int, window_id int) string {
	full_url, err := j.tab.url.Resolve(url)
	if err != nil {
		fmt.Println("Request failed: " + err.Error())
		return ""
	}
	if !j.tab.root_frame.allowed_request(full_url) {
		fmt.Println("Cross-origin XHR blocked by CSP")
		return ""
	}
	if full_url.Origin() != j.tab.url.Origin() {
		fmt.Println("Cross-origin XHR request not allowed")
		return ""
	}
	run_load := func() string {
		_, response, err := full_url.Request(j.tab.url, body)
		if err != nil {
			fmt.Println("Request failed: " + err.Error())
			return ""
		}
		task := task.NewTask(func(i ...interface{}) {
			j.dispatch_xhr_onload(string(response), handle, window_id)
		}, response, handle)
		j.tab.TaskRunner.ScheduleTask(task)
		return string(response)
	}
	if !is_async {
		return run_load()
	} else {
		go run_load()
		return ""
	}
}

func (j *JSContext) dispatch_xhr_onload(out string, handle int, window_id int) {
	if j.Discarded {
		return
	}
	j.tab.browser.measure.Time("eval_dispatch_xhr_onload")
	j.ctx.EvalString(j.wrap(fmt.Sprintf("window.__runXHROnload(%s, %d)", out, handle), window_id))
	j.tab.browser.measure.Stop("eval_dispatch_xhr_onload")
}

func (j *JSContext) setTimeout(handle int, t int, window_id int) {
	run_callback := func() {
		task := task.NewTask(func(i ...interface{}) {
			j.dispatch_settimeout(handle, window_id)
		}, handle)
		j.tab.TaskRunner.ScheduleTask(task)
	}
	time.AfterFunc(time.Duration(t)*time.Millisecond, run_callback)
}

func (j *JSContext) requestAnimationFrame() {
	j.tab.browser.SetNeedsAnimationFrame(j.tab)
}

func (j *JSContext) DispatchRAF(window_id int) {
	j.ctx.PevalString("window.__runRAFHandlers()")
}

func (j *JSContext) parent(window_id int) int {
	parent_frame := j.tab.window_id_to_frame[window_id].parent_frame
	if parent_frame == nil {
		return -1
	}
	return parent_frame.window_id
}

func (j *JSContext) throw_if_cross_origin(frame *Frame) {
	if frame.url.Origin() != j.origin {
		panic("Cross-origin access disallowed from script")
	}
}

func (j *JSContext) postMessage(target_window_id int, message, origin string) {
	task := task.NewTask(func(i ...interface{}) {
		j.tab.post_message(message, target_window_id)
	})
	j.tab.TaskRunner.ScheduleTask(task)
}

func (j *JSContext) dispatch_post_message(message string, window_id int) {
	j.ctx.EvalString(j.wrap(fmt.Sprintf("window.dispatchEvent(new window.MessageEvent(%s))", message), window_id))
}
