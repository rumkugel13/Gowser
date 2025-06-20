package browser

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
	HSTEP            = 13.
	VSTEP            = 18.
	INPUT_WIDTH_PX   = 200.
	IFRAME_WIDTH_PX  = 300.
	IFRAME_HEIGHT_PX = 150.
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
	wrap *LayoutNode
}

func NewDocumentLayout() *DocumentLayout {
	return &DocumentLayout{}
}

func (d *DocumentLayout) LayoutWithZoom(zoom float64) {
	d.wrap.Zoom = zoom

	var child *LayoutNode
	if len(d.wrap.Children) == 0 {
		child = NewLayoutNode(NewBlockLayout(), d.wrap.Node, d.wrap, nil, d.wrap.Frame)
	} else {
		child = d.wrap.Children[0]
	}
	d.wrap.Children = []*LayoutNode{child}

	d.wrap.Width = WIDTH - 2*dpx(HSTEP, d.wrap.Zoom)
	d.wrap.X = dpx(HSTEP, d.wrap.Zoom)
	d.wrap.Y = dpx(VSTEP, d.wrap.Zoom)
	child.Layout.Layout()
	d.wrap.Height = child.Height
}

func (d *DocumentLayout) Layout() {
	fmt.Println("Normal layout should not be called on DocumentLayout")
}

func (d *DocumentLayout) String() string {
	return fmt.Sprintf("DocumentLayout(x=%f, y=%f, width=%f, height=%f)", d.wrap.X, d.wrap.Y, d.wrap.Width, d.wrap.Height)
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
	d.wrap = wrap
}

type BlockLayout struct {
	cursor_x, cursor_y float64
	wrap               *LayoutNode
	children_dirty     bool
}

func NewBlockLayout() *BlockLayout {
	layout := &BlockLayout{
		cursor_x:       HSTEP,
		cursor_y:       VSTEP,
		children_dirty: true,
	}
	return layout
}

func (l *BlockLayout) Layout() {
	l.wrap.Zoom = l.wrap.Parent.Zoom
	if l.wrap.Previous != nil {
		l.wrap.Y = l.wrap.Previous.Y + l.wrap.Previous.Height
	} else {
		l.wrap.Y = l.wrap.Parent.Y
	}
	l.wrap.X = l.wrap.Parent.X
	l.wrap.Width = l.wrap.Parent.Width

	mode := l.layout_mode()
	if mode == "block" {
		if l.children_dirty {
			l.wrap.Children = make([]*LayoutNode, 0)
			var previous *LayoutNode
			for _, child := range l.wrap.Node.Children {
				next := NewLayoutNode(NewBlockLayout(), child, l.wrap, previous, l.wrap.Frame)
				l.wrap.Children = append(l.wrap.Children, next)
				previous = next
			}
			l.children_dirty = false
		}
	} else {
		if l.children_dirty {
			l.wrap.Children = make([]*LayoutNode, 0)
			l.new_line()
			l.recurse(l.wrap.Node)
			l.children_dirty = false
		}
	}

	if l.children_dirty {
		panic("children dirty")
	}
	for _, child := range l.wrap.Children {
		child.Layout.Layout()
	}

	if l.children_dirty {
		panic("children dirty")
	}
	var totalHeight float64
	for _, child := range l.wrap.Children {
		totalHeight += child.Height
	}
	l.wrap.Height = totalHeight
}

func (l *BlockLayout) String() string {
	return fmt.Sprintf("BlockLayout(mode=%s, x=%f, y=%f, width=%f, height=%f, node=%v, style=%v)", l.layout_mode(),
		l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.wrap.Node.Token, l.wrap.Node.Style)
}

