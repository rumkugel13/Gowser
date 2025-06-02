package layout

import (
	"fmt"
	"gowser/html"
	"slices"
	"strconv"
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

type Layout interface {
	Layout()
	String() string
	Paint() []Command
	Wrap(*LayoutNode)
}

type DocumentLayout struct {
	Wrapper *LayoutNode
}

func NewDocumentLayout() *DocumentLayout {
	return &DocumentLayout{}
}

func (d *DocumentLayout) Layout() {
	child := NewLayoutNode(NewBlockLayout(nil), d.Wrapper.Node, d.Wrapper)
	d.Wrapper.children = append(d.Wrapper.children, child)

	d.Wrapper.Width = DefaultWidth - 2*HSTEP
	d.Wrapper.X = HSTEP
	d.Wrapper.Y = VSTEP
	child.Layout.Layout()
	d.Wrapper.Height = child.Height
}

func (d *DocumentLayout) String() string {
	return fmt.Sprintf("DocumentLayout(x=%f, y=%f, width=%f, height=%f)", d.Wrapper.X, d.Wrapper.Y, d.Wrapper.Width, d.Wrapper.Height)
}

func (d *DocumentLayout) Paint() []Command {
	return []Command{}
}

func (d *DocumentLayout) Wrap(wrap *LayoutNode) {
	d.Wrapper = wrap
}

type BlockLayout struct {
	cursor_x, cursor_y float32
	wrap               *LayoutNode
	previous           *LayoutNode
}

func NewBlockLayout(previous *LayoutNode) *BlockLayout {
	layout := &BlockLayout{
		cursor_x: float32(HSTEP),
		cursor_y: float32(VSTEP),
		previous: previous,
	}
	return layout
}

func (l *BlockLayout) Layout() {
	if l.previous != nil {
		l.wrap.Y = l.previous.Y + l.previous.Height
	} else {
		l.wrap.Y = l.wrap.parent.Y
	}
	l.wrap.X = l.wrap.parent.X
	l.wrap.Width = l.wrap.parent.Width

	mode := l.layout_mode()
	if mode == "block" {
		var previous *LayoutNode
		for _, child := range l.wrap.Node.Children {
			next := NewLayoutNode(NewBlockLayout(previous), child, l.wrap)
			l.wrap.children = append(l.wrap.children, next)
			previous = next
		}
	} else {
		l.new_line()
		l.recurse(l.wrap.Node)
	}

	for _, child := range l.wrap.children {
		child.Layout.Layout()
	}

	var totalHeight float32
	for _, child := range l.wrap.children {
		totalHeight += child.Height
	}
	l.wrap.Height = totalHeight
}

func (l *BlockLayout) String() string {
	return fmt.Sprintf("BlockLayout(mode=%s, x=%f, y=%f, width=%f, height=%f, node=%v, style=%v)", l.layout_mode(),
		l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.wrap.Node.Token, l.wrap.Node.Style)
}

func (l *BlockLayout) Paint() []Command {
	cmds := make([]Command, 0)

	bgcolor, ok := l.wrap.Node.Style["background-color"]
	if !ok {
		bgcolor = "transparent"
	}
	if bgcolor != "transparent" && bgcolor != "" {
		rect := NewDrawRect(l.self_rect(), bgcolor)
		cmds = append(cmds, rect)
	}
	return cmds
}

func (d *BlockLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

func (l *BlockLayout) layout_mode() string {
	if _, ok := l.wrap.Node.Token.(html.TextToken); ok {
		return "inline"
	} else {
		for _, child := range l.wrap.Node.Children {
			if element, ok := child.Token.(html.ElementToken); ok && slices.Contains(BLOCK_ELEMENTS, element.Tag) {
				return "block"
			}
		}
		if len(l.wrap.Node.Children) > 0 {
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
			l.word(node, word)
		}
	} else {
		if element, ok := node.Token.(html.ElementToken); ok && element.Tag == "br" {
			l.new_line()
		}
		for _, child := range node.Children {
			l.recurse(child)
		}
	}
}

func (l *BlockLayout) word(node *html.Node, word string) {
	weight := node.Style["font-weight"]
	style := node.Style["font-style"]
	if style == "normal" {
		style = "roman"
	}
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(node.Style["font-size"], "px"), 32)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	size := int(float32(fSize) * 0.75)

	font := GetFont(size, weight, style)
	width := Measure(font, word)
	if l.cursor_x+width > l.wrap.Width {
		l.new_line()
	}

	line := l.wrap.children[len(l.wrap.children)-1]
	var previous_word *LayoutNode
	if len(line.children) > 0 {
		previous_word = line.children[len(line.children)-1]
	}
	text := NewLayoutNode(NewTextLayout(word, previous_word), node, line)
	line.children = append(line.children, text)
	l.cursor_x += width + Measure(font, " ")
}

func (l *BlockLayout) new_line() {
	l.cursor_x = 0
	var last_line *LayoutNode
	if len(l.wrap.children) > 0 {
		last_line = l.wrap.children[len(l.wrap.children)-1]
	}
	new_line := NewLayoutNode(NewLineLayout(last_line), l.wrap.Node, l.wrap)
	l.wrap.children = append(l.wrap.children, new_line)
}

func (l *BlockLayout) self_rect() *Rect {
	return NewRect(l.wrap.X, l.wrap.Y, l.wrap.X+l.wrap.Width, l.wrap.Y+l.wrap.Height)
}

func PaintTree(l *LayoutNode, displayList *[]Command) {
	*displayList = append(*displayList, l.Layout.Paint()...)
	for _, child := range l.children {
		PaintTree(child, displayList)
	}
}

func PrintTree(l *LayoutNode, indent int) {
	fmt.Println(strings.Repeat(" ", indent) + l.Layout.String())
	for _, child := range l.children {
		PrintTree(child, indent+2)
	}
}

func TreeToList(tree *LayoutNode, list *[]*LayoutNode) []*LayoutNode {
	*list = append(*list, tree)
	for _, child := range tree.children {
		TreeToList(child, list)
	}
	return *list
}

type LineLayout struct {
	wrap     *LayoutNode
	previous *LayoutNode
}

func NewLineLayout(previous *LayoutNode) *LineLayout {
	return &LineLayout{
		previous: previous,
	}
}

func (l *LineLayout) Layout() {
	l.wrap.Width = l.wrap.parent.Width
	l.wrap.X = l.wrap.parent.X

	if l.previous != nil {
		l.wrap.Y = l.previous.Y + l.previous.Height
	} else {
		l.wrap.Y = l.wrap.parent.Y
	}

	for _, word := range l.wrap.children {
		word.Layout.Layout()
	}

	var maxAscent float32
	for _, item := range l.wrap.children {
		maxAscent = max(maxAscent, float32(item.Layout.(*TextLayout).font.MetricsAscent(tk9_0.App)))
	}

	baseline := l.wrap.Y + 1.25*maxAscent
	for _, item := range l.wrap.children {
		item.Y = baseline - float32(item.Layout.(*TextLayout).font.MetricsAscent(tk9_0.App))
	}

	var maxDescent float32
	for _, item := range l.wrap.children {
		maxDescent = max(maxDescent, float32(item.Layout.(*TextLayout).font.MetricsDescent(tk9_0.App)))
	}

	l.wrap.Height = 1.25 * (maxAscent + maxDescent)
}

func (l *LineLayout) String() string {
	return fmt.Sprintf("LineLayout(x=%f, y=%f, width=%f, height=%f)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height)
}

func (l *LineLayout) Paint() []Command {
	return []Command{}
}

func (d *LineLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

type TextLayout struct {
	word     string
	wrap     *LayoutNode
	previous *LayoutNode
	font     *tk9_0.FontFace
}

func NewTextLayout(word string, previous *LayoutNode) *TextLayout {
	return &TextLayout{
		word:     word,
		previous: previous,
	}
}

func (l *TextLayout) Layout() {
	weight := l.wrap.Node.Style["font-weight"]
	style := l.wrap.Node.Style["font-style"]
	if style == "normal" {
		style = "roman"
	}
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(l.wrap.Node.Style["font-size"], "px"), 32)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	size := int(float32(fSize) * 0.75)
	l.font = GetFont(size, weight, style)

	l.wrap.Width = Measure(l.font, l.word)

	if l.previous != nil {
		space := Measure(l.previous.Layout.(*TextLayout).font, " ")
		l.wrap.X = l.previous.X + space + l.previous.Width
	} else {
		l.wrap.X = l.wrap.parent.X
	}

	l.wrap.Height = float32(l.font.MetricsLinespace(tk9_0.App))
}

func (l *TextLayout) String() string {
	return fmt.Sprintf("TextLayout(x=%f, y=%f, width=%f, height=%f, word='%s')", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.word)
}

func (l *TextLayout) Paint() []Command {
	color := l.wrap.Node.Style["color"]
	return []Command{NewDrawText(l.wrap.X, l.wrap.Y, l.word, l.font, color)}
}

func (d *TextLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}
