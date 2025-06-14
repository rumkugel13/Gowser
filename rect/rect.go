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

func NewRectEmpty() *Rect {
	return &Rect{}
}

func (r *Rect) Union(other *Rect) *Rect {
	if r.IsEmpty() && other.IsEmpty() {
		return NewRectEmpty()
	} else if r.IsEmpty() {
		return other.Clone()
	} else if other.IsEmpty() {
		return r.Clone()
	}
	return &Rect{
		Left:   min(r.Left, other.Left),
		Top:    min(r.Top, other.Top),
		Right:  max(r.Right, other.Right),
		Bottom: max(r.Bottom, other.Bottom),
	}
}

func (r *Rect) Intersect(other *Rect) *Rect {
	left := math.Max(r.Left, other.Left)
	top := math.Max(r.Top, other.Top)
	right := math.Min(r.Right, other.Right)
	bottom := math.Min(r.Bottom, other.Bottom)
	if left < right && top < bottom {
		return NewRect(left, top, right, bottom)
	}
	return NewRectEmpty()
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

func (r *Rect) Intersects(other *Rect) bool {
	return r.Left < other.Right && r.Right > other.Left &&
		r.Top < other.Bottom && r.Bottom > other.Top
}

func (r *Rect) ContainsPoint(x, y float64) bool {
	return x >= r.Left && x < r.Right &&
		y >= r.Top && y < r.Bottom
}

func (r *Rect) String() string {
	return fmt.Sprintf("Rect(left=%.2f, top=%.2f, right=%.2f, bottom=%.2f)", r.Left, r.Top, r.Right, r.Bottom)
}

func (r *Rect) Clone() *Rect {
	return &Rect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom}
}
