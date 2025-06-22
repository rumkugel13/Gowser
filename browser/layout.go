package browser

import (
	"fmt"
	fnt "gowser/font"
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
	Paint() []Command
	Wrap(*LayoutNode)
	ShouldPaint() bool
	PaintEffects([]Command) []Command
}

type DocumentLayout struct {
	wrap *LayoutNode
}

func NewDocumentLayout() *DocumentLayout {
	return &DocumentLayout{}
}

func (d *DocumentLayout) LayoutWithZoom(zoom float64) {
	if !d.wrap.layout_needed() {
		return
	}

	d.wrap.Zoom.Set(zoom)
	d.wrap.Width.Set(WIDTH - 2*dpx(HSTEP, zoom))

	var child *LayoutNode
	if d.wrap.Children.Dirty || len(d.wrap.Children.Get()) == 0 {
		child = NewLayoutNode(NewBlockLayout(), d.wrap.Node, d.wrap, nil, d.wrap.Frame)
		d.wrap.Height.SetDependencies([]ProtectedMarker{child.Height})
	} else {
		child = d.wrap.Children.Get()[0]
	}
	d.wrap.Children.Set([]*LayoutNode{child})

	d.wrap.X.Set(dpx(HSTEP, zoom))
	d.wrap.Y.Set(dpx(VSTEP, zoom))

	child.Layout.Layout()
	d.wrap.has_dirty_descendants = false

	d.wrap.Height.Copy(child.Height)
}

func (d *DocumentLayout) Layout() {
	panic("Layout should never be called on DocumentLayout")
}

func (d *DocumentLayout) String() string {
	return fmt.Sprintf("DocumentLayout(x=%f, y=%f, width=%f, height=%f)", d.wrap.X.Get(), d.wrap.Y.Get(), d.wrap.Width.Get(), d.wrap.Height.Get())
}

func (d *DocumentLayout) Paint() []Command {
	return []Command{}
}

func (d *DocumentLayout) PaintEffects(cmds []Command) []Command {
	if d.wrap.Frame != d.wrap.Frame.tab.root_frame && d.wrap.Frame.scroll != 0 {
		rect := rect.NewRect(d.wrap.X.Get(), d.wrap.Y.Get(),
			d.wrap.X.Get()+d.wrap.Width.Get(), d.wrap.Y.Get()+d.wrap.Height.Get())
		cmds = []Command{NewTransform(0, -d.wrap.Frame.scroll, rect, d.wrap.Node, cmds)}
	}
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
	temp_children      []*LayoutNode
	previous_word      *LayoutNode
}

func NewBlockLayout() *BlockLayout {
	layout := &BlockLayout{
		cursor_x: HSTEP,
		cursor_y: VSTEP,
	}
	return layout
}

func (l *BlockLayout) Layout() {
	if !l.wrap.layout_needed() {
		return
	}

	l.wrap.Zoom.Copy(l.wrap.Parent.Zoom)
	l.wrap.Width.Copy(l.wrap.Parent.Width)
	l.wrap.X.Copy(l.wrap.Parent.X)

	if l.wrap.Previous != nil {
		prev_y := l.wrap.Previous.Y.Read(l.wrap.Y)
		prev_height := l.wrap.Previous.Height.Read(l.wrap.Y)
		l.wrap.Y.Set(prev_y + prev_height)
	} else {
		l.wrap.Y.Copy(l.wrap.Parent.Y)
	}

	mode := l.layout_mode()
	if mode == "block" {
		if l.wrap.Children.Dirty {
			children := make([]*LayoutNode, 0)
			var previous *LayoutNode
			for _, child := range l.wrap.Node.Children {
				// Exercise 5-2: Hidden head (and also style and script)
				if element, ok := child.Token.(ElementToken); ok && slices.Contains([]string{"head", "style", "script"}, element.Tag) {
					continue
				}
				next := NewLayoutNode(NewBlockLayout(), child, l.wrap, previous, l.wrap.Frame)
				children = append(children, next)
				previous = next
			}
			l.wrap.Children.Set(children)

			height_dependencies := []ProtectedMarker{}
			for _, child := range children {
				height_dependencies = append(height_dependencies, child.Height)
			}
			height_dependencies = append(height_dependencies, l.wrap.Children)
			l.wrap.Height.SetDependencies(height_dependencies)
		}
	} else {
		if l.wrap.Children.Dirty {
			l.temp_children = make([]*LayoutNode, 0)
			l.new_line()
			l.recurse(l.wrap.Node)
			l.wrap.Children.Set(l.temp_children)

			height_dependencies := []ProtectedMarker{}
			for _, child := range l.temp_children {
				height_dependencies = append(height_dependencies, child.Height)
			}
			height_dependencies = append(height_dependencies, l.wrap.Children)
			l.wrap.Height.SetDependencies(height_dependencies)

			l.temp_children = nil
		}
	}

	for _, child := range l.wrap.Children.Get() {
		child.Layout.Layout()
	}

	l.wrap.has_dirty_descendants = false

	children := l.wrap.Children.Read(l.wrap.Height)
	var totalHeight float64
	for _, child := range children {
		totalHeight += child.Height.Read(l.wrap.Height)
	}
	l.wrap.Height.Set(totalHeight)
	l.wrap.has_dirty_descendants = false
}

