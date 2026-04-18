package lsp

import (
	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// computeDiagnostics runs the front-end passes over the document
// text and returns the LSP-shaped diagnostics. The server walks the
// whole parse → resolve → hir.Bridge → check pipeline so editor
// diagnostics match what `fuse build` would report. Each pass is
// gated by the previous one producing any output at all — if the
// parser rejected the file, resolve/check are skipped; otherwise
// they run even when earlier passes flagged soft errors, so the
// editor surfaces as many actionable diagnostics as possible in a
// single keystroke.
//
// A recover() guards the checker: it is designed for well-formed
// programs, so partial input mid-keystroke can panic. Recovering
// keeps the server alive and limits the blast radius to "this
// document's check pass produced no diagnostics this tick".
func (s *Server) computeDiagnostics(uri string) []Diagnostic {
	doc := s.docs.Get(uri)
	if doc == nil {
		return nil
	}
	path := uriPath(uri)
	file, parseDiags := parse.Parse(path, []byte(doc.Text))
	all := append([]lex.Diagnostic(nil), parseDiags...)
	if file == nil {
		return translateDiagnostics(doc.Text, all)
	}

	// Single-file resolve+bridge+check. Modules lean on the filename
	// as module path — editors operate per-document, and the pipeline
	// is tolerant to an empty module path.
	sources := []*resolve.SourceFile{{ModulePath: "", File: file}}
	resolveDiags, bridgeDiags, checkDiags := runDownstreamPasses(sources)
	all = append(all, resolveDiags...)
	all = append(all, bridgeDiags...)
	all = append(all, checkDiags...)
	return translateDiagnostics(doc.Text, all)
}

// runDownstreamPasses executes resolve → bridge → check and returns
// each pass's diagnostics. A panic in any pass is recovered to
// `[]lex.Diagnostic{}` rather than propagating — partial input at
// keystroke time occasionally hits invariant assertions in the
// checker, and the LSP must not crash.
func runDownstreamPasses(srcs []*resolve.SourceFile) (rd, bd, cd []lex.Diagnostic) {
	defer func() {
		if r := recover(); r != nil {
			// Pipeline panicked mid-run; whichever outputs we already
			// populated stay, the rest stay zero.
			_ = r
		}
	}()
	resolved, resolveDiags := resolve.Resolve(srcs, resolve.BuildConfig{})
	rd = resolveDiags
	if resolved == nil {
		return
	}
	tab := typetable.New()
	prog, bridgeDiags := hir.NewBridge(tab, resolved, srcs).Run()
	bd = bridgeDiags
	if prog == nil {
		return
	}
	cd = check.Check(prog)
	return
}

// translateDiagnostics converts compiler diagnostics to LSP
// diagnostics. The LSP Range is end-inclusive on the client side
// for highlighting, so we add one column when the compiler
// diagnostic collapses to a point span. Hints are concatenated into
// Message as ` hint: <text>` so the existing code-action extractor
// at handlers.go can parse them out without a schema change.
//
// Positions are emitted as (line, UTF-16 code-unit) pairs per LSP
// 3.17 §3.17/PositionEncodingKind. The lexer's byte-offset absolute
// positions are routed through byteOffsetToPosition so multi-byte
// UTF-8 runes (non-ASCII identifiers, raw-string content) produce
// correct editor highlights.
func translateDiagnostics(text string, diags []lex.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		start := byteOffsetToPosition(text, d.Span.Start.Offset)
		end := byteOffsetToPosition(text, d.Span.End.Offset)
		if end == start {
			end.Character++
		}
		msg := d.Message
		if d.Hint != "" {
			msg = msg + " hint: " + d.Hint
		}
		out = append(out, Diagnostic{
			Range:    Range{Start: start, End: end},
			Severity: SeverityError,
			Source:   "fuse",
			Message:  msg,
		})
	}
	return out
}

