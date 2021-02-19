package parser
var(
regLink= regexp.MustCompile(`$\[([a-zA-Z0-9][^\]]*)([^\s]+)`)
)

func(p *Parser) read() {
	if p.readpos >= len(p.pos) {
		p.ch= 0
	}else{
		p.ch= p.doc[p.readpos]
	}
	p.pos= p.readpos
	p.readpos++
}

func(p *Parser) peek() rune{
	return p.peekN(1)
}

func(p *Parser) peekN(n int) rune{
	if p.pos + n < 0 || p.pos + n >= len(p.doc) {
		return 0
	}
	return p.doc[p.pos+n]
}

func(p *Parser) isStartOfLine() bool{
	if p.pos==0 {
		return true
	}
	return p.doc[p.pos-1] == '\n'
}

func(p *Parser) searchLineUntil(c rune) (string, int) {
	end:= -1
	var char rune
	for i:=p.readpos; i<len(p.doc); i++{
		char= p.doc[i]
		if char == '\n' {
			return "", -1
		}
		if char == c{
			end= i
			break
		}
	}
	return string(p.doc[p.readpos:end]), end
}

func(p *Parser) lineFirstNonSpaceIndex() int{
	var first int
	var ch rune
	for i:= p.pos; i>=0; i--{
		ch= p.pos[i]
		if ch== '\n'{
			return first
		}
		if !unicode.IsSpace(ch) {
			first= i
		}
	}
	
	return first
}

func(p *Parser) lineLastNonSpaceIndex() int{
	var ch rune
	last:= p.pos
	for i:= p.pos; i< len(p.doc); i++{
		ch= p.doc[i]
		if ch== '\n' {
			return last
		}
		if !unicode.IsSpace(ch){
			last= i
		}
	}
	return last
}

func isBold(s string) (string, bool) {
	if len(s) < 3{
		return "", false
	}
	if s[0] == '_' && s[len(s)-1] == '_'{
		return s[1:len(s)-1], true
	}
	return "", false
}

func isItalic(s string) (string, bool) {
	if len(s) < 3{
		return "", false
	}
	if s[0] == '*' && s[len(s)-1] == '*'{
		return s[1:len(s)-1], true
	}
	return "", false
}

func(p *Parser) aheadIs(s string) bool{
	if len(s) + p.pos >= len(p.doc) {
		return false
	}
	return string(p.doc[p.pos:p.pos+len(s)]) == s
}

func(p *Parser) lineOnlyCharIs(c rune) bool{
	if unicode.IsSpace(c) {
		return false
	}
	found:= false
	var char rune
	for i:= p.pos; i>= 0; i--{
		char= p.doc[i]
		if char== '\n' {
			break
		}
		if char== c{
			if found {
				return false
			}
			found= true
			continue
		}
		if !unicode.IsSpace(char) {
			return false
		}
	}
	for i:= p.readpos; i< len(p.doc); i++{
		char= p.doc[i]
		if char== '\n' {
			return found
		}
		if char== c{
			if found {
				return false
			}
			found= true
			continue
		}
		if !unicode.IsSpace(char) {
			return false
		}
	}
	return found
}