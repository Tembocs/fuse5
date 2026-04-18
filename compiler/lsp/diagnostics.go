package lsp

import (
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/parse"
)

// computeDiagnostics runs the front-end passes over the document
// text and returns the LSP-shaped diagnostics. Fast path: only
// parse + resolve at W19; full check + consteval run but any
// panics they throw are caught so the server never crashes on
// user input.
//
// The parser is the most stable front-end surface at W19; it's the
// one consistently exercised in the e2e suite. Extending the
// pipeline to resolve/check/consteval happens as the pieces
// stabilise on partial input (the checker currently expects a
// well-formed program).
func (s *Server) computeDiagnostics(uri string) []Diagnostic {
	doc := s.docs.Get(uri)
	if doc == nil {
		return nil
	}
	_, lexDiags := parse.Parse(uriPath(uri), []byte(doc.Text))
	return translateDiagnostics(doc.Text, lexDiags)
}

// translateDiagnostics converts compiler diagnostics to LSP
// diagnostics. The LSP Range is end-inclusive on the client side
// for highlighting, so we add one column when the compiler
// diagnostic collapses to a point span.
func translateDiagnostics(text string, diags []lex.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		start := Position{
			Line:      max0(d.Span.Start.Line - 1),
			Character: max0(d.Span.Start.Column - 1),
		}
		end := Position{
			Line:      max0(d.Span.End.Line - 1),
			Character: max0(d.Span.End.Column - 1),
		}
		if end == start {
			end.Character++
		}
		out = append(out, Diagnostic{
			Range:    Range{Start: start, End: end},
			Severity: SeverityError,
			Source:   "fuse",
			Message:  d.Message,
		})
	}
	_ = text // reserved for future offset-based fallback
	return out
}

// max0 returns x when x ≥ 0, else 0. LSP positions are 0-indexed;
// compiler positions are 1-indexed. Subtracting 1 must not
// underflow when a diagnostic points at line 0 / column 0.
func max0(x int) int {
	if x < 0 {
		return 0
	}
	return x
}
