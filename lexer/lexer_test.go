package lexer

import (
	"testing"
	"typeup/token"
)

// TODO: rewrite a better test after some progress
func TestCodeAhead(t *testing.T) {
	doc := `===
line1
line2
===`
	l := New(doc)
	s, ok := l.codeAhead('=')
	if l.warnings != nil {
		t.Errorf("l.readCode produced warning: %s", l.warnings)
	}
	if !ok {
		t.Errorf("l.codeAhead returned false")
	}
	if s != "line1\nline2" {
		t.Errorf("l.codeAhead returned:\n%s\nexpected:\nline1\nline2", s)
	}
}

func TestNext(t *testing.T) {
	lf := token.Token{
		Type: token.LF, Literal: "\n",
	}
	tk := func(ty token.TokenType, s string) token.Token {
		return token.Token{
			Type:    ty,
			Literal: s,
		}
	}

	doc := `=# title
[
	1
	2
	3
]
@{x=y}
@{
	x=y
	a=b
}
### h3
===
line1
line2
===
# || {
	a || b || c
}
`
	tests := []token.Token{
		tk(token.TitleSpecial, "=#"),
		tk(token.Text, "title"),
		lf,
		tk(token.LBracket, "["),
		lf,
		tk(token.Text, "1"),
		lf,
		tk(token.Text, "2"),
		lf,
		tk(token.Text, "3"),
		lf,
		tk(token.RBracket, "]"),
		lf,
		tk(token.MetaOpen, "@{"),
		tk(token.Text, "x=y"),
		tk(token.RBrace, "}"),
		lf,
		tk(token.MetaOpen, "@{"),
		lf,
		tk(token.Text, "x=y"),
		lf,
		tk(token.Text, "a=b"),
		lf,
		tk(token.RBrace, "}"),
		lf,
		tk(token.Hash, "#"),
		tk(token.Hash, "#"),
		tk(token.Hash, "#"),
		tk(token.Text, "h3"),
		lf,
		tk(token.Code, "line1\nline2"),
		lf,
		tk(token.Hash, "#"),
		tk(token.Text, "||"),
		tk(token.LBrace, "{"),
		lf,
		tk(token.Text, "a"),
		tk(token.Text, "||"),
		tk(token.Text, "b"),
		tk(token.Text, "||"),
		tk(token.Text, "c"),
		lf,
		tk(token.RBrace, "}"),
		lf,
		tk(token.EOF, ""),
	}

	l := New(doc)
	for _, test := range tests {
		got := l.Next()
		if got.Type != test.Type {
			t.Errorf("type mismatch:\nexpected %s\ngot %s", test, got)
		}
		if test.Literal != got.Literal {
			t.Errorf("literal mismatch:\nexpected: %v\ngot: %v", got, test)
		}
	}
}
