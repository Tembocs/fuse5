package resolve

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/parse"
)

// TestModuleFirstFallback verifies the two-step module-first rule from
// reference §18.7: try the full dotted path as a module, and on miss
// retry with the last segment as an item inside the preceding module.
func TestModuleFirstFallback(t *testing.T) {
	cases := []struct {
		name    string
		srcs    []*SourceFile
		probe   string // name we expect to find in root module's scope after import
		target  string // target module path (or "" for an item binding)
		wantErr string // substring expected in diagnostics, or "" for clean
	}{
		{
			name: "full-path-is-module",
			srcs: []*SourceFile{
				makeSrc("", "lib.fuse", "import std.io;"),
				makeSrc("std", "std.fuse", ""),
				makeSrc("std.io", "io.fuse", "pub fn stdout() {}"),
			},
			probe:  "io",
			target: "std.io",
		},
		{
			name: "item-fallback",
			srcs: []*SourceFile{
				makeSrc("", "lib.fuse", "import std.io.stdout;"),
				makeSrc("std", "std.fuse", ""),
				makeSrc("std.io", "io.fuse", "pub fn stdout() {}"),
			},
			probe:  "stdout",
			target: "", // resolves to an item, not a module
		},
		{
			name: "unresolved-path",
			srcs: []*SourceFile{
				makeSrc("", "lib.fuse", "import std.io.missing;"),
				makeSrc("std", "std.fuse", ""),
				makeSrc("std.io", "io.fuse", "pub fn stdout() {}"),
			},
			wantErr: "no item",
		},
		{
			name: "totally-missing",
			srcs: []*SourceFile{
				makeSrc("", "lib.fuse", "import nowhere;"),
			},
			wantErr: "unresolved import",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, msgs := resolveStringsBare(t, tc.srcs)
			if tc.wantErr != "" {
				if !hasSubstring(msgs, tc.wantErr) {
					t.Fatalf("want diag containing %q, got %v", tc.wantErr, msgs)
				}
				return
			}
			if len(msgs) != 0 {
				t.Fatalf("unexpected diagnostics: %v", msgs)
			}
			root := out.Graph.Modules[""]
			if root == nil {
				t.Fatalf("root module not registered")
			}
			id := root.Scope.LookupLocal(tc.probe)
			if id == NoSymbol {
				t.Fatalf("%q not bound in root scope", tc.probe)
			}
			sym := out.Symbols.Get(id)
			if sym.Kind != SymImportAlias {
				t.Fatalf("expected SymImportAlias for %q, got %v", tc.probe, sym.Kind)
			}
			if sym.ModulePath != tc.target {
				t.Fatalf("import target module = %q, want %q", sym.ModulePath, tc.target)
			}
		})
	}
}

// TestImportCycleDetection asserts that a cyclic import graph produces
// a diagnostic naming the cycle and does NOT hang.
func TestImportCycleDetection(t *testing.T) {
	cases := []struct {
		name string
		srcs []*SourceFile
	}{
		{
			name: "two-module-cycle",
			srcs: []*SourceFile{
				makeSrc("a", "a.fuse", "import b;"),
				makeSrc("b", "b.fuse", "import a;"),
			},
		},
		{
			name: "three-module-cycle",
			srcs: []*SourceFile{
				makeSrc("a", "a.fuse", "import b;"),
				makeSrc("b", "b.fuse", "import c;"),
				makeSrc("c", "c.fuse", "import a;"),
			},
		},
		{
			name: "self-cycle",
			srcs: []*SourceFile{
				makeSrc("a", "a.fuse", "import a;"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, msgs := resolveStringsBare(t, tc.srcs)
			if !hasSubstring(msgs, "import cycle") {
				t.Fatalf("expected import cycle diagnostic, got %v", msgs)
			}
		})
	}
}

// TestImportCycleDetection_NoFalsePositive — a DAG must not trigger
// the cycle diagnostic.
func TestImportCycleDetection_NoFalsePositive(t *testing.T) {
	srcs := []*SourceFile{
		makeSrc("a", "a.fuse", "import b;\nimport c;"),
		makeSrc("b", "b.fuse", "import c;"),
		makeSrc("c", "c.fuse", ""),
	}
	_, msgs := resolveStringsBare(t, srcs)
	for _, m := range msgs {
		if indexOf(m, "import cycle") >= 0 {
			t.Fatalf("false positive cycle on DAG: %v", msgs)
		}
	}
}

// --- helpers ---

func makeSrc(modulePath, filename, src string) *SourceFile {
	// A dedicated version of mkSource that does not take *testing.T
	// because the table definitions run at package-init time. Parse
	// diagnostics here would indicate a malformed test fixture; the
	// caller's resolveStringsBare step exposes them if any leak past
	// parsing (Resolve surfaces lex diagnostics it receives).
	f, _ := parse.Parse(filename, []byte(src))
	return &SourceFile{ModulePath: modulePath, File: f}
}

func resolveStringsBare(t *testing.T, srcs []*SourceFile) (*Resolved, []string) {
	t.Helper()
	return resolveStrings(t, srcs, BuildConfig{})
}
