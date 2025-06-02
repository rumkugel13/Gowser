package browser

import (
	"fmt"
	"gowser/css"
	"gowser/html"
	"gowser/layout"
	"gowser/try"
	"gowser/url"
	"os"
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

var (
	DEFAULT_STYLE_SHEET []css.Rule
)

type Browser struct {
	window       *tk9_0.Window
	canvas       *tk9_0.CanvasWidget
	display_list []layout.Command
	scroll       float32
	document     *layout.LayoutNode
	url			 *url.URL
}

func NewBrowser() *Browser {
	load_default_style_sheet()
	browser := &Browser{}
	browser.canvas = tk9_0.Canvas(tk9_0.Width(DefaultWidth), tk9_0.Height(DefaultHeight), tk9_0.Background("white"))
	browser.window = tk9_0.App.Center()
	tk9_0.Pack(browser.canvas)
	browser.scroll = 0
	tk9_0.Bind(tk9_0.App, "<Down>", tk9_0.Command(browser.scrollDown))
	tk9_0.Bind(tk9_0.App, "<Button-1>", tk9_0.Command(browser.click))
	return browser
}

func (b *Browser) Load(url *url.URL) {
	b.url = url
	fmt.Println("Requesting URL:", url)
	start := time.Now()
	body := url.Request()
	fmt.Println("Request took:", time.Since(start))

	start = time.Now()
	nodes := html.NewHTMLParser(body).Parse()
	// nodes.PrintTree(0)
	fmt.Println("Parsing took:", time.Since(start))

	start = time.Now()
	rules := slices.Clone(DEFAULT_STYLE_SHEET)
	links := b.links(nodes)
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
			rules = append(rules, css.NewCSSParser(style_body).Parse()...)
		}
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return css.CascadePriority(rules[i]) < css.CascadePriority(rules[j])
	})
	css.Style(nodes, rules)
	fmt.Println("Styling took:", time.Since(start))

	start = time.Now()
	b.document = layout.NewLayoutNode(layout.NewDocumentLayout(), nodes, nil)
	b.document.Layout.Layout()
	// layout.PrintTree(b.document, 0)
	b.display_list = make([]layout.Command, 0)
	layout.PaintTree(b.document, &b.display_list)
	// layout.PrintCommands(b.display_list)
	fmt.Println("Layout took:", time.Since(start))

	start = time.Now()
	b.Draw()
	fmt.Println("Drawing took:", time.Since(start))
}

func (b *Browser) Draw() {
	b.canvas.Delete("all")
	for _, cmd := range b.display_list {
		if cmd.Top() > b.scroll+DefaultHeight {
			continue // Skip items that are outside the visible area
		}
		if cmd.Bottom() < b.scroll {
			continue // Skip items that are above the visible area
		}
		cmd.Execute(b.scroll, *b.canvas)
	}
}

func (b *Browser) links(nodes *html.Node) []string {
	flatNodes := html.TreeToList(nodes, &[]html.Node{})
	links := []string{}
	for _, node := range flatNodes {
		if tag, ok := node.Token.(html.TagToken); ok && tag.Tag == "link" {
			if rel, exists := tag.Attributes["rel"]; exists && rel == "stylesheet" {
				if href, exists := tag.Attributes["href"]; exists {
					links = append(links, href)
				}
			}
		}
	}
	return links
}

func (b *Browser) scrollDown() {
	max_y := max(b.document.Height+2*layout.VSTEP-DefaultHeight, 0)
	b.scroll = min(b.scroll+SCROLL_STEP, max_y)
	b.Draw()
}

func (b *Browser) click(e *tk9_0.Event) {
	x, y := float32(e.X), float32(e.Y)
	y += b.scroll
	objs := []*layout.LayoutNode{}
	for _, obj := range layout.TreeToList(b.document, &[]*layout.LayoutNode{}) {
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
		tag, ok := elt.Token.(html.TagToken)
		if !ok {
			// pass, text token
		} else if tag.Tag == "a" && tag.Attributes["href"] != "" {
			url := b.url.Resolve(tag.Attributes["href"])
			b.Load(url)
			return
		}
		elt = elt.Parent
	}
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
