package layout

import (
	"modernc.org/tk9.0"
)

var (
	FONT_CACHE = map[FontKey]FontItem{}
)

type FontKey struct {
	Size   int
	Weight string
	Style  string
}

type FontItem struct {
	Font  *tk9_0.FontFace
	Label string
}

func GetFont(size int, weight, style string) *tk9_0.FontFace {
	key := FontKey{Size: size, Weight: weight, Style: style}
	if fontItem, exists := FONT_CACHE[key]; exists {
		return fontItem.Font
	}

	fontFace := tk9_0.NewFont(tk9_0.Size(size), tk9_0.Slant(style), tk9_0.Weight(weight))
	label := fontFace.String()
	FONT_CACHE[key] = FontItem{Font: fontFace, Label: label}
	return fontFace
}
