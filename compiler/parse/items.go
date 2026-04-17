package parse

import (
	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// parseFile is the grammar entry rule. It consumes imports followed by items
// until EOF. Individual parse failures are recovered at the item boundary so
// one bad declaration does not hide the rest of the file.
func (p *parser) parseFile() *ast.File {
	start := p.cur()
	f := &ast.File{Filename: p.filename}

	for !p.check(lex.TokEOF) {
		before := p.pos
		if p.check(lex.TokKwImport) {
			if imp := p.parseImport(); imp != nil {
				f.Imports = append(f.Imports, imp)
			} else {
				p.synchronize()
			}
		} else if item := p.parseItem(); item != nil {
			f.Items = append(f.Items, item)
		} else {
			p.synchronize()
		}
		// Forcing-progress guard: synchronize() stops at `}` without
		// consuming it so inner block parsers can resume, but at file
		// level no outer block is waiting. If nothing was consumed this
		// iteration, advance one token so malformed input still
		// terminates (TestNopanicOnMalformed).
		if p.pos == before {
			p.advance()
		}
	}
	f.Span = p.spanFrom(start)
	return f
}

func (p *parser) parseImport() *ast.Import {
	startTok, _ := p.expect(lex.TokKwImport, "import declaration")
	imp := &ast.Import{}

	first, ok := p.expect(lex.TokIdent, "import path")
	if !ok {
		return nil
	}
	imp.Path = append(imp.Path, ast.Ident{Span: first.Span, Name: first.Text})
	for p.eat(lex.TokDot) {
		seg, ok := p.expect(lex.TokIdent, "import path segment")
		if !ok {
			return nil
		}
		imp.Path = append(imp.Path, ast.Ident{Span: seg.Span, Name: seg.Text})
	}

	if p.eat(lex.TokKwAs) {
		alias, ok := p.expect(lex.TokIdent, "import alias")
		if !ok {
			return nil
		}
		a := ast.Ident{Span: alias.Span, Name: alias.Text}
		imp.Alias = &a
	}
	p.expect(lex.TokSemi, "import declaration")
	imp.Span = p.spanFrom(startTok)
	return imp
}

// parseItem dispatches on the first token of an item declaration. Decorators
// and visibility prefixes are consumed up-front so the dispatch reflects the
// item's central keyword.
func (p *parser) parseItem() ast.Item {
	if !p.enter() {
		p.leave()
		return nil
	}
	defer p.leave()

	start := p.cur()
	decorators := p.parseDecorators()
	vis := p.parseVisibility()

	// `const fn` is a function; a bare `const` is a const-decl.
	switch p.cur().Kind {
	case lex.TokKwConst:
		if p.peek(1).Kind == lex.TokKwFn {
			return p.parseFnDecl(start, decorators, vis, true, false)
		}
		return p.parseConstDecl(start, vis, decorators)
	case lex.TokKwStatic:
		return p.parseStaticDecl(start, vis, false, decorators)
	case lex.TokKwType:
		return p.parseTypeDecl(start, vis)
	case lex.TokKwFn:
		return p.parseFnDecl(start, decorators, vis, false, false)
	case lex.TokKwStruct:
		return p.parseStructDecl(start, decorators, vis)
	case lex.TokKwEnum:
		return p.parseEnumDecl(start, decorators, vis)
	case lex.TokKwTrait:
		return p.parseTraitDecl(start, vis)
	case lex.TokKwImpl:
		if len(decorators) > 0 {
			p.errorAt(decorators[0].NodeSpan(),
				"decorators are not permitted on `impl` blocks",
				"move the decorator to the items inside the impl body")
		}
		if vis != ast.VisPrivate {
			p.errorAt(start.Span,
				"visibility is not permitted on `impl` blocks",
				"remove the `pub`; members of the impl carry their own visibility")
		}
		return p.parseImplDecl(start)
	case lex.TokKwExtern:
		if len(decorators) > 0 {
			p.errorAt(decorators[0].NodeSpan(),
				"decorators on extern items are handled per-item, not on the `extern` keyword",
				"move the decorator to the extern function or static it describes")
		}
		return p.parseExternDecl(start, vis)
	case lex.TokIdent:
		if p.cur().Text == "union" {
			return p.parseUnionDecl(start, decorators, vis)
		}
	}
	p.errorAt(p.cur().Span,
		"expected an item declaration (fn, struct, enum, trait, impl, const, static, type, extern, union, import)",
		"start the declaration with one of the listed keywords")
	return nil
}

func (p *parser) parseDecorators() []*ast.Decorator {
	var out []*ast.Decorator
	for p.check(lex.TokAt) {
		d := p.parseOneDecorator()
		if d == nil {
			return out
		}
		out = append(out, d)
	}
	return out
}

func (p *parser) parseOneDecorator() *ast.Decorator {
	start, _ := p.expect(lex.TokAt, "decorator")
	nameTok, ok := p.expect(lex.TokIdent, "decorator name")
	if !ok {
		return nil
	}
	d := &ast.Decorator{Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text}}
	if p.eat(lex.TokLParen) {
		for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
			arg := p.parseDecoratorArg()
			if arg != nil {
				d.Args = append(d.Args, arg)
			}
			if !p.eat(lex.TokComma) {
				break
			}
		}
		p.expect(lex.TokRParen, "decorator argument list")
	}
	d.Span = p.spanFrom(start)
	return d
}

