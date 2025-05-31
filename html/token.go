package html

import (
	"strconv"
)

type TokenType int

const (
	TextTokenType TokenType = iota
	TagTokenType
)

type Token interface {
	Value() string
	Type() TokenType
	String() string
}

type TextToken struct {
	Text string
}

func (t TextToken) Value() string {
	return t.Text
}

func (t TextToken) Type() TokenType {
	return TextTokenType
}

func (t TextToken) String() string {
	return strconv.Quote(t.Text)
}

type TagToken struct {
	Tag string
}

func (t TagToken) Value() string {
	return t.Tag
}

func (t TagToken) Type() TokenType {
	return TagTokenType
}

func (t TagToken) String() string {
	return "<" + t.Tag + ">"
}
