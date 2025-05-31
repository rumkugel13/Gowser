package token

type TokenType int

const (
	TextTokenType TokenType = iota
	TagTokenType
)

type Token interface {
	Value() string
	Type() TokenType
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

type TagToken struct {
	Tag string
}

func (t TagToken) Value() string {
	return t.Tag
}

func (t TagToken) Type() TokenType {
	return TagTokenType
}