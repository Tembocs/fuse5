package diagnostics

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/lex"
)

// TestTextRendering verifies the canonical `file:line:col: severity: msg`
// text layout plus the `  hint:` continuation when a suggestion is
// present. Bound by:
//
//	go test ./compiler/diagnostics/... -run TestTextRendering -v
func TestTextRendering(t *testing.T) {
	t.Run("error-with-hint", func(t *testing.T) {
		r := Rendered{
			File:       "main.fuse",
			Line:       12,
			Column:     7,
			Severity:   SeverityError.String(),
			Message:    "unexpected token",
			Suggestion: "did you mean `return`?",
		}
		got := RenderText(r)
		if !strings.HasPrefix(got, "main.fuse:12:7: error: unexpected token") {
			t.Errorf("header wrong: %q", got)
		}
		if !strings.Contains(got, "\n  hint: did you mean `return`?") {
			t.Errorf("hint continuation missing: %q", got)
		}
	})
	t.Run("error-without-hint", func(t *testing.T) {
		r := Rendered{
			File:     "x.fuse", Line: 1, Column: 1,
			Severity: SeverityError.String(),
			Message:  "boom",
		}
		got := RenderText(r)
		if strings.Contains(got, "hint:") {
			t.Errorf("no-hint diagnostic should not render `hint:`: %q", got)
		}
	})
	t.Run("missing-file-renders-unknown", func(t *testing.T) {
		r := Rendered{Severity: SeverityError.String(), Message: "no file"}
		got := RenderText(r)
		if !strings.HasPrefix(got, "<unknown>:0:0:") {
			t.Errorf("missing-file header should degrade gracefully: %q", got)
		}
	})
	t.Run("notes-append", func(t *testing.T) {
		r := Rendered{
			File: "a.fuse", Line: 3, Column: 4,
			Severity: SeverityError.String(),
			Message:  "m",
			Notes:    []string{"also here"},
		}
		got := RenderText(r)
		if !strings.Contains(got, "\n  note: also here") {
			t.Errorf("notes should append: %q", got)
		}
	})
}

// TestJsonRendering verifies the canonical JSON encoding. IDE / LSP
// consumers parse this directly — any field rename is a breaking
// change to the external contract.
func TestJsonRendering(t *testing.T) {
	r := Rendered{
		File: "main.fuse", Line: 5, Column: 9,
		Severity: "error", Message: "bad",
		Suggestion: "try this",
	}
	out, err := RenderJSON(r)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	// Decode and re-check on the field names — test on the
	// contract, not string containment, so whitespace variants
	// can't hide a rename.
	var round map[string]any
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, out)
	}
	for _, k := range []string{"file", "line", "column", "severity", "message", "suggestion"} {
		if _, ok := round[k]; !ok {
			t.Errorf("JSON missing field %q: %s", k, out)
		}
	}
	// Batch shape: empty slice serialises to "[]" or "null" — we
	// choose null-free for predictability.
	batch, err := RenderAll(nil, true)
	if err != nil {
		t.Fatalf("RenderAll: %v", err)
	}
	if string(batch) != "null" && string(batch) != "[]" {
		t.Errorf("empty batch should be null or [] (got %q)", batch)
	}
}

// TestDiagnosticQualityRule is the Rule 6.17 audit. It takes a
// representative sample of diagnostic emitters the compiler has
// produced since W01, runs them through AuditRule617, and asserts
// every message passes the load-bearing checks. This gate-tests
// the diagnostic surface for regressions: a diagnostic that
// regresses loses its primary span or explanation and the audit
// surfaces it.
func TestDiagnosticQualityRule(t *testing.T) {
	samples := []AuditDiagnostic{
		{
			Source: "parse",
			Diag: lex.Diagnostic{
				Span:    lex.Span{File: "x.fuse", Start: lex.Position{Line: 1, Column: 1}},
				Message: "expected identifier, got `;`",
				Hint:    "declare the identifier before the semicolon",
			},
			SuggestionAllowed: true,
		},
		{
			Source: "check",
			Diag: lex.Diagnostic{
				Span:    lex.Span{File: "x.fuse", Start: lex.Position{Line: 7, Column: 10}},
				Message: "cannot assign `Bool` to `I32`",
				Hint:    "cast via `as I32` or widen the target type",
			},
			SuggestionAllowed: true,
		},
		{
			Source: "lower",
			Diag: lex.Diagnostic{
				Span:    lex.Span{File: "x.fuse", Start: lex.Position{Line: 3, Column: 5}},
				Message: "spine does not yet lower generic fn bodies",
				Hint:    "monomorphization and generic instantiation arrive in W08",
			},
			SuggestionAllowed: true,
		},
		{
			Source: "runtime",
			Diag: lex.Diagnostic{
				Span:    lex.Span{File: "rt.fuse", Start: lex.Position{Line: 1, Column: 1}},
				Message: "runtime panic",
			},
			// Runtime panics have no caller-actionable
			// suggestion; the audit must not demand one.
			SuggestionAllowed: false,
		},
	}
	results := AuditRule617(samples)
	if len(results) != 0 {
		for _, r := range results {
			t.Errorf("audit failure: %+v", r)
		}
	}

	// Negative cases: diagnostics that SHOULD fail the audit.
	bad := []AuditDiagnostic{
		{
			Source: "bug",
			Diag:   lex.Diagnostic{Message: ""}, // empty message
		},
		{
			Source: "bug2",
			Diag:   lex.Diagnostic{Message: "has message, no span"},
		},
		{
			Source: "bug3",
			Diag: lex.Diagnostic{
				Span:    lex.Span{File: "y.fuse", Start: lex.Position{Line: 1, Column: 1}},
				Message: "has span, no suggestion",
			},
			SuggestionAllowed: true,
		},
	}
	results = AuditRule617(bad)
	if len(results) < 3 {
		t.Fatalf("audit missed a violation: %+v", results)
	}
	var fatals, soft int
	for _, r := range results {
		if r.Fatal {
			fatals++
		} else {
			soft++
		}
	}
	if fatals < 2 {
		t.Errorf("expected ≥2 fatal audit results for empty message + missing span, got %d", fatals)
	}
	if soft < 1 {
		t.Errorf("expected ≥1 soft audit result for missing suggestion, got %d", soft)
	}
}