func (l *BlockLayout) String() string {
	return fmt.Sprintf("BlockLayout(mode=%s, x=%f, y=%f, width=%f, height=%f, node=%v, style=%v)", l.layout_mode(),
		l.wrap.X.Get(), l.wrap.Y.Get(), l.wrap.Width.Get(), l.wrap.Height.Get(), l.wrap.Node.Token, l.wrap.Node.Style)
}

func (l *BlockLayout) Paint() []Command {
	cmds := make([]Command, 0)

	bgcolor := l.wrap.Node.Style["background-color"].Get()
	if bgcolor != "transparent" {
		radius := l.wrap.Node.Style["border-radius"].Get()
		actualRadius, err := strconv.ParseFloat(strings.TrimSuffix(radius, "px"), 32)
		if err != nil {
			actualRadius = 0 // Default radius size if parsing fails
		}
		rect := NewDrawRRect(l.wrap.self_rect(), actualRadius, bgcolor)
		cmds = append(cmds, rect)
	}
	return cmds
}

func NewDrawCursor(elt *LayoutNode, offset float64) *DrawLine {
	x := elt.X.Get() + offset
	return NewDrawLine(x, elt.Y.Get(), x, elt.Y.Get()+elt.Height.Get(), "red", 1)
}

func (l *BlockLayout) PaintEffects(cmds []Command) []Command {
	if _, ok := l.wrap.Node.Token.(ElementToken); ok && l.wrap.Node.Token.(ElementToken).IsFocused && l.wrap.Node.Token.(ElementToken).Attributes["contenteditable"] != "" {
		text_nodes := []*LayoutNode{}
		for _, t := range LayoutTreeToList(l.wrap) {
			if _, text := t.Node.Token.(TextToken); text {
				text_nodes = append(text_nodes, t)
			}
		}
		if len(text_nodes) > 0 {
			cmds = append(cmds, NewDrawCursor(text_nodes[len(text_nodes)-1], text_nodes[len(text_nodes)-1].Width.Get()))
		} else {
			cmds = append(cmds, NewDrawCursor(l.wrap, 0))
		}
	}
	cmds = paint_visual_effects(l.wrap.Node, cmds, l.wrap.self_rect())
	return cmds
}

func (d *BlockLayout) ShouldPaint() bool {
	if _, ok := d.wrap.Node.Token.(TextToken); ok || !slices.Contains([]string{"input", "button", "img", "iframe"}, d.wrap.Node.Token.(ElementToken).Tag) {
		return true
	}
	return false
}

func (d *BlockLayout) Wrap(wrap *LayoutNode) {
	d.wrap = wrap
}

func (l *BlockLayout) layout_mode() string {
	if _, ok := l.wrap.Node.Token.(TextToken); ok {
		return "inline"
	} else {
		for _, child := range l.wrap.Node.Children {
			if element, ok := child.Token.(ElementToken); ok && slices.Contains(BLOCK_ELEMENTS, element.Tag) {
				return "block"
			}
		}
		if len(l.wrap.Node.Children) > 0 || slices.Contains([]string{"input", "img", "iframe"}, l.wrap.Node.Token.(ElementToken).Tag) {
			return "inline"
		} else {
			return "block"
		}
	}
}

