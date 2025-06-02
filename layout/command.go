package layout

import (
	"fmt"

	tk9_0 "modernc.org/tk9.0"
)

type Command interface {
	Execute(float32, tk9_0.CanvasWidget)
	Top() float32
	Bottom() float32
	String() string
}

type DrawText struct {
	rect  *Rect
	text  string
	font  *tk9_0.FontFace
	color string
}

func NewDrawText(x1, y1 float32, text string, font *tk9_0.FontFace, color string) *DrawText {
	return &DrawText{
		rect:  NewRect(x1, y1, Measure(font, text), y1+float32(font.MetricsLinespace(tk9_0.App))),
		text:  text,
		font:  font,
		color: color,
	}
}

func (d *DrawText) Execute(scroll float32, canvas tk9_0.CanvasWidget) {
	canvas.CreateText(d.rect.Left, d.rect.Top-scroll, tk9_0.Txt(d.text), tk9_0.Anchor("nw"), tk9_0.Font(d.font), tk9_0.Fill(d.color))
}

func (d *DrawText) Top() float32 {
	return d.rect.Top
}

func (d *DrawText) Bottom() float32 {
	return d.rect.Bottom
}

func (d *DrawText) String() string {
	return fmt.Sprint("DrawText(rect=", d.rect, ", text='", d.text, "', font=", d.font.String(), ")")
}

type DrawRect struct {
	rect  *Rect
	color string
}

func NewDrawRect(rect *Rect, color string) *DrawRect {
	return &DrawRect{
		rect:  rect,
		color: color,
	}
}

func (d *DrawRect) Execute(scroll float32, canvas tk9_0.CanvasWidget) {
	canvas.CreateRectangle(d.rect.Left, d.rect.Top-scroll, d.rect.Right, d.rect.Bottom-scroll, tk9_0.Width(0), tk9_0.Fill(d.color))
}

func (d *DrawRect) Top() float32 {
	return d.rect.Top
}

func (d *DrawRect) Bottom() float32 {
	return d.rect.Bottom
}

func (d *DrawRect) String() string {
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
	thickness int
}

func NewDrawOutline(rect *Rect, color string, thickness int) *DrawOutline {
	return &DrawOutline{
		rect:      rect,
		color:     color,
		thickness: thickness,
	}
}

func (d *DrawOutline) Execute(scroll float32, canvas tk9_0.CanvasWidget) {
	canvas.CreateRectangle(d.rect.Left, d.rect.Top-scroll, d.rect.Right, d.rect.Bottom-scroll, tk9_0.Width(d.thickness), tk9_0.Outline(d.color))
}

func (d *DrawOutline) Top() float32 {
	return d.rect.Top
}

func (d *DrawOutline) Bottom() float32 {
	return d.rect.Bottom
}

func (d *DrawOutline) String() string {
	return fmt.Sprint("DrawOutline(rect=", d.rect, ", color='", d.color, "', thickness=", d.thickness, ")")
}

type DrawLine struct {
	rect      *Rect
	color     string
	thickness int
}

func NewDrawLine(x1, y1, x2, y2 float32, color string, thickness int) *DrawLine {
	return &DrawLine{
		rect:      NewRect(x1, y1, x2, y2),
		color:     color,
		thickness: thickness,
	}
}

func (d *DrawLine) Execute(scroll float32, canvas tk9_0.CanvasWidget) {
	canvas.CreateLine(d.rect.Left, d.rect.Top-scroll, d.rect.Right, d.rect.Bottom-scroll, tk9_0.Fill(d.color), tk9_0.Width(d.thickness))
}

func (d *DrawLine) Top() float32 {
	return d.rect.Top
}

func (d *DrawLine) Bottom() float32 {
	return d.rect.Bottom
}

func (d *DrawLine) String() string {
	return fmt.Sprint("DrawLine(rect=", d.rect, ", color='", d.color, "', thickness=", d.thickness, ")")
}
