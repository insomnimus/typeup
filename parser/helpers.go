package parser

import (
	"fmt"
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

func (p *Parser) lineFirstNonSpaceIndex() int {
	var first int
	var ch rune
	for i := p.pos; i >= 0; i-- {
		ch = p.doc[i]
		if ch == '\n' {
			return first
		}
		if !unicode.IsSpace(ch) {
			first = i
		}
	}

	return first
}

func (p *Parser) lineLastNonSpaceIndex() int {
	var ch rune
	last := p.pos
	for i := p.pos; i < len(p.doc); i++ {
		ch = p.doc[i]
		if ch == '\n' {
			return last
		}
		if !unicode.IsSpace(ch) {
			last = i
		}
	}
	return last
}

func isBold(s string) (string, bool) {
	if len(s) < 3 {
		return "", false
	}
	if s[0] == '_' && s[len(s)-1] == '_' {
		return s[1 : len(s)-1], true
	}
	return "", false
}

func isItalic(s string) (string, bool) {
	if len(s) < 3 {
		return "", false
	}
	if s[0] == '*' && s[len(s)-1] == '*' {
		return s[1 : len(s)-1], true
	}
	return "", false
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
	text := spaceRemover.ReplaceAllString(string(p.doc[start:end]), "")
	if len(text) != 1 {
		return false
	}
	return rune(text[0]) == char
}

func isEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

func (p *Parser) warnAt(pos int, format string, args ...interface{}) {
	if pos < 0 {
		p.warnings = append(p.warnings, fmt.Sprintf(format, args...))
		return
	}
	ln := 1
	for i := 0; i < pos && i < len(p.doc); i++ {
		if p.doc[i] == '\n' {
			ln++
		}
	}
	args = append([]interface{}{ln}, args...)
	p.warnings = append(p.warnings, fmt.Sprintf("line %d: "+format, args...))
}

func (p *Parser) setPos(pos int) {
	if pos < 0 {
		return
	}
	if pos >= len(p.doc) {
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
	if s[start] != '[' {
		return nil, -1
	}
	var (
		buff strings.Builder
		ch   rune
		pos  = start
	)
	for i := start; i < len(s); i++ {
		if i+1 == len(s) {
			return nil, -1
		}
		ch = s[i]
		if ch == ']' {
			pos = i + 1
			break
		}
		buff.WriteRune(ch)
	}
	if unicode.IsSpace(s[pos]) {
		return nil, -1
	}
	text := strings.TrimSpace(buff.String())
	buff.Reset()
	for i := pos; i < len(s); i++ {
		ch = s[i]
		if unicode.IsSpace(ch) {
			pos = i
			break
		}
		buff.WriteRune(ch)
	}
	href := buff.String()
	if href == "" {
		return nil, -1
	}
	if text == "" {
		text = href
	}
	return &ast.Anchor{Text: processText(text), URL: href}, pos
}

func hasItalic(s []rune, start int) (*ast.Text, int) {
	if s[start] != '*' || start+1 >= len(s) || start < 0 {
		return nil, -1
	}
	var (
		pos  = start + 1
		ch   = s[pos]
		buff strings.Builder
	)
	for i := start; i < len(s); i++ {
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

func hasBold(s []rune, start int) (*ast.Text, int) {
	if s[start] != '_' || start+1 >= len(s) || start < 0 {
		return nil, -1
	}
	var (
		pos  = start + 1
		ch   = s[pos]
		buff strings.Builder
	)
	for i := start; i < len(s); i++ {
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

func (p *Parser) readLineRest() string {
	var buff strings.Builder
	for p.ch != '\n' && p.ch != 0 {
		buff.WriteRune(p.ch)
		p.read()
	}
	return buff.String()
}

func hasInlineCode(s []rune, start int) (*ast.Code, int) {
	if s[start] != '`' || start+1 >= len(s) || start < 0 {
		return nil, -1
	}
	var (
		pos  = start + 1
		ch   = s[pos]
		buff strings.Builder
	)
	for i := pos; i < len(s); i++ {
		ch = s[i]
		if ch == '`' {
			pos = i
			break
		}
		buff.WriteRune(ch)
	}
	if pos == start+1 {
		return nil, -1
	}
	text := buff.String()
	if text == "" {
		return nil, -1
	}
	return &ast.Code{
		Text: text,
	}, pos
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

func (p *Parser) Warnings() []string {
	return p.warnings
}
