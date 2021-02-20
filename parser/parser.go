package parser

import (
	"strings"
	"typeup/ast"
	"unicode"
)

type Parser struct {
	doc              []rune
	ch               rune
	pos, readpos     int
	errors, warnings []string
	meta             map[string]string
}

func New(s string) *Parser {
	p := &Parser{
		doc:  []rune(s),
		meta: make(map[string]string),
	}
	p.read()
	return p
}

func (p *Parser) codeAhead(delim rune) (*ast.Code, bool) {
	if p.ch != delim || p.peek() != p.ch || p.peekN(2) != p.ch {
		return nil, false
	}
	backupPos := p.pos
	p.read()
	p.read()
	p.read()
	var buff strings.Builder
LOOP:
	for {
		switch p.ch {
		case delim:
			if p.peek() == delim && p.peekN(2) == delim {
				p.read()
				p.read()
				p.read()
				break LOOP
			}
			buff.WriteRune(p.ch)
		case 0:
			p.warnAt(p.pos, "possible code block not terminated")
			p.setPos(backupPos)
			return nil, false
		default:
			buff.WriteRune(p.ch)
		}
		p.read()
	}
	return &ast.Code{
		Text: buff.String(),
	}, true
}

func (p *Parser) linkAhead() (*ast.Anchor, bool) {
	if p.ch != '[' {
		return nil, false
	}
	var text, href strings.Builder
	var char rune
	var idx int
	for i := p.readpos; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '\n' {
			return nil, false
		}
		if char == ']' {
			idx = i + 1
			break
		}
		text.WriteRune(char)
	}
	if unicode.IsSpace(p.doc[idx]) {
		return nil, false
	}
	for i := idx; i < len(p.doc); i++ {
		char = p.doc[i]
		if unicode.IsSpace(char) {
			p.setPos(i)
			break
		}
		href.WriteRune(char)
	}
	return &ast.Anchor{
		Text: processText(text.String()),
		URL:  href.String(),
	}, true
}

func (p *Parser) headingAhead() (*ast.Heading, bool) {
	if p.ch != '#' || !p.isStartOfLine() {
		return nil, false
	}
	var char rune
	var idx, level int
	var buff strings.Builder
	for i := p.pos; i < len(p.doc); i++ {
		char = p.doc[i]
		if unicode.IsSpace(char) && char != '\n' {
			idx = i + 1 // set the cursor to the text
			break
		}
		if char == '\n' {
			p.warnAt(i, "heading possibly missing title")
			return nil, false
		}
		level++
	}
	if idx >= len(p.doc) {
		p.warnAt(idx, "line doesn2t make sense")
		return nil, false
	} else if p.doc[idx] == '\n' {
		p.warnAt(idx, "heading possibly missing title")
		return nil, false
	}
	if level > 6 {
		level = 6
		p.warnAt(idx, "too many '#' for a heading, maximum is 6")
	}
	for i := idx; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '\n' {
			p.setPos(i)
			break
		}
		buff.WriteRune(char)
	}
	return &ast.Heading{
		Level: level,
		Title: processText(buff.String()),
	}, true
}

func (p *Parser) tableAhead() (*ast.Table, bool) {
	backupPos := p.pos // to revert back to in case of faulty typeup file
	if p.ch != '#' || !p.isStartOfLine() {
		return nil, false
	}
	if p.lineLastChar() != '{' {
		return nil, false
	}
	var delim string
	var char rune
	pos := p.readpos
	for i := p.readpos; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '{' {
			pos = i + 1
			break
		}
		if char == '#' || char == '}' {
			p.warnAt(i, "%c is not allowed as a table delimiter", char)
			return nil, false
		}
		if char == '\n' {
			// todo: make sure it never comes to this block, it shouldn't
			return nil, false
		}
		delim += string(char)
	}
	// check if there're any non space chars after '{'
	if pos >= len(p.doc) {
		p.warnAt(pos, "unexpected EoF in possible table declaration")
		return nil, false
	}
	for i := pos; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '\n' {
			// it's good, jump over to the next line
			pos = i + 1
			break
		}
		if !unicode.IsSpace(char) {
			p.warnAt(i, "illegal character '%c' in the same line after opening brace in table", char)
			return nil, false
		}
	}
	if pos >= len(p.doc) {
		p.warnAt(pos, "unexpected EoF in possible table declaration")
		return nil, false
	}
	delim = strings.TrimSpace(delim)
	if delim == "" {
		p.warnAt(pos-1, "the table delimiter can't be empty")
		return nil, false
	}
	// prepare to read the table elements
	p.setPos(pos)
	// read until we find the closing brace
	var lines []string
	var buff strings.Builder
	for {
		if p.ch == 0 {
			p.warnAt(p.pos, "unexpected EoF in table")
			p.setPos(backupPos)
			return nil, false
		}
		if p.ch == '\n' {
			lines = append(lines, buff.String())
			buff.Reset()
		}
		if p.ch == '}' && p.lineOnlyCharIs(p.ch) {
			// table done, break out
			p.read()
			break
		}
		p.read()
	}
	if len(lines) == 0 {
		p.warnAt(p.pos, "table with no elements ignored")
		p.setPos(backupPos)
		return nil, false
	} else if len(lines) == 0 {
		p.warnAt(p.pos, "table missing cells")
		p.setPos(backupPos)
		return nil, false
	}
	// TODO(note): do this in place
	return parseTable(lines, delim), true
}