func (l *BlockLayout) recurse(node *HtmlNode) {
	if text, ok := node.Token.(TextToken); ok {
		words := strings.Fields(text.Text)
		for _, word := range words {
			l.word(node, word)
		}
	} else {
		element, _ := node.Token.(ElementToken)
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

func (l *BlockLayout) word(node *HtmlNode, word string) {
	zoom := l.wrap.Zoom.Read(l.wrap.Children)
	node_font := get_font(node.Style, zoom, l.wrap.Children)
	w := fnt.Measure(node_font, word)
	l.add_inline_child(node, w, "text", word, l.wrap.Frame)
}

func (l *BlockLayout) input(node *HtmlNode) {
	zoom := l.wrap.Zoom.Read(l.wrap.Children)
	w := dpx(INPUT_WIDTH_PX, zoom)
	l.add_inline_child(node, w, "input", "", l.wrap.Frame)
}

func (l *BlockLayout) image(node *HtmlNode) {
	zoom := l.wrap.Zoom.Read(l.wrap.Children)
	w := dpx(float64(node.Image.Bounds().Dx()), zoom)
	if val, ok := node.Token.(ElementToken).Attributes["width"]; ok {
		fVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			fVal = float64(node.Image.Bounds().Dx())
		}
		w = dpx(fVal, zoom)
	}
	l.add_inline_child(node, w, "image", "", l.wrap.Frame)
}

func (l *BlockLayout) iframe(node *HtmlNode) {
	zoom := l.wrap.Zoom.Read(l.wrap.Children)
	w := IFRAME_WIDTH_PX + dpx(2, zoom)
	if val, ok := node.Token.(ElementToken).Attributes["width"]; ok {
		fVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			fVal = float64(IFRAME_WIDTH_PX + 2)
		}
		w = dpx(fVal, zoom)
	}
	l.add_inline_child(node, w, "iframe", "", l.wrap.Frame)
}

func (l *BlockLayout) add_inline_child(node *HtmlNode, w float64, child_class, word string, frame *Frame) {
	width := l.wrap.Width.Read(l.wrap.Children)
	if l.cursor_x+w > width {
		l.new_line()
	}
	line := l.temp_children[len(l.temp_children)-1]
	var child *LayoutNode
	if child_class == "text" {
		child = NewLayoutNode(NewTextLayout(word), node, line, l.previous_word, frame)
	} else if child_class == "input" {
		child = NewLayoutNode(NewInputLayout(), node, line, l.previous_word, frame)
	} else if child_class == "image" {
		child = NewLayoutNode(NewImageLayout(), node, line, l.previous_word, frame)
	} else if child_class == "iframe" {
		child = NewLayoutNode(NewIframeLayout(), node, line, l.previous_word, frame)
	} else {
		panic("not implemented")
	}
	// warning: not using get
	line.Children.Set(append(line.Children.Value, child))
	l.previous_word = child
	zoom := l.wrap.Zoom.Read(l.wrap.Children)
	l.cursor_x += w + fnt.Measure(get_font(node.Style, zoom, l.wrap.Children), " ")
}

func (l *BlockLayout) new_line() {
	l.previous_word = nil
	l.cursor_x = 0
	var last_line *LayoutNode
	if len(l.temp_children) > 0 {
		last_line = l.temp_children[len(l.temp_children)-1]
	}
	new_line := NewLayoutNode(NewLineLayout(), l.wrap.Node, l.wrap, last_line, l.wrap.Frame)
	l.temp_children = append(l.temp_children, new_line)
}

type LineLayout struct {
	wrap               *LayoutNode
	initialized_fields bool
}

func NewLineLayout() *LineLayout {
	return &LineLayout{}
}

