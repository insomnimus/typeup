package ast

type TextStyle uint8

const(
NoStyle TextStyle = iota
Bold
Italic
BoldAndItalic
)

type Node interface{
	HTML() string
}

type ListItem interface{
	listHTML()
}

type TextNode interface{
	TextHTML() string
}

type Text struct{
	Style TextStyle
	Text string
}

type Paragraph struct{
	Words []TextNode
}

type UnorderedList struct{
	Items []ListItem
}

type OrderedList struct{
	Items []ListItem
}

type Link struct{
	Text TextNode
	URL string
}

type Image struct{
	Text string
	URL string
}

type Quote struct{
	Words []TextNode
}

type Heading struct{
	Level int
	Text TextNode
}

type CodeBlock struct{
	//Lang string
	Text string
}

type InlineCode string

type ThemeBreak struct{}
