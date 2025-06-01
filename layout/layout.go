package layout

import (
	"fmt"
	"gowser/html"
	"slices"
	"strings"

	tk9_0 "modernc.org/tk9.0"
)

const (
	HSTEP        = 13.
	VSTEP        = 18.
	DefaultWidth = 800.
)

var BLOCK_ELEMENTS = []string{
	"html", "body", "article", "section", "nav", "aside",
	"h1", "h2", "h3", "h4", "h5", "h6", "hgroup", "header",
	"footer", "address", "p", "hr", "pre", "blockquote",
	"ol", "ul", "menu", "li", "dl", "dt", "dd", "figure",
	"figcaption", "main", "div", "table", "form", "fieldset",
	"legend", "details", "summary",
}

type LayoutItem struct {
	Word string
	X    float32
	Y    float32
	Font *tk9_0.FontFace
}

type LineItem struct {
	X    float32
	Word string
	Font *tk9_0.FontFace
}

type Layout interface {
	Layout()
	String() string
	Parent() *Layout
	Children() *[]Layout
	X() float32
	Y() float32
	Width() float32
	Height() float32
	Paint() []Command
}

type DocumentLayout struct {
	node                *html.Node
	parent              *Layout
	children            []Layout
	x, y, width, height float32
}

func NewDocumentLayout(node *html.Node) *DocumentLayout {
	return &DocumentLayout{
		node:     node,
		parent:   nil,
		children: make([]Layout, 0),
	}
}

func (d *DocumentLayout) Layout() {
	child := NewBlockLayout(d.node, d, nil)
	d.children = append(d.children, child)

	d.width = DefaultWidth - 2*HSTEP
	d.x = HSTEP
	d.y = VSTEP
	child.Layout()
	d.height = child.Height()
}

func (d *DocumentLayout) String() string {
	return fmt.Sprintf("DocumentLayout(x=%f, y=%f, width=%f, height=%f)", d.x, d.y, d.width, d.height)
}

func (d *DocumentLayout) Parent() *Layout {
	return d.parent
}

func (d *DocumentLayout) Children() *[]Layout {
	return &d.children
}

func (d *DocumentLayout) X() float32 {
	return d.x
}

func (d *DocumentLayout) Y() float32 {
	return d.y
}

func (d *DocumentLayout) Width() float32 {
	return d.width
}

func (d *DocumentLayout) Height() float32 {
	return d.height
}

func (d *DocumentLayout) Paint() []Command {
	return []Command{}
}

type BlockLayout struct {
	display_list        []LayoutItem
	cursor_x, cursor_y  float32
	weight, style       string
	size                float32
	Line                []LineItem
	node                *html.Node
	parent, previous    *Layout
	children            []Layout
	x, y, width, height float32
}

func NewBlockLayout(tree *html.Node, parent Layout, previous Layout) *BlockLayout {
	layout := &BlockLayout{
		display_list: make([]LayoutItem, 0),
		cursor_x:     float32(HSTEP),
		cursor_y:     float32(VSTEP),
		weight:       tk9_0.NORMAL,
		style:        tk9_0.ROMAN,
		size:         12,
		Line:         make([]LineItem, 0),
		node:         tree,
		parent:       &parent,
		previous:     &previous,
		children:     make([]Layout, 0),
	}
	return layout
}

func (l *BlockLayout) Layout() {
	if *l.previous != nil {
		l.y = (*l.previous).Y() + (*l.previous).Height()
	} else {
		l.y = (*l.parent).Y()
	}
	l.x = (*l.parent).X()
	l.width = (*l.parent).Width()

	mode := l.layout_mode()
	if mode == "block" {
		var previous Layout
		for _, child := range *l.node.Children {
			next := NewBlockLayout(&child, l, previous)
			l.children = append(l.children, next)
			previous = next
		}
	} else {
		l.cursor_x = 0
		l.cursor_y = 0
		l.weight = tk9_0.NORMAL
		l.style = tk9_0.ROMAN
		l.size = 12

		l.Line = make([]LineItem, 0)
		l.recurse(l.node)
		l.flush()
	}

	for _, child := range l.children {
		child.Layout()
	}

	if mode == "block" {
		var totalHeight float32
		for _, child := range l.children {
			totalHeight += child.Height()
		}
		l.height = totalHeight
	} else {
		l.height = l.cursor_y
	}
}

func (l *BlockLayout) String() string {
	return fmt.Sprintf("BlockLayout(mode=%s, x=%f, y=%f, width=%f, height=%f, node=%v, style=%v)", l.layout_mode(), l.x, l.y, l.width, l.height, l.node.Token, l.node.Style)
}

func (l *BlockLayout) Parent() *Layout {
	return l.parent
}

func (l *BlockLayout) Children() *[]Layout {
	return &l.children
}

func (l *BlockLayout) X() float32 {
	return l.x
}

func (l *BlockLayout) Y() float32 {
	return l.y
}

func (l *BlockLayout) Width() float32 {
	return l.width
}

func (l *BlockLayout) Height() float32 {
	return l.height
}