func (l *LineLayout) Layout() {
	if !l.initialized_fields {
		ascent_dependencies := []ProtectedMarker{}
		for _, child := range l.wrap.Children.Value {
			ascent_dependencies = append(ascent_dependencies, child.Ascent)
		}
		l.wrap.Ascent.SetDependencies(ascent_dependencies)

		descent_dependencies := []ProtectedMarker{}
		for _, child := range l.wrap.Children.Value {
			descent_dependencies = append(descent_dependencies, child.Descent)
		}
		l.wrap.Descent.SetDependencies(descent_dependencies)

		l.initialized_fields = true
	}

	if !l.wrap.layout_needed() {
		return
	}

	l.wrap.Zoom.Copy(l.wrap.Parent.Zoom)
	l.wrap.Width.Copy(l.wrap.Parent.Width)
	l.wrap.X.Copy(l.wrap.Parent.X)

	if l.wrap.Previous != nil {
		prev_y := l.wrap.Previous.Y.Read(l.wrap.Y)
		prev_height := l.wrap.Previous.Height.Read(l.wrap.Y)
		l.wrap.Y.Set(prev_y + prev_height)
	} else {
		l.wrap.Y.Copy(l.wrap.Parent.Y)
	}

	for _, word := range l.wrap.Children.Value {
		word.Layout.Layout()
	}

	if len(l.wrap.Children.Value) == 0 {
		l.wrap.Ascent.Set(0)
		l.wrap.Descent.Set(0)
		l.wrap.Height.Set(0)
		l.wrap.has_dirty_descendants = false
		return
	}

	var maxAscent float64
	for _, item := range l.wrap.Children.Get() {
		maxAscent = max(maxAscent, item.Ascent.Read(l.wrap.Ascent))
	}
	l.wrap.Ascent.Set(maxAscent)

	var maxDescent float64
	for _, item := range l.wrap.Children.Get() {
		maxDescent = max(maxDescent, item.Descent.Read(l.wrap.Descent))
	}
	l.wrap.Descent.Set(maxDescent)

	for _, child := range l.wrap.Children.Get() {
		new_y := l.wrap.Y.Read(child.Y)
		new_y -= l.wrap.Ascent.Read(child.Y)
		// note: need negative since ascent is actually positive, but we want to be on same baseline
		switch child.Layout.(type) {
		case *TextLayout:
			new_y += child.Ascent.Read(child.Y) / 1.25
		default:
			new_y += child.Ascent.Read(child.Y)
		}
		child.Y.Set(new_y)
	}

	max_ascent := l.wrap.Ascent.Read(l.wrap.Height)
	max_descent := l.wrap.Descent.Read(l.wrap.Height)
	l.wrap.Height.Set(max_ascent + max_descent)

	l.wrap.has_dirty_descendants = false
}

func (l *LineLayout) String() string {
	return fmt.Sprintf("LineLayout(x=%f, y=%f, width=%f, height=%f, style=%v)", l.wrap.X.Get(), l.wrap.Y.Get(), l.wrap.Width.Get(), l.wrap.Height.Get(), l.wrap.Node.Style)
}

func (l *LineLayout) Paint() []Command {
	return []Command{}
}

func (l *LineLayout) PaintEffects(cmds []Command) []Command {
	outline_rect := rect.NewRectEmpty()
	var outline_node *HtmlNode
	for _, child := range l.wrap.Children.Get() {
		var outline_str string
		if child.Node.Parent != nil {
			outline_str = child.Node.Parent.Style["outline"].Get()
		}
		thickness, color := ParseOutline(outline_str)
		if thickness != 0 && color != "" {
			outline_rect = outline_rect.Union(child.self_rect())
			outline_node = child.Node.Parent
		}
	}
	if outline_node != nil {
		paint_outline(outline_node, &cmds, outline_rect, l.wrap.Zoom.Get())
	}
	return cmds
}

func (l *LineLayout) ShouldPaint() bool {
	return true
}

