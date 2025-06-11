package display

import (
	"gowser/rect"
	"image/color"

	"github.com/fogleman/gg"
)

type CompositedLayer struct {
	surface *gg.Context
	// skia_context not available
	DisplayItems []Command
}

func NewCompositedLayer(cmd Command) *CompositedLayer {
	return &CompositedLayer{
		surface:      nil,
		DisplayItems: []Command{cmd},
	}
}

func (c *CompositedLayer) Raster() {
	bounds := c.composited_bounds()
	if bounds.IsEmpty() {
		return
	}
	irect := bounds.RoundOutToInt()

	if c.surface == nil {
		c.surface = gg.NewContext(irect.Dx(), irect.Dy())
	}

	canvas := c.surface // same thing in gg
	canvas.SetColor(color.Transparent)
	canvas.Clear()
	canvas.Push()
	canvas.Translate(-bounds.Left, -bounds.Top)
	for _, item := range c.DisplayItems {
		item.Execute(canvas)
	}
	canvas.Pop()
}

func (c *CompositedLayer) composited_bounds() *rect.Rect {
	rect := rect.NewRect(0, 0, 0, 0)
	for _, item := range c.DisplayItems {
		rect = rect.Union(item.Rect())
	}
	rect.Inflate(1, 1)
	return rect
}