func (p *Parser) ulAhead() (*ast.UnorderedList, bool) {
	if p.ch != '[' {
		return nil, false
	}
	if !p.lineOnlyCharIs('[') {
		return nil, false
	}
	backupPos := p.pos
	// consume the line and get to the first element
	for p.ch != 0 && p.ch != '\n' {
		p.read()
	}
	if p.ch == 0 {
		p.warnAt(backupPos, "stray '['")
		p.setPos(backupPos)
		return nil, false
	}
	p.read()
	var (
		ln    string
		buff  strings.Builder
		items []ast.ListItem
	)
LOOP:
	for {
		switch p.ch {
		case '[':
			if item, ok := p.ulAhead(); ok {
				items = append(items, item)
			} else {
				buff.WriteRune(p.ch)
			}
		case '{':
			if item, ok := p.olAhead(); ok {
				items = append(items, item)
			} else {
				buff.WriteRune(p.ch)
			}
		case ']':
			if p.lineOnlyCharIs(p.ch) {
				p.read()
				break LOOP
			}
			buff.WriteRune(p.ch)
		case 0:
			p.warnAt(p.pos, "possible list not terminated with ']'")
			p.setPos(backupPos)
			return nil, false
		case '\n':
			ln = buff.String()
			buff.Reset()
			if !isEmpty(ln) {
				items = append(items, processText(ln))
			}
		default:
			buff.WriteRune(p.ch)
		}
		p.read()
	}
	return &ast.UnorderedList{
		Items: items,
	}, true
}

func (p *Parser) olAhead() (*ast.OrderedList, bool) {
	if p.ch != '{' {
		return nil, false
	}
	if !p.lineOnlyCharIs('{') {
		return nil, false
	}
	backupPos := p.pos
	// consume the line and get to the first element
	for p.ch != 0 && p.ch != '\n' {
		p.read()
	}
	if p.ch == 0 {
		p.warnAt(backupPos, "stray '{'")
		p.setPos(backupPos)
		return nil, false
	}
	p.read()
	var (
		ln    string
		buff  strings.Builder
		items []ast.ListItem
	)
LOOP:
	for {
		switch p.ch {
		case '[':
			if item, ok := p.ulAhead(); ok {
				items = append(items, item)
			} else {
				buff.WriteRune(p.ch)
			}
		case '{':
			if item, ok := p.olAhead(); ok {
				items = append(items, item)
			} else {
				buff.WriteRune(p.ch)
			}
		case '}':
			if p.lineOnlyCharIs(p.ch) {
				p.read()
				break LOOP
			}
			buff.WriteRune(p.ch)
		case 0:
			p.warnAt(p.pos, "possible list not terminated with '}'")
			p.setPos(backupPos)
			return nil, false
		case '\n':
			ln = buff.String()
			buff.Reset()
			if !isEmpty(ln) {
				items = append(items, processText(ln))
			}
		default:
			buff.WriteRune(p.ch)
		}
		p.read()
	}
	return &ast.OrderedList{
		Items: items,
	}, true
}

