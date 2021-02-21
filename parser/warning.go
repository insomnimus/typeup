package parser

import "fmt"

type Warning struct {
	doc    []rune
	pos    int
	format string
	args   []interface{}
}

func (w *Warning) String() string {
	if w.doc == nil ||
		w.pos < 0 ||
		w.pos >= len(w.doc) {
		return fmt.Sprintf(w.format, w.args...)
	}
	ln := 0
	for i, c := range w.doc {
		if c == '\n' {
			ln++
		}
		if i == w.pos {
			break
		}
	}
	w.args = append([]interface{}{ln}, w.args...)
	return fmt.Sprintf("line %d: "+w.format, w.args...)
}

func (p *Parser) warnAt(pos int, format string, args ...interface{}) {
	p.warnings = append(p.warnings, &Warning{
		doc:    p.doc,
		format: format,
		args:   args,
		pos:    pos,
	})
}

func (p *Parser) Warnings() []*Warning {
	return p.warnings
}