func (l *BlockLayout) Paint() []html.Command {
	if l.children_dirty {
		panic("children dirty")
	}
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
		rect := html.NewDrawRRect(l.wrap.self_rect(), actualRadius, bgcolor)
		cmds = append(cmds, rect)
	}
	if _, ok := l.wrap.Node.Token.(html.ElementToken); ok && l.wrap.Node.Token.(html.ElementToken).IsFocused && l.wrap.Node.Token.(html.ElementToken).Attributes["contenteditable"] != "" {
		text_nodes := []*LayoutNode{}
		for _, t := range LayoutTreeToList(l.wrap) {
			if _, text := t.Node.Token.(html.TextToken); text {
				text_nodes = append(text_nodes, t)
			}
		}
		if len(text_nodes) > 0 {
			cmds = append(cmds, NewDrawCursor(text_nodes[len(text_nodes)-1], text_nodes[len(text_nodes)-1].Width))
		} else {
			cmds = append(cmds, NewDrawCursor(l.wrap, 0))
		}
	}
	return cmds
}

func NewDrawCursor(elt *LayoutNode, offset float64) *html.DrawLine {
	x := elt.X + offset
	return html.NewDrawLine(x, elt.Y, x, elt.Y+elt.Height, "red", 1)
}

func (l *BlockLayout) PaintEffects(cmds []html.Command) []html.Command {
	cmds = paint_visual_effects(l.wrap.Node, cmds, l.wrap.self_rect())
	return cmds
}

func (d *BlockLayout) ShouldPaint() bool {
	if _, ok := d.wrap.Node.Token.(html.TextToken); ok || !slices.Contains([]string{"input", "button", "img", "iframe"}, d.wrap.Node.Token.(html.ElementToken).Tag) {
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
		if len(l.wrap.Node.Children) > 0 || slices.Contains([]string{"input", "img", "iframe"}, l.wrap.Node.Token.(html.ElementToken).Tag) {
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
		element, _ := node.Token.(html.ElementToken)
		if element.Tag == "br" {
			l.new_line()
		} else if element.Tag == "input" || element.Tag == "button" {
			l.input(node)
		} else if element.Tag == "img" {
			l.image(node)
		} else if element.Tag == "iframe" && element.Attributes["src"] != "" {
			l.iframe(node)
		} else {
			for _, child := range node.Children {
				l.recurse(child)
			}
		}
	}
}

func (l *BlockLayout) word(node *html.HtmlNode, word string) {
	node_font := get_font(node.Style, l.wrap.Zoom)
	w := fnt.Measure(node_font, word)
	l.add_inline_child(node, w, "text", word, l.wrap.Frame)
}

func (l *BlockLayout) input(node *html.HtmlNode) {
	w := dpx(INPUT_WIDTH_PX, l.wrap.Zoom)
	l.add_inline_child(node, w, "input", "", l.wrap.Frame)
}

func (l *BlockLayout) image(node *html.HtmlNode) {
	w := dpx(float64(node.Image.Bounds().Dx()), l.wrap.Zoom)
	if val, ok := node.Token.(html.ElementToken).Attributes["width"]; ok {
		fVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			fVal = float64(node.Image.Bounds().Dx())
		}
		w = dpx(fVal, l.wrap.Zoom)
	}
	l.add_inline_child(node, w, "image", "", l.wrap.Frame)
}

func (l *BlockLayout) iframe(node *html.HtmlNode) {
	w := IFRAME_WIDTH_PX + dpx(2, l.wrap.Zoom)
	if val, ok := node.Token.(html.ElementToken).Attributes["width"]; ok {
		fVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			fVal = float64(IFRAME_WIDTH_PX + 2)
		}
		w = dpx(fVal, l.wrap.Zoom)
	}
	l.add_inline_child(node, w, "iframe", "", l.wrap.Frame)
}

