package parser

import (
	"strings"
	"typeup/ast"
	"unicode"
)

type Parser struct {
	doc          []rune
	ch           rune
	pos, readpos int
	warnings     []*Warning
	meta         map[string]string
}

func New(s string) *Parser {
	s = strings.Replace(s, "\r", "\n", -1)
	p := &Parser{
		doc:  []rune(s),
		meta: make(map[string]string),
	}
	p.read()
	return p
}

func (p *Parser) Next() ast.Node {
	switch p.ch {
	case '[':
		if node, ok := p.ulAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case '#':
		if node, ok := p.headingAhead(); ok {
			return node
		}
		if node, ok := p.tableAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case '{':
		if node, ok := p.olAhead(); ok {
			return node
		}
		// TODO: implement meta blocks
		return p.readPlainText(true)
	case '=':
		if node, ok := p.headingShortAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case '-':
		if node, ok := p.themeBreakAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case '`', '\'':
		if node, ok := p.codeAhead(p.ch); ok {
			return node
		}
		return p.readPlainText(true)
	case '!':
		if node, ok := p.imageShortAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case 'i':
		if node, ok := p.imageAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case 'v':
		if node, ok := p.videoAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case 0:
		return nil
	default:
		return p.readPlainText(false)
	}
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
	if p.ch != '#' || !p.isStartOfLine() {
		return nil, false
	}
	// get past hash
	p.read()
	var (
		delim     string
		buff      strings.Builder
		backupPos = p.pos
	)
	for p.ch != '{' && p.ch != '\n' && p.ch != 0 {
		buff.WriteRune(p.ch)
		p.read()
	}
	delim = strings.TrimSpace(buff.String())
	buff.Reset()
	if delim == "" {
		p.warnAt(p.pos, "table delimiter empty")
		p.setPos(backupPos)
		return nil, false
	}
	// read until lf
	p.read()
	if p.ch != '\n' {
		p.read()
		for p.ch != '\n' {
			if p.ch == 0 {
				p.warnAt(p.pos, "unexpected eof")
				p.setPos(backupPos)
				return nil, false
			}
			if !unicode.IsSpace(p.ch) {
				p.warnAt(p.pos, "wrong table syntax")
				p.setPos(backupPos)
				return nil, false
			}
			p.read()
		}
		p.read()
	}
	// collect rows
	var rows []string
	var text string
LOOP:
	for {
		switch p.ch {
		case '\n':
			text = strings.TrimSpace(buff.String())
			if text != "" {
				rows = append(rows, text)
			}
			buff.Reset()
		case 0:
			p.warnAt(p.pos, "unexpected EoF in table")
			p.setPos(backupPos)
			return nil, false
		case '}':
			if p.lineOnlyCharIs('}') {
				p.read()
				text = strings.TrimSpace(buff.String())
				if text != "" {
					rows = append(rows, text)
				}
				break LOOP
			}
			buff.WriteRune(p.ch)
		default:
			buff.WriteRune(p.ch)
		}
		p.read()
	}
	if len(rows) == 0 {
		p.warnAt(p.pos, "table is empty")
		p.setPos(backupPos)
		return nil, false
	}

	return parseTable(rows, delim), true
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
	var (
		s     = []rune(source)
		buff  strings.Builder
		ch    rune
		items []ast.TextNode
		text  string
	)

LOOP:
	for i := 0; i < len(s); i++ {
		ch = s[i]
		switch ch {
		// TODO: implement inline code snips
		case '`', '\'':
			if code, pos := hasInlineCode(s, i); pos > i {
				text = strings.TrimSpace(buff.String())
				buff.Reset()
				if text != "" {
					items = append(items, &ast.Text{Text: text})
				}
				items = append(items, code)
				i = pos
			} else {
				buff.WriteRune(ch)
			}
		case '/':
			if (i > 0 && s[i-1] != ':' || i == 0) && (i+1 < len(s) && s[i+1] == ch || i+1 >= len(s)) {
				if italic, pos := hasItalicLong(s, i); pos > i {
					text = strings.TrimSpace(buff.String())
					buff.Reset()
					if text != "" {
						items = append(items, &ast.Text{Text: text})
					}
					items = append(items, italic)
					i = pos
				} else {
					buff.WriteRune(ch)
				}
			} else {
				buff.WriteRune(ch)
			}
		case '=':
			if (i > 0 && unicode.IsSpace(s[i-1]) || i == 0) && (i+1 < len(s) && s[i+1] == ch || i+1 >= len(s)) {
				if bold, pos := hasBoldLong(s, i); pos > i {
					text = strings.TrimSpace(buff.String())
					buff.Reset()
					if text != "" {
						items = append(items, &ast.Text{Text: text})
					}
					items = append(items, bold)
					i = pos
				} else {
					buff.WriteRune(ch)
				}
			} else {
				buff.WriteRune(ch)
			}
		case '[':
			if anchor, pos := hasAnchor(s, i); pos > i {
				text = strings.TrimSpace(buff.String())
				if text != "" {
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
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, &ast.Text{
						Text: text,
					})
				}
				buff.Reset()
				items = append(items, item)
				i = pos
				continue LOOP // maybe remove
			} else {
				buff.WriteRune(ch)
			}
		case '_':
			if item, pos := hasBold(s, i); pos > i {
				text = strings.TrimSpace(buff.String())
				if text != "" {
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
	text = strings.TrimSpace(buff.String())
	if text != "" {
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
	p.meta["title"] = node.Bare()
	return &ast.Heading{
		Level: 1,
		Title: node,
	}, true
}

func (p *Parser) readPlainText(force bool) *ast.TextBlock {
	var (
		backupPos = p.pos
		text      string
		buff      strings.Builder
		items     []ast.TextNode
	)

LOOP:
	for {
		switch p.ch {
		case '`', '\'':
			if force && p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.peek() == p.ch && p.peekN(2) == p.ch && p.isStartOfLine() {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case 'i':
			if force && p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.aheadIs("image[") {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '!':
			if force && p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.peek() == '[' {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case 'v':
			if force && p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.aheadIs("video[") {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '-':
			if force && p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.isStartOfLine() && p.peek() == '-' && p.peekN(2) == '-' {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '[':
			node, ok := p.linkAhead()
			text = strings.TrimSpace(buff.String())
			if text != "" {
				items = append(items, processText(text))
			}
			buff.Reset()
			if ok {
				items = append(items, node)
			} else if backupPos == p.pos && force {
				buff.WriteRune(p.ch)
			} else {
				break LOOP
			}
		case '{', '#':
			if force && p.pos == backupPos {
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
			if force && p.pos == backupPos {
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

func (p *Parser) themeBreakAhead() (*ast.ThemeBreak, bool) {
	if !p.isStartOfLine() ||
		p.ch != '-' ||
		p.peek() != '-' ||
		p.peekN(2) != '-' {
		return nil, false
	}
	backupPos := p.pos
	p.read()
	p.read()
	if !p.isSpaceUntilLF() {
		p.setPos(backupPos)
		p.warnAt(p.pos, "theme break line can only contain '-'")
		return nil, false
	}
	p.read()
	return &ast.ThemeBreak{}, true
}

func (p *Parser) imageAhead() (*ast.Image, bool) {
	if !p.isStartOfLine() || !p.aheadIs("image[") {
		return nil, false
	}
	var (
		backupPos = p.pos
		buff      strings.Builder
		href      string
	)
	for p.ch != '[' {
		p.read()
	}
	alt, pos := p.searchLineUntil(']')
	if pos <= p.pos {
		p.warnAt(p.pos, "image missing ']'")
		p.setPos(backupPos)
		return nil, false
	}
	p.setPos(pos)
	p.read()
	if p.ch == 0 {
		p.warnAt(p.pos, "image url missing")
		p.setPos(backupPos)
		return nil, false
	}
	alt = strings.TrimSpace(alt)
	if alt == "" {
		p.warnAt(p.pos, "image alt text missing")
	}
	for {
		if unicode.IsSpace(p.ch) {
			break
		}
		buff.WriteRune(p.ch)
		p.read()
	}
	href = strings.TrimSpace(buff.String())
	if href == "" {
		p.warnAt(p.pos, "image missing url")
		p.setPos(backupPos)
		return nil, false
	}
	return &ast.Image{Attrs: map[string]string{
		"src": href,
		"alt": alt,
	}}, true
}

func (p *Parser) videoAhead() (*ast.Video, bool) {
	if !p.isStartOfLine() || !p.aheadIs("video[") {
		return nil, false
	}
	backupPos := p.pos
	// read till '['
	for p.ch != '[' {
		p.read()
	}
	href, pos := p.searchLineUntil(']')
	if pos <= p.pos {
		p.warnAt(p.pos, "video missing source url")
		p.setPos(backupPos)
		return nil, false
	}
	p.setPos(pos)
	p.read()
	return &ast.Video{Source: href}, true
}

func (p *Parser) imageShortAhead() (*ast.Image, bool) {
	if p.ch != '!' || !p.isStartOfLine() || p.peek() != '[' {
		return nil, false
	}
	backupPos := p.pos
	var buff strings.Builder
	p.read()
	alt, pos := p.searchLineUntil(']')
	if pos <= p.pos {
		p.setPos(backupPos)
		return nil, false
	}
	alt = strings.TrimSpace(alt)
	if alt == "" {
		p.warnAt(p.pos, "image alt can't be empty")
		p.setPos(backupPos)
		return nil, false
	}
	p.setPos(pos)
	p.read()
	for {
		if unicode.IsSpace(p.ch) {
			break
		}
		buff.WriteRune(p.ch)
		p.read()
	}
	href := strings.TrimSpace(buff.String())
	if href == "" {
		p.warnAt(p.pos, "image source url missing")
		p.setPos(backupPos)
		return nil, false
	}
	return &ast.Image{Attrs: map[string]string{
		"src": href,
		"alt": alt,
	}}, true
}
