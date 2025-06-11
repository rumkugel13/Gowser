package rect

import (
	"fmt"
	"image"
	"math"
)

type Rect struct {
	Left, Top, Right, Bottom float64
}

func NewRect(left, top, right, bottom float64) *Rect {
	return &Rect{
		Left:   left,
		Top:    top,
		Right:  right,
		Bottom: bottom,
	}
}

func (r *Rect) Union(other *Rect) *Rect {
	return &Rect{
		Left:   min(r.Left, other.Left),
		Top:    min(r.Top, other.Top),
		Right:  max(r.Right, other.Right),
		Bottom: max(r.Bottom, other.Bottom),
	}
}

func (r *Rect) Inflate(dx, dy float64) {
	r.Left -= dx
	r.Top -= dy
	r.Right += dx
	r.Bottom += dy
}

func (r *Rect) IsEmpty() bool {
	return r.Left >= r.Right || r.Top >= r.Bottom
}

func (r *Rect) RoundOutToInt() image.Rectangle {
	return image.Rect(
		int(math.Floor(r.Left)),
		int(math.Floor(r.Top)),
		int(math.Ceil(r.Right)),
		int(math.Ceil(r.Bottom)),
	)
}

func (r *Rect) Width() float64 {
	return r.Right - r.Left
}

func (r *Rect) Height() float64 {
	return r.Bottom - r.Top
}

func (r *Rect) ContainsPoint(x, y float64) bool {
	return x >= r.Left && x < r.Right &&
		y >= r.Top && y < r.Bottom
}

func (r *Rect) String() string {
	return fmt.Sprintf("Rect(left=%f, top=%f, right=%f, bottom=%f)", r.Left, r.Top, r.Right, r.Bottom)
}
