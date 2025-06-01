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

func (t TextToken) String() string {
	return strconv.Quote(t.Text)
}

type TagToken struct {
	Tag string
	Attributes map[string]string
}

func (t TagToken) String() string {
	return "<" + t.Tag + ">"
}