func (l *BlockLayout) add_inline_child(node *html.HtmlNode, w float64, child_class, word string, frame *Frame) {
	if l.cursor_x+w > l.wrap.X+l.wrap.Width {
		l.new_line()
	}
	line := l.wrap.Children[len(l.wrap.Children)-1]
	var previous_word *LayoutNode
	if len(line.Children) > 0 {
		previous_word = line.Children[len(line.Children)-1]
	}
	var child *LayoutNode
	if child_class == "text" {
		child = NewLayoutNode(NewTextLayout(word), node, line, previous_word, frame)
	} else if child_class == "input" {
		child = NewLayoutNode(NewInputLayout(), node, line, previous_word, frame)
	} else if child_class == "image" {
		child = NewLayoutNode(NewImageLayout(), node, line, previous_word, frame)
	} else if child_class == "iframe" {
		child = NewLayoutNode(NewIframeLayout(), node, line, previous_word, frame)
	} else {
		panic("not implemented")
	}
	line.Children = append(line.Children, child)
	l.cursor_x += w + fnt.Measure(get_font(node.Style, l.wrap.Zoom), " ")
}

func (l *BlockLayout) new_line() {
	l.cursor_x = 0
	var last_line *LayoutNode
	if len(l.wrap.Children) > 0 {
		last_line = l.wrap.Children[len(l.wrap.Children)-1]
	}
	new_line := NewLayoutNode(NewLineLayout(), l.wrap.Node, l.wrap, last_line, l.wrap.Frame)
	l.wrap.Children = append(l.wrap.Children, new_line)
}

type LineLayout struct {
	wrap *LayoutNode
}

func NewLineLayout() *LineLayout {
	return &LineLayout{}
}

func (l *LineLayout) Layout() {
	l.wrap.Zoom = l.wrap.Parent.Zoom
	l.wrap.Width = l.wrap.Parent.Width
	l.wrap.X = l.wrap.Parent.X

	if l.wrap.Previous != nil {
		l.wrap.Y = l.wrap.Previous.Y + l.wrap.Previous.Height
	} else {
		l.wrap.Y = l.wrap.Parent.Y
	}

	for _, word := range l.wrap.Children {
		word.Layout.Layout()
	}

	var maxAscent float64
	for _, item := range l.wrap.Children {
		maxAscent = max(maxAscent, item.Ascent)
	}

	baseline := l.wrap.Y + maxAscent
	for _, item := range l.wrap.Children {
		switch item.Layout.(type) {
		case *TextLayout:
			item.Y = baseline - item.Ascent/1.25
		default:
			item.Y = baseline - item.Ascent
		}
	}

	var maxDescent float64
	for _, item := range l.wrap.Children {
		maxDescent = max(maxDescent, item.Descent)
	}

	l.wrap.Height = maxAscent + maxDescent
}

