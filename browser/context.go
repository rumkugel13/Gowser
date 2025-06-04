package browser

import (
	"fmt"
	"gowser/css"
	"gowser/html"
	"os"

	duk "gopkg.in/olebedev/go-duktape.v3"
)

var (
	RUNTIME_JS string
)

type JSContext struct {
	ctx            *duk.Context
	tab            *Tab
	node_to_handle map[*html.Node]int
	handle_to_node map[int]*html.Node
}

func NewJSContext(tab *Tab) *JSContext {
	load_runtime_js()
	js := &JSContext{
		ctx:            duk.New(),
		tab:            tab,
		node_to_handle: make(map[*html.Node]int),
		handle_to_node: make(map[int]*html.Node),
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
	err = js.ctx.PevalString(RUNTIME_JS)
	if err != nil {
		fmt.Println(err)
	}
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

func (j *JSContext) DispatchEvent(eventType string, elt *html.Node) bool {
	handle := -1
	if val, ok := j.node_to_handle[elt]; ok {
		handle = val
	}

	err := j.ctx.PevalString(fmt.Sprintf("new Node(%d).dispatchEvent(new Event(\"%s\"));", handle, eventType))
	if err != nil {
		fmt.Println("Error executing dispatchEvent:", err)
	}

	do_default := j.ctx.GetBoolean(-1)

	j.ctx.Pop() // pop Node
	return !do_default
}

func log(ctx *duk.Context) int {
	numArgs := ctx.GetTop()
	for i := range numArgs {
		fmt.Print(ctx.SafeToString(i), " ")
	}
	fmt.Println()
	return 0
}

func (j *JSContext) querySelectorAll(selector_text string) []*html.Node {
	selector := css.NewCSSParser(selector_text).Selector()
	var nodes []*html.Node
	for _, node := range html.TreeToList(j.tab.Nodes) {
		if selector.Matches(node) {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (j *JSContext) get_handle(elt *html.Node) int {
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

func (j *JSContext) innerHTML_set(handle int, s string) {
	doc := html.NewHTMLParser("<html><body>" + s + "</body></html>").Parse()
	new_nodes := doc.Children[0].Children
	elt := j.handle_to_node[handle]
	elt.Children = new_nodes
	for _, child := range elt.Children {
		child.Parent = elt
	}
	j.tab.render()
}

func load_runtime_js() {
	data, err := os.ReadFile("runtime.js")
	if err != nil {
		fmt.Println("Error loading js runtime:", err)
		return
	}

	fmt.Println("Loading js runtime runtime.js")
	RUNTIME_JS = string(data)
}
