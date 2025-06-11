package layout

import (
	"fmt"
	"gowser/display"
	"image"
	"image/color"
	// "image/draw"
	"strings"
	"gowser/rect"

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
	rect := rect.NewRect(x1, y1, x1+Measure(font, text), y1+Ascent(font)+Descent(font))
	return &DrawText{
		PaintCommand: PaintCommand{rect: rect},
		text:         text,
		font:         font,
		color:        color,
	}
}

func (d *DrawText) Execute(canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
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
	canvas.SetColor(display.ParseColor(d.color))
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
	canvas.SetColor(display.ParseColor(d.color))
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
	canvas.SetColor(display.ParseColor(d.color))
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
		PaintCommand:     PaintCommand{rect: composited_layer.composited_bounds()},
	}
}

func (d *DrawCompositedLayer) Execute(canvas *gg.Context) {
	layer := d.composited_layer
	bounds := layer.composited_bounds()

	canvas.DrawImage(layer.surface.Image(), int(bounds.Left), int(bounds.Top))
}

func (d *DrawCompositedLayer) String() string {
	return "DrawCompositedLayer()"
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
	Clone(child Command) VisualEffectCommand // Returns a new instance with updated children
}

type VisualEffect struct {
	rect     *rect.Rect
	children []Command
	parent   Command
}

func NewVisualEffect(rect *rect.Rect, children []Command) *VisualEffect {
	for _, child := range children {
		rect = rect.Union(child.Rect())
	}
	return &VisualEffect{
		rect:     rect,
		children: children,
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

type DrawBlend struct {
	PaintCommand
	*VisualEffect
	opacity     float64
	blend_mode  string
	should_save bool
}

func (d *DrawBlend) Children() *[]Command {
	return d.PaintCommand.Children()
}

func (d *DrawBlend) GetParent() Command {
	return d.PaintCommand.GetParent()
}

func (d *DrawBlend) Rect() *rect.Rect {
	return d.PaintCommand.Rect()
}

func (d *DrawBlend) SetParent(command Command) {
	d.PaintCommand.SetParent(command)
}

func NewDrawBlend(opacity float64, blend_mode string, children []Command) *DrawBlend {
	r := &rect.Rect{}
	for _, child := range children {
		r = r.Union(child.Rect())
	}
	return &DrawBlend{
		PaintCommand: PaintCommand{rect: r, children: children},
		VisualEffect: NewVisualEffect(&rect.Rect{}, children),
		opacity:      opacity,
		blend_mode:   blend_mode,
		should_save:  blend_mode != "" || opacity < 1.0,
	}
}

func (d *DrawBlend) Clone(child Command) *DrawBlend {
	return &DrawBlend{
		PaintCommand: d.PaintCommand,
		VisualEffect: NewVisualEffect(&rect.Rect{}, []Command{child}),
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

	// Create a new context for the layer
	layerContext := gg.NewContext(canvas.Width(), canvas.Height())

	// Execute each child command on the layer context
	for _, cmd := range d.VisualEffect.children {
		cmd.Execute(layerContext)
	}

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
	dst := canvas.Image().(*image.RGBA)

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
	// draw.Draw(dst, rect, blended, image.Point{X: int(d.VisualEffect.rect.Left), Y: int(d.VisualEffect.rect.Top)}, draw.Over)
	canvas.DrawImage(blended, 0, 0)
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
	return fmt.Sprint("DrawBlend(rect=", d.PaintCommand.rect, ", blend_mode='", d.blend_mode, "', opacity=", d.opacity, ", shoud_save=", d.should_save, ")")
}

func PrintCommands(list []Command, indent int) {
	for _, cmd := range list {
		fmt.Println(strings.Repeat(" ", indent) + cmd.String())
		if bl, ok := cmd.(*DrawBlend); ok {
			PrintCommands(*bl.Children(), indent+2)
		}
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
