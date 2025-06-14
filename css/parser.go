package css

import (
	"cmp"
	"gowser/try"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const (
	REFRESH_RATE_SEC = 0.033
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

func (p *CSSParser) pair(until ...rune) (string, string) {
	prop := p.word()
	p.whitespace()
	p.literal(':')
	p.whitespace()
	val := p.until_chars(until...)
	return strings.ToLower(prop), strings.TrimSpace(val)
}

func (p *CSSParser) Body() map[string]string {
	pairs := make(map[string]string)
	for p.i < len(p.style) && p.style[p.i] != '}' {
		err := try.Try(func() {
			prop, val := p.pair(';', '}')
			pairs[strings.ToLower(prop)] = val
			p.whitespace()
			p.literal(';')
			p.whitespace()
		})
		if err != nil {
			why := p.ignore_until(';', '}')
			if why == ';' {
				p.literal(';')
				p.whitespace()
			} else {
				break
			}
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

func (p *CSSParser) until_chars(chars ...rune) string {
	start := p.i
	for p.i < len(p.style) && !slices.Contains(chars, rune(p.style[p.i])) {
		p.i++
	}
	return p.style[start:p.i]
}

func ParseTransition(value string) map[string]int {
	properties := make(map[string]int)
	if value == "" {
		return properties
	}

	for _, item := range strings.Split(value, ",") {
		split := strings.Fields(item)
		property, duration := split[0], split[1]
		fVal, err := strconv.ParseFloat(duration[:len(duration)-1], 32)
		if err == nil {
			frames := int(fVal / REFRESH_RATE_SEC)
			properties[property] = frames
		}
	}
	return properties
}

func ParseTransform(value string) (float64, float64) {
	if !strings.Contains(value, "translate(") {
		return 0, 0
	}
	left_paren := strings.Index(value, "(")
	right_paren := strings.Index(value, ")")
	parts := strings.Split(value[left_paren+1:right_paren], ",")
	xVal, err1 := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(parts[0], "px")), 32)
	yVal, err2 := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(parts[1], "px")), 32)
	if err := cmp.Or(err1, err2); err != nil {
		return 0, 0
	}
	return xVal, yVal
}

func ParseOutline(outline_str string) (int, string) {
	// note: better error value?
	if outline_str == "" {
		return 0, ""
	}
	values := strings.Fields(outline_str)
	if len(values) != 3 {
		return 0, ""
	}
	if values[1] != "solid" {
		return 0, ""
	}
	iVal, err := strconv.Atoi(strings.TrimSuffix(values[0], "px"))
	if err != nil {
		return 0, ""
	}
	return iVal, values[2]
}

func (p *CSSParser) simple_selector() Selector {
	var out Selector = NewTagSelector(strings.ToLower(p.word()))
	if p.i < len(p.style) && p.style[p.i] == ':' {
		p.literal(':')
		pseudoclass := strings.ToLower(p.word())
		out = NewPseudoclassSelector(pseudoclass, out)
	}
	return out
}

func (p *CSSParser) Selector() Selector {
	out := p.simple_selector()
	p.whitespace()
	for p.i < len(p.style) && p.style[p.i] != '{' {
		descendant := p.simple_selector()
		out = NewDescendantSelector(out, descendant)
		p.whitespace()
	}
	return out
}

func (p *CSSParser) Parse() []Rule {
	rules := make([]Rule, 0)
	var media string
	for p.i < len(p.style) {
		err := try.Try(func() {
			p.whitespace()
			if p.style[p.i] == '@' && media == "" {
				prop, val := p.media_query()
				if prop == "prefers-color-scheme" && slices.Contains([]string{"dark", "light"}, val) {
					media = val
				}
				p.whitespace()
				p.literal('{')
				p.whitespace()
			} else if p.style[p.i] == '}' && media != "" {
				p.literal('}')
				media = ""
				p.whitespace()
			} else {
				selector := p.Selector()
				p.literal('{')
				p.whitespace()
				body := p.Body()
				p.literal('}')
				rules = append(rules, *NewRule(media, selector, body))
			}
		})
		if err != nil {
			why := p.ignore_until('}')
			if why == '}' {
				p.literal('}')
				p.whitespace()
			} else {
				break
			}
		}
	}
	return rules
}

func (p *CSSParser) media_query() (string, string) {
	p.literal('@')
	word := p.word()
	if word != "media" {
		panic("unsupported media query: " + word)
	}
	p.whitespace()
	p.literal('(')
	p.whitespace()
	prop, val := p.pair(')')
	p.whitespace()
	p.literal(')')
	return prop, val
}