func (p *parser) parseDecoratorArg() *ast.DecoratorArg {
	start := p.cur()
	// `IDENT = expr` named form.
	if p.check(lex.TokIdent) && p.peek(1).Kind == lex.TokEq {
		nameTok := p.advance()
		p.advance() // `=`
		expr := p.parseExpr()
		if expr == nil {
			return nil
		}
		n := ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
		a := &ast.DecoratorArg{Name: &n, Value: expr}
		a.Span = p.spanFrom(start)
		return a
	}
	// Positional — either a bare IDENT (flag like `packed`) or any expr.
	expr := p.parseExpr()
	if expr == nil {
		return nil
	}
	a := &ast.DecoratorArg{Value: expr}
	a.Span = p.spanFrom(start)
	return a
}

func (p *parser) parseVisibility() ast.Visibility {
	if !p.check(lex.TokKwPub) {
		return ast.VisPrivate
	}
	p.advance()
	if !p.eat(lex.TokLParen) {
		return ast.VisPub
	}
	if p.check(lex.TokKwMod) {
		p.advance()
		p.expect(lex.TokRParen, "`pub(mod)` visibility")
		return ast.VisPubMod
	}
	if p.check(lex.TokIdent) && p.cur().Text == "pkg" {
		p.advance()
		p.expect(lex.TokRParen, "`pub(pkg)` visibility")
		return ast.VisPubPkg
	}
	p.errorAt(p.cur().Span,
		"expected `mod` or `pkg` inside `pub(...)`",
		"valid forms are `pub`, `pub(mod)`, and `pub(pkg)`")
	// Try to recover to the closing paren.
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		p.advance()
	}
	p.eat(lex.TokRParen)
	return ast.VisPub
}

func (p *parser) parseFnDecl(start lex.Token, decorators []*ast.Decorator, vis ast.Visibility, isConst, isExtern bool) *ast.FnDecl {
	if isConst {
		p.expect(lex.TokKwConst, "const fn declaration")
	}
	p.expect(lex.TokKwFn, "function declaration")

	fn := &ast.FnDecl{
		Decorators: decorators,
		Vis:        vis,
		IsConst:    isConst,
		IsExtern:   isExtern,
	}
	nameTok, ok := p.expect(lex.TokIdent, "function name")
	if ok {
		fn.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	}
	if p.check(lex.TokLBracket) {
		fn.Generics = p.parseGenericParams()
	}
	p.expect(lex.TokLParen, "function parameter list")

	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		if isExtern && p.check(lex.TokIdent) && p.cur().Text == "..." {
			// ... as variadic would be a token pattern; reserved here.
		}
		if isExtern && p.check(lex.TokDotDot) && p.peek(1).Kind == lex.TokDot {
			// Represent `...` as DotDot+Dot from the lexer; accept as variadic.
			p.advance()
			p.advance()
			fn.Variadic = true
			break
		}
		param := p.parseParam()
		if param == nil {
			break
		}
		fn.Params = append(fn.Params, param)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "function parameter list")

	if p.eat(lex.TokArrow) {
		fn.Return = p.parseType()
	}
	if p.check(lex.TokKwWhere) {
		fn.Where = p.parseWhereClause()
	}
	if isExtern {
		p.expect(lex.TokSemi, "extern function declaration")
	} else {
		fn.Body = p.parseBlockExpr()
	}
	fn.Span = p.spanFrom(start)
	return fn
}

