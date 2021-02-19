package parser

type Parser struct {
	doc []rune
	ch, peek rune
	pos, readpos int
	errors, warnings []string
}

func New(s string) *Parser{
	p:= &Parser{doc: []rune(s)}
	p.read()
	return p
}

func(p *Parser) readCode(delim rune) ast.Node{
	var buff strings.Builder
	for i:=0; i<3; i++{
		if p.ch != delim{
			return &ast.Error{}
		}
		p.read()
	}
	LOOP:
	for{
		switch p.ch{
			case delim:
			if p.peek() == delim && p.peekN(2) == delim{
				break LOOP
			}
			buff.WriteRune(p.ch)
			case 0:
			return newError("code block not terminated")
			default:
			buff.WriteRune(p.ch)
		}
		p.read()
	}
	p.read()
	p.read()
	p.read()
	return &ast.Code{
		Text: buff.String()
	}
}

func(p *Parser) linkAhead() (*ast.Anchor, bool){
	if p.ch != '[' {
		return nil, false
	}
	var text, href strings.Builder
	var char rune
	var idx int
	for i:= p.readpos; i< len(p.doc); i++{
		char= p.doc[i]
		if char== '\n' {
			return nil, false
		}
		if char== ']' {
			idx= i+1
			break
		}
		text.WriteRune(char)
	}
	if unicode.IsSpace(p.doc[idx]) {
		return nil, false
	}
	for i:= idx; i< len(p.doc); i++{
		char= p.doc[i]
		if unicode.IsSpace(char) {
			p.setPos(i)
			break
		}
		href.WriteRune(char)
	}
	return &ast.Anchor{
		Text: processText(text.String()),
		URL: href.String(),
	}, true
}

func(p *Parser) headingAhead() (*ast.Heading, bool) {
	if p.ch!= '#' || !p.isStartOfLine() {
		return nil, false
	}
	var char rune
	var idx, level int
	var buff strings.Builder
	for i:= p.pos; i<len(p.doc); i++{
		char= p.doc[i]
		if unicode.IsSpace(char) && char!= '\n' {
			idx= i+ 1 // set the cursor to the text
			break
		}
		if char== '\n' {
			p.warnat(i, "heading possibly missing title")
			return nil, false
		}
		level++
	}
	if idx >= len(p.doc) {
		p.warnat(idx, "line doesn2t make sense")
		return nil, false
	}else if p.doc[idx] == '\n' {
		p.warnat(idx, "heading possibly missing title")
		return nil, false
	}
	if level> 6{
		level= 6
		p.warnAt(idx, "too many '#' for a heading, maximum is 6")
	}
	for i:= idx; i<len(p.doc); i++{
		char= p.doc[i]
		if char== '\n' {
			p.setPos(i)
			break
		}
		buff.writeRune(char)
	}
	return &ast.Heading{
		Level: level,
		Text: processText(buff.String()),
	}, true
}

func(p *Parser) tableAhead() (*ast.Table, bool) {
	backupPos:= p.pos // to revert back to in case of faulty typeup file
	if p.ch != '#' || !p.isStartOfLine() {
		return nil, false
	}
	if p.lineLastChar() != '{' {
		return nil, false
	}
	var delim string
	lBraceN:=0
	var char rune
	pos:= p.readpos
	for i:=p.readpos; i< len(p.doc); i++{
		char= p.doc[i]
		if char== '{' {
			pos= i + 1
			break
		}
		if char == '#' || char== '}' {
			p.warnAt(i, "%c is not allowed as a table delimiter", char)
			return nil, false
		}
		if char== '\n' {
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
	for i:= pos; i< len(p.doc); i++{
		char= p.doc[i]
		if char== '\n' {
			// it's good, jump over to the next line
			pos= i + 1
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
	delim= strings.TrimSpace(delim)
	if delim == ""{
		p.warnAt(pos - 1, "the table delimiter can't be empty")
		return nil, false
	}
	// prepare to read the table elements
	p.setPos(pos)
	// read until we find the closing brace
	var lines []string
	var buff strings.Builder
	for{
		if p.ch== 0 {
			p.warnAt(p.pos, "unexpected EoF in table")
			p.setPos(backupPos)
			return nil, false
		}
		if p.ch== '\n' {
			lines= append(lines, buff.String())
			buff.Reset()
		}
	if p.ch== '}' && p.onlyCharInLine(p.ch) {
		// table done, break out
		p.read()
		break
	}
		p.read()
	}
	if len(lines) == 0{
		p.warnAt(p.pos, "table with no elements ignored")
		p.setPos(backupPos)
		return nil, false
	}
	// TODO(note): do this in place
	return parseTable(lines, delim), true
}

func(p *Parser) ulAhead() (*ast.UnorderedList, bool) {
	if p.ch!= '[' {
		return nil, false
	}
	if !p.lineOnlyCharIs('[') {
		return nil, false
	}
	backupPos:= p.pos
	// consume the line and get to the first element
	for p.ch!= 0 && p.ch!= '\n' {
		p.read()
	}
	if p.ch== 0{
		p.warnAt(backupPos, "stray '['")
		p.setPos(backupPos)
		return nil, false
	}
	p.read()
	var(
	ln string
	buff strings.Builder
	items []ast.ListItem
	)
	LOOP:
	for{
		switch p.ch{
			case '[':
			if p.lineOnlyCharIs(p.ch) {
				if item, ok:= p.ulAhead(); ok{
					items= append(items, item)
				}else{
					buff.WriteRune(p.ch)
				}
				case '{':
				if item, ok:= p.olAhead(); ok{
					items= append(items, item)
				}else{
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
			}
			case '\n':
			ln= buff.String()
			buff.Reset()
			if !isEmpty(ln){
				items= append(items, processText(ln))
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

func(p *Parser) olAhead() (*ast.OrderedList, bool) {
	if p.ch!= '{' {
		return nil, false
	}
	if !p.lineOnlyCharIs('{') {
		return nil, false
	}
	backupPos:= p.pos
	// consume the line and get to the first element
	for p.ch!= 0 && p.ch!= '\n' {
		p.read()
	}
	if p.ch== 0{
		p.warnAt(backupPos, "stray '{'")
		p.setPos(backupPos)
		return nil, false
	}
	p.read()
	var(
	ln string
	buff strings.Builder
	items []ast.ListItem
	)
	LOOP:
	for{
		switch p.ch{
			case '[':
			if p.lineOnlyCharIs(p.ch) {
				if item, ok:= p.ulAhead(); ok{
					items= append(items, item)
				}else{
					buff.WriteRune(p.ch)
				}
				case '{':
				if item, ok:= p.olAhead(); ok{
					items= append(items, item)
				}else{
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
			}
			case '\n':
			ln= buff.String()
			buff.Reset()
			if !isEmpty(ln){
				items= append(items, processText(ln))
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