func (l *LineLayout) Wrap(wrap *LayoutNode) {
	l.wrap = wrap
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
	if !l.wrap.layout_needed() {
		return
	}

	l.wrap.Zoom.Copy(l.wrap.Parent.Zoom)
	zoom := l.wrap.Zoom.Read(l.wrap.Font)
	l.wrap.Font.Set(get_font(l.wrap.Node.Style, zoom, l.wrap.Font))

	f := l.wrap.Font.Read(l.wrap.Width)
	l.wrap.Width.Set(fnt.Measure(f, l.word))

	f = l.wrap.Font.Read(l.wrap.Ascent)
	l.wrap.Ascent.Set(fnt.Ascent(f) * 1.25)

	f = l.wrap.Font.Read(l.wrap.Descent)
	l.wrap.Descent.Set(fnt.Descent(f) * 1.25)

	f = l.wrap.Font.Read(l.wrap.Height)
	l.wrap.Height.Set(fnt.Linespace(f) * 1.25)

	if l.wrap.Previous != nil {
		prev_x := l.wrap.Previous.X.Read(l.wrap.X)
		prev_font := l.wrap.Previous.Font.Read(l.wrap.X)
		prev_width := l.wrap.Previous.Width.Read(l.wrap.X)
		l.wrap.X.Set(prev_x + fnt.Measure(prev_font, " ") + prev_width)
	} else {
		l.wrap.X.Copy(l.wrap.Parent.X)
	}

	l.wrap.has_dirty_descendants = false
}

func (l *TextLayout) String() string {
	return fmt.Sprintf("TextLayout(x=%f, y=%f, width=%f, height=%f, word='%s', style=%v)", l.wrap.X.Get(), l.wrap.Y.Get(), l.wrap.Width.Get(), l.wrap.Height.Get(), l.word, l.wrap.Node.Style)
}

func (l *TextLayout) Paint() []Command {
	leading := l.wrap.Height.Get() / 1.25 * .25 / 2

	color := l.wrap.Node.Style["color"].Get()
	return []Command{NewDrawText(l.wrap.X.Get(), l.wrap.Y.Get()+leading, l.word, l.wrap.Font.Get(), color)}
}

func (d *TextLayout) PaintEffects(cmds []Command) []Command {
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
	l.wrap.Zoom.Copy(l.wrap.Parent.Zoom)
	zoom := l.wrap.Zoom.Read(l.wrap.Font)
	l.wrap.Font.Set(get_font(l.wrap.Node.Style, zoom, l.wrap.Font))

	if l.wrap.Previous != nil {
		prev_x := l.wrap.Previous.X.Read(l.wrap.X)
		prev_font := l.wrap.Previous.Font.Read(l.wrap.X)
		prev_width := l.wrap.Previous.Width.Read(l.wrap.X)
		l.wrap.X.Set(prev_x + fnt.Measure(prev_font, " ") + prev_width)
	} else {
		l.wrap.X.Copy(l.wrap.Parent.X)
	}

	l.wrap.has_dirty_descendants = false
}

func (l *EmbedLayout) ShouldPaint() bool {
	return true
}

func (l *EmbedLayout) Paint() []Command {
	return []Command{}
}

func (l *EmbedLayout) PaintEffects(cmds []Command) []Command {
	return cmds
}

func (l *EmbedLayout) String() string {
	return "EmbedLayout()"
}

func (l *EmbedLayout) Wrap(node *LayoutNode) {
	l.wrap = node
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
	if !l.wrap.layout_needed() {
		return
	}

	l.EmbedLayout.Layout()

	zoom := l.wrap.Zoom.Read(l.wrap.Width)
	l.wrap.Width.Set(dpx(INPUT_WIDTH_PX, zoom))

	font := l.wrap.Font.Read(l.wrap.Height)
	l.wrap.Height.Set(fnt.Linespace(font))

	height := l.wrap.Height.Read(l.wrap.Ascent)
	l.wrap.Ascent.Set(height)
	l.wrap.Descent.Set(0)
}

func (l *InputLayout) String() string {
	return fmt.Sprintf("InputLayout(x=%f, y=%f, width=%f, height=%f, style=%v)", l.wrap.X.Get(), l.wrap.Y.Get(), l.wrap.Width.Get(), l.wrap.Height.Get(), l.wrap.Node.Style)
}

