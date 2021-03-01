package ast

type Text struct {
	Style TextStyle
	Text  string
}

type TextBlock struct {
	Items []TextNode
}

type Heading struct {
	Title TextNode
	Level int
}

type OrderedList struct {
	Items []ListItem
}

type UnorderedList struct {
	Items []ListItem
}

type Anchor struct {
	Text TextNode
	URL  string
}

type Table struct {
	Headers []TextNode
	Rows    [][]TextNode
}

type Code struct {
	Text string
}

type Video struct {
	Source string
	Attrs  map[string]string // not implemented
}

type Image struct {
	Attrs map[string]string
}

type BlockQuote struct {
	Text TextNode
}

type ThemeBreak struct{}
type LineBreak struct{}

type InlineCode struct {
	Text string
}
