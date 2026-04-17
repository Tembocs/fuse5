// Package parse converts a Fuse token stream into an AST (reference
// Appendix C).
//
// The parser is a recursive-descent implementation with precedence climbing
// for expressions. It is syntax-only: no resolved symbols, no types, no pass
// metadata (Rule 3.2 — disjoint IR type families).
//
// Error handling: the parser never panics on malformed input. Syntactic
// errors are accumulated as lex.Diagnostic values and the parser advances to
// a synchronization point (`;`, `}`, start-of-item keyword) before continuing.
// `TestNopanicOnMalformed` is the forcing function for that invariant.
package parse

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// Parse tokenizes the source and returns the AST plus any diagnostics. The
// returned file is never nil (it may contain no items if the source is
// empty or entirely malformed). This keeps downstream code from needing a
// nil-guard at every use site.
func Parse(filename string, src []byte) (*ast.File, []lex.Diagnostic) {
	sc := lex.NewScanner(filename, src)
	diags := sc.Run()
	p := newParser(filename, sc.Tokens(), diags)
	file := p.parseFile()
	return file, p.diags
}

// ParseTokens parses an already-tokenized stream. Useful when the caller has
// lex diagnostics it wants to merge manually, or wants to re-parse the same
// buffer with different entry rules.
func ParseTokens(filename string, toks []lex.Token, lexDiags []lex.Diagnostic) (*ast.File, []lex.Diagnostic) {
	p := newParser(filename, toks, lexDiags)
	file := p.parseFile()
	return file, p.diags
}

// maxRecursionDepth guards against pathological nested input (e.g. deeply
// nested parens) so the parser terminates cleanly instead of overflowing the
// stack. The bound is large enough for any realistic source.
const maxRecursionDepth = 256

// parser is the mutable scanning state. Zero value is not usable; always
// construct via newParser.
type parser struct {
	filename string
	toks     []lex.Token
	pos      int
	diags    []lex.Diagnostic
	depth    int
}

func newParser(filename string, toks []lex.Token, initialDiags []lex.Diagnostic) *parser {
	// Defensive copy so callers can safely reuse their slice.
	d := make([]lex.Diagnostic, len(initialDiags))
	copy(d, initialDiags)
	return &parser{filename: filename, toks: toks, diags: d}
}

// peek returns the token k positions ahead of the current position. An out-of-
// range index returns the terminating EOF token so callers never need a
// length check — Rule 6.9 forbids silent defaults, and EOF is the
// well-defined end-of-stream marker.
func (p *parser) peek(k int) lex.Token {
	if p.pos+k >= len(p.toks) {
		// lex.Scanner always emits a terminal EOF, so this branch is only
		// reached on an empty token slice. Produce a synthetic EOF anchored
		// at the last known position so span reporting stays sane.
		return p.eofToken()
	}
	return p.toks[p.pos+k]
}

func (p *parser) eofToken() lex.Token {
	if len(p.toks) == 0 {
		return lex.Token{Kind: lex.TokEOF, Span: lex.Span{File: p.filename}}
	}
	return p.toks[len(p.toks)-1]
}

// cur returns the current token.
func (p *parser) cur() lex.Token { return p.peek(0) }

// advance moves one token forward and returns the consumed token.
func (p *parser) advance() lex.Token {
	t := p.cur()
	if t.Kind != lex.TokEOF {
		p.pos++
	}
	return t
}

// check reports whether the current token has kind k (no consume).
func (p *parser) check(k lex.TokenKind) bool { return p.cur().Kind == k }

// eat consumes one token if it matches k and returns true; otherwise it
// returns false without advancing.
func (p *parser) eat(k lex.TokenKind) bool {
	if p.check(k) {
		p.advance()
		return true
	}
	return false
}

// expect consumes one token of kind k or records a diagnostic. The returned
// token is valid only if ok is true — callers that read Text on failure must
// handle the fallback path explicitly.
func (p *parser) expect(k lex.TokenKind, context string) (lex.Token, bool) {
	if p.check(k) {
		return p.advance(), true
	}
	p.errorAt(p.cur().Span, fmt.Sprintf("expected %s (for %s), got %s",
		kindDescription(k), context, kindDescription(p.cur().Kind)),
		fmt.Sprintf("insert or correct the %s before proceeding", kindDescription(k)))
	return p.cur(), false
}

// errorAt records a diagnostic at the given span. Every diagnostic carries a
// primary span, a one-line message, and a suggestion per Rule 6.17.
func (p *parser) errorAt(sp lex.Span, msg, hint string) {
	p.diags = append(p.diags, lex.Diagnostic{Span: sp, Message: msg, Hint: hint})
}

// synchronize advances until it reaches a token that plausibly starts a new
// top-level construct or terminates the current one. Used to recover after
// an item-level error so a single broken declaration does not cascade into
// every subsequent token being rejected.
func (p *parser) synchronize() {
	for !p.check(lex.TokEOF) {
		switch p.cur().Kind {
		case lex.TokSemi:
			p.advance()
			return
		case lex.TokRBrace:
			return
		case lex.TokKwFn, lex.TokKwStruct, lex.TokKwEnum, lex.TokKwTrait,
			lex.TokKwImpl, lex.TokKwConst, lex.TokKwStatic, lex.TokKwType,
			lex.TokKwExtern, lex.TokKwImport, lex.TokKwPub, lex.TokKwUse,
			lex.TokKwMod, lex.TokAt:
			return
		}
		p.advance()
	}
}

// enter / leave manage the recursion-depth counter.
func (p *parser) enter() bool {
	p.depth++
	if p.depth > maxRecursionDepth {
		p.errorAt(p.cur().Span,
			"expression or type nesting too deep",
			"simplify the nested structure or split it across intermediate bindings")
		return false
	}
	return true
}

func (p *parser) leave() { p.depth-- }

// spanFrom builds a span starting at `start` and ending at the position of
// the last consumed token. The returned span covers the whole production.
func (p *parser) spanFrom(start lex.Token) lex.Span {
	// If nothing was consumed after start, the span is zero-width at start.
	endOff := start.Span.End
	if p.pos > 0 {
		endOff = p.toks[p.pos-1].Span.End
	}
	return lex.Span{File: p.filename, Start: start.Span.Start, End: endOff}
}

// kindDescription returns a human-readable name for a TokenKind used in
// diagnostics.
func kindDescription(k lex.TokenKind) string {
	switch k {
	case lex.TokEOF:
		return "end of file"
	case lex.TokSemi:
		return "`;`"
	case lex.TokComma:
		return "`,`"
	case lex.TokColon:
		return "`:`"
	case lex.TokLBrace:
		return "`{`"
	case lex.TokRBrace:
		return "`}`"
	case lex.TokLParen:
		return "`(`"
	case lex.TokRParen:
		return "`)`"
	case lex.TokLBracket:
		return "`[`"
	case lex.TokRBracket:
		return "`]`"
	case lex.TokArrow:
		return "`->`"
	case lex.TokFatArrow:
		return "`=>`"
	case lex.TokEq:
		return "`=`"
	case lex.TokIdent:
		return "identifier"
	}
	return k.String()
}
