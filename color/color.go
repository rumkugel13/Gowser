package color

import (
	col "image/color"

	"github.com/mazznoer/csscolorparser"
)

func ParseColor(color string) col.Color {
	c, err := csscolorparser.Parse(color)
	if err != nil {
		return col.Black
	}
	r, g, b, a := c.RGBA255()
	return col.RGBA{r, g, b, a}
}
