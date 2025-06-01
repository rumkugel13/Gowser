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
	top, left, bottom float32
	text              string
	font              *tk9_0.FontFace
}

func NewDrawText(x1, y1 float32, text string, font *tk9_0.FontFace) *DrawText {
	return &DrawText{
		top:    y1,
		left:   x1,
		text:   text,
		font:   font,
		bottom: y1 + float32(font.MetricsLinespace(tk9_0.App)),
	}
}

func (d *DrawText) Execute(scroll float32, canvas tk9_0.CanvasWidget) {
	canvas.CreateText(d.left, d.top-scroll, tk9_0.Txt(d.text), tk9_0.Anchor("nw"), tk9_0.Font(d.font))
}

func (d *DrawText) Top() float32 {
	return d.top
}

func (d *DrawText) Bottom() float32 {
	return d.bottom
}

func (d *DrawText) String() string {
	return fmt.Sprint("DrawText(top=", d.top, ", left=", d.left, ", bottom=", d.bottom, ", text='", d.text, "', font=", d.font.String(), ")")
}

type DrawRect struct {
	top, left, bottom, right float32
	color                    string
}

func NewDrawRect(x1, y1, x2, y2 float32, color string) *DrawRect {
	return &DrawRect{
		top:    y1,
		left:   x1,
		bottom: y2,
		right:  x2,
		color:  color,
	}
}

func (d *DrawRect) Execute(scroll float32, canvas tk9_0.CanvasWidget) {
	canvas.CreateRectangle(d.left, d.top-scroll, d.right, d.bottom-scroll, tk9_0.Width(0), tk9_0.Fill(d.color))
}

func (d *DrawRect) Top() float32 {
	return d.top
}

func (d *DrawRect) Bottom() float32 {
	return d.bottom
}

func (d *DrawRect) String() string {
	return fmt.Sprint("DrawRect(top=", d.top, ", left=", d.left, ", bottom=", d.bottom, ", right=", d.right, ", color='", d.color, "')")
}
