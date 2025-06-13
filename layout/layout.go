package layout

import (
	"fmt"
	"gowser/css"
	fnt "gowser/font"
	"gowser/html"
	"gowser/rect"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/image/font"
)

const (
	HSTEP          = 13.
	VSTEP          = 18.
	WIDTH          = 800.
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
	Paint() []html.Command
	Wrap(*LayoutNode)
	ShouldPaint() bool
	PaintEffects([]html.Command) []html.Command
}

type DocumentLayout struct {
	Wrapper *LayoutNode
}

func NewDocumentLayout() *DocumentLayout {
	return &DocumentLayout{}
}

func (d *DocumentLayout) LayoutWithZoom(zoom float64) {
	d.Wrapper.Zoom = zoom
	child := NewLayoutNode(NewBlockLayout(nil), d.Wrapper.Node, d.Wrapper)
	d.Wrapper.children = append(d.Wrapper.children, child)

	d.Wrapper.Width = WIDTH - 2*dpx(HSTEP, d.Wrapper.Zoom)
	d.Wrapper.X = dpx(HSTEP, d.Wrapper.Zoom)
	d.Wrapper.Y = dpx(VSTEP, d.Wrapper.Zoom)
	child.Layout.Layout()
	d.Wrapper.Height = child.Height
}

func (d *DocumentLayout) Layout() {
	fmt.Println("Normal layout should not be called on DocumentLayout")
}

func (d *DocumentLayout) String() string {
	return fmt.Sprintf("DocumentLayout(x=%f, y=%f, width=%f, height=%f)", d.Wrapper.X, d.Wrapper.Y, d.Wrapper.Width, d.Wrapper.Height)
}

func (d *DocumentLayout) Paint() []html.Command {
	return []html.Command{}
}

func (d *DocumentLayout) PaintEffects(cmds []html.Command) []html.Command {
	return cmds
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
	l.wrap.Zoom = l.wrap.parent.Zoom
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

func (l *BlockLayout) Paint() []html.Command {
	cmds := make([]html.Command, 0)

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
		rect := html.NewDrawRRect(l.self_rect(), actualRadius, bgcolor)
		cmds = append(cmds, rect)
	}
	return cmds
}

func (d *BlockLayout) PaintEffects(cmds []html.Command) []html.Command {
	cmds = paint_visual_effects(d.wrap.Node, cmds, d.self_rect())
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

func (l *BlockLayout) recurse(node *html.HtmlNode) {
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

func (l *BlockLayout) word(node *html.HtmlNode, word string) {
	weight := node.Style["font-weight"]
	style := node.Style["font-style"]
	if style == "normal" {
		style = "roman"
	}
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(node.Style["font-size"], "px"), 32)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	size := dpx(fSize*0.75, l.wrap.Zoom)

	font := fnt.GetFont(size, weight, style)
	width := fnt.Measure(font, word)
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
	l.cursor_x += width + fnt.Measure(font, " ")
}

func (l *BlockLayout) input(node *html.HtmlNode) {
	w := dpx(INPUT_WIDTH_PX, l.wrap.Zoom)
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
	size := dpx(fSize*0.75, l.wrap.Zoom)
	font := fnt.GetFont(size, weight, style)
	l.cursor_x += w + fnt.Measure(font, " ")
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

func (l *BlockLayout) self_rect() *rect.Rect {
	return rect.NewRect(l.wrap.X, l.wrap.Y, l.wrap.X+l.wrap.Width, l.wrap.Y+l.wrap.Height)
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
	l.wrap.Zoom = l.wrap.parent.Zoom
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
			maxAscent = max(maxAscent, fnt.Ascent(l.font))
		case *InputLayout:
			maxAscent = max(maxAscent, fnt.Ascent(l.font))
		}
	}

	baseline := l.wrap.Y + 1.25*maxAscent
	for _, item := range l.wrap.children {
		switch l := item.Layout.(type) {
		case *TextLayout:
			item.Y = baseline - fnt.Ascent(l.font)
		case *InputLayout:
			item.Y = baseline - fnt.Ascent(l.font)
		}
	}

	var maxDescent float64
	for _, item := range l.wrap.children {
		switch l := item.Layout.(type) {
		case *TextLayout:
			maxDescent = max(maxDescent, fnt.Descent(l.font))
		case *InputLayout:
			maxDescent = max(maxDescent, fnt.Descent(l.font))
		}
	}

	l.wrap.Height = 1.25 * (maxAscent + maxDescent)
}

func (l *LineLayout) String() string {
	return fmt.Sprintf("LineLayout(x=%f, y=%f, width=%f, height=%f, style=%v)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.wrap.Node.Style)
}

func (l *LineLayout) Paint() []html.Command {
	return []html.Command{}
}

func (d *LineLayout) PaintEffects(cmds []html.Command) []html.Command {
	return cmds
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
	l.wrap.Zoom = l.wrap.parent.Zoom
	weight := l.wrap.Node.Style["font-weight"]
	style := l.wrap.Node.Style["font-style"]
	if style == "normal" {
		style = "roman"
	}
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(l.wrap.Node.Style["font-size"], "px"), 32)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	size := dpx(fSize*0.75, l.wrap.Zoom)
	l.font = fnt.GetFont(size, weight, style)

	l.wrap.Width = fnt.Measure(l.font, l.word)

	if l.previous != nil {
		switch t := l.previous.Layout.(type) {
		case *TextLayout:
			space := fnt.Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		case *InputLayout:
			space := fnt.Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		default:
			l.wrap.X = l.wrap.parent.X
		}
	} else {
		l.wrap.X = l.wrap.parent.X
	}

	l.wrap.Height = fnt.Linespace(l.font)
}