func (p *parser) parseParam() *ast.Param {
	start := p.cur()
	own := ast.OwnNone
	switch p.cur().Kind {
	case lex.TokKwRef:
		own = ast.OwnRef
		p.advance()
	case lex.TokKwMutref:
		own = ast.OwnMutref
		p.advance()
	case lex.TokKwOwned:
		own = ast.OwnOwned
		p.advance()
	}
	var nameTok lex.Token
	var ok bool
	if p.check(lex.TokKwSelfVal) {
		nameTok = p.advance()
		ok = true
	} else {
		nameTok, ok = p.expect(lex.TokIdent, "parameter name")
	}
	if !ok {
		return nil
	}
	// `self` may omit the `: Self` annotation — this is the standard receiver
	// shorthand matching reference §9 method examples. For all other names
	// the type annotation is required.
	var ty ast.Type
	if nameTok.Kind == lex.TokKwSelfVal && !p.check(lex.TokColon) {
		// leave ty nil; resolver fills `Self` type from the enclosing impl
	} else {
		p.expect(lex.TokColon, "parameter type annotation")
		ty = p.parseType()
	}
	pr := &ast.Param{
		Ownership: own,
		Name:      ast.Ident{Span: nameTok.Span, Name: nameTok.Text},
		Type:      ty,
	}
	pr.Span = p.spanFrom(start)
	return pr
}

func (p *parser) parseGenericParams() []*ast.GenericParam {
	p.expect(lex.TokLBracket, "generic parameter list")
	var out []*ast.GenericParam
	for !p.check(lex.TokRBracket) && !p.check(lex.TokEOF) {
		start := p.cur()
		nameTok, ok := p.expect(lex.TokIdent, "generic parameter name")
		if !ok {
			break
		}
		gp := &ast.GenericParam{
			Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text},
		}
		if p.eat(lex.TokColon) {
			gp.Bounds = p.parseTypeList()
		}
		gp.Span = p.spanFrom(start)
		out = append(out, gp)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRBracket, "generic parameter list")
	return out
}

func (p *parser) parseWhereClause() []*ast.WherePred {
	p.expect(lex.TokKwWhere, "where clause")
	var out []*ast.WherePred
	for {
		start := p.cur()
		target := p.parseType()
		if target == nil {
			break
		}
		wp := &ast.WherePred{Target: target}
		if p.eat(lex.TokColon) {
			wp.Bounds = p.parseTypeList()
		} else if p.eat(lex.TokEq) {
			wp.IsEq = true
			wp.Eq = p.parseType()
		}
		wp.Span = p.spanFrom(start)
		out = append(out, wp)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	return out
}

func (p *parser) parseStructDecl(start lex.Token, decorators []*ast.Decorator, vis ast.Visibility) *ast.StructDecl {
	p.expect(lex.TokKwStruct, "struct declaration")
	s := &ast.StructDecl{Decorators: decorators, Vis: vis}
	nameTok, _ := p.expect(lex.TokIdent, "struct name")
	s.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	if p.check(lex.TokLBracket) {
		s.Generics = p.parseGenericParams()
	}
	switch p.cur().Kind {
	case lex.TokLBrace:
		p.advance()
		s.Fields = p.parseFieldList(lex.TokRBrace)
		p.expect(lex.TokRBrace, "struct body")
	case lex.TokLParen:
		p.advance()
		for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
			ty := p.parseType()
			if ty == nil {
				break
			}
			s.Tuple = append(s.Tuple, ty)
			if !p.eat(lex.TokComma) {
				break
			}
		}
		p.expect(lex.TokRParen, "tuple struct")
		p.expect(lex.TokSemi, "tuple struct")
	case lex.TokSemi:
		p.advance()
		s.IsUnit = true
	default:
		p.errorAt(p.cur().Span,
			"expected `{`, `(`, or `;` after struct name",
			"use `{ fields }`, `(tuple, types);`, or `;` for a unit struct")
	}
	s.Span = p.spanFrom(start)
	return s
}

