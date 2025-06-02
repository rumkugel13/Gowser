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

type TagToken struct {
	Tag        string
	Attributes map[string]string
}

func NewTagToken(tag string, attributes map[string]string) TagToken {
	return TagToken{
		Tag:        tag,
		Attributes: attributes,
	}
}

func (t TagToken) String() string {
	return "<" + t.Tag + ">"
}
