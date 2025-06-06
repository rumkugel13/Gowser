package layout

import (
	"fmt"
	"image/color"

	"github.com/adrg/sysfont"
	"github.com/tdewolff/canvas"
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
	Font  *canvas.Font
	Label string
}

func GetFont(size float64, weight, style string, color color.Color) *canvas.FontFace {
	key := FontKey{Size: size, Weight: weight, Style: style}
	if fontItem, exists := FONT_CACHE[key]; exists {
		return fontItem.Font.Face(size*float64(canvas.DefaultResolution), color)
	}

	cStyle := canvas.FontRegular
	if weight == "bold" {
		cStyle = canvas.FontBold
	}
	if style == "italic" {
		cStyle |= canvas.FontItalic
	}

	fontPath := sysfont.NewFinder(nil).Match(weight + " " + style)
	font, err := canvas.LoadFontFile(fontPath.Filename, cStyle)
	if err != nil {
		panic(fmt.Sprint("Error loading font:", font))
	}
	fmt.Println("Loading font:", font, "at size", size)

	FONT_CACHE[key] = FontItem{Font: font, Label: font.Name()}
	return font.Face(size*float64(canvas.DefaultResolution), color)
}

func Measure(font *canvas.FontFace, text string) float64 {
	return font.TextWidth(text)
}

func Linespace(font *canvas.FontFace) float64 {
	// note: without the scaling factor, the lines are too narrow
	return font.LineHeight()
}

func Ascent(font *canvas.FontFace) float64 {
	return font.Metrics().Ascent
}

func Descent(font *canvas.FontFace) float64 {
	return font.Metrics().Descent
}