func (p *parser) parseFieldList(terminator lex.TokenKind) []*ast.Field {
	var out []*ast.Field
	for !p.check(terminator) && !p.check(lex.TokEOF) {
		start := p.cur()
		vis := p.parseVisibility()
		nameTok, ok := p.expect(lex.TokIdent, "field name")
		if !ok {
			break
		}
		p.expect(lex.TokColon, "field type annotation")
		ty := p.parseType()
		f := &ast.Field{
			Vis:  vis,
			Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text},
			Type: ty,
		}
		f.Span = p.spanFrom(start)
		out = append(out, f)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	return out
}

func (p *parser) parseEnumDecl(start lex.Token, decorators []*ast.Decorator, vis ast.Visibility) *ast.EnumDecl {
	p.expect(lex.TokKwEnum, "enum declaration")
	e := &ast.EnumDecl{Decorators: decorators, Vis: vis}
	nameTok, _ := p.expect(lex.TokIdent, "enum name")
	e.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	if p.check(lex.TokLBracket) {
		e.Generics = p.parseGenericParams()
	}
	p.expect(lex.TokLBrace, "enum body")
	for !p.check(lex.TokRBrace) && !p.check(lex.TokEOF) {
		v := p.parseVariant()
		if v == nil {
			break
		}
		e.Variants = append(e.Variants, v)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRBrace, "enum body")
	e.Span = p.spanFrom(start)
	return e
}

func (p *parser) parseVariant() *ast.Variant {
	start := p.cur()
	nameTok, ok := p.expect(lex.TokIdent, "enum variant name")
	if !ok {
		return nil
	}
	v := &ast.Variant{Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text}}
	switch p.cur().Kind {
	case lex.TokEq:
		p.advance()
		v.Explicit = p.parseExpr()
	case lex.TokLParen:
		p.advance()
		for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
			ty := p.parseType()
			if ty == nil {
				break
			}
			v.Tuple = append(v.Tuple, ty)
			if !p.eat(lex.TokComma) {
				break
			}
		}
		p.expect(lex.TokRParen, "tuple variant")
	case lex.TokLBrace:
		p.advance()
		v.Fields = p.parseFieldList(lex.TokRBrace)
		p.expect(lex.TokRBrace, "struct variant")
	}
	v.Span = p.spanFrom(start)
	return v
}

func (p *parser) parseTraitDecl(start lex.Token, vis ast.Visibility) *ast.TraitDecl {
	p.expect(lex.TokKwTrait, "trait declaration")
	td := &ast.TraitDecl{Vis: vis}
	nameTok, _ := p.expect(lex.TokIdent, "trait name")
	td.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	if p.check(lex.TokLBracket) {
		td.Generics = p.parseGenericParams()
	}
	if p.eat(lex.TokColon) {
		td.Supertrs = p.parseTypeList()
	}
	if p.check(lex.TokKwWhere) {
		td.Where = p.parseWhereClause()
	}
	p.expect(lex.TokLBrace, "trait body")
	for !p.check(lex.TokRBrace) && !p.check(lex.TokEOF) {
		item := p.parseTraitItem()
		if item != nil {
			td.Items = append(td.Items, item)
		} else {
			p.synchronize()
		}
	}
	p.expect(lex.TokRBrace, "trait body")
	td.Span = p.spanFrom(start)
	return td
}

func (p *parser) parseTraitItem() ast.Item {
	start := p.cur()
	decorators := p.parseDecorators()
	switch p.cur().Kind {
	case lex.TokKwFn:
		return p.parseTraitFnItem(start, decorators)
	case lex.TokKwConst:
		if p.peek(1).Kind == lex.TokKwFn {
			return p.parseFnDecl(start, decorators, ast.VisPrivate, true, false)
		}
		return p.parseTraitConstItem(start)
	case lex.TokKwType:
		return p.parseTraitTypeItem(start)
	}
	p.errorAt(p.cur().Span,
		"expected `fn`, `type`, or `const` inside a trait body",
		"trait items are function signatures, associated types, or associated constants")
	return nil
}