func (l *InputLayout) Paint() []Command {
	cmds := []Command{}
	bgcolor := l.wrap.Node.Style["background-color"].Get()
	if bgcolor != "transparent" {
		radius := l.wrap.Node.Style["border-radius"].Get()
		actualRadius, err := strconv.ParseFloat(strings.TrimSuffix(radius, "px"), 32)
		if err != nil {
			actualRadius = 0 // Default radius size if parsing fails
		}
		rect := NewDrawRRect(l.wrap.self_rect(), actualRadius, bgcolor)
		cmds = append(cmds, rect)
	}

	var text string
	if l.wrap.Node.Token.(ElementToken).Tag == "input" {
		text = l.wrap.Node.Token.(ElementToken).Attributes["value"]
	} else if l.wrap.Node.Token.(ElementToken).Tag == "button" {
		if len(l.wrap.Node.Children) == 1 {
			if txt, ok := l.wrap.Node.Children[0].Token.(TextToken); ok {
				text = txt.Text
			} else {
				fmt.Println("Ignoring HTML contents inside button")
			}
		} else {
			fmt.Println("Ignoring HTML contents inside button")
		}
	}

	color := l.wrap.Node.Style["color"].Get()
	cmds = append(cmds, NewDrawText(l.wrap.X.Get(), l.wrap.Y.Get(), text, l.wrap.Font.Get(), color))

	if l.wrap.Node.Token.(ElementToken).IsFocused && l.wrap.Node.Token.(ElementToken).Tag == "input" {
		cmds = append(cmds, NewDrawCursor(l.wrap, fnt.Measure(l.wrap.Font.Get(), text)))
	}

	return cmds
}

func (l *InputLayout) PaintEffects(cmds []Command) []Command {
	cmds = paint_visual_effects(l.wrap.Node, cmds, l.wrap.self_rect())
	paint_outline(l.wrap.Node, &cmds, l.wrap.self_rect(), l.wrap.Zoom.Get())
	return cmds
}

func (l *InputLayout) ShouldPaint() bool {
	return true
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
	if !l.wrap.layout_needed() {
		return
	}

	l.EmbedLayout.Layout()

	width_attr := l.wrap.Node.Token.(ElementToken).Attributes["width"]
	height_attr := l.wrap.Node.Token.(ElementToken).Attributes["height"]
	image_width := l.wrap.Node.Image.Bounds().Dx()
	image_height := l.wrap.Node.Image.Bounds().Dy()
	aspect_ratio := float64(image_width) / float64(image_height)

	w_zoom := l.wrap.Zoom.Read(l.wrap.Width)
	h_zoom := l.wrap.Zoom.Read(l.wrap.Height)
	if width_attr != "" && height_attr != "" {
		fValW, err := strconv.ParseFloat(width_attr, 64)
		if err != nil {
			fValW = float64(image_width)
		}
		fValH, err := strconv.ParseFloat(height_attr, 64)
		if err != nil {
			fValH = float64(image_height)
		}
		l.wrap.Width.Set(dpx(fValW, w_zoom))
		l.img_height = dpx(fValH, h_zoom)
	} else if width_attr != "" {
		fValW, err := strconv.ParseFloat(width_attr, 64)
		if err != nil {
			fValW = float64(image_width)
		}
		l.wrap.Width.Set(dpx(fValW, w_zoom))
		w := l.wrap.Width.Read(l.wrap.Height)
		l.img_height = w / aspect_ratio
	} else if height_attr != "" {
		fValH, err := strconv.ParseFloat(height_attr, 64)
		if err != nil {
			fValH = float64(image_height)
		}
		l.img_height = dpx(fValH, h_zoom)
		l.wrap.Width.Set(l.img_height * aspect_ratio)
	} else {
		l.wrap.Width.Set(dpx(float64(image_width), w_zoom))
		l.img_height = dpx(float64(image_height), h_zoom)
	}

	font := l.wrap.Font.Read(l.wrap.Height)
	l.wrap.Height.Set(max(l.img_height, fnt.Linespace(font)))

	height := l.wrap.Height.Read(l.wrap.Ascent)
	l.wrap.Ascent.Set(height)
	l.wrap.Descent.Set(0)
}

func (l *ImageLayout) String() string {
	return fmt.Sprintf("ImageLayout(x=%f, y=%f, width=%f, height=%f, img_height=%f, style=%v)", l.wrap.X.Get(), l.wrap.Y.Get(), l.wrap.Width.Get(), l.wrap.Height.Get(), l.img_height, l.wrap.Node.Style)
}

func (l *ImageLayout) Paint() []Command {
	cmds := []Command{}
	rect := rect.NewRect(l.wrap.X.Get(), l.wrap.Y.Get()+l.wrap.Height.Get()-l.img_height,
		l.wrap.X.Get()+l.wrap.Width.Get(), l.wrap.Y.Get()+l.wrap.Height.Get())
	quality := l.wrap.Node.Style["image-rendering"].Get()
	if quality == "" {
		quality = "auto"
	}
	cmds = append(cmds, NewDrawImage(l.wrap.Node.Image, rect, quality))
	return cmds
}

