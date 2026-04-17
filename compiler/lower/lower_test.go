package lower

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/mir"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// pipeline runs parse → resolve → bridge → lower on the given
// source and returns the lowered MIR plus any diagnostics.
func pipeline(t *testing.T, src string) (*mir.Module, []Diagnostic) {
	t.Helper()
	f, pd := parse.Parse("test.fuse", []byte(src))
	if len(pd) != 0 {
		t.Fatalf("parse: %v", pd)
	}
	srcs := []*resolve.SourceFile{{ModulePath: "", File: f}}
	r, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		t.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, r, srcs).Run()
	if len(bd) != 0 {
		t.Fatalf("bridge: %v", bd)
	}
	return Lower(prog)
}

// TestMinimalLowerIntReturn exercises the supported W05 surface:
// a zero-arg fn that returns an integer expression built from
// literals and `+`/`-`/`*`/`/`/`%`.
func TestMinimalLowerIntReturn(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want int64 // expected final register's const value (for const-only programs)
	}{
		{"literal-zero", `fn main() -> I32 { return 0; }`, 0},
		{"literal-42", `fn main() -> I32 { return 42; }`, 42},
		{"add-simple", `fn main() -> I32 { return 20 + 22; }`, 42},
		{"mixed-arith", `fn main() -> I32 { return 1 + 2 * 3; }`, 0 /* not a const-only check */},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mod, diags := pipeline(t, tc.src)
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if len(mod.Functions) != 1 {
				t.Fatalf("expected 1 function, got %d", len(mod.Functions))
			}
			fn := mod.Functions[0]
			if fn.Name != "main" {
				t.Fatalf("fn.Name = %q, want main", fn.Name)
			}
			if err := fn.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			// For literal-only programs, the block's first
			// instruction must be ConstInt with the expected value.
			if tc.name == "literal-42" {
				first := fn.Blocks[0].Insts[0]
				if first.Op != mir.OpConstInt || first.IntValue != 42 {
					t.Fatalf("expected ConstInt 42, got %+v", first)
				}
			}
		})
	}
}

// TestMinimalLowerIntReturn_Rejects verifies that each unsupported
// W05 shape produces a diagnostic rather than a silent default.
func TestMinimalLowerIntReturn_Rejects(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantMsg string
	}{
		{"multiple-statements",
			`fn main() -> I32 {
				let a: I32 = 1;
				return a;
			}`,
			"exactly one statement"},
		{"trailing-expr",
			`fn main() -> I32 { 42 }`,
			"trailing block expressions"},
		{"string-literal",
			`fn main() -> I32 { return "nope"; }`,
			"only lowers integer and boolean literals"},
		{"logical-op",
			`fn main() -> I32 { return 1 && 2; }`,
			"does not yet lower binary operator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := pipeline(t, tc.src)
			if len(diags) == 0 {
				t.Fatalf("expected diagnostic containing %q, got none", tc.wantMsg)
			}
			found := false
			for _, d := range diags {
				if contains(d.Message, tc.wantMsg) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("no diagnostic contained %q; got %v", tc.wantMsg, diags)
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