func (p *parser) parseTraitFnItem(start lex.Token, decorators []*ast.Decorator) *ast.FnDecl {
	// Trait functions may end with `;` (no body) or `{ body }`.
	p.expect(lex.TokKwFn, "trait function signature")
	fn := &ast.FnDecl{Decorators: decorators}
	nameTok, _ := p.expect(lex.TokIdent, "trait function name")
	fn.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	if p.check(lex.TokLBracket) {
		fn.Generics = p.parseGenericParams()
	}
	p.expect(lex.TokLParen, "trait function parameter list")
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		param := p.parseParam()
		if param == nil {
			break
		}
		fn.Params = append(fn.Params, param)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "trait function parameter list")
	if p.eat(lex.TokArrow) {
		fn.Return = p.parseType()
	}
	if p.check(lex.TokKwWhere) {
		fn.Where = p.parseWhereClause()
	}
	if p.check(lex.TokLBrace) {
		fn.Body = p.parseBlockExpr()
	} else {
		p.expect(lex.TokSemi, "trait function signature")
	}
	fn.Span = p.spanFrom(start)
	return fn
}

func (p *parser) parseTraitTypeItem(start lex.Token) *ast.TraitTypeItem {
	p.expect(lex.TokKwType, "trait associated type")
	nameTok, _ := p.expect(lex.TokIdent, "associated type name")
	ti := &ast.TraitTypeItem{Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text}}
	if p.eat(lex.TokColon) {
		ti.Bounds = p.parseTypeList()
	}
	p.expect(lex.TokSemi, "trait associated type")
	ti.Span = p.spanFrom(start)
	return ti
}

func (p *parser) parseTraitConstItem(start lex.Token) *ast.TraitConstItem {
	p.expect(lex.TokKwConst, "trait associated const")
	nameTok, _ := p.expect(lex.TokIdent, "associated const name")
	tc := &ast.TraitConstItem{Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text}}
	p.expect(lex.TokColon, "trait associated const type")
	tc.Type = p.parseType()
	if p.eat(lex.TokEq) {
		tc.Default = p.parseExpr()
	}
	p.expect(lex.TokSemi, "trait associated const")
	tc.Span = p.spanFrom(start)
	return tc
}

func (p *parser) parseImplDecl(start lex.Token) *ast.ImplDecl {
	p.expect(lex.TokKwImpl, "impl block")
	impl := &ast.ImplDecl{}
	if p.check(lex.TokLBracket) {
		impl.Generics = p.parseGenericParams()
	}
	impl.Target = p.parseType()
	if p.eat(lex.TokColon) {
		// `impl Target : Trait` — per grammar, the body's type is Target and
		// the trait is after the colon. Swap to the Rust-like order.
		impl.Trait = impl.Target
		impl.Target = p.parseType()
	}
	if p.check(lex.TokKwWhere) {
		impl.Where = p.parseWhereClause()
	}
	p.expect(lex.TokLBrace, "impl body")
	for !p.check(lex.TokRBrace) && !p.check(lex.TokEOF) {
		item := p.parseImplItem()
		if item != nil {
			impl.Items = append(impl.Items, item)
		} else {
			p.synchronize()
		}
	}
	p.expect(lex.TokRBrace, "impl body")
	impl.Span = p.spanFrom(start)
	return impl
}

func (p *parser) parseImplItem() ast.Item {
	start := p.cur()
	decorators := p.parseDecorators()
	vis := p.parseVisibility()
	switch p.cur().Kind {
	case lex.TokKwFn:
		return p.parseFnDecl(start, decorators, vis, false, false)
	case lex.TokKwConst:
		if p.peek(1).Kind == lex.TokKwFn {
			return p.parseFnDecl(start, decorators, vis, true, false)
		}
		return p.parseConstDecl(start, vis, decorators)
	case lex.TokKwType:
		return p.parseImplTypeItem(start)
	}
	p.errorAt(p.cur().Span,
		"expected `fn`, `const`, or `type` in impl body",
		"impl items are methods, associated constants, or associated type bindings")
	return nil
}

