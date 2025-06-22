package browser

import (
	"slices"
	"strings"
)

var (
	VOID_TAGS = []string{
		"area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr"}
	HEAD_TAGS = []string{
		"base", "basefont", "bgsound", "noscript",
		"link", "meta", "title", "style", "script"}
)

type HTMLParser struct {
	body       string
	unfinished []*HtmlNode
}

func NewHTMLParser(body string) *HTMLParser {
	return &HTMLParser{
		body:       body,
		unfinished: []*HtmlNode{},
	}
}

func (p *HTMLParser) Parse() *HtmlNode {
	buffer := strings.Builder{}
	inTag := false
	for _, char := range p.body {
		if char == '<' {
			inTag = true
			if buffer.Len() > 0 {
				p.add_text(buffer.String())
				buffer.Reset()
			}
		} else if char == '>' {
			inTag = false
			p.add_tag(buffer.String())
			buffer.Reset()
		} else {
			buffer.WriteRune(char)
		}
	}
	if !inTag && buffer.Len() > 0 {
		p.add_text(buffer.String())
	}
	return p.finish()
}

func (p *HTMLParser) add_text(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	p.implicit_tags("")
	parent := p.unfinished[len(p.unfinished)-1]
	node := NewNode(NewTextToken(text), parent)
	parent.Children = append(parent.Children, node)
}

func (p *HTMLParser) add_tag(tag string) {
	tag, attributes := p.get_attributes(tag)
	if strings.HasPrefix(tag, "!") {
		return
	}
	p.implicit_tags(tag)

	if strings.HasPrefix(tag, "/") {
		if len(p.unfinished) == 1 {
			return
		}
		node := p.unfinished[len(p.unfinished)-1]
		p.unfinished = p.unfinished[:len(p.unfinished)-1] // pop
		parent := p.unfinished[len(p.unfinished)-1]
		parent.Children = append(parent.Children, node)
	} else if slices.Contains(VOID_TAGS, tag) {
		parent := p.unfinished[len(p.unfinished)-1]
		node := NewNode(NewElementToken(tag, attributes), parent)
		parent.Children = append(parent.Children, node)
	} else {
		var parent *HtmlNode
		if len(p.unfinished) == 0 {
			parent = nil
		} else {
			parent = p.unfinished[len(p.unfinished)-1]
		}
		node := NewNode(NewElementToken(tag, attributes), parent)
		p.unfinished = append(p.unfinished, node)
	}
}

func (p *HTMLParser) finish() *HtmlNode {
	if len(p.unfinished) == 0 {
		p.implicit_tags("")
	}
	for len(p.unfinished) > 1 {
		node := p.unfinished[len(p.unfinished)-1]
		p.unfinished = p.unfinished[:len(p.unfinished)-1] // pop
		parent := p.unfinished[len(p.unfinished)-1]
		parent.Children = append(parent.Children, node)
	}
	node := p.unfinished[len(p.unfinished)-1]
	p.unfinished = p.unfinished[:len(p.unfinished)-1] // pop
	return node
}

func isWhitespace(char rune) bool {
	switch char {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	default:
		return false
	}
}

func (p *HTMLParser) get_attributes(text string) (string, map[string]string) {
	attributes := make(map[string]string)

	split := strings.FieldsFunc(text, isWhitespace)
	if len(split) == 0 {
		return "", attributes
	}

	tag := strings.ToLower(split[0])
	if len(split) == 1 {
		return tag, attributes
	}

	rest := ""
	if len(text) > len(split[0]) {
		rest = strings.TrimSpace(text[len(split[0]):])
	}

	attr_str := rest
	start := 0
	cur := 0
	for {
		for start < len(attr_str) && isWhitespace(rune(attr_str[start])) {
			start++
		}
		for cur < len(attr_str) && attr_str[cur] != '=' {
			cur++
		}
		key := strings.ToLower(attr_str[start:cur])
		cur++
		start = cur // skip =
		if cur < len(attr_str) && (attr_str[cur] == '\'' || attr_str[cur] == '"') {
			quot := attr_str[cur]
			cur++ // skip quot
			for cur < len(attr_str) && attr_str[cur] != quot {
				cur++
			}
			val := attr_str[start+1 : cur]
			attributes[key] = val
			cur++ // skip quot
			start = cur
		} else if cur < len(attr_str) && !isWhitespace(rune(attr_str[cur])) {
			for cur < len(attr_str) && !isWhitespace(rune(attr_str[cur])) {
				cur++
			}
			val := attr_str[start:cur]
			attributes[key] = val
			start = cur
		} else {
			if key != "" && key != "/" {
				attributes[key] = ""
			}
			break
		}
	}
	return tag, attributes
}

func (p *HTMLParser) implicit_tags(tag string) {
	for {
		open_tags := []string{}
		for _, node := range p.unfinished {
			open_tags = append(open_tags, node.Token.(ElementToken).Tag)
		}
		if len(open_tags) == 0 && tag != "html" {
			p.add_tag("html")
		} else if len(open_tags) == 1 && open_tags[0] == "html" &&
			!slices.Contains([]string{"head", "body", "/html"}, tag) {
			if slices.Contains(HEAD_TAGS, tag) {
				p.add_tag("head")
			} else {
				p.add_tag("body")
			}
		} else if len(open_tags) == 2 && open_tags[0] == "html" &&
			open_tags[1] == "head" && !slices.Contains(append(HEAD_TAGS, "/head"), tag) {
			p.add_tag("/head")
		} else {
			break
		}
	}
}
