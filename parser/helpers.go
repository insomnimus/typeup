package parser

import (
	"regexp"
	"strings"
	"typeup/ast"
	"unicode"
)

var spaceRemover = regexp.MustCompile(`\s+`)

func (p *Parser) read() {
	if p.readpos >= len(p.doc) {
		p.ch = 0
	} else {
		p.ch = p.doc[p.readpos]
	}
	p.pos = p.readpos
	p.readpos++
}

func (p *Parser) peek() rune {
	return p.peekN(1)
}

func (p *Parser) peekN(n int) rune {
	if p.pos+n < 0 || p.pos+n >= len(p.doc) {
		return 0
	}
	return p.doc[p.pos+n]
}

func (p *Parser) isStartOfLine() bool {
	if p.pos == 0 {
		return true
	}
	return p.doc[p.pos-1] == '\n'
}

func (p *Parser) searchLineUntil(c rune) (string, int) {
	end := -1
	var char rune
	for i := p.readpos; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '\n' {
			break
		}
		if char == c {
			end = i
			break
		}
	}
	return string(p.doc[p.readpos:end]), end
}

func (p *Parser) aheadIs(s string) bool {
	if len(s)+p.pos >= len(p.doc) {
		return false
	}
	return string(p.doc[p.pos:p.pos+len(s)]) == s
}

func (p *Parser) lineOnlyCharIs(char rune) bool {
	var start, end int
	for i := p.pos; i >= 0; i-- {
		if p.doc[i] == '\n' {
			start = i + 1
			break
		}
	}
	if p.readpos < len(p.doc) {
		for i := p.readpos; i < len(p.doc); i++ {
			if i+1 >= len(p.doc) {
				end = len(p.doc)
				break
			}
			if p.doc[i] == '\n' {
				end = i
				break
			}
		}
	} else {
		end = len(p.doc)
	}
	text := spaceRemover.ReplaceAllString(string(p.doc[start:end]), "")
	if len(text) != 1 {
		return false
	}
	return rune(text[0]) == char
}

func (p *Parser) setPos(pos int) {
	if pos < 0 ||
		pos >= len(p.doc) {
		return
	}
	p.pos = pos
	p.readpos = pos + 1
	p.ch = p.doc[pos]
}

func (p *Parser) lineLastChar() rune {
	char := p.ch
	lastChar := p.ch
	for i := p.pos; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '\n' {
			return lastChar
		}
		if !unicode.IsSpace(char) {
			lastChar = char
		}
	}
	return lastChar
}

func hasAnchor(s []rune, start int) (*ast.Anchor, int) {
	if s[start] != '[' || start+1 >= len(s) || start < 0 {
		return nil, -1
	}
	var (
		buff strings.Builder
		ch   rune
		pos  = start + 1
	)

	for i := pos; i < len(s); i++ {
		ch = s[i]
		if ch == '\n' {
			return nil, -1
		}
		if ch == ']' {
			pos = i
			break
		}
		buff.WriteRune(ch)
	}
	if pos == start+1 {
		return nil, -1
	}
	text := strings.TrimSpace(buff.String())
	if text == "" {
		return nil, -1
	}
	if strings.Contains(text, "|") {
		fields := strings.SplitN(text, "|", 2)
		switch len(fields) {
		case 1:
			return &ast.Anchor{
				Text: &ast.Text{Text: strings.TrimSpace(fields[0])},
				URL:  strings.TrimSpace(fields[0]),
			}, pos
		case 2:
			return &ast.Anchor{
				Text: processText(strings.TrimSpace(fields[0])),
				URL:  strings.TrimSpace(fields[1]),
			}, pos
		}
	}
	fields := strings.Fields(text)
	switch len(fields) {
	case 0: // impossible
		return nil, -1
	case 1:
		return &ast.Anchor{
			Text: &ast.Text{Text: text},
			URL:  text,
		}, pos
	default:
		text = strings.Join(fields[:len(fields)-1], " ")
		return &ast.Anchor{
			Text: processText(text),
			URL:  fields[len(fields)-1],
		}, pos
	}
}

func hasBold(s []rune, start int) (*ast.Text, int) {
	switch s[start] {
	case '=':
		return hasBoldLong(s, start)
	case '*':
		return hasBoldShort(s, start)
	default:
		return nil, -1
	}
}

func hasItalic(s []rune, start int) (*ast.Text, int) {
	switch s[start] {
	case '/':
		return hasItalicLong(s, start)
	case '_':
		return hasItalicShort(s, start)
	default:
		return nil, -1
	}
}

func hasBoldShort(s []rune, start int) (*ast.Text, int) {
	if s[start] != '*' || start+1 >= len(s) || start < 0 {
		return nil, -1
	}
	var (
		pos  = start + 1
		ch   = s[pos]
		buff strings.Builder
	)
	for i := pos; i < len(s); i++ {
		ch = s[i]
		if ch == '*' {
			pos = i
			break
		}
		buff.WriteRune(ch)
	}
	if pos == start+1 {
		return nil, -1
	}
	text := strings.TrimSpace(buff.String())
	if text == "" {
		return nil, -1
	}
	if item, xpos := hasBold([]rune(text), 0); xpos != -1 {
		item.Style = ast.BoldAndItalic
		return item, pos
	}
	return &ast.Text{Style: ast.Italic, Text: text}, pos
}

