package html

import (
	"gowser/rect"
	"image/color"

	// "image/color"

	"github.com/fogleman/gg"
)

const (
	SHOW_COMPOSITED_LAYER_BORDERS = true
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
	bounds := c.CompositedBounds()
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
	canvas.Translate(-float64(irect.Min.X), -float64(irect.Min.Y))
	for _, item := range c.DisplayItems {
		item.Execute(canvas)
	}
	canvas.Pop()
	if SHOW_COMPOSITED_LAYER_BORDERS {
		border_rect := rect.NewRect(1, 1, 1+float64(irect.Dx())-2, 1+float64(irect.Dy())-2)
		NewDrawOutline(border_rect, "red", 1).Execute(canvas)
	}
}

func (c *CompositedLayer) Add(display_item Command) {
	c.DisplayItems = append(c.DisplayItems, display_item)
}

func (c *CompositedLayer) CanMerge(display_item Command) bool {
	return display_item.GetParent() == c.DisplayItems[0].GetParent()
}

func (c *CompositedLayer) CompositedBounds() *rect.Rect {
	rect := rect.NewRectEmpty()
	for _, item := range c.DisplayItems {
		rect = rect.Union(AbsoluteToLocal(item, LocalToAbsolute(item, item.Rect())))
	}
	rect.Inflate(1, 1)
	return rect
}

func (c *CompositedLayer) AbsoluteBounds() *rect.Rect {
	rect := rect.NewRectEmpty()
	for _, item := range c.DisplayItems {
		rect = rect.Union(LocalToAbsolute(item, item.Rect()))
	}
	return rect
}

func LocalToAbsolute(display_item Command, rct *rect.Rect) *rect.Rect {
	for display_item.GetParent() != nil {
		rct = display_item.GetParent().(VisualEffectCommand).Map(rct)
		display_item = display_item.GetParent()
	}
	return rct
}

func AbsoluteToLocal(display_item Command, rct *rect.Rect) *rect.Rect {
	parent_chain := []Command{}
	for display_item.GetParent() != nil {
		parent_chain = append(parent_chain, display_item.GetParent())
		display_item = display_item.GetParent()
	}
	for i := len(parent_chain) - 1; i >= 0; i-- {
		parent := parent_chain[i]
		rct = parent.(VisualEffectCommand).Unmap(rct)
	}
	return rct
}
