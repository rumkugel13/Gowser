package css

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type CSSParser struct {
	style string
	i     int
}

func NewCSSParser(style string) *CSSParser {
	return &CSSParser{
		style: style,
		i:     0,
	}
}

func (p *CSSParser) whitespace() {
	for p.i < len(p.style) && unicode.IsSpace(rune(p.style[p.i])) {
		p.i++
	}
}

func (p *CSSParser) word() string {
	start := p.i
	for p.i < len(p.style) {
		if unicode.IsLetter(rune(p.style[p.i])) || unicode.IsDigit(rune(p.style[p.i])) || slices.Contains([]rune{'#', '-', '.', '%'}, rune(p.style[p.i])) {
			p.i++
		} else {
			break
		}
	}
	if !(p.i > start) {
		panic("Expected a word at position " + strconv.Itoa(p.i))
	}
	return p.style[start:p.i]
}

func (p *CSSParser) literal(literal rune) {
	if !(p.i < len(p.style) && rune(p.style[p.i]) == literal) {
		panic("Expected literal '" + string(literal) + "' at position " + strconv.Itoa(p.i))
	}
	p.i++
}

func (p *CSSParser) pair() (string, string) {
	prop := p.word()
	p.whitespace()
	p.literal(':')
	p.whitespace()
	val := p.word()
	return strings.ToLower(prop), val
}

func (p *CSSParser) body() map[string]string {
	pairs := make(map[string]string)
	for p.i < len(p.style) {
		shouldBreak := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					// this represents the catch block
					why := p.ignore_until(';')
					if why == ';' {
						p.literal(';')
						p.whitespace()
					} else {
						shouldBreak = true
					}
				}
			}()
			// this represents the try block
			prop, val := p.pair()
			pairs[strings.ToLower(prop)] = val
			p.whitespace()
			p.literal(';')
			p.whitespace()
		}()
		if shouldBreak {
			break
		}
	}
	return pairs
}

func (p *CSSParser) ignore_until(chars ...rune) rune {
	for p.i < len(p.style) {
		if slices.Contains(chars, rune(p.style[p.i])) {
			return rune(p.style[p.i])
		} else {
			p.i++
		}
	}
	return 0
}
