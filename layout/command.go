package layout

import (
	"fmt"
	"gowser/display"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type Command interface {
	Execute(float64, *gg.Context)
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

func (d *DrawText) Execute(scroll float64, canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.SetFontFace(d.font)
	canvas.DrawStringAnchored(d.text, d.rect.Left, d.rect.Top-scroll, 0, 1)
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
	rect  *Rect
	radius float64
	color string
}

func NewDrawRRect(rect *Rect, radius float64, color string) *DrawRRect {
	return &DrawRRect{
		rect:  rect,
		radius: radius,
		color: color,
	}
}

func (d *DrawRRect) Execute(scroll float64, canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.DrawRoundedRectangle(d.rect.Left, d.rect.Top-scroll, d.rect.Right-d.rect.Left, d.rect.Bottom-d.rect.Top, d.radius)
	canvas.Fill()
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

func PrintCommands(list []Command) {
	for _, cmd := range list {
		fmt.Println("Command:", cmd)
	}
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

func (d *DrawOutline) Execute(scroll float64, canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.DrawRectangle(d.rect.Left, d.rect.Top-scroll, d.rect.Right-d.rect.Left, d.rect.Bottom-d.rect.Top)
	canvas.SetLineWidth(d.thickness)
	canvas.Stroke()
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

func (d *DrawLine) Execute(scroll float64, canvas *gg.Context) {
	canvas.SetColor(display.ParseColor(d.color))
	canvas.SetLineWidth(d.thickness)
	canvas.DrawLine(d.rect.Left, d.rect.Top-scroll, d.rect.Right, d.rect.Bottom-scroll)
	canvas.Stroke()
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
