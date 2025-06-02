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

func Measure(font *tk9_0.FontFace, text string) float32 {
	// Measure the width of the text using the font metrics
	// This is a simplified version of text width measurement based on character widths.
	// In a real implementation, you would use the font's metrics to get accurate widths.
	var width float32
	ascent := float32(font.MetricsAscent(tk9_0.App))
	for _, r := range text {
		switch r {
		case '!', '\'', '`', ',', '.', 'i', 'l', ':', ';', '|':
			width += ascent * 0.2
		case '"', '(', ')', '[', ']', '{', '}', 'f', 'I', 'j', 'r', 't', '\\', '/', ' ':
			width += ascent * 0.35
		case '*', '+', '-', '=', '<', '>', 'a', 'b', 'c', 'd', 'e', 'g', 'h', 'k', 'o', 'p', 'q', 's', 'u', 'v', 'x', 'y', 'z', '~':
			width += ascent * 0.55
		case '#', '$', '%', '&', '?', '@', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'K', 'L', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'X', 'Y', 'Z':
			width += ascent * 0.7
		case 'M', 'W', 'm', 'w', '—':
			width += ascent * 0.9
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			width += ascent * 0.6
		default:
			width += ascent * 0.6
		}
	}
	return width
}
