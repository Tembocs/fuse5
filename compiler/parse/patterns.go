package parse

import (
	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// parsePattern parses a top-level pattern and assembles or-patterns when
// followed by `|`. Range patterns (`lo..hi`, `lo..=hi`) are detected when the
// first parsed element is an expression followed by `..`/`..=`.
func (p *parser) parsePattern() ast.Pat {
	if !p.enter() {
		p.leave()
		return nil
	}
	defer p.leave()

	start := p.cur()
	first := p.parsePatternAtom(start)
	if first == nil {
		return nil
	}
	// Or-pattern.
	if p.check(lex.TokPipe) {
		alts := []ast.Pat{first}
		for p.eat(lex.TokPipe) {
			alts = append(alts, p.parsePatternAtom(p.cur()))
		}
		or := &ast.OrPat{Alts: alts}
		or.Span = p.spanFrom(start)
		return or
	}
	return first
}

// parsePatternAtom parses one pattern element — everything except the
// or-chain. At-patterns (`name @ pat`) are detected when a bind-shape
// identifier is immediately followed by `@`.
func (p *parser) parsePatternAtom(start lex.Token) ast.Pat {
	switch p.cur().Kind {
	case lex.TokIdent:
		// Wildcard: the identifier `_` is the wildcard pattern per reference
		// grammar (`_` is not a reserved word; see reference §1.9 and the
		// lexer comment).
		if p.cur().Text == "_" {
			tok := p.advance()
			wp := &ast.WildcardPat{}
			wp.Span = tok.Span
			return wp
		}
		return p.parseIdentOrCtorOrAtPat(start)
	case lex.TokKwNone:
		tok := p.advance()
		cp := &ast.CtorPat{Path: []ast.Ident{{Span: tok.Span, Name: tok.Text}}}
		cp.Span = tok.Span
		return cp
	case lex.TokKwSome:
		return p.parseCtorPatTail(start)
	case lex.TokLParen:
		return p.parseTuplePat(start)
	case lex.TokInt, lex.TokFloat, lex.TokString, lex.TokRawString,
		lex.TokCString, lex.TokChar, lex.TokTrue, lex.TokFalse:
		return p.parseLiteralOrRangePat(start)
	case lex.TokMinus:
		// Negative numeric literal as pattern head — used for both literal
		// patterns and range-pattern bounds.
		return p.parseLiteralOrRangePat(start)
	}
	p.errorAt(p.cur().Span,
		"expected a pattern",
		"patterns are literals, identifiers, `_`, tuples `(...)`, constructors `Name { ... }`, or ranges `lo..hi`")
	p.advance()
	return nil
}

func (p *parser) parseIdentOrCtorOrAtPat(start lex.Token) ast.Pat {
	nameTok := p.advance()
	name := ast.Ident{Span: nameTok.Span, Name: nameTok.Text}

	// `name @ pat`
	if p.eat(lex.TokAt) {
		inner := p.parsePattern()
		ap := &ast.AtPat{Name: name, Pattern: inner}
		ap.Span = p.spanFrom(start)
		return ap
	}

	// Path-prefix for constructor pattern: `Foo.Bar` / `Foo.Bar(x)` / `Foo.Bar { .. }`.
	segs := []ast.Ident{name}
	for p.check(lex.TokDot) && p.peek(1).Kind == lex.TokIdent {
		p.advance()
		s := p.advance()
		segs = append(segs, ast.Ident{Span: s.Span, Name: s.Text})
	}

	switch p.cur().Kind {
	case lex.TokLParen:
		return p.parseCtorTuplePatTail(start, segs)
	case lex.TokLBrace:
		return p.parseCtorStructPatTail(start, segs)
	}
	if len(segs) > 1 {
		cp := &ast.CtorPat{Path: segs}
		cp.Span = p.spanFrom(start)
		return cp
	}
	bp := &ast.BindPat{Name: name}
	bp.Span = name.Span
	return bp
}

func (p *parser) parseCtorPatTail(start lex.Token) ast.Pat {
	head := p.advance() // `Some` / constructor keyword
	segs := []ast.Ident{{Span: head.Span, Name: head.Text}}
	for p.check(lex.TokDot) && p.peek(1).Kind == lex.TokIdent {
		p.advance()
		s := p.advance()
		segs = append(segs, ast.Ident{Span: s.Span, Name: s.Text})
	}
	switch p.cur().Kind {
	case lex.TokLParen:
		return p.parseCtorTuplePatTail(start, segs)
	case lex.TokLBrace:
		return p.parseCtorStructPatTail(start, segs)
	}
	cp := &ast.CtorPat{Path: segs}
	cp.Span = p.spanFrom(start)
	return cp
}

func (p *parser) parseCtorTuplePatTail(start lex.Token, segs []ast.Ident) ast.Pat {
	p.advance() // `(`
	var pats []ast.Pat
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		pats = append(pats, p.parsePattern())
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "tuple-constructor pattern")
	cp := &ast.CtorPat{Path: segs, Tuple: pats}
	cp.Span = p.spanFrom(start)
	return cp
}

