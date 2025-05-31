package layout

import (
	. "gowser/token"
	"strings"

	tk9_0 "modernc.org/tk9.0"
)

const (
	HSTEP        = 13.
	VSTEP        = 18.
	DefaultWidth = 800.
)

type LayoutItem struct {
	Word string
	X    float32
	Y    float32
	Font *tk9_0.FontFace
}

type LineItem struct {
	X    float32
	Word string
	Font *tk9_0.FontFace
}

type Layout struct {
	Display_list       []LayoutItem
	cursor_x, cursor_y float32
	weight, style      string
	size               float32
	Line               []LineItem
}

func NewLayout(tokens []Token) *Layout {
	layout := &Layout{
		Display_list: make([]LayoutItem, 0),
		cursor_x:     float32(HSTEP),
		cursor_y:     float32(VSTEP),
		style:        tk9_0.ROMAN,
		weight:       tk9_0.NORMAL,
		size:         12,
		Line:         make([]LineItem, 0),
	}
	for _, t := range tokens {
		layout.token(t)
	}
	layout.flush()
	return layout
}

func (l *Layout) token(token Token) {
	if token.Type() == TextTokenType {
		words := strings.Fields(token.Value())
		for _, word := range words {
			l.word(word)
		}
	} else if token.Value() == "i" {
		l.style = tk9_0.ITALIC
	} else if token.Value() == "/i" {
		l.style = tk9_0.ROMAN
	} else if token.Value() == "b" {
		l.weight = tk9_0.BOLD
	} else if token.Value() == "/b" {
		l.weight = tk9_0.NORMAL
	} else if token.Value() == "small" {
		l.size -= 2
	} else if token.Value() == "/small" {
		l.size += 2
	} else if token.Value() == "big" {
		l.size += 4
	} else if token.Value() == "/big" {
		l.size -= 4
	} else if token.Value() == "br" {
		l.flush()
	} else if token.Value() == "/p" {
		l.flush()
		l.cursor_y += float32(VSTEP)
	}
}

func (l *Layout) word(word string) {
	font := GetFont(l.size, l.weight, l.style)
	width := measure(font, word)
	if l.cursor_x+width >= DefaultWidth-HSTEP {
		l.flush()
	}
	l.Line = append(l.Line, LineItem{l.cursor_x, word, font})
	l.cursor_x += width + measure(font, " ")
}

func (l *Layout) flush() {
	if len(l.Line) == 0 {
		return
	}
	var maxAscent float32
	for _, item := range l.Line {
		maxAscent = max(maxAscent, float32(item.Font.MetricsAscent(tk9_0.App)))
	}

	baseline := l.cursor_y + maxAscent*1.25
	for _, item := range l.Line {
		l.Display_list = append(l.Display_list, LayoutItem{
			Word: item.Word,
			X:    item.X,
			Y:    baseline - float32(item.Font.MetricsAscent(tk9_0.App)),
			Font: item.Font,
		})
	}

	var maxDescent float32
	for _, item := range l.Line {
		maxDescent = max(maxDescent, float32(item.Font.MetricsDescent(tk9_0.App)))
	}
	l.cursor_y = baseline + maxDescent*1.25
	l.cursor_x = float32(HSTEP)
	l.Line = make([]LineItem, 0)
}

func measure(font *tk9_0.FontFace, text string) float32 {
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