func (l *ImageLayout) ShouldPaint() bool {
	return true
}

type IframeLayout struct {
	EmbedLayout
	// parent_frame *HtmlNode
}

func NewIframeLayout() *IframeLayout {
	return &IframeLayout{
		EmbedLayout: *NewEmbedLayout(),
	}
}

func (l *IframeLayout) Layout() {
	if !l.wrap.layout_needed() {
		return
	}

	l.EmbedLayout.Layout()

	width_attr := l.wrap.Node.Token.(ElementToken).Attributes["width"]
	height_attr := l.wrap.Node.Token.(ElementToken).Attributes["height"]

	w_zoom := l.wrap.Zoom.Read(l.wrap.Width)
	if width_attr != "" {
		fValW, err := strconv.ParseFloat(width_attr, 64)
		if err != nil {
			fValW = float64(IFRAME_WIDTH_PX)
		}
		l.wrap.Width.Set(dpx(fValW+2, w_zoom))
	} else {
		l.wrap.Width.Set(dpx(IFRAME_WIDTH_PX+2, w_zoom))
	}

	h_zoom := l.wrap.Zoom.Read(l.wrap.Height)
	if height_attr != "" {
		fValH, err := strconv.ParseFloat(height_attr, 64)
		if err != nil {
			fValH = float64(IFRAME_HEIGHT_PX)
		}
		l.wrap.Height.Set(dpx(fValH+2, h_zoom))
	} else {
		l.wrap.Height.Set(dpx(IFRAME_HEIGHT_PX+2, h_zoom))
	}

	if l.wrap.Node.Frame != nil && l.wrap.Node.Frame.Loaded {
		l.wrap.Node.Frame.frame_height = l.wrap.Height.Get() - dpx(2, l.wrap.Zoom.Get())
		l.wrap.Node.Frame.frame_width = l.wrap.Width.Get() - dpx(2, l.wrap.Zoom.Get())
		l.wrap.Node.Frame.Document.Width.Mark()
	}

	height := l.wrap.Height.Read(l.wrap.Ascent)
	l.wrap.Ascent.Set(height)
	l.wrap.Descent.Set(0)
}

func (l *IframeLayout) String() string {
	return fmt.Sprintf("IframeLayout(x=%f, y=%f, width=%f, height=%f, style=%v)", l.wrap.X.Get(), l.wrap.Y.Get(), l.wrap.Width.Get(), l.wrap.Height.Get(), l.wrap.Node.Style)
}

func (l *IframeLayout) Paint() []Command {
	cmds := []Command{}
	rect := rect.NewRect(l.wrap.X.Get(), l.wrap.Y.Get()+l.wrap.Height.Get(),
		l.wrap.X.Get()+l.wrap.Width.Get(), l.wrap.Y.Get()+l.wrap.Height.Get())

	bgcolor := l.wrap.Node.Style["background-color"].Get()
	if bgcolor != "transparent" {
		radius := l.wrap.Node.Style["border-radius"].Get()
		actualRadius, err := strconv.ParseFloat(strings.TrimSuffix(radius, "px"), 32)
		if err != nil {
			actualRadius = 0 // Default radius size if parsing fails
		}
		rect := NewDrawRRect(rect, dpx(actualRadius, l.wrap.Zoom.Get()), bgcolor)
		cmds = append(cmds, rect)
	}

	return cmds
}

func (l *IframeLayout) ShouldPaint() bool {
	return true
}

