package lexer

import (
	"fmt"
	"strings"
	"typeup/token"
	"unicode"
)

type Lexer struct {
	doc          []rune
	ch           rune
	pos, readpos int
	line, col    int
	warnings     []string
}

func New(s string) *Lexer {
	s = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(s)
	l := &Lexer{
		doc: []rune(s),
	}
	l.read()
	return l
}

func (l *Lexer) Next() token.Token {
	l.skipSpace()
	var t token.Token
	p := l.getpos()
	switch l.ch {
	case 0:
		return newToken(p, token.EOF, "")
	case '|':
		if l.startOfLine() {
			t = newToken(p, token.Pipe, "|")
			break
		}
		return newToken(p, token.Text, l.readText())
	case '"':
		if l.startOfLine() &&
			l.peek() == '"' &&
			l.peekN(2) == '"' {
			l.read()
			l.read()
			t = newToken(p, token.TripleQuote, `"""`)
			break
		}
		return newToken(p, token.Text, l.readText())
	case '=':
		if s, ok := l.codeAhead(l.ch); ok {
			return newToken(p, token.Code, s)
		}
		if l.startOfLine() &&
			l.peek() == '#' {
			l.read()
			t = newToken(p, token.TitleSpecial, "=#")
			break
		}
		return newToken(p, token.Text, l.readText())
	case '\n':
		t = newToken(p, token.LF, "\n")
	case '`':
		if s, ok := l.codeAhead(l.ch); ok {
			return newToken(p, token.Code, s)
		}
		return newToken(p, token.Text, l.readText())
	case '#':
		t = newToken(p, token.Hash, "#")
	case '[':
		t = newToken(p, token.LBracket, "[")
	case ']':
		t = newToken(p, token.RBracket, "]")
	case '{':
		t = newToken(p, token.LBrace, "{")
	case '}':
		t = newToken(p, token.RBrace, "}")
	case '@':
		if l.startOfLine() &&
			l.peek() == '{' {
			l.read()
			t = newToken(p, token.MetaOpen, "@{")
			break
		}
		return newToken(p, token.Text, l.readText())
	case '!':
		if l.startOfLine() &&
			l.peek() == '[' {
			l.read()
			t = newToken(p, token.MediaOpen, "![")
			break
		}
		return newToken(p, token.Text, l.readText())
	default:
		return newToken(p, token.Text, l.readText())
	}
	l.read()
	return t
}

func (l *Lexer) codeAhead(ch rune) (code string, yes bool) {
	p := l.getpos()
	defer func() {
		if !yes {
			l.setpos(p)
		}
	}()
	// sanity check
	if l.ch != ch {
		panicf(p, "l.codeAhead called on char %q, expected %q instead", l.ch, ch)
	}
	if !l.startOfLine() {
		return
	}

	if ch == '`' && l.peek() != ch {
		return l.inlineCodeAhead()
	}
	for i := 0; i < 3; i++ {
		if l.ch != ch {
			return
		}
		l.read()
	}
	if !l.ignoreSpace() {
		l.warn(p, "non space characters are not allowed after '%c%c%c' in code blocks", ch, ch, ch)
		return
	}

	var buff strings.Builder

LOOP:
	for {
		switch l.ch {
		case 0:
			l.warn(p, "unexpected EOF in code block")
			return
		case ch:
			if l.startOfLine() && l.peek() == ch && l.peekN(2) == ch {
				l.read()
				l.read()
				l.read()
				if !l.ignoreSpace() {
					fmt.Fprintf(&buff, "%c%c%c", ch, ch, ch)
					l.warn(l.getpos(), "no characters allowed in the same line after '%c%c%c'", ch, ch, ch)
					continue LOOP
				}
				break LOOP
			}
			buff.WriteRune(l.ch)
		default:
			buff.WriteRune(l.ch)
		}
		l.read()
	}
	return strings.Trim(buff.String(), "\n"), true
}

func (l *Lexer) inlineCodeAhead() (code string, yes bool) {
	p := l.getpos()
	defer func() {
		if !yes {
			l.setpos(p)
		}
	}()
	// sanity check
	if l.ch != '`' {
		panicf(p, "l.inlineCodeAhead called on char %q, expected '`' instead", l.ch)
	}
	l.read()
	var buff strings.Builder
LOOP:
	for {
		switch l.ch {
		case 0, '\n':
			return
		case '\\':
			if l.peek() == '`' {
				buff.WriteRune('`')
				l.read()
				break
			}
			buff.WriteRune(l.ch)
		case '`':
			break LOOP
		default:
			buff.WriteRune(l.ch)
		}
		l.read()
	}
	l.read()
	yes = true
	return buff.String(), true
}

func (l *Lexer) readText() string {
	var buff strings.Builder
	buff.WriteRune(l.ch)
	l.read()

LOOP:
	for {
		switch l.ch {
		case '#':
			if l.startOfLine() {
				break LOOP
			}
			buff.WriteRune(l.ch)
		case '[', ']', '{', '}', 0:
			break LOOP
		case '@':
			if l.startOfLine() && l.peek() == '{' {
				break LOOP
			}
			buff.WriteRune(l.ch)
		case '"':
			if l.startOfLine() &&
				l.peek() == '"' &&
				l.peekN(2) == '"' {
				break LOOP
			}
			buff.WriteRune(l.ch)
		case '|':
			if l.startOfLine() {
				break LOOP
			}
			buff.WriteRune(l.ch)
		case '=':
			if l.startOfLine() {
				if l.peek() == '#' {
					break LOOP
				}
				if l.peek() == '=' &&
					l.peekN(2) == '=' {
					break LOOP
				}
			}
			buff.WriteRune(l.ch)
		case '`':
			break LOOP
		default:
			if unicode.IsSpace(l.ch) {
				break LOOP
			}
			buff.WriteRune(l.ch)
		}
		l.read()
	}
	return buff.String()
}
