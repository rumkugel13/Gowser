package layout

import (
	"fmt"
	"gowser/display"
	"image"
	"image/color"
	"strings"
	"sync"

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
		rect:  NewRect(x1, y1, x1+Measure(font, text), y1-Linespace(font)),
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
	return fmt.Sprint("DrawRect(rect=", d.rect, ", color='", d.color, "')")
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

type DrawOpacity struct {
	opacity  float64
	rect     *Rect
	children []Command
}

func NewOpacity(opacity float64, children []Command) *DrawOpacity {
	var rect Rect
	for _, child := range children {
		rect = rect.Union(child.Rect())
	}
	return &DrawOpacity{
		opacity:  opacity,
		rect:     &rect,
		children: children,
	}
}

func (d *DrawOpacity) Execute(canvas *gg.Context) {
	if d.opacity == 1.0 {
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
	layerImage := layerContext.Image().(*image.RGBA)
	bounds := layerImage.Bounds()
	imgWithOpacity := image.NewRGBA(bounds)

	var wg sync.WaitGroup
	rowChan := make(chan int, bounds.Dy())

	// Start worker goroutines
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rowChan {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					origColor := layerImage.RGBAAt(x, y)
					r, g, b, a := origColor.RGBA()
					imgWithOpacity.SetRGBA(x, y, color.RGBA{
						R: uint8(r >> 8),
						G: uint8(g >> 8),
						B: uint8(b >> 8),
						A: uint8(float64(a>>8) * d.opacity),
					})
				}
			}
		}()
	}

	// Send rows to the worker goroutines
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		rowChan <- y
	}
	close(rowChan)
	wg.Wait()

	// Draw the image with opacity onto the main canvas
	canvas.DrawImage(imgWithOpacity, 0, 0)
}

func (d *DrawOpacity) Rect() Rect {
	return *d.rect
}

func (d *DrawOpacity) Top() float64 {
	return d.rect.Top
}

func (d *DrawOpacity) Bottom() float64 {
	return d.rect.Bottom
}

func (d *DrawOpacity) String() string {
	return fmt.Sprint("DrawOpacity(rect=", d.rect, ", opacity='", d.opacity, "')")
}

func PrintCommands(list []Command, indent int) {
	for _, cmd := range list {
		fmt.Println(strings.Repeat(" ", indent) + cmd.String())
		if op, ok := cmd.(*DrawOpacity); ok {
			PrintCommands(op.children, indent+2)
		}
	}
}