func processText(source string) ast.TextNode {
	s := []rune(source)
	var (
		buff  strings.Builder
		ch    rune
		items []ast.TextNode
		text  string
	)

LOOP:
	for i := 0; i < len(s); i++ {
		ch = s[i]
		switch ch {
		case '[':
			if anchor, pos := hasAnchor(s, i); pos > i {
				text = buff.String()
				if !isEmpty(text) {
					items = append(items, &ast.Text{
						Text: text,
					})
				}
				buff.Reset()
				items = append(items, anchor)
				i = pos
				continue LOOP
			} else {
				buff.WriteRune(ch)
			}
		case '*':
			if item, pos := hasItalic(s, i); pos > i {
				text = buff.String()
				if !isEmpty(text) {
					items = append(items, &ast.Text{
						Text: text,
					})
				}
				buff.Reset()
				items = append(items, item)
				i = pos
				continue LOOP
			} else {
				buff.WriteRune(ch)
			}
		case '_':
			if item, pos := hasBold(s, i); pos > i {
				text = buff.String()
				if !isEmpty(text) {
					items = append(items, &ast.Text{
						Text: text,
					})
				}
				buff.Reset()
				items = append(items, item)
				i = pos
				continue LOOP
			} else {
				buff.WriteRune(ch)
			}
		default:
			buff.WriteRune(ch)
		}
	}
	text = buff.String()
	if !isEmpty(text) {
		items = append(items, &ast.Text{
			Text: text,
		})
	}
	return &ast.TextBlock{
		Items: items,
	}
}

func parseTable(lines []string, delim string) *ast.Table {
	headers := strings.Split(lines[0], delim)
	var rows [][]string
	for _, row := range lines[1:] {
		rows = append(rows,
			strings.Split(row, delim))
	}
	var table ast.Table
	for _, x := range headers {
		table.Headers = append(table.Headers,
			processText(x))
	}
	for _, row := range rows {
		var cells []ast.TextNode
		for _, x := range row {
			cells = append(cells, processText(x))
		}
		table.Rows = append(table.Rows, cells)
	}
	return &table
}

func (p *Parser) headingShortAhead() (*ast.Heading, bool) {
	if p.ch != '=' || p.peek() != '#' {
		return nil, false
	}
	// read past '#='
	backupPos := p.pos
	p.read()
	p.read()
	if !unicode.IsSpace(p.ch) {
		p.warnAt(p.pos, "heading declaration possibly missing a space")
		p.setPos(backupPos)
		return nil, false
	}
	p.read()
	// read until end of line for the text
	var buff strings.Builder
LOOP:
	for {
		switch p.ch {
		case '\n':
			p.read()
			break LOOP
		case 0:
			p.warnAt(p.pos, "unexpected EoF")
			break LOOP
		default:
			buff.WriteRune(p.ch)
		}
		p.read()
	}
	text := strings.TrimSpace(buff.String())
	if text == "" {
		p.warnAt(p.pos, "heading missing text")
		p.setPos(backupPos)
		return nil, false
	}
	node := processText(text)
	// TODO: implement this
	// p.meta["title"]= node.BareText()
	return &ast.Heading{
		Level: 1,
		Title: node,
	}, true
}

func (p *Parser) readPlainText() *ast.TextBlock {
	// NOTE: this func should eat any special character if it's the first iteration
	// the reason being, this func should only be called as a fallback
	var (
		backupPos = p.pos
		text      string
		buff      strings.Builder
		items     []ast.TextNode
	)

LOOP:
	for {
		switch p.ch {
		case '[':
			node, ok := p.linkAhead()
			text = strings.TrimSpace(buff.String())
			if text != "" {
				items = append(items, processText(text))
			}
			buff.Reset()
			if ok {
				items = append(items, node)
			} else if backupPos == p.pos {
				buff.WriteRune(p.ch)
			} else {
				break LOOP
			}
		case '{', '#':
			if p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.isStartOfLine() {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '=':
			if p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.isStartOfLine() && p.peek() == '#' {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '|':
			if p.isStartOfLine() && p.peek() != '\n' && unicode.IsSpace(p.peek()) {
				text = strings.TrimSpace(buff.String())
				buff.Reset()
				if text != "" {
					items = append(items, processText(text))
				}
				p.read()
				text = strings.TrimSpace(p.readLineRest())
				if text != "" {
					items = append(items, &ast.BlockQuote{
						Text: processText(text),
					})
				}
			} else {
				buff.WriteRune(p.ch)
			}
		case 0:
			text = strings.TrimSpace(buff.String())
			if text != "" {
				items = append(items, processText(text))
			}
			break LOOP
		default:
			buff.WriteRune(p.ch)
		}
		p.read()
	}
	return &ast.TextBlock{
		Items: items,
	}
}

