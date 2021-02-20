package ast

import (
	"fmt"
	"html"
	"strings"
)

var escape = html.EscapeString

type TextStyle int

const (
	NoStyle = iota
	Bold
	Italic
	BoldAndItalic
)

type Node interface {
	HTML() string
}

type TextNode interface {
	textHTML() string
	listHTML() string
}

type ListItem interface {
	listHTML() string
}

type Text struct {
	Style TextStyle
	Text  string
}

func (t *Text) textHTML() string {
	switch t.Style {
	case Bold:
		return fmt.Sprintf("<b> %s </b>", escape(t.Text))
	case Italic:
		return fmt.Sprintf("<i> %s </i>", escape(t.Text))
	case BoldAndItalic:
		return fmt.Sprintf("<b><i> %s </i></b>", escape(t.Text))
	default:
		return escape(t.Text)
	}
}
func (t *Text) listHTML() string { return t.textHTML() }

type TextBlock struct {
	Items []TextNode
}

func (t *TextBlock) HTML() string {
	var out strings.Builder
	out.WriteString("<p>\n")
	for _, text := range t.Items {
		out.WriteString(text.textHTML())
		out.WriteRune('\n')
	}
	out.WriteString("</p>")
	return out.String()
}

func (tb *TextBlock) textHTML() string {
	var out strings.Builder
	for _, t := range tb.Items {
		out.WriteString(t.textHTML())
		out.WriteRune('\n')
	}
	return out.String()
}

func (tb *TextBlock) listHTML() string {
	var out strings.Builder
	for _, t := range tb.Items {
		fmt.Fprintf(&out, "%s ", t.listHTML())
	}
	return out.String()
}

type Heading struct {
	Title TextNode
	Level int
}

func (h *Heading) HTML() string {
	return fmt.Sprintf("<h%d> %s </h%d>", h.Level, h.Title.textHTML(), h.Level)
}

type OrderedList struct {
	Items []ListItem
}

func (ol *OrderedList) HTML() string {
	var out strings.Builder
	out.WriteString("<ol>\n")
	for _, x := range ol.Items {
		fmt.Fprintf(&out, "<li> %s </li>\n", x.listHTML())
	}
	out.WriteString("</ol>")
	return out.String()
}

func (ol *OrderedList) listHTML() string {
	return ol.HTML()
}

type UnorderedList struct {
	Items []ListItem
}

func (ul *UnorderedList) HTML() string {
	var out strings.Builder
	out.WriteString("<ul>\n")
	for _, x := range ul.Items {
		fmt.Fprintf(&out, "<li> %s </li>\n", x.listHTML())
	}
	out.WriteString("</ul>")
	return out.String()
}

func (ul *UnorderedList) listHTML() string { return ul.HTML() }

type Anchor struct {
	Text TextNode
	URL  string
}

func (a *Anchor) textHTML() string {
	return fmt.Sprintf("<a href=%q> %s </a>",
		a.URL,
		a.Text.textHTML())
}

func (a *Anchor) listHTML() string { return a.textHTML() }

type Table struct {
	Headers []TextNode
	Rows    [][]TextNode
}

func (t *Table) HTML() string {
	var out strings.Builder
	out.WriteString(`<table style="width:100%">`)
	out.WriteString("\n<tr>\n")
	for _, x := range t.Headers {
		fmt.Fprintf(&out, "<th> %s </th>\n", x.textHTML())
	}
	out.WriteString("</tr>\n")
	for _, row := range t.Rows {
		out.WriteString("<tr>\n")
		for _, x := range row {
			fmt.Fprintf(&out, "<td> %s </td>\n", x.textHTML())
		}
		out.WriteString("</tr>\n")
	}
	out.WriteString("</table>")
	return out.String()
}

type Code struct {
	Text string
}

func (c *Code) HTML() string {
	return fmt.Sprintf("<pre><code>\n%s\n</code></pre>",
		escape(c.Text))
}

func (c *Code) textHTML() string { return c.HTML() }
func (c *Code) listHTML() string { return c.HTML() }

type Video struct {
	Source string
	Attrs  map[string]string // not implemented
}

func (v *Video) HTML() string {
	return fmt.Sprintf("<video><source src=%q></video>", v.Source)
}

type Image struct {
	Attrs map[string]string
}

func (img *Image) HTML() string {
	var out strings.Builder
	out.WriteString("<img")
	for key, val := range img.Attrs {
		fmt.Fprintf(&out, " %s=%q", key, val)
	}
	out.WriteRune('>')
	return out.String()
}

type BlockQuote struct {
	Text TextNode
}

func (bq *BlockQuote) HTML() string {
	return fmt.Sprintf("<blockquote> %s </blockquote>",
		bq.Text.textHTML())
}

func (bq *BlockQuote) textHTML() string {
	return fmt.Sprintf("<blockquote> %s </blockquote>", bq.Text.textHTML())
}

func (bq *BlockQuote) listHTML() string {
	return fmt.Sprintf("<blockquote> %s </blockquote>", bq.listHTML())
}

// EoF is actually not a node but just there to signal the end of the file
type EoF struct{}

func (_ *EoF) HTML() string     { return "" }
func (_ *EoF) textHTML() string { return "" }
func (_ *EoF) listHTML() string { return "" }

type ThemeBreak struct{}

func (_ *ThemeBreak) HTML() string { return "<hr>" }

type LineBreak struct{}

func (_ *LineBreak) HTML() string     { return "<br>" }
func (_ *LineBreak) textHTML() string { return "<br>" }

type Document struct {
	Nodes []Node
	Meta  map[string]string
}
