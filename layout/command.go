package layout

import (
	"fmt"
	"gowser/display"

	"github.com/tdewolff/canvas"
)

type Command interface {
	Execute(*canvas.Context)
	Top() float64
	Bottom() float64
	String() string
}

type DrawText struct {
	rect  *Rect
	text  string
	font  *canvas.FontFace
	color string
}

func NewDrawText(x1, y1 float64, text string, font *canvas.FontFace, color string) *DrawText {
	return &DrawText{
		rect:  NewRect(x1, y1, x1+font.TextWidth(text), y1-font.LineHeight()),
		text:  text,
		font:  font,
		color: color,
	}
}

func (d *DrawText) Execute(ctx *canvas.Context) {
	ctx.SetFillColor(display.ParseColor(d.color))
	// text renders at baseline, not topleft
	ctx.DrawText(d.rect.Left, d.rect.Top+d.font.Metrics().Ascent, canvas.NewTextLine(d.font, d.text, canvas.Top))
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

func (d *DrawRRect) Execute(ctx *canvas.Context) {
	ctx.SetFillColor(display.ParseColor(d.color))
	ctx.DrawPath(d.rect.Left, d.rect.Right, canvas.RoundedRectangle(d.rect.Right-d.rect.Left, d.rect.Bottom-d.rect.Top, d.radius))
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

func (d *DrawOutline) Execute(ctx *canvas.Context) {
	ctx.SetStrokeWidth(d.thickness)
	ctx.SetStrokeColor(display.ParseColor(d.color))
	ctx.MoveTo(d.rect.Left, d.rect.Top)
	ctx.LineTo(d.rect.Right, d.rect.Top)
	ctx.LineTo(d.rect.Right, d.rect.Bottom)
	ctx.LineTo(d.rect.Left, d.rect.Bottom)
	ctx.Close()
	ctx.Stroke()
	// ctx.DrawPath(d.rect.Left, d.rect.Top, canvas.Rectangle(d.rect.Right-d.rect.Left, d.rect.Bottom-d.rect.Top))
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

func (d *DrawLine) Execute(ctx *canvas.Context) {
	ctx.SetStrokeColor(display.ParseColor(d.color))
	ctx.SetStrokeWidth(d.thickness)
	ctx.MoveTo(d.rect.Left, d.rect.Top)
	ctx.LineTo(d.rect.Right, d.rect.Bottom)
	ctx.Stroke()
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
