package layout

import (
	"fmt"
	"gowser/display"
	"image"
	"image/color"
	"image/draw"

	"strings"

	"github.com/anthonynsimon/bild/adjust"
	"github.com/anthonynsimon/bild/blend"
	"github.com/anthonynsimon/bild/fcolor"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type Command interface {
	Execute(*gg.Context)
	Rect() Rect
	Top() float64
	Bottom() float64
	String() string
}

type DrawText struct {
	rect  *Rect
	text  string
	font  font.Face
	color string
}

func NewDrawText(x1, y1 float64, text string, font font.Face, color string) *DrawText {
	return &DrawText{
		rect:  NewRect(x1, y1, x1+Measure(font, text), y1+Ascent(font)+Descent(font)),
		text:  text,
		font:  font,
		color: color,
	}
}

func (d *DrawText) Execute(canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.SetFontFace(d.font)
	canvas.DrawStringAnchored(d.text, d.rect.Left, d.rect.Top, 0, 1)
}

func (d *DrawText) Rect() Rect {
	return *d.rect
}

func (d *DrawText) Top() float64 {
	return d.rect.Top
}

func (d *DrawText) Bottom() float64 {
	return d.rect.Bottom
}

func (d *DrawText) String() string {
	return fmt.Sprint("DrawText(rect=", d.rect, ", text='", d.text, "', color='", d.color, "')")
}

type DrawRRect struct {
	rect   *Rect
	radius float64
	color  string
}

func NewDrawRRect(rect *Rect, radius float64, color string) *DrawRRect {
	return &DrawRRect{
		rect:   rect,
		radius: radius,
		color:  color,
	}
}

func (d *DrawRRect) Execute(canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.DrawRoundedRectangle(d.rect.Left, d.rect.Top, d.rect.Right-d.rect.Left, d.rect.Bottom-d.rect.Top, d.radius)
	canvas.Fill()
}

func (d *DrawRRect) Rect() Rect {
	return *d.rect
}

func (d *DrawRRect) Top() float64 {
	return d.rect.Top
}

func (d *DrawRRect) Bottom() float64 {
	return d.rect.Bottom
}

func (d *DrawRRect) String() string {
	return fmt.Sprint("DrawRRect(rect=", d.rect, ", radius=", d.radius, ", color='", d.color, "')")
}

type DrawOutline struct {
	rect      *Rect
	color     string
	thickness float64
}

func NewDrawOutline(rect *Rect, color string, thickness float64) *DrawOutline {
	return &DrawOutline{
		rect:      rect,
		color:     color,
		thickness: thickness,
	}
}

func (d *DrawOutline) Execute(canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.DrawRectangle(d.rect.Left, d.rect.Top, d.rect.Right-d.rect.Left, d.rect.Bottom-d.rect.Top)
	canvas.SetLineWidth(d.thickness)
	canvas.Stroke()
}

func (d *DrawOutline) Rect() Rect {
	return *d.rect
}

func (d *DrawOutline) Top() float64 {
	return d.rect.Top
}

func (d *DrawOutline) Bottom() float64 {
	return d.rect.Bottom
}

func (d *DrawOutline) String() string {
	return fmt.Sprint("DrawOutline(rect=", d.rect, ", color='", d.color, "', thickness=", d.thickness, ")")
}

type DrawLine struct {
	rect      *Rect
	color     string
	thickness float64
}

func NewDrawLine(x1, y1, x2, y2 float64, color string, thickness float64) *DrawLine {
	return &DrawLine{
		rect:      NewRect(x1, y1, x2, y2),
		color:     color,
		thickness: thickness,
	}
}

func (d *DrawLine) Execute(canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.SetLineWidth(d.thickness)
	canvas.DrawLine(d.rect.Left, d.rect.Top, d.rect.Right, d.rect.Bottom)
	canvas.Stroke()
}

func (d *DrawLine) Rect() Rect {
	return *d.rect
}

func (d *DrawLine) Top() float64 {
	return d.rect.Top
}

func (d *DrawLine) Bottom() float64 {
	return d.rect.Bottom
}

func (d *DrawLine) String() string {
	return fmt.Sprint("DrawLine(rect=", d.rect, ", color='", d.color, "', thickness=", d.thickness, ")")
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

type DrawBlend struct {
	opacity     float64
	blend_mode  string
	should_save bool
	children    []Command
	rect        *Rect
}

func NewDrawBlend(opacity float64, blend_mode string, children []Command) *DrawBlend {
	var rect Rect
	for _, child := range children {
		rect = rect.Union(child.Rect())
	}
	return &DrawBlend{
		opacity:     opacity,
		blend_mode:  blend_mode,
		should_save: blend_mode != "" || opacity < 1.0,
		children:    children,
		rect:        &rect,
	}
}

func (d *DrawBlend) Execute(canvas *gg.Context) {
	if !d.should_save {
		for _, cmd := range d.children {
			cmd.Execute(canvas)
		}
		return
	}

	// Create a new context for the layer
	layerContext := gg.NewContext(canvas.Width(), canvas.Height())

	// Execute each child command on the layer context
	for _, cmd := range d.children {
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
	switch d.blend_mode {
	case "difference":
		blended = blend.Difference(dst, src)
	case "multiply":
		blended = blend.Multiply(dst, src)
	case "destination-in":
		// DestinationIn:  Show destination only where source exists
		blended = destinationInBlend(dst, src)
	default: // source-over
		// SourceOver: Show source over destination
		blended = src
	}

	// Draw the image with opacity onto the main canvas
	rect := image.Rect(int(d.rect.Left), int(d.rect.Top), int(d.rect.Right), int(d.rect.Bottom))
	draw.Draw(dst, rect, blended, image.Point{X: int(d.rect.Left), Y: int(d.rect.Top)}, draw.Over)
	// canvas.DrawImage(blended, 0, 0)
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

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (d *DrawBlend) Rect() Rect {
	return *d.rect
}

func (d *DrawBlend) Top() float64 {
	return d.rect.Top
}

func (d *DrawBlend) Bottom() float64 {
	return d.rect.Bottom
}

func (d *DrawBlend) String() string {
	return fmt.Sprint("DrawBlend(rect=", d.rect, ", blend_mode='", d.blend_mode, "', opacity=", d.opacity, ", shoud_save=", d.should_save, ")")
}

func PrintCommands(list []Command, indent int) {
	for _, cmd := range list {
		fmt.Println(strings.Repeat(" ", indent) + cmd.String())
		if bl, ok := cmd.(*DrawBlend); ok {
			PrintCommands(bl.children, indent+2)
		}
	}
}
