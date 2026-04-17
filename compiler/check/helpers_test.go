package check

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// checkSource drives the full parse → resolve → bridge → check
// pipeline over an in-memory Fuse source and returns the resulting
// Program and check diagnostics. The test harness treats parse /
// resolve / bridge failures as fatal so check-level assertions
// stay focused on the checker's behavior.
func checkSource(t *testing.T, modPath, filename, src string) (*hir.Program, []Diagnostic) {
	t.Helper()
	f, pd := parse.Parse(filename, []byte(src))
	if len(pd) != 0 {
		t.Fatalf("parse: %v", pd)
	}
	srcs := []*resolve.SourceFile{{ModulePath: modPath, File: f}}
	resolved, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		t.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, resolved, srcs).Run()
	if len(bd) != 0 {
		t.Fatalf("bridge: %v", bd)
	}
	diags := Check(prog)
	return prog, diags
}

// checkMulti is checkSource for builds with more than one module.
func checkMulti(t *testing.T, files map[string]string) (*hir.Program, []Diagnostic) {
	t.Helper()
	var srcs []*resolve.SourceFile
	for modPath, src := range files {
		f, pd := parse.Parse(modPath+".fuse", []byte(src))
		if len(pd) != 0 {
			t.Fatalf("parse %s: %v", modPath, pd)
		}
		srcs = append(srcs, &resolve.SourceFile{ModulePath: modPath, File: f})
	}
	resolved, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		t.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, resolved, srcs).Run()
	if len(bd) != 0 {
		t.Fatalf("bridge: %v", bd)
	}
	return prog, Check(prog)
}

// wantClean fails the test when diags is non-empty.
func wantClean(t *testing.T, diags []Diagnostic) {
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

// wantDiag asserts that at least one diagnostic contains substr.
func wantDiag(t *testing.T, diags []Diagnostic, substr string) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, substr) {
			return
		}
	}
	var sb strings.Builder
	for _, d := range diags {
		sb.WriteString("  - ")
		sb.WriteString(d.Message)
		sb.WriteString("\n")
	}
	t.Fatalf("expected diagnostic containing %q; got:\n%s", substr, sb.String())
}

// findFn locates an FnDecl by name in the first module of prog.
func findFn(t *testing.T, prog *hir.Program, name string) *hir.FnDecl {
	t.Helper()
	for _, modPath := range prog.Order {
		for _, it := range prog.Modules[modPath].Items {
			if fn, ok := it.(*hir.FnDecl); ok && fn.Name == name {
				return fn
			}
		}
	}
	t.Fatalf("fn %q not found", name)
	return nil
}
