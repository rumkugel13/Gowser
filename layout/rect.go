package layout

import "fmt"

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

func (r *Rect) ContainsPoint(x, y float64) bool {
	return x >= r.Left && x < r.Right &&
		y >= r.Top && y < r.Bottom
}

func (r *Rect) String() string {
	return fmt.Sprintf("Rect(left=%f, top=%f, right=%f, bottom=%f)", r.Left, r.Top, r.Right, r.Bottom)
}
