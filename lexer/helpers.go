package lexer

import (
	"fmt"
	"typeup/token"
	"unicode"
)

type position struct {
	index int
	line  int
	col   int
}

func (l *Lexer) read() {
	if l.readpos >= len(l.doc) {
		l.ch = 0
	} else {
		l.ch = l.doc[l.readpos]
	}
	l.col++
	if l.ch == '\n' {
		l.line++
		l.col = 0
	}
	l.pos = l.readpos
	l.readpos++
}

func (l *Lexer) peekN(n int) rune {
	n += l.pos
	if n < 0 || n >= len(l.doc) {
		return 0
	}
	return l.doc[n]
}

func (l *Lexer) peek() rune {
	if l.readpos >= len(l.doc) {
		return 0
	}
	return l.doc[l.readpos]
}

func (l *Lexer) getpos() position {
	return position{
		index: l.pos,
		line:  l.line,
		col:   l.col,
	}
}

func (l *Lexer) setpos(p position) {
	l.pos = p.index
	l.readpos = l.pos + 1
	l.ch = l.doc[l.pos]
	l.line = p.line
	l.col = p.col
}

func (l *Lexer) startOfLine() bool {
	if l.pos == 0 {
		return true
	}

	for i := l.pos - 1; i >= 0; i-- {
		if l.doc[i] == '\n' {
			return true
		}
		if !unicode.IsSpace(l.doc[i]) {
			return false
		}
	}
	return true
}

func (l *Lexer) ignoreSpace() (yes bool) {
	if l.ch == 0 || l.ch == '\n' {
		return true
	}
	if !unicode.IsSpace(l.ch) {
		return
	}
	p := l.getpos()
	defer func() {
		if !yes {
			l.setpos(p)
		}
	}()

	for l.ch != 0 && l.ch != '\n' {
		if !unicode.IsSpace(l.ch) {
			return
		}

		l.read()
	}
	return true
}

func (l *Lexer) warn(p position, format string, args ...interface{}) {
	format = "%s: " + format
	args = append([]interface{}{p}, args...)
	l.warnings = append(l.warnings, fmt.Sprintf(format, args...))
}

func panicf(p position, format string, args ...interface{}) {
	format = "internal error: %s: " + format
	args = append([]interface{}{p}, args...)
	panic(fmt.Sprintf(format, args...))
}

func (p position) String() string {
	return fmt.Sprintf("line %d:%d", p.line, p.col)
}

func newToken(p position, t token.TokenType, s string) token.Token {
	return token.Token{
		Type:    t,
		Literal: s,
		Line:    p.line,
		Col:     p.col,
	}
}

func (l *Lexer) skipSpace() {
	for l.ch != '\n' && unicode.IsSpace(l.ch) && l.ch != 0 {
		l.read()
	}
}
