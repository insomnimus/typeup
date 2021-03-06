package parser

import (
	"github.com/insomnimus/typeup/ast"
	"strings"
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
	s = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(s)
	p := &Parser{
		doc:  []rune(s),
		meta: make(map[string]string),
	}
	p.read()
	return p
}

func (p *Parser) Next() ast.Node {
	switch p.ch {
	case '"':
		if node, ok := p.multilineQuoteAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case '@':
		if p.metaAhead() {
			return p.Next()
		}
		return p.readPlainText(true)
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
		return p.readPlainText(true)
	case '=':
		if node, ok := p.headingShortAhead(); ok {
			return node
		}
		if node, ok := p.codeAhead(p.ch); ok {
			return node
		}
		return p.readPlainText(true)
	case '-':
		if node, ok := p.themeBreakAhead(); ok {
			return node
		}
		return p.readPlainText(true)
	case '`':
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
		if p.ignoreAhead() {
			return p.Next()
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
			if p.isStartOfLine() && p.peek() == delim && p.peekN(2) == delim {
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
	var (
		backupPos = p.pos
		buff      strings.Builder
		text      string
	)
	p.read()
	for p.ch != ']' {
		if p.ch == '\n' || p.ch == 0 {
			p.setPos(backupPos)
			return nil, false
		}
		buff.WriteRune(p.ch)
		p.read()
	}

	p.read() // consume the ']'
	text = strings.TrimSpace(buff.String())
	if text == "" {
		p.setPos(backupPos)
		return nil, false
	}
	if strings.Contains(text, "|") {
		var t, href string
		for i := len(text) - 1; i >= 0; i-- {
			if text[i] == '|' {
				t = strings.TrimSpace(text[:i])
				href = strings.TrimSpace(text[i+1:])
				break
			}
		}
		if href != "" {
			if t == "" && href != "" {
				return &ast.Anchor{
					Text: &ast.Text{Text: t},
					URL:  href,
				}, true
			}
			if href != "" {
				return &ast.Anchor{
					Text: processText(t),
					URL:  href,
				}, true
			}
		}
	}
	fields := strings.Fields(text)
	switch len(fields) {
	case 0: // not possible but don2t wanna take chances
		p.setPos(backupPos)
		return nil, false
	case 1:
		return &ast.Anchor{
			Text: &ast.Text{Text: text},
			URL:  text,
		}, true
	default:
		text = strings.Join(fields[:len(fields)-1], " ")
		return &ast.Anchor{
			Text: processText(text),
			URL:  fields[len(fields)-1],
		}, true
	}
}

func (p *Parser) headingAhead() (head *ast.Heading, yes bool) {
	if p.ch != '#' || !p.isStartOfLine() {
		return
	}
	var (
		char       rune
		idx, level int
		buff       strings.Builder
	)
	for i := p.pos; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '{' {
			return
		}
		if char == '\n' {
			p.warnAt(i, "heading possibly missing title")
			return nil, false
		}
		if char != '#' {
			idx = i + 1 // set the cursor to the text
			break
		}

		level++
	}

	if idx >= len(p.doc) {
		p.warnAt(idx, "line doesn't make sense")
		return
	}
	if p.doc[idx] == '\n' {
		p.warnAt(idx, "heading possibly missing title")
		return
	}

	if level > 6 {
		level = 6
		p.warnAt(idx, "too many '#' for a heading, maximum is 6")
	}

	// read the title text
	for i := idx; i < len(p.doc); i++ {
		char = p.doc[i]
		if char == '{' {
			return
		}
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
		case '"':
			if force && p.pos == backupPos {
				buff.WriteRune(p.ch)
			} else if p.isStartOfLine() && p.aheadIs(`"""`) {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '@':
			if p.pos == backupPos && force {
				buff.WriteRune(p.ch)
			} else if p.isStartOfLine() && p.peek() == '{' {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '`':
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
			} else if p.isStartOfLine() && (p.aheadIs("image[") || p.aheadIs("ignore{")) {
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
			} else if p.isStartOfLine() && p.peek() == '[' {
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
			} else if p.isStartOfLine() && p.aheadIs("video[") {
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
				continue LOOP
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
			} else if p.isStartOfLine() && p.peek() == '=' && p.peekN(2) == '=' {
				text = strings.TrimSpace(buff.String())
				if text != "" {
					items = append(items, processText(text))
				}
				break LOOP
			} else {
				buff.WriteRune(p.ch)
			}
		case '|':
			if node, ok := p.blockQuoteAhead(); ok {
				text = strings.TrimSpace(buff.String())
				buff.Reset()
				if text != "" {
					items = append(items, processText(text))
				}
				items = append(items, node)
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
	if !p.isStartOfLine() || !(p.aheadIs("image[") || p.aheadIs("img[")) {
		return nil, false
	}
	var (
		backupPos = p.pos
		buff      strings.Builder
		text      string
	)
	for p.ch != '[' {
		p.read()
	}
	p.read()
	if p.ch == ']' {
		p.warnAt(p.pos, "image missing content")
		p.setPos(backupPos)
		return nil, false
	}

	for p.ch != ']' {
		if p.ch == 0 {
			p.warnAt(p.pos, "image missing ']'")
			p.setPos(backupPos)
			return nil, false
		}
		if p.ch == '\n' {
			p.warnAt(p.pos, "unexpected newline in image")
			p.setPos(backupPos)
			return nil, false
		}
		buff.WriteRune(p.ch)
		p.read()
	}
	p.read()
	text = strings.TrimSpace(buff.String())
	if strings.Contains(text, "|") {
		var alt, href string
		for i := len(text) - 1; i >= 0; i-- {
			if text[i] == '|' {
				text = strings.TrimSpace(text[:i])
				href = strings.TrimSpace(text[i+1:])
				break
			}
		}
		if alt == "" && href == "" {
			p.warnAt(p.pos, "image missing src and alt text")
			p.setPos(backupPos)
			return nil, false
		}
		if alt != "" && href != "" {
			return &ast.Image{Attrs: map[string]string{
				"src": href,
				"alt": alt,
			}}, true
		}
	}

	fields := strings.Fields(text)
	switch len(fields) {
	case 0: // impossible
		p.warnAt(p.pos, "image missing src and alt")
		p.setPos(backupPos)
		return nil, false
	case 1:
		p.warnAt(p.pos, "image missing src attribute")
		p.setPos(backupPos)
		return nil, false
	default:
		return &ast.Image{Attrs: map[string]string{
			"src": fields[len(fields)-1],
			"alt": strings.Join(fields[:len(fields)-1], " "),
		}}, true
	}
}

func (p *Parser) videoAhead() (*ast.Video, bool) {
	if !p.isStartOfLine() || !(p.aheadIs("video[") || p.aheadIs("vid[")) {
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

func (p *Parser) imageShortAhead() (img *ast.Image, yes bool) {
	if p.ch != '!' || !p.isStartOfLine() || p.peek() != '[' {
		return nil, false
	}
	pos := p.pos
	defer func() {
		if !yes {
			p.setPos(pos)
		}
	}()

	var buff strings.Builder
	p.read()
	p.read()
	for p.ch != ']' {
		if p.ch == 0 {
			p.warnAt(p.pos, "unexpected EoF")
			return
		}
		if p.ch == '\n' {
			p.warnAt(p.pos, "possibly image syntax error")
			return
		}
		buff.WriteRune(p.ch)
		p.read()
	}
	p.read()
	text := strings.TrimSpace(buff.String())
	if text == "" {
		p.warnAt(p.pos, "image syntax possibly incorrect")
		return
	}

	if strings.Contains(text, "|") {
		var alt, href string
		// split from the last '|'
		for i := len(text) - 1; i >= 0; i-- {
			if text[i] == '|' {
				alt = strings.TrimSpace(text[:i])
				href = strings.TrimSpace(text[i+1:])
				break
			}
		}
		if href != "" && alt != "" {
			return &ast.Image{Attrs: map[string]string{
				"src": href,
				"alt": alt,
			}}, true
		}
	}

	fields := strings.Fields(text)
	switch len(fields) {
	case 0: // impossible but still
		p.warnAt(pos, "invalid image syntax")
		return
	default:
		return &ast.Image{Attrs: map[string]string{
			"src": fields[len(fields)-1],
			"alt": strings.Join(fields[:len(fields)-1], " "),
		}}, true
	}
}

func (p *Parser) ignoreAhead() bool {
	if !p.isStartOfLine() || !p.aheadIs("ignore") {
		return false
	}
	backupPos := p.pos
	for range "ignore" {
		p.read()
	}
	for p.ch != '{' {
		if p.ch == '\n' || p.ch == 0 {
			p.setPos(backupPos)
			return false
		}
		if !unicode.IsSpace(p.ch) {
			p.setPos(backupPos)
			return false
		}
		p.read()
	}
	p.read()
	for {
		if p.ch == 0 {
			p.setPos(backupPos)
			return false
		}
		if p.ch == '\n' {
			break
		}
		if !unicode.IsSpace(p.ch) {
			p.setPos(backupPos)
			return false
		}
		p.read()
	}

	for {
		if p.ch == '}' && p.lineOnlyCharIs('}') {
			p.read()
			return true
		}
		if p.ch == 0 {
			p.setPos(backupPos)
			return false
		}
		p.read()
	}
}

func (p *Parser) metaAhead() bool {
	if !p.isStartOfLine() || p.ch != '@' || p.peek() != '{' {
		return false
	}
	var (
		buff      strings.Builder
		backupPos = p.pos
		text      string
	)
	p.read()
	p.read()
	if p.ch == '\n' {
		// is multiline
		p.read()
		var lines []string
	LOOP:
		for {
			switch p.ch {
			case '}':
				if p.lineOnlyCharIs(p.ch) {
					p.read()
					break LOOP
				}
				buff.WriteRune(p.ch)
			case '\n':
				text = strings.TrimSpace(buff.String())
				buff.Reset()
				if text != "" {
					lines = append(lines, text)
				}
			case 0:
				p.warnAt(p.pos, "unexpected EoF in multiline meta block")
				p.setPos(backupPos)
				return false
			default:
				buff.WriteRune(p.ch)
			}
			p.read()
		}
		if len(lines) == 0 {
			p.setPos(backupPos)
			p.warnAt(p.pos, "meta block empty")
			return false
		}
		var fields []string
		var key string
		for _, s := range lines {
			fields = strings.SplitN(s, "=", 2)
			if len(fields) != 2 {
				continue
			}
			key = strings.TrimSpace(fields[0])
			if key == "" {
				continue
			}
			p.meta[key] = strings.TrimSpace(fields[1])
		}
		return true
	}
	// means it's a one liner
	for {
		if p.ch == 0 {
			p.warnAt(p.pos, "unexpected EoF in meta block")
			p.setPos(backupPos)
			return false
		}
		if p.ch == '\n' {
			text = strings.TrimSpace(buff.String())
			if text != "" {
				p.warnAt(p.pos, "linebreak in one liner meta block not allowed")
			} else {
				p.warnAt(p.pos, "space after '@{' not allowed in multiline meta blocks")
			}
			p.setPos(backupPos)
			return false
		}
		if p.ch == '}' {
			p.read()
			break
		}
		buff.WriteRune(p.ch)
		p.read()
	}
	text = strings.TrimSpace(buff.String())
	if text == "" || !strings.Contains(text, "=") {
		p.warnAt(backupPos, "invalid meta block syntax")
		p.setPos(backupPos)
		return false
	}
	fields := strings.SplitN(text, "=", 2)
	if len(fields) != 2 {
		p.setPos(backupPos)
		p.warnAt(p.pos, "invalid meta block syntax")
		return false
	}
	key := strings.TrimSpace(fields[0])
	if key == "" {
		p.warnAt(p.pos, "meta key can't be empty")
		p.setPos(backupPos)
		return false
	}
	p.meta[key] = strings.TrimSpace(fields[1])
	return true
}

func (p *Parser) blockQuoteAhead() (*ast.BlockQuote, bool) {
	if p.ch != '|' || !p.isStartOfLine() {
		return nil, false
	}
	if p.peek() == '\n' {
		return nil, false
	}
	var (
		buff      strings.Builder
		backupPos = p.pos
	)
	p.read()
	for {
		if p.ch == 0 {
			break
		}
		if p.ch == '\n' {
			if p.peek() != '|' {
				break
			}
			buff.WriteRune(p.ch)
			p.read()
			p.read()
			continue
		}
		buff.WriteRune(p.ch)
		p.read()
	}
	text := strings.TrimSpace(buff.String())
	if text == "" {
		p.setPos(backupPos)
		p.warnAt(p.pos, "empty block quote")
		return nil, false
	}
	return &ast.BlockQuote{
		Text: processText(text),
	}, true
}

func (p *Parser) multilineQuoteAhead() (*ast.BlockQuote, bool) {
	if !p.isStartOfLine() || p.ch != '"' || p.peek() != '"' || p.peekN(2) != '"' {
		return nil, false
	}
	var (
		backupPos = p.pos
		buff      strings.Builder
	)
	for range `"""` {
		p.read()
	}
	if p.ch != '\n' {
		p.warnAt(p.pos, `no characters allowed after '"""' in multiline block quotes`)
		p.setPos(backupPos)
		return nil, false
	}
	p.read()
	for {
		if p.ch == '"' && p.isStartOfLine() && p.aheadIs(`"""`) {
			p.read()
			p.read()
			p.read()
			if p.ch == '\n' || p.ch == 0 {
				break
			}
			p.warnAt(p.pos, `no characters allowed in the same line after closing '"""' in block quote`)
			buff.WriteString(`"""`)
		}
		if p.ch == 0 {
			p.setPos(backupPos)
			p.warnAt(p.pos, "unexpected EoF in multiline block quote")
			return nil, false
		}
		buff.WriteRune(p.ch)
		p.read()
	}

	text := strings.TrimSpace(buff.String())
	if text == "" {
		p.setPos(backupPos)
		p.warnAt(p.pos, "multiline block quote is empty")
		return nil, false
	}
	return &ast.BlockQuote{
		Text: processText(text),
	}, true
}