func hasBoldLong(s []rune, start int) (*ast.Text, int) {
	if start+4 >= len(s) ||
		s[start] != '=' || s[start+1] != '=' {
		return nil, -1
	}
	var (
		pos  = start + 2
		buff strings.Builder
		ch   rune
	)
	for i := pos; i < len(s); i++ {
		ch = s[i]
		if ch == '\n' {
			return nil, -1
		}
		if ch == '=' &&
			i+1 < len(s) &&
			s[i+1] == '=' {
			pos = i + 1
			break
		}
		buff.WriteRune(ch)
	}
	if pos == start+2 {
		return nil, -1
	}
	text := []rune(strings.TrimSpace(buff.String()))
	if len(text) == 0 {
		return nil, -1
	}
	if node, ps := hasItalic(text, 0); len(text) == ps+1 {
		node.Style = ast.BoldAndItalic
		return node, pos
	}
	return &ast.Text{
		Text:  string(text),
		Style: ast.Bold,
	}, pos
}

func hasItalicShort(s []rune, start int) (*ast.Text, int) {
	if s[start] != '_' || start+1 >= len(s) || start < 0 {
		return nil, -1
	}
	var (
		pos  = start + 1
		ch   = s[pos]
		buff strings.Builder
	)
	for i := pos; i < len(s); i++ {
		ch = s[i]
		if ch == '_' {
			pos = i
			break
		}
		buff.WriteRune(ch)
	}
	if pos == start+1 {
		return nil, -1
	}
	text := strings.TrimSpace(buff.String())
	if text == "" {
		return nil, -1
	}
	if item, xpos := hasItalic([]rune(text), 0); xpos != -1 {
		item.Style = ast.BoldAndItalic
		return item, pos
	}
	return &ast.Text{Style: ast.Bold, Text: text}, pos
}

func hasItalicLong(s []rune, start int) (*ast.Text, int) {
	if start+4 >= len(s) ||
		s[start] != '/' || s[start+1] != '/' {
		return nil, -1
	}
	var (
		buff strings.Builder
		pos  = start + 2
		ch   rune
	)
	for i := pos; i < len(s); i++ {
		ch = s[i]
		if ch == '\n' {
			return nil, -1
		}
		if ch == '/' &&
			i+1 < len(s) && s[i+1] == '/' {
			pos = i + 1
			break
		}
		buff.WriteRune(ch)
	}
	if pos == start+2 {
		return nil, -1
	}
	text := []rune(strings.TrimSpace(buff.String()))
	if len(text) == 0 {
		return nil, -1
	}
	if node, ps := hasBold(text, 0); ps == len(text)-1 {
		node.Style = ast.BoldAndItalic
		return node, pos
	}
	return &ast.Text{
		Text:  string(text),
		Style: ast.Italic,
	}, pos
}

func (p *Parser) readLineRest() string {
	var buff strings.Builder
	for p.ch != '\n' && p.ch != 0 {
		buff.WriteRune(p.ch)
		p.read()
	}
	return buff.String()
}

func (p *Parser) isSpaceUntilLF() bool {
	if p.readpos >= len(p.doc) {
		return true
	}
	if p.peek() == '\n' || p.peek() == 0 {
		return true
	}
	var ch rune
	for i := p.readpos; i < len(p.doc); i++ {
		ch = p.doc[i]
		if ch == '\n' {
			return true
		}
		if !unicode.IsSpace(ch) {
			return false
		}
	}
	return true
}

func hasInlineCode(s []rune, start int) (*ast.InlineCode, int) {
	if start < 0 || start >= len(s) {
		return nil, -1
	}
	var (
		buff strings.Builder
		ch   rune
		pos  int
	)
	switch s[start] {
	case '`':
		pos = start + 1
		if pos >= len(s) {
			return nil, -1
		}
		for i := pos; i < len(s); i++ {
			ch = s[i]
			if ch == '\n' {
				return nil, -1
			}
			if ch == '`' {
				pos = i
				break
			}
			buff.WriteRune(ch)
		}
		if pos == start+1 {
			return nil, -1
		}
		return &ast.InlineCode{
			Text: buff.String(),
		}, pos
	case '\'':
		pos = start + 2
		if pos >= len(s) {
			return nil, -1
		}
		for i := pos; i < len(s); i++ {
			ch = s[i]
			if ch == '\n' {
				return nil, -1
			}
			if ch == '\'' &&
				i+1 < len(s) &&
				s[i+1] == '\'' {
				pos = i + 1
				break
			}
			buff.WriteRune(ch)
		}
		if pos == start+2 {
			return nil, -1
		}
		return &ast.InlineCode{
			Text: buff.String(),
		}, pos
	default:
		return nil, -1
	}
}
