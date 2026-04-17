package parse

import (
	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// parseExpr parses an expression (reference grammar `expr` / `assignment_expr`).
// Expressions use precedence climbing per Appendix B. Struct-literal
// disambiguation uses a parse-mode flag that blocks `IDENT {` from starting a
// struct literal when the context forbids it (e.g. the condition of an `if`).
func (p *parser) parseExpr() ast.Expr {
	return p.parseExprCtx(ctxNormal)
}

// parseExprNoStruct parses an expression where `IDENT {` must not start a
// struct literal (reference §10.7). Used for `if`/`while`/`for` conditions
// and `match` scrutinee.
func (p *parser) parseExprNoStruct() ast.Expr {
	return p.parseExprCtx(ctxNoStruct)
}

type parseCtx int

const (
	ctxNormal parseCtx = iota
	ctxNoStruct
)

func (p *parser) parseExprCtx(ctx parseCtx) ast.Expr {
	if !p.enter() {
		p.leave()
		return nil
	}
	defer p.leave()
	return p.parseAssignment(ctx)
}

func (p *parser) parseAssignment(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseLogicOr(ctx)
	if lhs == nil {
		return nil
	}
	op, ok := assignOpFromKind(p.cur().Kind)
	if !ok {
		return lhs
	}
	opTok := p.advance()
	rhs := p.parseAssignment(ctx)
	ae := &ast.AssignExpr{Op: op, OpSpan: opTok.Span, Lhs: lhs, Rhs: rhs}
	ae.Span = p.spanFrom(start)
	return ae
}

func (p *parser) parseLogicOr(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseLogicAnd(ctx)
	for lhs != nil && p.check(lex.TokPipePipe) {
		opTok := p.advance()
		rhs := p.parseLogicAnd(ctx)
		lhs = binaryExpr(start, ast.BinLogOr, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseLogicAnd(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseCompare(ctx)
	for lhs != nil && p.check(lex.TokAmpAmp) {
		opTok := p.advance()
		rhs := p.parseCompare(ctx)
		lhs = binaryExpr(start, ast.BinLogAnd, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseCompare(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseBitOr(ctx)
	for lhs != nil {
		op, ok := compareOpFromKind(p.cur().Kind)
		if !ok {
			break
		}
		opTok := p.advance()
		rhs := p.parseBitOr(ctx)
		lhs = binaryExpr(start, op, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseBitOr(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseBitXor(ctx)
	for lhs != nil && p.check(lex.TokPipe) {
		opTok := p.advance()
		rhs := p.parseBitXor(ctx)
		lhs = binaryExpr(start, ast.BinOr, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseBitXor(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseBitAnd(ctx)
	for lhs != nil && p.check(lex.TokCaret) {
		opTok := p.advance()
		rhs := p.parseBitAnd(ctx)
		lhs = binaryExpr(start, ast.BinXor, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseBitAnd(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseShift(ctx)
	for lhs != nil && p.check(lex.TokAmp) {
		opTok := p.advance()
		rhs := p.parseShift(ctx)
		lhs = binaryExpr(start, ast.BinAnd, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseShift(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseAdditive(ctx)
	for lhs != nil {
		var op ast.BinaryOp
		switch p.cur().Kind {
		case lex.TokShl:
			op = ast.BinShl
		case lex.TokShr:
			op = ast.BinShr
		default:
			return lhs
		}
		opTok := p.advance()
		rhs := p.parseAdditive(ctx)
		lhs = binaryExpr(start, op, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseAdditive(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseMultiplicative(ctx)
	for lhs != nil {
		var op ast.BinaryOp
		switch p.cur().Kind {
		case lex.TokPlus:
			op = ast.BinAdd
		case lex.TokMinus:
			op = ast.BinSub
		default:
			return lhs
		}
		opTok := p.advance()
		rhs := p.parseMultiplicative(ctx)
		lhs = binaryExpr(start, op, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseMultiplicative(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseCast(ctx)
	for lhs != nil {
		var op ast.BinaryOp
		switch p.cur().Kind {
		case lex.TokStar:
			op = ast.BinMul
		case lex.TokSlash:
			op = ast.BinDiv
		case lex.TokPercent:
			op = ast.BinMod
		default:
			return lhs
		}
		opTok := p.advance()
		rhs := p.parseCast(ctx)
		lhs = binaryExpr(start, op, opTok.Span, lhs, rhs, p)
	}
	return lhs
}

func (p *parser) parseCast(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parseUnary(ctx)
	for lhs != nil && p.check(lex.TokKwAs) {
		p.advance()
		ty := p.parseType()
		ce := &ast.CastExpr{Expr: lhs, Type: ty}
		ce.Span = p.spanFrom(start)
		lhs = ce
	}
	return lhs
}

func (p *parser) parseUnary(ctx parseCtx) ast.Expr {
	start := p.cur()
	var op ast.UnaryOp
	var have bool
	switch p.cur().Kind {
	case lex.TokBang:
		op, have = ast.UnNot, true
	case lex.TokMinus:
		op, have = ast.UnNeg, true
	case lex.TokStar:
		op, have = ast.UnDeref, true
	case lex.TokAmp:
		op, have = ast.UnAddr, true
	}
	if have {
		opTok := p.advance()
		inner := p.parseUnary(ctx)
		ue := &ast.UnaryExpr{Op: op, OpSpan: opTok.Span, Operand: inner}
		ue.Span = p.spanFrom(start)
		return ue
	}
	return p.parsePostfix(ctx)
}

func (p *parser) parsePostfix(ctx parseCtx) ast.Expr {
	start := p.cur()
	lhs := p.parsePrimary(ctx)
	for lhs != nil {
		switch p.cur().Kind {
		case lex.TokDot:
			p.advance()
			nameTok, ok := p.expect(lex.TokIdent, "field name after `.`")
			if !ok {
				return lhs
			}
			fe := &ast.FieldExpr{
				Receiver: lhs,
				Name:     ast.Ident{Span: nameTok.Span, Name: nameTok.Text},
			}
			fe.Span = p.spanFrom(start)
			lhs = fe
		case lex.TokQuestionDot:
			p.advance()
			nameTok, ok := p.expect(lex.TokIdent, "field name after `?.`")
			if !ok {
				return lhs
			}
			of := &ast.OptFieldExpr{
				Receiver: lhs,
				Name:     ast.Ident{Span: nameTok.Span, Name: nameTok.Text},
			}
			of.Span = p.spanFrom(start)
			lhs = of
		case lex.TokQuestion:
			p.advance()
			te := &ast.TryExpr{Receiver: lhs}
			te.Span = p.spanFrom(start)
			lhs = te
		case lex.TokLParen:
			lhs = p.parseCallTail(start, lhs)
		case lex.TokLBracket:
			lhs = p.parseIndexTail(start, lhs)
		default:
			return lhs
		}
	}
	return lhs
}

func (p *parser) parseCallTail(start lex.Token, callee ast.Expr) ast.Expr {
	p.advance() // `(`
	var args []ast.Expr
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		a := p.parseExpr()
		if a == nil {
			break
		}
		args = append(args, a)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "call argument list")
	ce := &ast.CallExpr{Callee: callee, Args: args}
	ce.Span = p.spanFrom(start)
	return ce
}

func (p *parser) parseIndexTail(start lex.Token, receiver ast.Expr) ast.Expr {
	p.advance() // `[`
	// Leading `..` form: `[..hi]` or `[..=hi]`.
	if p.check(lex.TokDotDot) || p.check(lex.TokDotDotEq) {
		inclusive := p.check(lex.TokDotDotEq)
		p.advance()
		var hi ast.Expr
		if !p.check(lex.TokRBracket) {
			hi = p.parseExpr()
		}
		p.expect(lex.TokRBracket, "slice range index")
		ir := &ast.IndexRangeExpr{Receiver: receiver, High: hi, Inclusive: inclusive}
		ir.Span = p.spanFrom(start)
		return ir
	}
	first := p.parseExpr()
	if p.check(lex.TokDotDot) || p.check(lex.TokDotDotEq) {
		inclusive := p.check(lex.TokDotDotEq)
		p.advance()
		var hi ast.Expr
		if !p.check(lex.TokRBracket) {
			hi = p.parseExpr()
		}
		p.expect(lex.TokRBracket, "slice range index")
		ir := &ast.IndexRangeExpr{Receiver: receiver, Low: first, High: hi, Inclusive: inclusive}
		ir.Span = p.spanFrom(start)
		return ir
	}
	p.expect(lex.TokRBracket, "index expression")
	ie := &ast.IndexExpr{Receiver: receiver, Index: first}
	ie.Span = p.spanFrom(start)
	return ie
}

func (p *parser) parsePrimary(ctx parseCtx) ast.Expr {
	start := p.cur()
	switch p.cur().Kind {
	case lex.TokInt, lex.TokFloat, lex.TokString, lex.TokRawString,
		lex.TokCString, lex.TokChar, lex.TokTrue, lex.TokFalse, lex.TokKwNone:
		return p.parseLiteral()
	case lex.TokLParen:
		return p.parseParenOrTupleExpr(start)
	case lex.TokLBrace:
		return p.parseBlockExpr()
	case lex.TokKwIf:
		return p.parseIfExpr(start)
	case lex.TokKwMatch:
		return p.parseMatchExpr(start)
	case lex.TokKwLoop:
		return p.parseLoopExpr(start)
	case lex.TokKwWhile:
		return p.parseWhileExpr(start)
	case lex.TokKwFor:
		return p.parseForExpr(start)
	case lex.TokKwFn:
		return p.parseClosureExpr(start, false)
	case lex.TokKwMove:
		p.advance()
		if !p.check(lex.TokKwFn) {
			p.errorAt(p.cur().Span,
				"expected `fn` after `move`",
				"`move` is only valid as a prefix on a closure (`move fn(...)`)")
			return nil
		}
		return p.parseClosureExpr(start, true)
	case lex.TokKwSpawn:
		return p.parseSpawnExpr(start)
	case lex.TokKwUnsafe:
		return p.parseUnsafeExpr(start)
	case lex.TokIdent, lex.TokKwSelfVal, lex.TokKwSelfType, lex.TokKwSome:
		return p.parsePathOrStructLit(start, ctx)
	}
	p.errorAt(p.cur().Span,
		"expected an expression",
		"expression starts are literals, identifiers, `(`, `{`, `if`, `match`, `loop`, `while`, `for`, `fn`, `move`, `spawn`, or `unsafe`")
	// Advance one token so we make progress on malformed input.
	p.advance()
	return nil
}

func (p *parser) parseLiteral() ast.Expr {
	tok := p.advance()
	lit := &ast.LiteralExpr{Text: tok.Text}
	switch tok.Kind {
	case lex.TokInt:
		lit.Kind = ast.LitInt
	case lex.TokFloat:
		lit.Kind = ast.LitFloat
	case lex.TokString:
		lit.Kind = ast.LitString
	case lex.TokRawString:
		lit.Kind = ast.LitRawString
	case lex.TokCString:
		lit.Kind = ast.LitCString
	case lex.TokChar:
		lit.Kind = ast.LitChar
	case lex.TokTrue:
		lit.Kind = ast.LitBool
		lit.Value = true
	case lex.TokFalse:
		lit.Kind = ast.LitBool
		lit.Value = false
	case lex.TokKwNone:
		lit.Kind = ast.LitNone
	}
	lit.Span = tok.Span
	return lit
}

func (p *parser) parseParenOrTupleExpr(start lex.Token) ast.Expr {
	p.advance() // `(`
	if p.eat(lex.TokRParen) {
		// `()` is the unit literal, which at AST level is a zero-arg tuple.
		te := &ast.TupleExpr{}
		te.Span = p.spanFrom(start)
		return te
	}
	first := p.parseExpr()
	if !p.eat(lex.TokComma) {
		p.expect(lex.TokRParen, "parenthesized expression")
		pe := &ast.ParenExpr{Inner: first}
		pe.Span = p.spanFrom(start)
		return pe
	}
	// 1-tuple: `(x,)` — comma consumed, immediate `)`.
	if p.eat(lex.TokRParen) {
		te := &ast.TupleExpr{Elements: []ast.Expr{first}}
		te.Span = p.spanFrom(start)
		return te
	}
	elems := []ast.Expr{first}
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		elems = append(elems, p.parseExpr())
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "tuple expression")
	te := &ast.TupleExpr{Elements: elems}
	te.Span = p.spanFrom(start)
	return te
}

// parsePathOrStructLit parses a path expression and, if the next token is
// `{`, decides whether to upgrade it to a struct literal. Reference §10.7
// requires the brace body to be empty or to begin with `IDENT :` (or `..`)
// for the literal to be selected; otherwise the path is returned as-is and
// the `{` belongs to a surrounding block.
func (p *parser) parsePathOrStructLit(start lex.Token, ctx parseCtx) ast.Expr {
	path := p.parsePathExpr(start)
	if path == nil {
		return nil
	}
	if ctx == ctxNoStruct {
		return path
	}
	if !p.check(lex.TokLBrace) {
		return path
	}
	if !p.looksLikeStructLitBody() {
		return path
	}
	return p.parseStructLitTail(start, path)
}

// looksLikeStructLitBody peeks past the `{` to decide whether the brace body
// starts a struct literal. Rules from reference §10.7:
//   - `{}` is a struct literal (empty field list)
//   - `{ IDENT :` starts a struct literal (first field declared value)
//   - `{ IDENT ,` or `{ IDENT }` is the shorthand field form
//   - `{ .. expr ...` starts a struct update literal
// Anything else means the identifier is just an expression and the `{`
// opens a block.
func (p *parser) looksLikeStructLitBody() bool {
	if !p.check(lex.TokLBrace) {
		return false
	}
	k1 := p.peek(1).Kind
	if k1 == lex.TokRBrace {
		return true
	}
	if k1 == lex.TokDotDot {
		return true
	}
	if k1 != lex.TokIdent {
		return false
	}
	k2 := p.peek(2).Kind
	return k2 == lex.TokColon || k2 == lex.TokComma || k2 == lex.TokRBrace
}

func (p *parser) parsePathExpr(start lex.Token) *ast.PathExpr {
	// An expression-position path is a single identifier (or `self` / `Self`
	// / a reserved constructor). Multi-segment access like `x.y.z` is handled
	// by the postfix layer as FieldExpr chains, so struct-literal
	// disambiguation looks at the token right after this one identifier.
	first := p.advance()
	segs := []ast.Ident{{Span: first.Span, Name: first.Text}}
	pe := &ast.PathExpr{Segments: segs}
	pe.Span = p.spanFrom(start)
	return pe
}

func (p *parser) parseStructLitTail(start lex.Token, path *ast.PathExpr) ast.Expr {
	p.advance() // `{`
	sl := &ast.StructLitExpr{Path: path}
	for !p.check(lex.TokRBrace) && !p.check(lex.TokEOF) {
		if p.check(lex.TokDotDot) {
			p.advance()
			sl.Base = p.parseExpr()
			break
		}
		fstart := p.cur()
		nameTok, ok := p.expect(lex.TokIdent, "struct literal field name")
		if !ok {
			break
		}
		field := &ast.StructLitField{
			Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text},
		}
		if p.eat(lex.TokColon) {
			field.Value = p.parseExpr()
		} else {
			field.Shorthand = true
		}
		field.Span = p.spanFrom(fstart)
		sl.Fields = append(sl.Fields, field)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRBrace, "struct literal")
	sl.Span = p.spanFrom(start)
	return sl
}

func (p *parser) parseBlockExpr() *ast.BlockExpr {
	start, _ := p.expect(lex.TokLBrace, "block")
	b := &ast.BlockExpr{}
	for !p.check(lex.TokRBrace) && !p.check(lex.TokEOF) {
		stmt, trailing := p.parseStmtOrTrailing()
		if trailing != nil {
			b.Trailing = trailing
			break
		}
		if stmt == nil {
			p.synchronize()
			continue
		}
		b.Stmts = append(b.Stmts, stmt)
	}
	p.expect(lex.TokRBrace, "block")
	b.Span = p.spanFrom(start)
	return b
}

// parseStmtOrTrailing returns exactly one of (stmt, nil) or (nil, trailing).
// A trailing expression is the last expression in a block with no terminating
// `;` — it becomes the block's value.
func (p *parser) parseStmtOrTrailing() (ast.Stmt, ast.Expr) {
	start := p.cur()
	switch p.cur().Kind {
	case lex.TokKwLet:
		return p.parseLetStmt(start), nil
	case lex.TokKwVar:
		return p.parseVarStmt(start), nil
	case lex.TokKwReturn:
		return p.parseReturnStmt(start), nil
	case lex.TokKwBreak:
		return p.parseBreakStmt(start), nil
	case lex.TokKwContinue:
		return p.parseContinueStmt(start), nil
	case lex.TokKwFn, lex.TokKwStruct, lex.TokKwEnum, lex.TokKwTrait,
		lex.TokKwImpl, lex.TokKwConst, lex.TokKwStatic, lex.TokKwType,
		lex.TokKwExtern, lex.TokAt, lex.TokKwPub:
		item := p.parseItem()
		if item == nil {
			return nil, nil
		}
		is := &ast.ItemStmt{Item: item}
		is.Span = p.spanFrom(start)
		return is, nil
	}
	e := p.parseExpr()
	if e == nil {
		return nil, nil
	}
	if p.eat(lex.TokSemi) {
		es := &ast.ExprStmt{Expr: e}
		es.Span = p.spanFrom(start)
		return es, nil
	}
	// No terminating `;` — this expression is either the trailing value or a
	// block-expression statement (`if`, `match`, `loop`, `while`, `for`,
	// `{...}`) that permits a following stmt without a semicolon. We treat
	// block-expressions as statements when another token follows the `}`.
	if isBlockExpr(e) && !p.check(lex.TokRBrace) {
		es := &ast.ExprStmt{Expr: e}
		es.Span = p.spanFrom(start)
		return es, nil
	}
	return nil, e
}

func isBlockExpr(e ast.Expr) bool {
	switch e.(type) {
	case *ast.BlockExpr, *ast.IfExpr, *ast.MatchExpr, *ast.LoopExpr,
		*ast.WhileExpr, *ast.ForExpr, *ast.UnsafeExpr:
		return true
	}
	return false
}

func (p *parser) parseLetStmt(start lex.Token) *ast.LetStmt {
	p.advance() // `let`
	pat := p.parsePattern()
	ls := &ast.LetStmt{Pattern: pat}
	if p.eat(lex.TokColon) {
		ls.Type = p.parseType()
	}
	if p.eat(lex.TokEq) {
		ls.Value = p.parseExpr()
	}
	p.expect(lex.TokSemi, "let statement")
	ls.Span = p.spanFrom(start)
	return ls
}

func (p *parser) parseVarStmt(start lex.Token) *ast.VarStmt {
	p.advance() // `var`
	nameTok, _ := p.expect(lex.TokIdent, "var name")
	vs := &ast.VarStmt{Name: ast.Ident{Span: nameTok.Span, Name: nameTok.Text}}
	if p.eat(lex.TokColon) {
		vs.Type = p.parseType()
	}
	p.expect(lex.TokEq, "var initializer")
	vs.Value = p.parseExpr()
	p.expect(lex.TokSemi, "var statement")
	vs.Span = p.spanFrom(start)
	return vs
}

func (p *parser) parseReturnStmt(start lex.Token) *ast.ReturnStmt {
	p.advance() // `return`
	rs := &ast.ReturnStmt{}
	if !p.check(lex.TokSemi) {
		rs.Value = p.parseExpr()
	}
	p.expect(lex.TokSemi, "return statement")
	rs.Span = p.spanFrom(start)
	return rs
}

func (p *parser) parseBreakStmt(start lex.Token) *ast.BreakStmt {
	p.advance() // `break`
	bs := &ast.BreakStmt{}
	if !p.check(lex.TokSemi) {
		bs.Value = p.parseExpr()
	}
	p.expect(lex.TokSemi, "break statement")
	bs.Span = p.spanFrom(start)
	return bs
}

func (p *parser) parseContinueStmt(start lex.Token) *ast.ContinueStmt {
	p.advance() // `continue`
	p.expect(lex.TokSemi, "continue statement")
	cs := &ast.ContinueStmt{}
	cs.Span = p.spanFrom(start)
	return cs
}

func (p *parser) parseIfExpr(start lex.Token) ast.Expr {
	p.advance() // `if`
	cond := p.parseExprNoStruct()
	then := p.parseBlockExpr()
	ie := &ast.IfExpr{Cond: cond, Then: then}
	if p.eat(lex.TokKwElse) {
		if p.check(lex.TokKwIf) {
			ie.Else = p.parseIfExpr(p.cur())
		} else {
			ie.Else = p.parseBlockExpr()
		}
	}
	ie.Span = p.spanFrom(start)
	return ie
}

func (p *parser) parseMatchExpr(start lex.Token) ast.Expr {
	p.advance() // `match`
	scrut := p.parseExprNoStruct()
	p.expect(lex.TokLBrace, "match body")
	me := &ast.MatchExpr{Scrutinee: scrut}
	for !p.check(lex.TokRBrace) && !p.check(lex.TokEOF) {
		arm := p.parseMatchArm()
		if arm == nil {
			p.synchronize()
			continue
		}
		me.Arms = append(me.Arms, arm)
		// Optional separator: a comma between arms.
		p.eat(lex.TokComma)
	}
	p.expect(lex.TokRBrace, "match body")
	me.Span = p.spanFrom(start)
	return me
}

func (p *parser) parseMatchArm() *ast.MatchArm {
	start := p.cur()
	pat := p.parsePattern()
	if pat == nil {
		return nil
	}
	arm := &ast.MatchArm{Pattern: pat}
	if p.eat(lex.TokKwIf) {
		arm.Guard = p.parseExprNoStruct()
	}
	// Grammar: arms are `pattern [guard] block_expr`. Accept the Rust-like
	// `=>` separator too for author convenience; it is equivalent to `{`.
	if p.eat(lex.TokFatArrow) {
		body := &ast.BlockExpr{}
		bstart := p.cur()
		e := p.parseExpr()
		if e != nil {
			body.Trailing = e
		}
		body.Span = p.spanFrom(bstart)
		arm.Body = body
	} else {
		arm.Body = p.parseBlockExpr()
	}
	arm.Span = p.spanFrom(start)
	return arm
}

func (p *parser) parseLoopExpr(start lex.Token) ast.Expr {
	p.advance() // `loop`
	body := p.parseBlockExpr()
	le := &ast.LoopExpr{Body: body}
	le.Span = p.spanFrom(start)
	return le
}

func (p *parser) parseWhileExpr(start lex.Token) ast.Expr {
	p.advance() // `while`
	cond := p.parseExprNoStruct()
	body := p.parseBlockExpr()
	we := &ast.WhileExpr{Cond: cond, Body: body}
	we.Span = p.spanFrom(start)
	return we
}

func (p *parser) parseForExpr(start lex.Token) ast.Expr {
	p.advance() // `for`
	pat := p.parsePattern()
	p.expect(lex.TokKwIn, "`for` iterator")
	iter := p.parseExprNoStruct()
	body := p.parseBlockExpr()
	fe := &ast.ForExpr{Pattern: pat, Iter: iter, Body: body}
	fe.Span = p.spanFrom(start)
	return fe
}

func (p *parser) parseClosureExpr(start lex.Token, isMove bool) ast.Expr {
	p.advance() // `fn`
	p.expect(lex.TokLParen, "closure parameter list")
	ce := &ast.ClosureExpr{IsMove: isMove}
	for !p.check(lex.TokRParen) && !p.check(lex.TokEOF) {
		pr := p.parseParam()
		if pr == nil {
			break
		}
		ce.Params = append(ce.Params, pr)
		if !p.eat(lex.TokComma) {
			break
		}
	}
	p.expect(lex.TokRParen, "closure parameter list")
	if p.eat(lex.TokArrow) {
		ce.Return = p.parseType()
	}
	ce.Body = p.parseBlockExpr()
	ce.Span = p.spanFrom(start)
	return ce
}

func (p *parser) parseSpawnExpr(start lex.Token) ast.Expr {
	p.advance() // `spawn`
	inner, ok := p.parseClosureExpr(p.cur(), false).(*ast.ClosureExpr)
	if !ok {
		return nil
	}
	se := &ast.SpawnExpr{Inner: inner}
	se.Span = p.spanFrom(start)
	return se
}

func (p *parser) parseUnsafeExpr(start lex.Token) ast.Expr {
	p.advance() // `unsafe`
	body := p.parseBlockExpr()
	ue := &ast.UnsafeExpr{Body: body}
	ue.Span = p.spanFrom(start)
	return ue
}

// binaryExpr builds a BinaryExpr with a span spanning from start to the last
// consumed token.
func binaryExpr(start lex.Token, op ast.BinaryOp, opSp lex.Span, lhs, rhs ast.Expr, p *parser) ast.Expr {
	be := &ast.BinaryExpr{Op: op, OpSpan: opSp, Lhs: lhs, Rhs: rhs}
	be.Span = p.spanFrom(start)
	return be
}

func compareOpFromKind(k lex.TokenKind) (ast.BinaryOp, bool) {
	switch k {
	case lex.TokEqEq:
		return ast.BinEq, true
	case lex.TokBangEq:
		return ast.BinNe, true
	case lex.TokLt:
		return ast.BinLt, true
	case lex.TokLe:
		return ast.BinLe, true
	case lex.TokGt:
		return ast.BinGt, true
	case lex.TokGe:
		return ast.BinGe, true
	}
	return 0, false
}

func assignOpFromKind(k lex.TokenKind) (ast.AssignOp, bool) {
	switch k {
	case lex.TokEq:
		return ast.AssignEq, true
	case lex.TokPlusEq:
		return ast.AssignAdd, true
	case lex.TokMinusEq:
		return ast.AssignSub, true
	case lex.TokStarEq:
		return ast.AssignMul, true
	case lex.TokSlashEq:
		return ast.AssignDiv, true
	case lex.TokPercentEq:
		return ast.AssignMod, true
	case lex.TokAmpEq:
		return ast.AssignAnd, true
	case lex.TokPipeEq:
		return ast.AssignOr, true
	case lex.TokCaretEq:
		return ast.AssignXor, true
	case lex.TokShlEq:
		return ast.AssignShl, true
	case lex.TokShrEq:
		return ast.AssignShr, true
	}
	return 0, false
}