func (l *TextLayout) String() string {
	return fmt.Sprintf("TextLayout(x=%f, y=%f, width=%f, height=%f, word='%s', style=%v)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.word, l.wrap.Node.Style)
}

func (l *TextLayout) Paint() []html.Command {
	color := l.wrap.Node.Style["color"]
	return []html.Command{html.NewDrawText(l.wrap.X, l.wrap.Y, l.word, l.font, color)}
}

func (d *TextLayout) PaintEffects(cmds []html.Command) []html.Command {
	return cmds
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
	l.wrap.Zoom = l.wrap.parent.Zoom
	weight := l.wrap.Node.Style["font-weight"]
	style := l.wrap.Node.Style["font-style"]
	if style == "normal" {
		style = "roman"
	}
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(l.wrap.Node.Style["font-size"], "px"), 32)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	size := dpx(fSize*0.75, l.wrap.Zoom)
	l.font = fnt.GetFont(size, weight, style)

	l.wrap.Width = INPUT_WIDTH_PX

	if l.previous != nil {
		switch t := l.previous.Layout.(type) {
		case *TextLayout:
			space := fnt.Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		case *InputLayout:
			space := fnt.Measure(t.font, " ")
			l.wrap.X = l.previous.X + space + l.previous.Width
		default:
			l.wrap.X = l.wrap.parent.X
		}
	} else {
		l.wrap.X = l.wrap.parent.X
	}

	l.wrap.Height = fnt.Linespace(l.font)
}

func (l *InputLayout) String() string {
	return fmt.Sprintf("InputLayout(x=%f, y=%f, width=%f, height=%f, style=%v)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.wrap.Node.Style)
}

func (l *InputLayout) Paint() []html.Command {
	cmds := []html.Command{}
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
		rect := html.NewDrawRRect(l.self_rect(), actualRadius, bgcolor)
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
	cmds = append(cmds, html.NewDrawText(l.wrap.X, l.wrap.Y, text, l.font, color))

	if l.wrap.Node.Token.(html.ElementToken).IsFocused {
		cx := l.wrap.X + fnt.Measure(l.font, text)
		cmds = append(cmds, html.NewDrawLine(cx, l.wrap.Y, cx, l.wrap.Y+l.wrap.Height, "black", 1))
	}

	return cmds
}

func (d *InputLayout) PaintEffects(cmds []html.Command) []html.Command {
	cmds = paint_visual_effects(d.wrap.Node, cmds, d.self_rect())
	return cmds
}

func (d *InputLayout) ShouldPaint() bool {
	return true
}

func (d *InputLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

func (l *InputLayout) self_rect() *rect.Rect {
	return rect.NewRect(l.wrap.X, l.wrap.Y, l.wrap.X+l.wrap.Width, l.wrap.Y+l.wrap.Height)
}

func PaintTree(l *LayoutNode, displayList *[]html.Command) {
	var cmds []html.Command
	if l.Layout.ShouldPaint() {
		cmds = l.Layout.Paint()
	}
	for _, child := range l.children {
		PaintTree(child, &cmds)
	}

	if l.Layout.ShouldPaint() {
		cmds = l.Layout.PaintEffects(cmds)
	}
	*displayList = append(*displayList, cmds...)
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

func paint_visual_effects(node *html.HtmlNode, cmds []html.Command, rect *rect.Rect) []html.Command {
	opacity := 1.0
	if val, ok := node.Style["opacity"]; ok {
		fval, err := strconv.ParseFloat(val, 32)
		if err == nil {
			opacity = fval
		}
	}
	var blend_mode string
	if val, ok := node.Style["mix-blend-mode"]; ok {
		blend_mode = val
	}

	overflow := "visible"
	if val, ok := node.Style["overflow"]; ok {
		overflow = val
	}

	var dx, dy float64
	if val, ok := node.Style["transform"]; ok {
		dx, dy = css.ParseTransform(val)
	}

	if overflow == "clip" {
		border_radius := "0px"
		if val, ok := node.Style["border-radius"]; ok {
			border_radius = val
		}
		if blend_mode == "" {
			blend_mode = "source-over"
		}
		fVal, err := strconv.ParseFloat(strings.TrimSuffix(border_radius, "px"), 32)
		if err == nil {
			cmds = append(cmds, html.NewDrawBlend(1.0, "destination-in", node, []html.Command{html.NewDrawRRect(rect, fVal, "white")}))
		}
	}

	blend_op := html.NewDrawBlend(opacity, blend_mode, node, cmds)
	node.BlendOp = blend_op
	return []html.Command{html.NewTransform(dx, dy, rect, node, []html.Command{blend_op})}
}

// css pixel -> device pixel
func dpx(css_px, zoom float64) float64 {
	return css_px * zoom
}
