package layout

import (
	"fmt"
	"gowser/html"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/image/font"
)

const (
	HSTEP          = 13.
	VSTEP          = 18.
	DefaultWidth   = 800.
	INPUT_WIDTH_PX = 200.
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
	ShouldPaint() bool
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

func (d *DocumentLayout) ShouldPaint() bool {
	return true
}

func (d *DocumentLayout) Wrap(wrap *LayoutNode) {
	d.Wrapper = wrap
}

type BlockLayout struct {
	cursor_x, cursor_y float64
	wrap               *LayoutNode
	previous           *LayoutNode
}

func NewBlockLayout(previous *LayoutNode) *BlockLayout {
	layout := &BlockLayout{
		cursor_x: HSTEP,
		cursor_y: VSTEP,
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

	var totalHeight float64
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
	if bgcolor != "transparent" {
		radius, ok := l.wrap.Node.Style["border-radius"]
		if !ok {
			radius = "0px"
		}
		actualRadius, err := strconv.ParseFloat(strings.TrimSuffix(radius, "px"), 32)
		if err != nil {
			actualRadius = 0 // Default radius size if parsing fails
		}
		rect := NewDrawRRect(l.self_rect(), actualRadius, bgcolor)
		cmds = append(cmds, rect)
	}
	return cmds
}

func (d *BlockLayout) ShouldPaint() bool {
	if _, ok := d.wrap.Node.Token.(html.TextToken); ok || (d.wrap.Node.Token.(html.ElementToken).Tag != "input" && d.wrap.Node.Token.(html.ElementToken).Tag != "button") {
		return true
	}
	return false
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
		if len(l.wrap.Node.Children) > 0 || l.wrap.Node.Token.(html.ElementToken).Tag == "input" {
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
		} else if element, ok := node.Token.(html.ElementToken); ok && (element.Tag == "input" || element.Tag == "button") {
			l.input(node)
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
	size := fSize * 0.75

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

func (l *BlockLayout) input(node *html.Node) {
	w := INPUT_WIDTH_PX
	if l.cursor_x+w > l.wrap.Width {
		l.new_line()
	}
	line := l.wrap.children[len(l.wrap.children)-1]
	var previous_word *LayoutNode
	if len(line.children) > 0 {
		previous_word = line.children[len(line.children)-1]
	}
	input := NewLayoutNode(NewInputLayout(previous_word), node, line)
	line.children = append(line.children, input)

	weight := node.Style["font-weight"]
	style := node.Style["font-style"]
	if style == "normal" {
		style = "roman"
	}
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(node.Style["font-size"], "px"), 32)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	size := fSize * 0.75
	font := GetFont(size, weight, style)
	l.cursor_x += w + Measure(font, " ")
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
	if l.Layout.ShouldPaint() {
		*displayList = append(*displayList, l.Layout.Paint()...)
	}
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

func TreeToList(tree *LayoutNode) []*LayoutNode {
	list := []*LayoutNode{tree}
	for _, child := range tree.children {
		list = append(list, TreeToList(child)...)
	}
	return list
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

	var maxAscent float64
	for _, item := range l.wrap.children {
		switch l := item.Layout.(type) {
		case *TextLayout:
			maxAscent = max(maxAscent, Ascent(l.font))
		case *InputLayout:
			maxAscent = max(maxAscent, Ascent(l.font))
		}
	}

	baseline := l.wrap.Y + 1.25*maxAscent
	for _, item := range l.wrap.children {
		switch l := item.Layout.(type) {
		case *TextLayout:
			item.Y = baseline - Ascent(l.font)
		case *InputLayout:
			item.Y = baseline - Ascent(l.font)
		}
	}

	var maxDescent float64
	for _, item := range l.wrap.children {
		switch l := item.Layout.(type) {
		case *TextLayout:
			maxDescent = max(maxDescent, Descent(l.font))
		case *InputLayout:
			maxDescent = max(maxDescent, Descent(l.font))
		}
	}

	l.wrap.Height = 1.25 * (maxAscent + maxDescent)
}

func (l *LineLayout) String() string {
	return fmt.Sprintf("LineLayout(x=%f, y=%f, width=%f, height=%f)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height)
}

func (l *LineLayout) Paint() []Command {
	return []Command{}
}

func (d *LineLayout) ShouldPaint() bool {
	return true
}

func (d *LineLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

type TextLayout struct {
	word     string
	wrap     *LayoutNode
	previous *LayoutNode
	font     font.Face
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
	size := fSize * 0.75
	l.font = GetFont(size, weight, style)

	l.wrap.Width = Measure(l.font, l.word)

	if l.previous != nil {
		switch t := l.previous.Layout.(type) {
		case *TextLayout:
			space := Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		case *InputLayout:
			space := Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		default:
			l.wrap.X = l.wrap.parent.X
		}
	} else {
		l.wrap.X = l.wrap.parent.X
	}

	l.wrap.Height = Linespace(l.font)
}

func (l *TextLayout) String() string {
	return fmt.Sprintf("TextLayout(x=%f, y=%f, width=%f, height=%f, word='%s')", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.word)
}

func (l *TextLayout) Paint() []Command {
	color := l.wrap.Node.Style["color"]
	return []Command{NewDrawText(l.wrap.X, l.wrap.Y, l.word, l.font, color)}
}

func (d *TextLayout) ShouldPaint() bool {
	return true
}

func (d *TextLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

type InputLayout struct {
	wrap     *LayoutNode
	previous *LayoutNode
	font     font.Face
}

func NewInputLayout(previous *LayoutNode) *InputLayout {
	return &InputLayout{
		previous: previous,
	}
}

func (l *InputLayout) Layout() {
	weight := l.wrap.Node.Style["font-weight"]
	style := l.wrap.Node.Style["font-style"]
	if style == "normal" {
		style = "roman"
	}
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(l.wrap.Node.Style["font-size"], "px"), 32)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	size := fSize * 0.75
	l.font = GetFont(size, weight, style)

	l.wrap.Width = INPUT_WIDTH_PX

	if l.previous != nil {
		switch t := l.previous.Layout.(type) {
		case *TextLayout:
			space := Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		case *InputLayout:
			space := Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		default:
			l.wrap.X = l.wrap.parent.X
		}
	} else {
		l.wrap.X = l.wrap.parent.X
	}

	l.wrap.Height = Linespace(l.font)
}

func (l *InputLayout) String() string {
	return fmt.Sprintf("InputLayout(x=%f, y=%f, width=%f, height=%f)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height)
}

func (l *InputLayout) Paint() []Command {
	cmds := []Command{}
	bgcolor, ok := l.wrap.Node.Style["background-color"]
	if !ok {
		bgcolor = "transparent"
	}
	if bgcolor != "transparent" {
		radius, ok := l.wrap.Node.Style["border-radius"]
		if !ok {
			radius = "0px"
		}
		actualRadius, err := strconv.ParseFloat(strings.TrimSuffix(radius, "px"), 32)
		if err != nil {
			actualRadius = 0 // Default radius size if parsing fails
		}
		rect := NewDrawRRect(l.self_rect(), actualRadius, bgcolor)
		cmds = append(cmds, rect)
	}

	var text string
	if l.wrap.Node.Token.(html.ElementToken).Tag == "input" {
		text = l.wrap.Node.Token.(html.ElementToken).Attributes["value"]
	} else if l.wrap.Node.Token.(html.ElementToken).Tag == "button" {
		if len(l.wrap.Node.Children) == 1 {
			if txt, ok := l.wrap.Node.Children[0].Token.(html.TextToken); ok {
				text = txt.Text
			} else {
				fmt.Println("Ignoring HTML contents inside button")
			}
		} else {
			fmt.Println("Ignoring HTML contents inside button")
		}
	}

	color := l.wrap.Node.Style["color"]
	cmds = append(cmds, NewDrawText(l.wrap.X, l.wrap.Y, text, l.font, color))

	if l.wrap.Node.Token.(html.ElementToken).IsFocused {
		cx := l.wrap.X + Measure(l.font, text)
		cmds = append(cmds, NewDrawLine(cx, l.wrap.Y, cx, l.wrap.Y+l.wrap.Height, "black", 1))
	}

	return cmds
}

func (d *InputLayout) ShouldPaint() bool {
	return true
}

func (d *InputLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

func (l *InputLayout) self_rect() *Rect {
	return NewRect(l.wrap.X, l.wrap.Y, l.wrap.X+l.wrap.Width, l.wrap.Y+l.wrap.Height)
}
