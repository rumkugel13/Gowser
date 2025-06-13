package html

import (
	"fmt"
	col "gowser/color"
	"image"
	"image/color"
	"slices"

	// "image/draw"
	fnt "gowser/font"
	"gowser/rect"
	"strings"

	"github.com/anthonynsimon/bild/adjust"
	"github.com/anthonynsimon/bild/blend"
	"github.com/anthonynsimon/bild/fcolor"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type Command interface {
	Execute(*gg.Context)
	Rect() *rect.Rect
	String() string
	Children() *[]Command
	SetParent(Command)
	GetParent() Command
}

type PaintCommand struct {
	rect     *rect.Rect
	children []Command
	parent   Command
}

func (p *PaintCommand) Rect() *rect.Rect {
	return p.rect
}

func (p *PaintCommand) Children() *[]Command {
	return &p.children
}

func (p *PaintCommand) SetParent(parent Command) {
	p.parent = parent
}

func (p *PaintCommand) GetParent() Command {
	return p.parent
}

type DrawText struct {
	PaintCommand
	text  string
	font  font.Face
	color string
}

func NewDrawText(x1, y1 float64, text string, font font.Face, color string) *DrawText {
	rect := rect.NewRect(x1, y1, x1+fnt.Measure(font, text), y1+fnt.Ascent(font)+fnt.Descent(font))
	return &DrawText{
		PaintCommand: PaintCommand{rect: rect},
		text:         text,
		font:         font,
		color:        color,
	}
}

func (d *DrawText) Execute(canvas *gg.Context) {
	canvas.SetColor(col.ParseColor(d.color))
	canvas.SetFontFace(d.font)
	canvas.DrawStringAnchored(d.text, d.PaintCommand.rect.Left, d.PaintCommand.rect.Top, 0, 1)
}

func (d *DrawText) String() string {
	return fmt.Sprint("DrawText(rect=", d.PaintCommand.rect, ", text='", d.text, "', color='", d.color, "')")
}

type DrawRRect struct {
	PaintCommand
	radius float64
	color  string
}

func NewDrawRRect(rect *rect.Rect, radius float64, color string) *DrawRRect {
	return &DrawRRect{
		PaintCommand: PaintCommand{rect: rect},
		radius:       radius,
		color:        color,
	}
}

func (d *DrawRRect) Execute(canvas *gg.Context) {
	canvas.SetColor(col.ParseColor(d.color))
	canvas.DrawRoundedRectangle(d.PaintCommand.rect.Left, d.PaintCommand.rect.Top, d.PaintCommand.rect.Right-d.PaintCommand.rect.Left, d.PaintCommand.rect.Bottom-d.PaintCommand.rect.Top, d.radius)
	canvas.Fill()
}

func (d *DrawRRect) String() string {
	return fmt.Sprint("DrawRRect(rect=", d.PaintCommand.rect, ", radius=", d.radius, ", color='", d.color, "')")
}

type DrawOutline struct {
	PaintCommand
	color     string
	thickness float64
}

func NewDrawOutline(rect *rect.Rect, color string, thickness float64) *DrawOutline {
	return &DrawOutline{
		PaintCommand: PaintCommand{rect: rect},
		color:        color,
		thickness:    thickness,
	}
}

func (d *DrawOutline) Execute(canvas *gg.Context) {
	canvas.SetColor(col.ParseColor(d.color))
	canvas.DrawRectangle(d.PaintCommand.rect.Left, d.PaintCommand.rect.Top, d.PaintCommand.rect.Right-d.PaintCommand.rect.Left, d.PaintCommand.rect.Bottom-d.PaintCommand.rect.Top)
	canvas.SetLineWidth(d.thickness)
	canvas.Stroke()
}

func (d *DrawOutline) String() string {
	return fmt.Sprint("DrawOutline(rect=", d.PaintCommand.rect, ", color='", d.color, "', thickness=", d.thickness, ")")
}

type DrawLine struct {
	PaintCommand
	color     string
	thickness float64
}

func NewDrawLine(x1, y1, x2, y2 float64, color string, thickness float64) *DrawLine {
	return &DrawLine{
		PaintCommand: PaintCommand{rect: rect.NewRect(x1, y1, x2, y2)},
		color:        color,
		thickness:    thickness,
	}
}

func (d *DrawLine) Execute(canvas *gg.Context) {
	canvas.SetColor(col.ParseColor(d.color))
	canvas.SetLineWidth(d.thickness)
	canvas.DrawLine(d.PaintCommand.rect.Left, d.PaintCommand.rect.Top, d.PaintCommand.rect.Right, d.PaintCommand.rect.Bottom)
	canvas.Stroke()
}

func (d *DrawLine) String() string {
	return fmt.Sprint("DrawLine(rect=", d.PaintCommand.rect, ", color='", d.color, "', thickness=", d.thickness, ")")
}

type DrawCompositedLayer struct {
	PaintCommand
	composited_layer *CompositedLayer
}

func NewDrawCompositedLayer(composited_layer *CompositedLayer) *DrawCompositedLayer {
	return &DrawCompositedLayer{
		composited_layer: composited_layer,
		PaintCommand:     PaintCommand{rect: composited_layer.CompositedBounds()},
	}
}

func (d *DrawCompositedLayer) Execute(canvas *gg.Context) {
	layer := d.composited_layer
	bounds := layer.CompositedBounds()

	canvas.DrawImage(layer.surface.Image(), int(bounds.Left), int(bounds.Top))
}

func (d *DrawCompositedLayer) String() string {
	return fmt.Sprint("DrawCompositedLayer(rect=", d.PaintCommand.rect, ", composited_layer bounds=", d.composited_layer.CompositedBounds(), ")")
}

type BlendMode uint

const (
	BlendModeSourceOver BlendMode = iota
	BlendModeDifference
	BlendModeMultiply
	BlendModeDestinationIn
)

func parse_blend_mode(blend_mode string) BlendMode {
	if blend_mode == "multiply" {
		return BlendModeMultiply
	} else if blend_mode == "difference" {
		return BlendModeDifference
	} else if blend_mode == "destination-in" {
		return BlendModeDestinationIn
	} else if blend_mode == "source-over" {
		return BlendModeSourceOver
	} else {
		return BlendModeSourceOver
	}
}

type VisualEffectCommand interface {
	Command
	GetNode() *HtmlNode
	Clone(child Command) VisualEffectCommand // Returns a new instance with updated children
	NeedsCompositing() bool
	Map(*rect.Rect) *rect.Rect
	Unmap(*rect.Rect) *rect.Rect
}

type VisualEffect struct {
	rect              *rect.Rect
	children          []Command
	parent            Command
	Node              *HtmlNode
	needs_compositing bool
}

func NewVisualEffect(rect *rect.Rect, node *HtmlNode, children []Command) *VisualEffect {
	for _, child := range children {
		rect = rect.Union(child.Rect())
	}
	needs_compositing := slices.ContainsFunc(children, func(child Command) bool {
		if v, ok := child.(*VisualEffect); ok {
			return v.needs_compositing
		}
		return false
	})
	return &VisualEffect{
		rect:              rect,
		children:          children,
		needs_compositing: needs_compositing,
		Node:              node,
	}
}

func (p *VisualEffect) Children() *[]Command {
	return &p.children
}

func (p *VisualEffect) Rect() *rect.Rect {
	return p.rect
}

func (p *VisualEffect) SetParent(parent Command) {
	p.parent = parent
}

func (p *VisualEffect) GetParent() Command {
	return p.parent
}

func (d *VisualEffect) Execute(canvas *gg.Context) {
}

func (d *VisualEffect) String() string {
	return "VisualEffect()"
}

func (d *VisualEffect) NeedsCompositing() bool {
	return d.needs_compositing
}

type DrawBlend struct {
	VisualEffect
	opacity     float64
	blend_mode  string
	should_save bool
}

func (d *DrawBlend) Children() *[]Command {
	return d.VisualEffect.Children()
}

func (d *DrawBlend) GetParent() Command {
	return d.VisualEffect.GetParent()
}

func (d *DrawBlend) Rect() *rect.Rect {
	return d.VisualEffect.Rect()
}

func (d *DrawBlend) SetParent(command Command) {
	d.VisualEffect.SetParent(command)
}

func NewDrawBlend(opacity float64, blend_mode string, node *HtmlNode, children []Command) *DrawBlend {
	r := rect.NewRectEmpty()
	for _, child := range children {
		r = r.Union(child.Rect())
	}
	blend := &DrawBlend{
		VisualEffect: *NewVisualEffect(rect.NewRectEmpty(), node, children),
		opacity:      opacity,
		blend_mode:   blend_mode,
		should_save:  blend_mode != "" || opacity < 1.0,
	}
	if blend.should_save {
		blend.needs_compositing = true
	}
	return blend
}

func (d *DrawBlend) GetNode() *HtmlNode {
	return d.VisualEffect.Node
}

func (d *DrawBlend) Clone(child Command) VisualEffectCommand {
	return &DrawBlend{
		VisualEffect: *NewVisualEffect(rect.NewRectEmpty(), d.Node, []Command{child}),
		opacity:      d.opacity,
		blend_mode:   d.blend_mode,
		should_save:  d.should_save,
	}
}

func (d *DrawBlend) Execute(canvas *gg.Context) {
	if !d.should_save {
		for _, cmd := range d.VisualEffect.children {
			cmd.Execute(canvas)
		}
		return
	}

	rect := d.rect.RoundOutToInt()
	// Create a new context for the layer
	layerContext := gg.NewContext(rect.Dx(), rect.Dy())
	layerContext.SetColor(color.Transparent)
	layerContext.Clear()
	layerContext.Push()
	layerContext.Translate(-d.rect.Left, -d.rect.Top)

	// Execute each child command on the layer context
	for _, cmd := range d.VisualEffect.children {
		cmd.Execute(layerContext)
	}
	layerContext.Pop()

	// Get the image from the layer context
	src := layerContext.Image().(*image.RGBA)

	// Apply opacity to the source image BEFORE blending
	if d.opacity < 1.0 {
		src = adjust.Apply(src, func(r color.RGBA) color.RGBA {
			r.A = uint8(float64(r.A) * d.opacity)
			return r
		})
	}

	// Get the destination image from the canvas
	dst := canvas.Image().(*image.RGBA).SubImage(rect)

	var blended image.Image

	// Perform blending based on the blend mode
	switch parse_blend_mode(d.blend_mode) {
	case BlendModeDifference:
		blended = blend.Difference(dst, src)
	case BlendModeMultiply:
		blended = blend.Multiply(dst, src)
	case BlendModeDestinationIn:
		// DestinationIn:  Show destination only where source exists
		blended = destinationInBlend(dst, src)
	default: // source-over
		// SourceOver: Show source over destination
		blended = src
	}

	// Draw the image with opacity onto the main canvas
	// rect := image.Rect(int(d.VisualEffect.rect.Left), int(d.VisualEffect.rect.Top), int(d.VisualEffect.rect.Right), int(d.VisualEffect.rect.Bottom))
	// draw.Draw(dst.(*image.RGBA), dst.Bounds(), blended, image.ZP, draw.Src)
	canvas.DrawImage(blended, rect.Min.X, rect.Min.Y)
}

func destinationInBlend(dst image.Image, src image.Image) *image.RGBA {
	// Define the custom blend function for DestinationIn
	destinationInFunc := func(bg fcolor.RGBAF64, fg fcolor.RGBAF64) fcolor.RGBAF64 {
		// If the source (foreground) pixel is opaque, return the destination (background) pixel;
		// otherwise, return transparent.
		if fg.A > 0 {
			return bg
		}
		return fcolor.RGBAF64{R: 0, G: 0, B: 0, A: 0}
	}

	// Use the blend.Blend function with our custom blend function
	return blend.Blend(dst, src, destinationInFunc)
}

func (d *DrawBlend) String() string {
	return fmt.Sprint("DrawBlend(rect=", d.rect, ", blend_mode='", d.blend_mode, "', opacity=", d.opacity, ", shoud_save=", d.should_save, ")")
}

func (d *DrawBlend) Map(rct *rect.Rect) *rect.Rect {
	if len(d.children) > 0 {
		if b, ok := d.children[len(d.children)-1].(*DrawBlend); ok && b.blend_mode == "destination-in" {
			bounds := rct.Clone()
			bounds.Intersect(b.rect)
			return bounds
		}
		return rct
	}
	return rct
}

func (d *DrawBlend) Unmap(rct *rect.Rect) *rect.Rect {
	return rct
}

type Transform struct {
	VisualEffect
	dx, dy    float64
	self_rect *rect.Rect
}

func (t *Transform) Children() *[]Command {
	return t.VisualEffect.Children()
}

func (t *Transform) GetParent() Command {
	return t.VisualEffect.GetParent()
}

func (t *Transform) Rect() *rect.Rect {
	return t.VisualEffect.Rect()
}

func (t *Transform) SetParent(command Command) {
	t.VisualEffect.SetParent(command)
}

func NewTransform(dx, dy float64, rct *rect.Rect, node *HtmlNode, children []Command) *Transform {
	for _, child := range children {
		rct = rct.Union(child.Rect())
	}
	return &Transform{
		dx:           dx,
		dy:           dy,
		self_rect:    rct,
		VisualEffect: *NewVisualEffect(rect.NewRectEmpty(), node, children),
	}
}

func (t *Transform) GetNode() *HtmlNode {
	return t.VisualEffect.Node
}

func (t *Transform) Clone(child Command) VisualEffectCommand {
	return &Transform{
		VisualEffect: *NewVisualEffect(rect.NewRectEmpty(), t.Node, []Command{child}),
		dx:           t.dx,
		dy:           t.dy,
		self_rect:    t.self_rect,
	}
}

func (t *Transform) Execute(canvas *gg.Context) {
	if t.dx != 0 || t.dy != 0 {
		canvas.Push()
		canvas.Translate(t.dx, t.dy)
	}
	for _, cmd := range t.children {
		cmd.Execute(canvas)
	}
	if t.dx != 0 || t.dy != 0 {
		canvas.Pop()
	}
}

func (t *Transform) String() string {
	return fmt.Sprintf("Transform(dx=%.2f, dy=%.2f, self_rect=%v)", t.dx, t.dy, t.self_rect)
}

func (t *Transform) Map(rct *rect.Rect) *rect.Rect {
	return MapTranslation(rct, t.dx, t.dy, false)
}

func (t *Transform) Unmap(rct *rect.Rect) *rect.Rect {
	return MapTranslation(rct, t.dx, t.dy, true)
}

func MapTranslation(rct *rect.Rect, dx, dy float64, reversed bool) *rect.Rect {
	if dx == 0 && dy == 0 {
		return rct
	} else {
		matrix := gg.Identity()
		if reversed {
			matrix = matrix.Translate(-dx, -dy)
		} else {
			matrix = matrix.Translate(dx, dy)
		}
		left, top := matrix.TransformPoint(rct.Left, rct.Top)
		right, bottom := matrix.TransformPoint(rct.Right, rct.Bottom)
		return rect.NewRect(left, top, right, bottom)
	}
}

func PrintCommands(list []Command, indent int) {
	for _, cmd := range list {
		fmt.Println(strings.Repeat(" ", indent) + cmd.String())
		PrintCommands(*cmd.Children(), indent+2)
	}
}

func CommandTreeToList(tree Command) []Command {
	list := []Command{tree}
	for _, child := range *tree.Children() {
		list = append(list, CommandTreeToList(child)...)
	}
	return list
}

func IsPaintCommand(cmd Command) bool {
	switch cmd.(type) {
	case *DrawLine, *DrawRRect, *DrawText, *DrawOutline:
		return true // These embed PaintCommand
	default:
		return false
	}
}
