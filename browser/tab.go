package browser

import (
	"fmt"
	"gowser/css"
	"gowser/html"
	"gowser/layout"
	"gowser/try"
	u "gowser/url"
	urllib "net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tdewolff/canvas"
)

const (
	WIDTH       = 800.
	HEIGHT      = 600.
	SCROLL_STEP = 100.
)

type Tab struct {
	display_list    []layout.Command
	scroll          float64
	document        *layout.LayoutNode
	url             *u.URL
	tab_height      float64
	history         []*u.URL
	Nodes           *html.Node
	rules           []css.Rule
	focus           *html.Node
	js              *JSContext
	allowed_origins []string
}

func NewTab(tab_height float64) *Tab {
	return &Tab{
		scroll:     0,
		tab_height: tab_height,
		history:    make([]*u.URL, 0),
	}
}

func (t *Tab) Load(url *u.URL, payload string) {
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
	// t.nodes.PrintTree(0)
	fmt.Println("Parsing took:", time.Since(start))

	start = time.Now()
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
			t.js.Run(script, code)
		}
	}
	fmt.Println("Eval took:", time.Since(start))

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
	t.render()
}

func (t *Tab) Raster(canvas *canvas.Context) {
	// layout.PrintCommands(t.display_list)
	for _, cmd := range t.display_list {
		cmd.Execute(canvas)
	}
}

func (t *Tab) scripts(nodes *html.Node) []string {
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

func (t *Tab) links(nodes *html.Node) []string {
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

func (t *Tab) scrollDown() {
	max_y := max(t.document.Height+2*layout.VSTEP-t.tab_height, 0)
	t.scroll = min(t.scroll+SCROLL_STEP, max_y)
}

func (t *Tab) click(x, y float64) {
	if t.focus != nil {
		tok := t.focus.Token.(html.ElementToken)
		tok.IsFocused = false
		t.focus.Token = tok
	}
	t.focus = nil

	y += t.scroll
	objs := []*layout.LayoutNode{}
	for _, obj := range layout.TreeToList(t.document) {
		if obj.X <= x && x < obj.X+obj.Width &&
			obj.Y <= y && y < obj.Y+obj.Height {
			objs = append(objs, obj)
		}
	}

	if len(objs) == 0 {
		return
	}

	elt := objs[len(objs)-1].Node
	for elt != nil {
		element, ok := elt.Token.(html.ElementToken)
		if !ok {
			// pass, text token
		} else if element.Tag == "a" && element.Attributes["href"] != "" {
			if t.js.DispatchEvent("click", elt) {
				return
			}
			url := t.url.Resolve(element.Attributes["href"])
			t.Load(url, "")
			return
		} else if element.Tag == "input" {
			if t.js.DispatchEvent("click", elt) {
				return
			}
			t.focus = elt

			tok := elt.Token.(html.ElementToken)
			tok.Attributes["value"] = ""
			tok.IsFocused = true
			elt.Token = tok

			t.render()
			return
		} else if element.Tag == "button" {
			if t.js.DispatchEvent("click", elt) {
				return
			}
			for elt != nil {
				if elt.Token.(html.ElementToken).Tag == "form" && elt.Token.(html.ElementToken).Attributes["action"] != "" {
					t.submit_form(elt)
					return
				}
				elt = elt.Parent
			}
		}
		elt = elt.Parent
	}
	t.render()
}

func (t *Tab) go_back() {
	if len(t.history) > 1 {
		t.history = t.history[:len(t.history)-1] // pop
		back := t.history[len(t.history)-1]
		t.history = t.history[:len(t.history)-1] // pop
		t.Load(back, "")
	}
}

func (t *Tab) render() {
	start := time.Now()
	sort.SliceStable(t.rules, func(i, j int) bool {
		return css.CascadePriority(t.rules[i]) < css.CascadePriority(t.rules[j])
	})
	css.Style(t.Nodes, t.rules)
	fmt.Println("Styling took:", time.Since(start))

	start = time.Now()
	t.document = layout.NewLayoutNode(layout.NewDocumentLayout(), t.Nodes, nil)
	t.document.Layout.Layout()
	// layout.PrintTree(b.document, 0)
	t.display_list = make([]layout.Command, 0)
	layout.PaintTree(t.document, &t.display_list)
	// layout.PrintCommands(b.display_list)
	fmt.Println("Layout took:", time.Since(start))
}

func (t *Tab) keypress(char rune) {
	if t.focus != nil {
		if t.js.DispatchEvent("keydown", t.focus) {
			return
		}
		t.focus.Token.(html.ElementToken).Attributes["value"] += string(char)
		t.render()
	}
}

func (t *Tab) submit_form(elt *html.Node) {
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

func (t *Tab) allowed_request(url *u.URL) bool {
	return t.allowed_origins == nil || slices.Contains(t.allowed_origins, url.Origin())
}
