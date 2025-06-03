package html

import (
	"strconv"
)

type Token interface {
	String() string
}

type TextToken struct {
	Text string
}

func NewTextToken(text string) TextToken {
	return TextToken{
		Text: text,
	}
}

func (t TextToken) String() string {
	return strconv.Quote(t.Text)
}

type ElementToken struct {
	Tag        string
	Attributes map[string]string
	IsFocused  bool
}

func NewElementToken(tag string, attributes map[string]string) ElementToken {
	return ElementToken{
		Tag:        tag,
		Attributes: attributes,
	}
}

func (e ElementToken) String() string {
	return "<" + e.Tag + ">"
}