func (l *IframeLayout) PaintEffects(cmds []Command) []Command {
	rct := rect.NewRect(l.wrap.X.Get(), l.wrap.Y.Get()+l.wrap.Height.Get(),
		l.wrap.X.Get()+l.wrap.Width.Get(), l.wrap.Y.Get()+l.wrap.Height.Get())

	diff := dpx(1, l.wrap.Zoom.Get())
	offsetX, offsetY := l.wrap.X.Get()+diff, l.wrap.Y.Get()+diff
	cmds = []Command{NewTransform(offsetX, offsetY, rct, l.wrap.Node, cmds)}
	inner_rect := rect.NewRect(
		l.wrap.X.Get()+diff, l.wrap.Y.Get()+diff,
		l.wrap.X.Get()+l.wrap.Width.Get()-diff, l.wrap.Y.Get()+l.wrap.Height.Get()-diff,
	)
	internal_cmds := cmds
	internal_cmds = append(internal_cmds, NewDrawBlend(1.0, "destination-in", nil, []Command{NewDrawRRect(inner_rect, 0, "white")}))
	cmds = []Command{NewDrawBlend(1.0, "source-over", l.wrap.Node, internal_cmds)}
	paint_outline(l.wrap.Node, &cmds, rct, l.wrap.Zoom.Get())
	cmds = paint_visual_effects(l.wrap.Node, cmds, rct)
	return cmds
}

func PaintTree(l *LayoutNode, displayList *[]Command) {
	var cmds []Command
	if l.Layout.ShouldPaint() {
		cmds = l.Layout.Paint()
	}

	if iframe, ok := l.Layout.(*IframeLayout); ok && iframe.wrap.Node.Frame != nil && iframe.wrap.Node.Frame.Loaded {
		PaintTree(iframe.wrap.Node.Frame.Document, &cmds)
	} else {
		for _, child := range l.Children.Get() {
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
	for _, child := range l.Children.Get() {
		PrintTree(child, indent+2)
	}
}

func LayoutTreeToList(tree *LayoutNode) []*LayoutNode {
	list := []*LayoutNode{tree}
	for _, child := range tree.Children.Get() {
		list = append(list, LayoutTreeToList(child)...)
	}
	return list
}

func paint_visual_effects(node *HtmlNode, cmds []Command, rect *rect.Rect) []Command {
	opacity := 1.0
	if val := node.Style["opacity"].Get(); val != "" {
		fval, err := strconv.ParseFloat(val, 32)
		if err == nil {
			opacity = fval
		}
	}
	var blend_mode string
	if val := node.Style["mix-blend-mode"].Get(); val != "" {
		blend_mode = val
	}

	overflow := "visible"
	if val := node.Style["overflow"].Get(); val != "" {
		overflow = val
	}

	var dx, dy float64
	if val := node.Style["transform"].Get(); val != "" {
		dx, dy = ParseTransform(val)
	}

	if overflow == "clip" {
		border_radius := "0px"
		if val := node.Style["border-radius"].Get(); val != "" {
			border_radius = val
		}
		if blend_mode == "" {
			blend_mode = "source-over"
		}
		fVal, err := strconv.ParseFloat(strings.TrimSuffix(border_radius, "px"), 32)
		if err == nil {
			cmds = []Command{NewDrawBlend(1.0, "source-over", node,
				append(cmds, NewDrawBlend(1.0, "destination-in", nil,
					[]Command{NewDrawRRect(rect, fVal, "white")})))}
		}
	}

	blend_op := NewDrawBlend(opacity, blend_mode, node, cmds)
	node.BlendOp = blend_op
	return []Command{NewTransform(dx, dy, rect, node, []Command{blend_op})}
}

// css pixel -> device pixel
func dpx(css_px, zoom float64) float64 {
	return css_px * zoom
}

func paint_outline(node *HtmlNode, cmds *[]Command, rct *rect.Rect, zoom float64) {
	thickness, color := ParseOutline(node.Style["outline"].Get())
	if thickness == 0 || color == "" {
		return
	}
	*cmds = append(*cmds, NewDrawOutline(rct, color, dpx(float64(thickness), zoom)))
}

func get_font[T any](css_style map[string]*ProtectedField[string], zoom float64, notify *ProtectedField[T]) font.Face {
	family := css_style["font-family"].Read(notify)
	weight := css_style["font-weight"].Read(notify)
	style := css_style["font-style"].Read(notify)
	fSize, err := strconv.ParseFloat(strings.TrimSuffix(css_style["font-size"].Read(notify), "px"), 64)
	if err != nil {
		fSize = 16 // Default font size if parsing fails
	}
	font_size := dpx(fSize*0.75, zoom)
	return fnt.GetFont(family, font_size, weight, style)
}