func (l *LineLayout) String() string {
	return fmt.Sprintf("LineLayout(x=%f, y=%f, width=%f, height=%f, style=%v)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.wrap.Node.Style)
}

func (l *LineLayout) Paint() []html.Command {
	return []html.Command{}
}

func (d *LineLayout) PaintEffects(cmds []html.Command) []html.Command {
	outline_rect := rect.NewRectEmpty()
	var outline_node *html.HtmlNode
	for _, child := range d.wrap.Children {
		var outline_str string
		if child.Node.Parent != nil {
			outline_str = child.Node.Parent.Style["outline"]
		}
		thickness, color := css.ParseOutline(outline_str)
		if thickness != 0 && color != "" {
			outline_rect = outline_rect.Union(child.self_rect())
			outline_node = child.Node.Parent
		}
	}
	if outline_node != nil {
		paint_outline(outline_node, &cmds, outline_rect, d.wrap.Zoom)
	}
	return cmds
}

func (d *LineLayout) ShouldPaint() bool {
	return true
}

func (d *LineLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

type TextLayout struct {
	word string
	wrap *LayoutNode
}

func NewTextLayout(word string) *TextLayout {
	return &TextLayout{
		word: word,
	}
}

func (l *TextLayout) Layout() {
	l.wrap.Zoom = l.wrap.Parent.Zoom
	l.wrap.Font = get_font(l.wrap.Node.Style, l.wrap.Zoom)

	l.wrap.Width = fnt.Measure(l.wrap.Font, l.word)

	if l.wrap.Previous != nil {
		space := fnt.Measure(l.wrap.Previous.Font, " ")
		l.wrap.X = l.wrap.Previous.X + space + l.wrap.Previous.Width
	} else {
		l.wrap.X = l.wrap.Parent.X
	}

	l.wrap.Height = fnt.Linespace(l.wrap.Font)
	l.wrap.Ascent = fnt.Ascent(l.wrap.Font) * 1.25
	l.wrap.Descent = fnt.Descent(l.wrap.Font) * 1.25
}

func (l *TextLayout) String() string {
	return fmt.Sprintf("TextLayout(x=%f, y=%f, width=%f, height=%f, word='%s', style=%v)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.word, l.wrap.Node.Style)
}

func (l *TextLayout) Paint() []html.Command {
	color := l.wrap.Node.Style["color"]
	return []html.Command{html.NewDrawText(l.wrap.X, l.wrap.Y, l.word, l.wrap.Font, color)}
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

type EmbedLayout struct {
	wrap *LayoutNode
}

func NewEmbedLayout() *EmbedLayout {
	return &EmbedLayout{}
}

func (l *EmbedLayout) Layout() {
	l.wrap.Zoom = l.wrap.Parent.Zoom
	l.wrap.Font = get_font(l.wrap.Node.Style, l.wrap.Zoom)

	if l.wrap.Previous != nil {
		space := fnt.Measure(l.wrap.Previous.Font, " ")
		l.wrap.X = l.wrap.Previous.X + space + l.wrap.Previous.Width
	} else {
		l.wrap.X = l.wrap.Parent.X
	}
}

func (l EmbedLayout) ShouldPaint() bool {
	return true
}

type InputLayout struct {
	EmbedLayout
}

func NewInputLayout() *InputLayout {
	return &InputLayout{
		EmbedLayout: *NewEmbedLayout(),
	}
}

func (l *InputLayout) Layout() {
	l.EmbedLayout.Layout()
	l.wrap.Width = dpx(INPUT_WIDTH_PX, l.wrap.Zoom)
	l.wrap.Height = fnt.Linespace(l.wrap.Font)
	l.wrap.Ascent = l.wrap.Height
	l.wrap.Descent = 0
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
	cmds = append(cmds, html.NewDrawText(l.wrap.X, l.wrap.Y, text, l.wrap.Font, color))

	if l.wrap.Node.Token.(html.ElementToken).IsFocused && l.wrap.Node.Token.(html.ElementToken).Tag == "input" {
		cmds = append(cmds, NewDrawCursor(l.wrap, fnt.Measure(l.wrap.Font, text)))
	}

	return cmds
}

func (d *InputLayout) PaintEffects(cmds []html.Command) []html.Command {
	cmds = paint_visual_effects(d.wrap.Node, cmds, d.self_rect())
	paint_outline(d.wrap.Node, &cmds, d.self_rect(), d.wrap.Zoom)
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

type ImageLayout struct {
	EmbedLayout
	img_height float64
}

func NewImageLayout() *ImageLayout {
	return &ImageLayout{
		EmbedLayout: *NewEmbedLayout(),
	}
}

func (l *ImageLayout) Layout() {
	l.EmbedLayout.Layout()

	width_attr := l.wrap.Node.Token.(html.ElementToken).Attributes["width"]
	height_attr := l.wrap.Node.Token.(html.ElementToken).Attributes["height"]
	image_width := l.wrap.Node.Image.Bounds().Dx()
	image_height := l.wrap.Node.Image.Bounds().Dy()
	aspect_ratio := float64(image_width) / float64(image_height)

	if width_attr != "" && height_attr != "" {
		fValW, err := strconv.ParseFloat(width_attr, 64)
		if err != nil {
			fValW = float64(image_width)
		}
		fValH, err := strconv.ParseFloat(height_attr, 64)
		if err != nil {
			fValH = float64(image_height)
		}
		l.wrap.Width = dpx(fValW, l.wrap.Zoom)
		l.img_height = dpx(fValH, l.wrap.Zoom)
	} else if width_attr != "" {
		fValW, err := strconv.ParseFloat(width_attr, 64)
		if err != nil {
			fValW = float64(image_width)
		}
		l.wrap.Width = dpx(fValW, l.wrap.Zoom)
		l.img_height = l.wrap.Width / aspect_ratio
	} else if height_attr != "" {
		fValH, err := strconv.ParseFloat(height_attr, 64)
		if err != nil {
			fValH = float64(image_height)
		}
		l.img_height = dpx(fValH, l.wrap.Zoom)
		l.wrap.Width = l.img_height * aspect_ratio
	} else {
		l.wrap.Width = dpx(float64(image_width), l.wrap.Zoom)
		l.img_height = dpx(float64(image_height), l.wrap.Zoom)
	}
	l.wrap.Height = max(l.img_height, fnt.Linespace(l.wrap.Font))
	l.wrap.Ascent = l.wrap.Height
	l.wrap.Descent = 0
}

func (l *ImageLayout) String() string {
	return fmt.Sprintf("ImageLayout(x=%f, y=%f, width=%f, height=%f, img_height=%f, style=%v)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.img_height, l.wrap.Node.Style)
}

func (l *ImageLayout) Paint() []html.Command {
	cmds := []html.Command{}
	rect := rect.NewRect(l.wrap.X, l.wrap.Y+l.wrap.Height-l.img_height,
		l.wrap.X+l.wrap.Width, l.wrap.Y+l.wrap.Height)
	quality := l.wrap.Node.Style["image-rendering"]
	if quality == "" {
		quality = "auto"
	}
	cmds = append(cmds, html.NewDrawImage(l.wrap.Node.Image, rect, quality))
	return cmds
}

func (l *ImageLayout) Wrap(wrap *LayoutNode) {
	l.wrap = wrap
}

func (l *ImageLayout) ShouldPaint() bool {
	return true
}

func (l *ImageLayout) PaintEffects(cmds []html.Command) []html.Command {
	return cmds
}

type IframeLayout struct {
	EmbedLayout
	parent_frame *html.HtmlNode
}

func NewIframeLayout() *IframeLayout {
	return &IframeLayout{
		EmbedLayout: *NewEmbedLayout(),
	}
}

func (l *IframeLayout) Layout() {
	l.EmbedLayout.Layout()

	width_attr := l.wrap.Node.Token.(html.ElementToken).Attributes["width"]
	height_attr := l.wrap.Node.Token.(html.ElementToken).Attributes["height"]

	if width_attr != "" {
		fValW, err := strconv.ParseFloat(width_attr, 64)
		if err != nil {
			fValW = float64(IFRAME_WIDTH_PX)
		}
		l.wrap.Width = dpx(fValW+2, l.wrap.Zoom)
	} else {
		l.wrap.Width = dpx(IFRAME_WIDTH_PX+2, l.wrap.Zoom)
	}

	if height_attr != "" {
		fValH, err := strconv.ParseFloat(height_attr, 64)
		if err != nil {
			fValH = float64(IFRAME_HEIGHT_PX)
		}
		l.wrap.Height = dpx(fValH+2, l.wrap.Zoom)
	} else {
		l.wrap.Height = dpx(IFRAME_HEIGHT_PX+2, l.wrap.Zoom)
	}

	l.wrap.Ascent = l.wrap.Height
	l.wrap.Descent = 0

	if l.wrap.Node.Frame != nil && l.wrap.Node.Frame.(*Frame).Loaded {
		l.wrap.Node.Frame.(*Frame).frame_height = l.wrap.Height - dpx(2, l.wrap.Zoom)
		l.wrap.Node.Frame.(*Frame).frame_width = l.wrap.Width - dpx(2, l.wrap.Zoom)
	}
}

func (l *IframeLayout) String() string {
	return fmt.Sprintf("IframeLayout(x=%f, y=%f, width=%f, height=%f, style=%v)", l.wrap.X, l.wrap.Y, l.wrap.Width, l.wrap.Height, l.wrap.Node.Style)
}

func (l *IframeLayout) Paint() []html.Command {
	cmds := []html.Command{}
	rect := rect.NewRect(l.wrap.X, l.wrap.Y+l.wrap.Height,
		l.wrap.X+l.wrap.Width, l.wrap.Y+l.wrap.Height)

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
		rect := html.NewDrawRRect(rect, actualRadius, bgcolor)
		cmds = append(cmds, rect)
	}

	return cmds
}

func (l *IframeLayout) Wrap(wrap *LayoutNode) {
	l.wrap = wrap
}

func (l *IframeLayout) ShouldPaint() bool {
	return true
}

func (l *IframeLayout) PaintEffects(cmds []html.Command) []html.Command {
	rct := rect.NewRect(l.wrap.X, l.wrap.Y+l.wrap.Height,
		l.wrap.X+l.wrap.Width, l.wrap.Y+l.wrap.Height)

	diff := dpx(1, l.wrap.Zoom)
	offsetX, offsetY := l.wrap.X+diff, l.wrap.Y+diff
	cmds = []html.Command{html.NewTransform(offsetX, offsetY, rct, l.wrap.Node, cmds)}
	inner_rect := rect.NewRect(
		l.wrap.X+diff, l.wrap.Y+diff,
		l.wrap.X+l.wrap.Width-diff, l.wrap.Y+l.wrap.Height-diff,
	)
	internal_cmds := cmds
	internal_cmds = append(internal_cmds, html.NewDrawBlend(1.0, "destination-in", nil, []html.Command{html.NewDrawRRect(inner_rect, 0, "white")}))
	cmds = []html.Command{html.NewDrawBlend(1.0, "source-over", l.wrap.Node, internal_cmds)}
	paint_outline(l.wrap.Node, &cmds, rct, l.wrap.Zoom)
	cmds = paint_visual_effects(l.wrap.Node, cmds, rct)
	return cmds
}

func PaintTree(l *LayoutNode, displayList *[]html.Command) {
	var cmds []html.Command
	if l.Layout.ShouldPaint() {
		cmds = l.Layout.Paint()
	}

	if iframe, ok := l.Layout.(*IframeLayout); ok && iframe.wrap.Node.Frame != nil && iframe.wrap.Node.Frame.(*Frame).Loaded {
		PaintTree(iframe.wrap.Node.Frame.(*Frame).Document, &cmds)
	} else {
		for _, child := range l.Children {
			PaintTree(child, &cmds)
		}
	}

	if l.Layout.ShouldPaint() {
		cmds = l.Layout.PaintEffects(cmds)
	}
	*displayList = append(*displayList, cmds...)
}

func PrintTree(l *LayoutNode, indent int) {
	fmt.Println(strings.Repeat(" ", indent) + l.Layout.String())
	for _, child := range l.Children {
		PrintTree(child, indent+2)
	}
}

func LayoutTreeToList(tree *LayoutNode) []*LayoutNode {
	list := []*LayoutNode{tree}
	for _, child := range tree.Children {
		list = append(list, LayoutTreeToList(child)...)
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

func paint_outline(node *html.HtmlNode, cmds *[]html.Command, rct *rect.Rect, zoom float64) {
	thickness, color := css.ParseOutline(node.Style["outline"])
	if thickness == 0 || color == "" {
		return
	}
	*cmds = append(*cmds, html.NewDrawOutline(rct, color, dpx(float64(thickness), zoom)))
}

func get_font(style map[string]string, zoom float64) font.Face {
	weight := style["font-weight"]
	variant := style["font-style"]
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(style["font-size"], "px"), 64)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	font_size := dpx(fSize*0.75, zoom)
	return fnt.GetFont(font_size, weight, variant)
}
