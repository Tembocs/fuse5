// Package diagnostics owns the rendering and quality-audit surface
// for Fuse compiler diagnostics. Every diagnostic the compiler
// emits flows through this package for:
//
//   - text rendering for terminal output
//   - JSON rendering for IDE / LSP consumers
//   - Rule 6.17 quality audit: primary span, one-line explanation,
//     suggestion where applicable
//
// Reference: docs/rules.md §6.17 + §Repository layout 4.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tembocs/fuse5/compiler/lex"
)

// Severity classifies how hard a diagnostic hits the user's flow.
// Error stops compilation; Warning lets it continue but raises a
// UI flag; Hint is suggestion-only.
type Severity int

const (
	// SeverityUnknown is the zero value. Rendering treats it as
	// Error to fail loudly rather than silently upgrade to a hint.
	SeverityUnknown Severity = iota
	SeverityError
	SeverityWarning
	SeverityHint
)

// String renders the severity for text / JSON output. Stable so
// IDE consumers can parse on it.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityHint:
		return "hint"
	}
	return "error"
}

// Rendered is the externally-consumable form of a diagnostic. It
// decouples the renderer's view from lex.Diagnostic's in-memory
// shape: `lex.Diagnostic` is the wire between passes, Rendered is
// the wire between the compiler and the user.
type Rendered struct {
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Column      int      `json:"column"`
	EndLine     int      `json:"end_line,omitempty"`
	EndColumn   int      `json:"end_column,omitempty"`
	Severity    string   `json:"severity"`
	Message     string   `json:"message"`
	Suggestion  string   `json:"suggestion,omitempty"`
	Code        string   `json:"code,omitempty"`
	Notes       []string `json:"notes,omitempty"`
}

// FromLex converts one lex.Diagnostic into a Rendered form. The
// converter is the single source of truth for the span → file/line
// mapping so every subcommand sees the same format.
//
// Severity defaults to Error — the W05-W17 diagnostic surface is
// all errors; W20+ stdlib warnings will use the Severity override.
func FromLex(d lex.Diagnostic) Rendered {
	return Rendered{
		File:       spanFile(d.Span),
		Line:       spanLine(d.Span),
		Column:     spanColumn(d.Span),
		EndLine:    spanEndLine(d.Span),
		EndColumn:  spanEndColumn(d.Span),
		Severity:   SeverityError.String(),
		Message:    d.Message,
		Suggestion: d.Hint,
	}
}

// RenderText formats a single Rendered diagnostic as the text the
// user sees at the terminal. The layout is stable:
//
//     <file>:<line>:<col>: <severity>: <message>
//       hint: <suggestion>
//
// A diagnostic with no suggestion prints just the first line. A
// diagnostic with no file prints `<unknown>:0:0:` so downstream
// tools that parse on the shape still get a well-formed header.
func RenderText(r Rendered) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s:%d:%d: %s: %s", renderPath(r.File), r.Line, r.Column, r.Severity, r.Message)
	if r.Suggestion != "" {
		fmt.Fprintf(&sb, "\n  hint: %s", r.Suggestion)
	}
	for _, note := range r.Notes {
		fmt.Fprintf(&sb, "\n  note: %s", note)
	}
	return sb.String()
}

// RenderJSON returns the canonical JSON encoding of r. Every
// consumer (LSP, IDE) gets the same bytes for the same Rendered.
// Uses json.Marshal with no indentation so the output is a single
// line — IDEs that want pretty-printing can re-encode.
func RenderJSON(r Rendered) ([]byte, error) {
	return json.Marshal(r)
}

// RenderAll turns a batch of lex.Diagnostic into text or JSON. The
// text path joins with newlines; the JSON path emits a single
// array so multi-diagnostic runs remain one parse target.
func RenderAll(diags []lex.Diagnostic, asJSON bool) ([]byte, error) {
	rs := make([]Rendered, len(diags))
	for i, d := range diags {
		rs[i] = FromLex(d)
	}
	if asJSON {
		return json.Marshal(rs)
	}
	var sb strings.Builder
	for i, r := range rs {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(RenderText(r))
	}
	return []byte(sb.String()), nil
}

// renderPath substitutes `<unknown>` for empty filenames so text
// rendering always has a stable four-field header.
func renderPath(p string) string {
	if p == "" {
		return "<unknown>"
	}
	return p
}

// Span-accessor helpers — the lex.Span layout evolves; routing
// access through these wrappers means diagnostics.go changes in
// one place if the span surface changes.
func spanFile(s lex.Span) string    { return s.File }
func spanLine(s lex.Span) int       { return s.Start.Line }
func spanColumn(s lex.Span) int     { return s.Start.Column }
func spanEndLine(s lex.Span) int {
	if s.End == s.Start {
		return 0
	}
	return s.End.Line
}
func spanEndColumn(s lex.Span) int {
	if s.End == s.Start {
		return 0
	}
	return s.End.Column
}

// AuditDiagnostic represents one diagnostic the quality-audit pass
// inspects for Rule 6.17 compliance.
type AuditDiagnostic struct {
	// Source names the emitter (e.g. "parse", "check",
	// "monomorph") so audit failures point at the right package.
	Source string
	// Diag is the diagnostic itself.
	Diag lex.Diagnostic
	// SuggestionAllowed marks this diagnostic as a case where
	// the Rule 6.17 "suggestion where possible" clause applies.
	// When false the absence of Diag.Hint is not a violation.
	SuggestionAllowed bool
}

// AuditResult is the outcome of one Rule 6.17 inspection.
type AuditResult struct {
	Source   string
	Message  string
	Reason   string // why this diagnostic failed the audit
	Fatal    bool   // true when missing primary span / explanation
}

// AuditRule617 walks a batch of diagnostics and reports every Rule
// 6.17 violation. Load-bearing checks:
//
//   - Primary span must be non-zero (File != nil or Start/End set)
//   - Message must be non-empty
//   - When SuggestionAllowed is true and Hint is empty the audit
//     records a soft failure the caller can surface.
//
// A fatal failure is a production blocker; a non-fatal one is a
// polish finding. Both are returned so callers can choose their
// own gating threshold.
func AuditRule617(diags []AuditDiagnostic) []AuditResult {
	var out []AuditResult
	for _, a := range diags {
		if a.Diag.Message == "" {
			out = append(out, AuditResult{
				Source:  a.Source,
				Message: a.Diag.Message,
				Reason:  "empty message (Rule 6.17 requires a one-line explanation)",
				Fatal:   true,
			})
			continue
		}
		if a.Diag.Span.IsZero() {
			out = append(out, AuditResult{
				Source:  a.Source,
				Message: a.Diag.Message,
				Reason:  "missing primary span (Rule 6.17 requires the most specific source location)",
				Fatal:   true,
			})
			continue
		}
		if a.SuggestionAllowed && strings.TrimSpace(a.Diag.Hint) == "" {
			out = append(out, AuditResult{
				Source:  a.Source,
				Message: a.Diag.Message,
				Reason:  "no suggestion offered (Rule 6.17 requires one where possible)",
				Fatal:   false,
			})
		}
	}
	return out
}
