package token

import "fmt"

type TokenType uint8

//go:generate stringer -type=TokenType

const (
	_ TokenType= iota
	Illegal
	EOF
	LF
	Code
	Text
	TitleSpecial
	TripleDash

	Hash
	Pipe
	LBracket
	RBracket
	LBrace
	RBrace
	MediaOpen
	TripleQuote
	MetaOpen
	Equals
)

type Token struct {
	Type    TokenType
	Literal string
	Line, Col int
}

func (t Token) GoString() string {
	return fmt.Sprintf("Token{\n"+
	"\tType: %s,\n"+
	"\tLiteral: %q,\n"+
	"\tLine: %d,\n"+
	"\tCol: %d,\n}", t.Type, t.Literal, t.Line, t.Col)
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q)", t.Type, t.Literal)
}
