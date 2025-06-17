package browser

import (
	"fmt"
	"gowser/css"
	"gowser/html"
	"gowser/task"
	"os"
	"time"

	duk "gopkg.in/olebedev/go-duktape.v3"
)

var (
	RUNTIME_JS string
)

func init() {
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
	node_to_handle map[*html.HtmlNode]int
	handle_to_node map[int]*html.HtmlNode
	Discarded      bool
}

func NewJSContext(tab *Tab) *JSContext {
	js := &JSContext{
		ctx:            duk.New(),
		tab:            tab,
		node_to_handle: make(map[*html.HtmlNode]int),
		handle_to_node: make(map[int]*html.HtmlNode),
		Discarded:      false,
	}
	js.ctx.PushGlobalGoFunction("_log", log)
	_, err := js.ctx.PushGlobalGoFunction("_querySelectorAll", func(ctx *duk.Context) int {
		selector_text := ctx.SafeToString(0)
		nodes := js.querySelectorAll(selector_text)
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
		js.innerHTML_set(handle, s)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_style_set", func(ctx *duk.Context) int {
		handle := ctx.GetInt(0)
		s := ctx.GetString(1)
		js.style_set(handle, s)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_XMLHttpRequest_send", func(ctx *duk.Context) int {
		method := ctx.GetString(0) // 0 is bottom of stack [0, .., -1]
		url := ctx.GetString(-4)   // -2 is second to top, in this case [0, -2, -1]
		body := ctx.GetString(-3)  // -1 is top of stack is stack is laid out correctly
		is_async := ctx.GetBoolean(-2)
		handle := ctx.GetInt(-1)
		out := js.xmlHttpRequest_send(method, url, body, is_async, handle)
		ctx.PushString(out)
		return 1
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = js.ctx.PushGlobalGoFunction("_setTimeout", func(ctx *duk.Context) int {
		handle := ctx.GetInt(-2)
		time := ctx.GetInt(-1)
		js.setTimeout(handle, time)
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
		handle := ctx.GetInt(-3)
		attr := ctx.GetString(-2)
		value := ctx.GetString(-1)
		js.setAttribute(handle, attr, value)
		return 0
	})
	if err != nil {
		fmt.Println(err)
	}
	tab.browser.measure.Time("eval_runtime_js")
	err = js.ctx.PevalString(RUNTIME_JS)
	if err != nil {
		fmt.Println(err)
	}
	tab.browser.measure.Stop("eval_runtime_js")
	return js
}

func (j *JSContext) Run(script, code string) (string, error) {
	err := j.ctx.PevalString(code)
	if err != nil {
		fmt.Println("Script", script, "crashed", err)
		return "", err
	}
	val := j.ctx.SafeToString(-1)
	j.ctx.Pop()
	return val, nil
}

func (j *JSContext) DispatchEvent(eventType string, elt *html.HtmlNode) bool {
	handle := -1
	if val, ok := j.node_to_handle[elt]; ok {
		handle = val
	}

	j.tab.browser.measure.Time("eval_dispatch_event")
	err := j.ctx.PevalString(fmt.Sprintf("new Node(%d).dispatchEvent(new Event(\"%s\"));", handle, eventType))
	if err != nil {
		fmt.Println("Error executing dispatchEvent:", err)
	}
	j.tab.browser.measure.Stop("eval_dispatch_event")

	do_default := j.ctx.GetBoolean(-1)

	j.ctx.Pop() // pop Node
	return !do_default
}

func (j *JSContext) dispatch_settimeout(handle int) {
	if j.Discarded {
		return
	}
	j.tab.browser.measure.Time("eval_set_timeout")
	j.ctx.PevalString(fmt.Sprintf("__runSetTimeout(%d)", handle))
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

func (j *JSContext) querySelectorAll(selector_text string) []*html.HtmlNode {
	selector := css.NewCSSParser(selector_text).Selector()
	var nodes []*html.HtmlNode
	for _, node := range html.TreeToList(j.tab.Nodes) {
		if selector.Matches(node) {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (j *JSContext) get_handle(elt *html.HtmlNode) int {
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
	attr := elt.Token.(html.ElementToken).Attributes[attribute]
	return attr
}

func (j *JSContext) setAttribute(handle int, attr, value string) {
	elt := j.handle_to_node[handle]
	elt.Token.(html.ElementToken).Attributes[attr] = value
	j.tab.SetNeedsRender()
}

func (j *JSContext) innerHTML_set(handle int, s string) {
	doc := html.NewHTMLParser("<html><body>" + s + "</body></html>").Parse()
	new_nodes := doc.Children[0].Children
	elt := j.handle_to_node[handle]
	elt.Children = new_nodes
	for _, child := range elt.Children {
		child.Parent = elt
	}
	j.tab.SetNeedsRender()
}

func (j *JSContext) style_set(handle int, s string) {
	elt := j.handle_to_node[handle]
	elt.Token.(html.ElementToken).Attributes["style"] = s
	j.tab.SetNeedsRender()
}

func (j *JSContext) xmlHttpRequest_send(method string, url string, body string, is_async bool, handle int) string {
	full_url, err := j.tab.url.Resolve(url)
	if err != nil {
		fmt.Println("Request failed: " + err.Error())
		return ""
	}
	if !j.tab.allowed_request(full_url) {
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
			j.dispatch_xhr_onload(string(response), handle)
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

func (j *JSContext) dispatch_xhr_onload(out string, handle int) {
	if j.Discarded {
		return
	}
	j.tab.browser.measure.Time("eval_dispatch_xhr_onload")
	j.ctx.EvalString(fmt.Sprintf("__runXHROnload(%s, %d)", out, handle))
	j.tab.browser.measure.Stop("eval_dispatch_xhr_onload")
}

func (j *JSContext) setTimeout(handle int, t int) {
	run_callback := func() {
		task := task.NewTask(func(i ...interface{}) {
			j.dispatch_settimeout(handle)
		}, handle)
		j.tab.TaskRunner.ScheduleTask(task)
	}
	time.AfterFunc(time.Duration(t)*time.Millisecond, run_callback)
}

func (j *JSContext) requestAnimationFrame() {
	j.tab.browser.SetNeedsAnimationFrame(j.tab)
}
