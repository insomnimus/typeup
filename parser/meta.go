package parser

func (p *Parser) Meta(key string) (string, bool) {
	val, ok := p.meta[key]
	return val, ok
}
