package resolve

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/parse"
)

// parseSource parses a single in-memory source file and asserts there
// were no parse diagnostics. It returns the file ready to be placed
// into a SourceFile.
func parseSource(t *testing.T, filename, src string) *ast.File {
	t.Helper()
	f, diags := parse.Parse(filename, []byte(src))
	if len(diags) != 0 {
		t.Fatalf("parse of %q failed: %v", filename, diags)
	}
	if f == nil {
		t.Fatalf("parse of %q returned nil file", filename)
	}
	return f
}

// mkSource is a convenience constructor for table-driven tests.
func mkSource(t *testing.T, modulePath, filename, src string) *SourceFile {
	return &SourceFile{ModulePath: modulePath, File: parseSource(t, filename, src)}
}

// resolveStrings turns a SourceFile list plus BuildConfig into (Resolved,
// diagnostic-messages) — useful in table-driven tests that want to
// match on message substrings without caring about full Diagnostic{}
// equality.
func resolveStrings(t *testing.T, srcs []*SourceFile, cfg BuildConfig) (*Resolved, []string) {
	t.Helper()
	out, diags := Resolve(srcs, cfg)
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	return out, msgs
}

// assertNoDiags fails the test when diags is non-empty, printing each
// diagnostic's span and message for debuggability.
func assertNoDiags(t *testing.T, diags []lex.Diagnostic) {
	t.Helper()
	if len(diags) == 0 {
		return
	}
	var sb strings.Builder
	for _, d := range diags {
		sb.WriteString(d.Span.String())
		sb.WriteString(": ")
		sb.WriteString(d.Message)
		if d.Hint != "" {
			sb.WriteString(" (hint: ")
			sb.WriteString(d.Hint)
			sb.WriteString(")")
		}
		sb.WriteString("\n")
	}
	t.Fatalf("unexpected diagnostics:\n%s", sb.String())
}

// diagContains returns true when any diagnostic in ds contains substr
// in its message.
func diagContains(ds []lex.Diagnostic, substr string) bool {
	for _, d := range ds {
		if strings.Contains(d.Message, substr) {
			return true
		}
	}
	return false
}
