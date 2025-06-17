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
	d.Wrapper.Children = append(d.Wrapper.Children, child)

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
	l.wrap.Zoom = l.wrap.Parent.Zoom
	if l.previous != nil {
		l.wrap.Y = l.previous.Y + l.previous.Height
	} else {
		l.wrap.Y = l.wrap.Parent.Y
	}
	l.wrap.X = l.wrap.Parent.X
	l.wrap.Width = l.wrap.Parent.Width

	mode := l.layout_mode()
	if mode == "block" {
		var previous *LayoutNode
		for _, child := range l.wrap.Node.Children {
			next := NewLayoutNode(NewBlockLayout(previous), child, l.wrap)
			l.wrap.Children = append(l.wrap.Children, next)
			previous = next
		}
	} else {
		l.new_line()
		l.recurse(l.wrap.Node)
	}

	for _, child := range l.wrap.Children {
		child.Layout.Layout()
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
	return cmds
}

func (d *BlockLayout) PaintEffects(cmds []html.Command) []html.Command {
	cmds = paint_visual_effects(d.wrap.Node, cmds, d.wrap.self_rect())
	return cmds
}

func (d *BlockLayout) ShouldPaint() bool {
	if _, ok := d.wrap.Node.Token.(html.TextToken); ok || !slices.Contains([]string{"input", "button", "img"}, d.wrap.Node.Token.(html.ElementToken).Tag) {
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
		if len(l.wrap.Node.Children) > 0 || slices.Contains([]string{"input", "img"}, l.wrap.Node.Token.(html.ElementToken).Tag) {
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
	l.add_inline_child(node, w, "text", word)
}

func (l *BlockLayout) input(node *html.HtmlNode) {
	w := dpx(INPUT_WIDTH_PX, l.wrap.Zoom)
	l.add_inline_child(node, w, "input", "")
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
	l.add_inline_child(node, w, "image", "")
}

func (l *BlockLayout) add_inline_child(node *html.HtmlNode, w float64, child_class, word string) {
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
		child = NewLayoutNode(NewTextLayout(word, previous_word), node, line)
	} else if child_class == "input" {
		child = NewLayoutNode(NewInputLayout(previous_word), node, line)
	} else if child_class == "image" {
		child = NewLayoutNode(NewImageLayout(previous_word), node, line)
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
	new_line := NewLayoutNode(NewLineLayout(last_line), l.wrap.Node, l.wrap)
	l.wrap.Children = append(l.wrap.Children, new_line)
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
	l.wrap.Zoom = l.wrap.Parent.Zoom
	l.wrap.Width = l.wrap.Parent.Width
	l.wrap.X = l.wrap.Parent.X

	if l.previous != nil {
		l.wrap.Y = l.previous.Y + l.previous.Height
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
	word     string
	wrap     *LayoutNode
	previous *LayoutNode
}

func NewTextLayout(word string, previous *LayoutNode) *TextLayout {
	return &TextLayout{
		word:     word,
		previous: previous,
	}
}

func (l *TextLayout) Layout() {
	l.wrap.Zoom = l.wrap.Parent.Zoom
	l.wrap.Font = get_font(l.wrap.Node.Style, l.wrap.Zoom)

	l.wrap.Width = fnt.Measure(l.wrap.Font, l.word)

	if l.previous != nil {
		space := fnt.Measure(l.previous.Font, " ")
		l.wrap.X = l.previous.X + space + l.previous.Width
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
	wrap     *LayoutNode
	previous *LayoutNode
}

func NewEmbedLayout(previous *LayoutNode) *EmbedLayout {
	return &EmbedLayout{
		previous: previous,
	}
}

func (l *EmbedLayout) Layout() {
	l.wrap.Zoom = l.wrap.Parent.Zoom
	l.wrap.Font = get_font(l.wrap.Node.Style, l.wrap.Zoom)

	if l.previous != nil {
		space := fnt.Measure(l.previous.Font, " ")
		l.wrap.X = l.previous.X + space + l.previous.Width
	} else {
		l.wrap.X = l.wrap.Parent.X
	}
}

type InputLayout struct {
	EmbedLayout
}

func NewInputLayout(previous *LayoutNode) *InputLayout {
	return &InputLayout{
		EmbedLayout: *NewEmbedLayout(previous),
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
		cx := l.wrap.X + fnt.Measure(l.wrap.Font, text)
		cmds = append(cmds, html.NewDrawLine(cx, l.wrap.Y, cx, l.wrap.Y+l.wrap.Height, "black", 1))
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

func NewImageLayout(previous *LayoutNode) *ImageLayout {
	return &ImageLayout{
		EmbedLayout: *NewEmbedLayout(previous),
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

func PaintTree(l *LayoutNode, displayList *[]html.Command) {
	var cmds []html.Command
	if l.Layout.ShouldPaint() {
		cmds = l.Layout.Paint()
	}
	for _, child := range l.Children {
		PaintTree(child, &cmds)
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

func TreeToList(tree *LayoutNode) []*LayoutNode {
	list := []*LayoutNode{tree}
	for _, child := range tree.Children {
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