func (p *parser) parseCtorStructPatTail(start lex.Token, segs []ast.Ident) ast.Pat {
	p.advance() // `{`
	var fields []*ast.FieldPat
	hasRest := false
	for !p.check(lex.TokRBrace) && !p.check(lex.TokEOF) {
		if p.eat(lex.TokDotDot) {
			hasRest = true
			break
		}
		fstart := p.cur()
		nameTok, ok := p.expect(lex.TokIdent, "struct-pattern field name")
		if !ok {
			break
		}
		fp := &ast.FieldPat{Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text}}
		if p.eat(lex.TokColon) {
			fp.Pattern = p.parsePattern()
		}
		fp.Span = p.spanFrom(fstart)
		fields = append(fields, fp)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRBrace, "struct-constructor pattern")
	cp := &ast.CtorPat{Path: segs, Struct: fields, HasRest: hasRest}
	cp.Span = p.spanFrom(start)
	return cp
}

func (p *parser) parseTuplePat(start lex.Token) ast.Pat {
	p.advance() // `(`
	var pats []ast.Pat
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		pats = append(pats, p.parsePattern())
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "tuple pattern")
	tp := &ast.TuplePat{Elements: pats}
	tp.Span = p.spanFrom(start)
	return tp
}

// parseLiteralOrRangePat parses a literal pattern, optionally upgraded to a
// range pattern when `..` / `..=` follows.
func (p *parser) parseLiteralOrRangePat(start lex.Token) ast.Pat {
	lo := p.parseLiteralPatExpr()
	if lo == nil {
		return nil
	}
	if p.check(lex.TokDotDot) || p.check(lex.TokDotDotEq) {
		inclusive := p.check(lex.TokDotDotEq)
		p.advance()
		hi := p.parseLiteralPatExpr()
		rp := &ast.RangePat{Lo: lo, Hi: hi, Inclusive: inclusive}
		rp.Span = p.spanFrom(start)
		return rp
	}
	lit, ok := lo.(*ast.LiteralExpr)
	if ok {
		lp := &ast.LiteralPat{Value: lit}
		lp.Span = lit.NodeSpan()
		return lp
	}
	// Negative literal falls here — wrap in LiteralPat with the unary node
	// inside Value is not allowed (Value is *LiteralExpr only). For W02 we
	// treat negative literal patterns as literal-of-text; the resolver will
	// re-interpret.
	return nil
}

// parseLiteralPatExpr parses a literal or a unary-minus-literal and returns
// it as a LiteralExpr. Range-pattern bounds accept `-42` and that form needs
// a stable AST representation.
func (p *parser) parseLiteralPatExpr() ast.Expr {
	neg := false
	start := p.cur()
	if p.check(lex.TokMinus) {
		p.advance()
		neg = true
	}
	switch p.cur().Kind {
	case lex.TokInt, lex.TokFloat, lex.TokString, lex.TokRawString,
		lex.TokCString, lex.TokChar, lex.TokTrue, lex.TokFalse:
		e := p.parseLiteral().(*ast.LiteralExpr)
		if neg {
			e.Text = "-" + e.Text
			e.Span = lex.Span{
				File:  e.Span.File,
				Start: start.Span.Start,
				End:   e.Span.End,
			}
		}
		return e
	}
	p.errorAt(p.cur().Span,
		"expected a literal in pattern",
		"patterns accept integer, float, string, char, or bool literals")
	return nil
}
