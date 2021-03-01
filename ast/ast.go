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
	Bare() string
	textHTML() string
	listHTML() string
}

type ListItem interface {
	listHTML() string
}

// Text

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
func (t *Text) Bare() string     { return t.Text }

// TextBlock

func (t *TextBlock) HTML() string {
	var out strings.Builder
	out.WriteString("<p>\n")
	var tmp string
	for _, text := range t.Items {
		tmp = strings.TrimSpace(text.textHTML())
		if tmp == "" {
			continue
		}
		out.WriteString(tmp)
		out.WriteRune('\n')
	}
	out.WriteString("</p>")
	return out.String()
}

func (tb *TextBlock) Bare() string {
	var elems []string
	for _, x := range tb.Items {
		elems = append(elems, x.Bare())
	}
	return strings.Join(elems, " ")
}

func (tb *TextBlock) textHTML() string {
	var out strings.Builder
	var tmp string
	for _, t := range tb.Items {
		tmp = strings.TrimSpace(t.textHTML())
		if tmp == "" {
			continue
		}
		out.WriteString(tmp)
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

// Heading

func (h *Heading) HTML() string {
	return fmt.Sprintf("<h%d> %s </h%d>", h.Level,
		strings.ReplaceAll(h.Title.textHTML(), "\n", ""), h.Level)
}

// OrderedList

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

// UnorderedList

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

// Anchor

func (a *Anchor) Bare() string { return a.Text.Bare() }
func (a *Anchor) textHTML() string {
	return fmt.Sprintf("<a href=%q> %s </a>",
		a.URL,
		a.Text.textHTML())
}

func (a *Anchor) listHTML() string { return a.textHTML() }

// Table

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

// Code

func (c *Code) HTML() string {
	return fmt.Sprintf("<pre><code>%s</code></pre>",
		escape(c.Text))
}

func (c *Code) textHTML() string { return c.HTML() }
func (c *Code) listHTML() string { return c.HTML() }
func (c *Code) Bare() string     { return c.Text }

// Video

func (v *Video) HTML() string {
	return fmt.Sprintf("<video><source src=%q></video>", v.Source)
}

// Image

func (img *Image) HTML() string {
	var out strings.Builder
	out.WriteString("<img")
	for key, val := range img.Attrs {
		fmt.Fprintf(&out, " %s=%q", key, val)
	}
	out.WriteRune('>')
	return out.String()
}

// BlockQuote

func (bq *BlockQuote) Bare() string { return bq.Text.Bare() }
func (bq *BlockQuote) HTML() string {
	return fmt.Sprintf("<blockquote> %s </blockquote>",
		bq.Text.textHTML())
}

func (bq *BlockQuote) textHTML() string {
	return fmt.Sprintf("<blockquote> %s </blockquote>", bq.Text.textHTML())
}

func (bq *BlockQuote) listHTML() string {
	return fmt.Sprintf("<blockquote> %s </blockquote>", bq.Text.listHTML())
}

// ThemeBreak

func (*ThemeBreak) HTML() string { return "<hr>" }

// LineBreak

func (*LineBreak) HTML() string     { return "<br>" }
func (*LineBreak) textHTML() string { return "<br>" }
func (*LineBreak) Bare() string     { return "" }

// InlineCode

func (c *InlineCode) textHTML() string {
	return fmt.Sprintf("<code> %s </code>", escape(c.Text))
}

func (c *InlineCode) listHTML() string { return c.textHTML() }
func (c *InlineCode) Bare() string     { return c.Text }
