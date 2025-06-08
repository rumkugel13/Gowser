package layout

import (
	"fmt"
	"gowser/display"
	"image"
	"image/color"
	"strings"

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
		rect:  NewRect(x1, y1, x1+Measure(font, text), y1-Ascent(font)+Descent(font)),
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
	return fmt.Sprint("DrawText(rect=", d.rect, ", text='", d.text, "')")
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
	bounds := src.Bounds()
	dst := canvas.Image().(*image.RGBA)

	result := image.NewRGBA(bounds)
	for y := int(d.rect.Top); y < int(d.rect.Bottom); y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			srcColor := src.RGBAAt(x, y)
			destColor := dst.RGBAAt(x, y)

			if parse_blend_mode(d.blend_mode) == BlendModeDifference {
				// Difference blending
				r := uint8(abs(int(srcColor.R) - int(destColor.R)))
				g := uint8(abs(int(srcColor.G) - int(destColor.G)))
				b := uint8(abs(int(srcColor.B) - int(destColor.B)))
				a := uint8(float64(srcColor.A) * d.opacity)

				result.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
			} else if parse_blend_mode(d.blend_mode) == BlendModeMultiply {
				// Multiply blending
				r := uint8(float64(srcColor.R) / 255 * float64(destColor.R) / 255 * 255.0)
				g := uint8(float64(srcColor.G) / 255 * float64(destColor.G) / 255 * 255.0)
				b := uint8(float64(srcColor.B) / 255 * float64(destColor.B) / 255 * 255.0)
				a := uint8(float64(srcColor.A) * d.opacity)

				result.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
			} else if parse_blend_mode(d.blend_mode) == BlendModeDestinationIn {
				// Destination-In blending: Keep the destination where it overlaps with the source
				// The alpha of the result is the alpha of the destination multiplied by the alpha of the source
				sourceAlpha := float64(srcColor.A) / 255.0 * d.opacity
				destAlpha := float64(destColor.A) / 255.0

				alpha := uint8((sourceAlpha * destAlpha) * 255.0)
				if alpha > 0 {
					result.SetRGBA(x, y, color.RGBA{
						R: destColor.R,
						G: destColor.G,
						B: destColor.B,
						A: alpha,
					})
				} else {
					// If alpha is zero, set the pixel to fully transparent
					result.SetRGBA(x, y, color.RGBA{A: 0})
				}
			} else { // source-over and transparency
				result.SetRGBA(x, y, color.RGBA{
					R: srcColor.R,
					G: srcColor.G,
					B: srcColor.B,
					A: uint8(float64(srcColor.A) * d.opacity),
				})
			}
		}
	}

	// Draw the image with opacity onto the main canvas
	canvas.DrawImage(result, 0, 0)
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
	return fmt.Sprint("DrawBlend(rect=", d.rect, ", blend_mode='", d.blend_mode, "', opacity=", d.opacity, ")")
}

func PrintCommands(list []Command, indent int) {
	for _, cmd := range list {
		fmt.Println(strings.Repeat(" ", indent) + cmd.String())
		if bl, ok := cmd.(*DrawBlend); ok {
			PrintCommands(bl.children, indent+2)
		}
	}
}
