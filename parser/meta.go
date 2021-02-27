package parser

func (p *Parser) Metas() map[string]string {
	return p.meta
}

func (p *Parser) Meta(key string) (string, bool) {
	val, ok := p.meta[key]
	return val, ok
}
