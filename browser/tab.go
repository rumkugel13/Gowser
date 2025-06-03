package browser

import (
	"fmt"
	"gowser/css"
	"gowser/html"
	"gowser/layout"
	"gowser/try"
	"gowser/url"
	"slices"
	"sort"
	"time"

	tk9_0 "modernc.org/tk9.0"
)

const (
	DefaultWidth  = 800.
	DefaultHeight = 600.
	SCROLL_STEP   = 100.
)

type Tab struct {
	display_list []layout.Command
	scroll       float32
	document     *layout.LayoutNode
	url          *url.URL
	tab_height   float32
	history      []*url.URL
	nodes        *html.Node
	rules        []css.Rule
	focus        *html.Node
}

func NewTab(tab_height float32) *Tab {
	return &Tab{
		scroll:     0,
		tab_height: tab_height,
		history:    make([]*url.URL, 0),
	}
}

func (t *Tab) Load(url *url.URL) {
	t.history = append(t.history, url)
	t.url = url
	fmt.Println("Requesting URL:", url)
	start := time.Now()
	body := url.Request()
	fmt.Println("Request took:", time.Since(start))

	start = time.Now()
	t.nodes = html.NewHTMLParser(body).Parse()
	// t.nodes.PrintTree(0)
	fmt.Println("Parsing took:", time.Since(start))

	start = time.Now()
	t.rules = slices.Clone(DEFAULT_STYLE_SHEET)
	links := t.links(t.nodes)
	for _, link := range links {
		style_url := url.Resolve(link)
		fmt.Println("Loading stylesheet:", style_url)
		var style_body string
		err := try.Try(func() {
			style_body = style_url.Request()
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

func (t *Tab) Draw(canvas *tk9_0.CanvasWidget, offset float32) {
	start := time.Now()
	for _, cmd := range t.display_list {
		if cmd.Top() > t.scroll+t.tab_height {
			continue // Skip items that are outside the visible area
		}
		if cmd.Bottom() < t.scroll {
			continue // Skip items that are above the visible area
		}
		cmd.Execute(t.scroll-offset, *canvas)
	}
	fmt.Println("Drawing took:", time.Since(start))
}

func (t *Tab) links(nodes *html.Node) []string {
	flatNodes := html.TreeToList(nodes, &[]html.Node{})
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

func (t *Tab) click(x, y float32) {
	if t.focus != nil {
		tok := t.focus.Token.(html.ElementToken)
		tok.IsFocused = false
		t.focus.Token = tok
	}
	t.focus = nil

	y += t.scroll
	objs := []*layout.LayoutNode{}
	for _, obj := range layout.TreeToList(t.document, &[]*layout.LayoutNode{}) {
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
			url := t.url.Resolve(element.Attributes["href"])
			t.Load(url)
			return
		} else if element.Tag == "input" {
			t.focus = elt
			element.Attributes["value"] = ""

			tok := elt.Token.(html.ElementToken)
			tok.IsFocused = true
			elt.Token = tok
			
			t.render()
			return
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
		t.Load(back)
	}
}

func (t *Tab) render() {
	start := time.Now()
	sort.SliceStable(t.rules, func(i, j int) bool {
		return css.CascadePriority(t.rules[i]) < css.CascadePriority(t.rules[j])
	})
	css.Style(t.nodes, t.rules)
	fmt.Println("Styling took:", time.Since(start))

	start = time.Now()
	t.document = layout.NewLayoutNode(layout.NewDocumentLayout(), t.nodes, nil)
	t.document.Layout.Layout()
	// layout.PrintTree(b.document, 0)
	t.display_list = make([]layout.Command, 0)
	layout.PaintTree(t.document, &t.display_list)
	// layout.PrintCommands(b.display_list)
	fmt.Println("Layout took:", time.Since(start))
}

func (t *Tab) keypress(char rune) {
	if t.focus != nil {
		t.focus.Token.(html.ElementToken).Attributes["value"] += string(char)
		t.render()
	}
}
