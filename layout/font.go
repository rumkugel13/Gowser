package layout

import (
	"fmt"
	"math"

	"github.com/adrg/sysfont"
	"github.com/fogleman/gg"
	fnt "golang.org/x/image/font"
)

var (
	FONT_CACHE = map[FontKey]FontItem{}
)

type FontKey struct {
	Size   float64
	Weight string
	Style  string
}

type FontItem struct {
	Font  fnt.Face
	Label string
}

func GetFont(size float64, weight, style string) fnt.Face {
	key := FontKey{Size: size, Weight: weight, Style: style}
	if fontItem, exists := FONT_CACHE[key]; exists {
		return fontItem.Font
	}

	font := sysfont.NewFinder(nil).Match(weight + " " + style)
	fontFace, err := gg.LoadFontFace(font.Filename, size)
	if err != nil {
		panic(fmt.Sprint("Error loading font:", font))
	}
	fmt.Println("Loading font:", font, "at size", size)

	label := font.Name
	FONT_CACHE[key] = FontItem{Font: fontFace, Label: label}
	return fontFace
}

func Measure(font fnt.Face, text string) float64 {
	return math.Ceil(float64(fnt.MeasureString(font, text)) / 64.0)
}

func Linespace(font fnt.Face) float64 {
	// note: without the scaling factor, the lines are too narrow
	return math.Ceil(float64(font.Metrics().Height) / 64.0 * 96 / 72)
}

func Ascent(font fnt.Face) float64 {
	return float64(font.Metrics().Ascent) / 64.0
}

func Descent(font fnt.Face) float64 {
	return float64(font.Metrics().Descent) / 64.0
}