func (l *BlockLayout) Paint() []Command {
	cmds := make([]Command, 0)
	// if tag, ok := l.node.Token.(html.TagToken); ok && tag.Tag == "pre" {
	// 	x2, y2 := l.x+l.width, l.y+l.height
	// 	rect := NewDrawRect(l.x, l.y, x2, y2, "gray")
	// 	cmds = append(cmds, rect)
	// }

	bgcolor, ok := l.node.Style["background-color"]
	if !ok {
		bgcolor = "transparent"
	}
	if bgcolor != "transparent" && bgcolor != "" {
		x2, y2 := l.x+l.width, l.y+l.height
		rect := NewDrawRect(l.x, l.y, x2, y2, bgcolor)
		cmds = append(cmds, rect)
	}

	if l.layout_mode() == "inline" {
		for _, item := range l.display_list {
			cmds = append(cmds, NewDrawText(item.X, item.Y, item.Word, item.Font))
		}
	}
	return cmds
}

func (l *BlockLayout) layout_mode() string {
	if _, ok := l.node.Token.(html.TextToken); ok {
		return "inline"
	} else {
		for _, child := range *l.node.Children {
			if elem, ok := child.Token.(html.TagToken); ok && slices.Contains(BLOCK_ELEMENTS, elem.Tag) {
				return "block"
			}
		}
		if len(*l.node.Children) > 0 {
			return "inline"
		} else {
			return "block"
		}
	}
}

func (l *BlockLayout) recurse(node *html.Node) {
	if text, ok := node.Token.(html.TextToken); ok {
		words := strings.Fields(text.Text)
		for _, word := range words {
			l.word(word)
		}
	} else if tag, ok := node.Token.(html.TagToken); ok {
		l.open_tag(tag.Tag)
		for _, child := range *node.Children {
			l.recurse(&child)
		}
		l.close_tag(tag.Tag)
	}
}

func (l *BlockLayout) open_tag(tag string) {
	if tag == "i" {
		l.style = tk9_0.ITALIC
	} else if tag == "b" {
		l.weight = tk9_0.BOLD
	} else if tag == "small" {
		l.size -= 2
	} else if tag == "big" {
		l.size += 4
	} else if tag == "br" {
		l.flush()
	}
}

func (l *BlockLayout) close_tag(tag string) {
	if tag == "i" {
		l.style = tk9_0.ROMAN
	} else if tag == "b" {
		l.weight = tk9_0.NORMAL
	} else if tag == "small" {
		l.size += 2
	} else if tag == "big" {
		l.size -= 4
	} else if tag == "p" {
		l.flush()
		l.cursor_y += float32(VSTEP)
	}
}

func (l *BlockLayout) word(word string) {
	font := GetFont(l.size, l.weight, l.style)
	width := measure(font, word)
	if l.cursor_x+width > l.width {
		l.flush()
	}
	l.Line = append(l.Line, LineItem{l.cursor_x, word, font})
	l.cursor_x += width + measure(font, " ")
}

func (l *BlockLayout) flush() {
	if len(l.Line) == 0 {
		return
	}
	var maxAscent float32
	for _, item := range l.Line {
		maxAscent = max(maxAscent, float32(item.Font.MetricsAscent(tk9_0.App)))
	}

	baseline := l.cursor_y + maxAscent*1.25
	for _, item := range l.Line {
		l.display_list = append(l.display_list, LayoutItem{
			Word: item.Word,
			X:    l.x + item.X,
			Y:    l.y + baseline - float32(item.Font.MetricsAscent(tk9_0.App)),
			Font: item.Font,
		})
	}

	var maxDescent float32
	for _, item := range l.Line {
		maxDescent = max(maxDescent, float32(item.Font.MetricsDescent(tk9_0.App)))
	}
	l.cursor_y = baseline + maxDescent*1.25
	l.cursor_x = 0
	l.Line = make([]LineItem, 0)
}

func measure(font *tk9_0.FontFace, text string) float32 {
	// Measure the width of the text using the font metrics
	// This is a simplified version of text width measurement based on character widths.
	// In a real implementation, you would use the font's metrics to get accurate widths.
	var width float32
	ascent := float32(font.MetricsAscent(tk9_0.App))
	for _, r := range text {
		switch r {
		case '!', '\'', '`', ',', '.', 'i', 'l', ':', ';', '|':
			width += ascent * 0.2
		case '"', '(', ')', '[', ']', '{', '}', 'f', 'I', 'j', 'r', 't', '\\', '/', ' ':
			width += ascent * 0.35
		case '*', '+', '-', '=', '<', '>', 'a', 'b', 'c', 'd', 'e', 'g', 'h', 'k', 'o', 'p', 'q', 's', 'u', 'v', 'x', 'y', 'z', '~':
			width += ascent * 0.55
		case '#', '$', '%', '&', '?', '@', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'K', 'L', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'X', 'Y', 'Z':
			width += ascent * 0.7
		case 'M', 'W', 'm', 'w', '—':
			width += ascent * 0.9
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			width += ascent * 0.6
		default:
			width += ascent * 0.6
		}
	}
	return width
}

func PaintTree(l Layout, displayList *[]Command) {
	*displayList = append(*displayList, l.Paint()...)
	for _, child := range *l.Children() {
		PaintTree(child, displayList)
	}
}

func PrintTree(l Layout, indent int) {
	fmt.Println(strings.Repeat(" ", indent) + l.String())
	for _, child := range *l.Children() {
		PrintTree(child, indent+2)
	}
}