func (p *parser) parseImplTypeItem(start lex.Token) *ast.ImplTypeItem {
	p.expect(lex.TokKwType, "impl associated type")
	nameTok, _ := p.expect(lex.TokIdent, "impl associated type name")
	ti := &ast.ImplTypeItem{Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text}}
	p.expect(lex.TokEq, "impl associated type")
	ti.Target = p.parseType()
	p.expect(lex.TokSemi, "impl associated type")
	ti.Span = p.spanFrom(start)
	return ti
}

func (p *parser) parseConstDecl(start lex.Token, vis ast.Visibility, decorators []*ast.Decorator) *ast.ConstDecl {
	p.expect(lex.TokKwConst, "const declaration")
	c := &ast.ConstDecl{Decorators: decorators, Vis: vis}
	nameTok, _ := p.expect(lex.TokIdent, "const name")
	c.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	p.expect(lex.TokColon, "const type annotation")
	c.Type = p.parseType()
	p.expect(lex.TokEq, "const initializer")
	c.Value = p.parseExpr()
	p.expect(lex.TokSemi, "const declaration")
	c.Span = p.spanFrom(start)
	return c
}

func (p *parser) parseStaticDecl(start lex.Token, vis ast.Visibility, isExtern bool, decorators []*ast.Decorator) *ast.StaticDecl {
	p.expect(lex.TokKwStatic, "static declaration")
	s := &ast.StaticDecl{Decorators: decorators, Vis: vis, IsExtern: isExtern}
	nameTok, _ := p.expect(lex.TokIdent, "static name")
	s.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	p.expect(lex.TokColon, "static type annotation")
	s.Type = p.parseType()
	if !isExtern {
		p.expect(lex.TokEq, "static initializer")
		s.Value = p.parseExpr()
	}
	p.expect(lex.TokSemi, "static declaration")
	s.Span = p.spanFrom(start)
	return s
}

func (p *parser) parseTypeDecl(start lex.Token, vis ast.Visibility) *ast.TypeDecl {
	p.expect(lex.TokKwType, "type alias")
	td := &ast.TypeDecl{Vis: vis}
	nameTok, _ := p.expect(lex.TokIdent, "type alias name")
	td.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	if p.check(lex.TokLBracket) {
		td.Generics = p.parseGenericParams()
	}
	p.expect(lex.TokEq, "type alias initializer")
	td.Target = p.parseType()
	p.expect(lex.TokSemi, "type alias")
	td.Span = p.spanFrom(start)
	return td
}

func (p *parser) parseExternDecl(start lex.Token, vis ast.Visibility) *ast.ExternDecl {
	p.expect(lex.TokKwExtern, "extern declaration")
	e := &ast.ExternDecl{}
	switch p.cur().Kind {
	case lex.TokKwFn:
		e.Item = p.parseFnDecl(start, nil, vis, false, true)
	case lex.TokKwStatic:
		e.Item = p.parseStaticDecl(start, vis, true, nil)
	default:
		p.errorAt(p.cur().Span,
			"expected `fn` or `static` after `extern`",
			"extern declarations name an FFI function signature or static binding")
		return nil
	}
	e.Span = p.spanFrom(start)
	return e
}

func (p *parser) parseUnionDecl(start lex.Token, decorators []*ast.Decorator, vis ast.Visibility) *ast.UnionDecl {
	// `union` is matched as an identifier at the dispatch layer; consume it.
	p.advance()
	u := &ast.UnionDecl{Decorators: decorators, Vis: vis}
	nameTok, _ := p.expect(lex.TokIdent, "union name")
	u.Name = ast.Ident{Span: nameTok.Span, Name: nameTok.Text}
	if p.check(lex.TokLBracket) {
		u.Generics = p.parseGenericParams()
	}
	p.expect(lex.TokLBrace, "union body")
	u.Fields = p.parseFieldList(lex.TokRBrace)
	p.expect(lex.TokRBrace, "union body")
	u.Span = p.spanFrom(start)
	return u
}
