package parse

import (
	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// parseType parses a single type expression (reference grammar `type_expr`).
func (p *parser) parseType() ast.Type {
	if !p.enter() {
		p.leave()
		return nil
	}
	defer p.leave()

	start := p.cur()
	switch p.cur().Kind {
	case lex.TokLParen:
		return p.parseParenOrTupleOrUnitType(start)
	case lex.TokLBracket:
		return p.parseArrayOrSliceType(start)
	case lex.TokKwFn:
		return p.parseFnType(start)
	case lex.TokIdent:
		// `dyn` is a contextual keyword in type position (reference §1.9 does
		// not list it, so the lexer emits it as an identifier — the parser
		// recognizes it here).
		if p.cur().Text == "dyn" {
			return p.parseDynType(start)
		}
		// `Ptr[T]` is a path-start that we special-case because the grammar
		// names it explicitly.
		if p.cur().Text == "Ptr" && p.peek(1).Kind == lex.TokLBracket {
			return p.parsePtrType(start)
		}
		return p.parsePathType(start)
	case lex.TokKwSelfType:
		// `Self` is reserved (reference §1.9) but appears in type position as
		// a path root.
		return p.parsePathType(start)
	case lex.TokKwImpl:
		return p.parseImplType(start)
	}
	p.errorAt(p.cur().Span,
		"expected a type expression",
		"valid type starts are identifiers, `(`, `[`, `fn`, `dyn`, `impl`, or `Ptr[...]`")
	return nil
}

func (p *parser) parseParenOrTupleOrUnitType(start lex.Token) ast.Type {
	p.advance() // `(`
	if p.eat(lex.TokRParen) {
		ut := &ast.UnitType{}
		ut.Span = p.spanFrom(start)
		return ut
	}
	first := p.parseType()
	if p.eat(lex.TokRParen) {
		// A parenthesized single type is not a tuple. We return it as-is
		// (parentheses disappear at the AST level for types).
		return first
	}
	if !p.eat(lex.TokComma) {
		p.expect(lex.TokRParen, "type expression")
		return first
	}
	elems := []ast.Type{first}
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		elems = append(elems, p.parseType())
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "tuple type")
	tt := &ast.TupleType{Elements: elems}
	tt.Span = p.spanFrom(start)
	return tt
}

func (p *parser) parseArrayOrSliceType(start lex.Token) ast.Type {
	p.advance() // `[`
	elem := p.parseType()
	if p.eat(lex.TokSemi) {
		length := p.parseExpr()
		p.expect(lex.TokRBracket, "array type")
		at := &ast.ArrayType{Element: elem, Length: length}
		at.Span = p.spanFrom(start)
		return at
	}
	p.expect(lex.TokRBracket, "slice type")
	st := &ast.SliceType{Element: elem}
	st.Span = p.spanFrom(start)
	return st
}

func (p *parser) parsePtrType(start lex.Token) ast.Type {
	p.advance() // `Ptr`
	p.expect(lex.TokLBracket, "`Ptr[T]`")
	inner := p.parseType()
	p.expect(lex.TokRBracket, "`Ptr[T]`")
	pt := &ast.PtrType{Pointee: inner}
	pt.Span = p.spanFrom(start)
	return pt
}

func (p *parser) parseFnType(start lex.Token) ast.Type {
	p.advance() // `fn`
	p.expect(lex.TokLParen, "fn type parameter list")
	var params []ast.Type
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		params = append(params, p.parseType())
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "fn type parameter list")
	var ret ast.Type
	if p.eat(lex.TokArrow) {
		ret = p.parseType()
	}
	ft := &ast.FnType{Params: params, Return: ret}
	ft.Span = p.spanFrom(start)
	return ft
}

func (p *parser) parseDynType(start lex.Token) ast.Type {
	p.advance() // `dyn`
	first := p.parseType()
	traits := []ast.Type{first}
	for p.eat(lex.TokPlus) {
		traits = append(traits, p.parseType())
	}
	dt := &ast.DynType{Traits: traits}
	dt.Span = p.spanFrom(start)
	return dt
}

func (p *parser) parseImplType(start lex.Token) ast.Type {
	p.advance() // `impl`
	inner := p.parseType()
	it := &ast.ImplType{Trait: inner}
	it.Span = p.spanFrom(start)
	return it
}

func (p *parser) parsePathType(start lex.Token) ast.Type {
	segs := []ast.Ident{}
	var first lex.Token
	if p.check(lex.TokKwSelfType) {
		first = p.advance()
	} else {
		tok, ok := p.expect(lex.TokIdent, "type name")
		if !ok {
			return nil
		}
		first = tok
	}
	segs = append(segs, ast.Ident{Span: first.Span, Name: first.Text})
	for p.check(lex.TokDot) && p.peek(1).Kind == lex.TokIdent {
		p.advance()
		s := p.advance()
		segs = append(segs, ast.Ident{Span: s.Span, Name: s.Text})
	}
	var args []ast.Type
	if p.check(lex.TokLBracket) {
		args = p.parseTypeArgs()
	}
	pt := &ast.PathType{Segments: segs, Args: args}
	pt.Span = p.spanFrom(start)
	return pt
}

func (p *parser) parseTypeArgs() []ast.Type {
	p.expect(lex.TokLBracket, "generic type arguments")
	var out []ast.Type
	for !p.check(lex.TokRBracket) && !p.check(lex.TokEOF) {
		out = append(out, p.parseType())
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRBracket, "generic type arguments")
	return out
}

// parseTypeList parses `T1, T2, ...` without surrounding delimiters — used
// for supertrait bounds, generic-parameter bounds, and `where` clauses.
func (p *parser) parseTypeList() []ast.Type {
	out := []ast.Type{p.parseType()}
	for p.eat(lex.TokPlus) {
		out = append(out, p.parseType())
	}
	return out
}
