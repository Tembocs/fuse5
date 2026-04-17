package consteval

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// buildProgram drives parse → resolve → bridge → check on an
// in-memory Fuse source and returns the HIR program ready for the
// evaluator. Any diagnostic before consteval is a test bug, not a
// test outcome, so the helper fails fatally.
func buildProgram(t *testing.T, src string) *hir.Program {
	t.Helper()
	f, pd := parse.Parse("t.fuse", []byte(src))
	if len(pd) != 0 {
		t.Fatalf("parse: %v", pd)
	}
	srcs := []*resolve.SourceFile{{ModulePath: "", File: f}}
	resolved, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		t.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, resolved, srcs).Run()
	if len(bd) != 0 {
		t.Fatalf("bridge: %v", bd)
	}
	if cd := check.Check(prog); len(cd) != 0 {
		t.Fatalf("check: %v", cd)
	}
	return prog
}

// findConstValue looks up an evaluated constant by name.
func findConstValue(prog *hir.Program, res *Result, name string) (Value, bool) {
	for _, mp := range prog.Order {
		for _, it := range prog.Modules[mp].Items {
			if c, ok := it.(*hir.ConstDecl); ok && c.Name == name {
				v, ok := res.ConstValues[c.SymID]
				return v, ok
			}
		}
	}
	return Value{}, false
}

// findStaticValue looks up an evaluated static by name.
func findStaticValue(prog *hir.Program, res *Result, name string) (Value, bool) {
	for _, mp := range prog.Order {
		for _, it := range prog.Modules[mp].Items {
			if s, ok := it.(*hir.StaticDecl); ok && s.Name == name {
				v, ok := res.StaticValues[s.SymID]
				return v, ok
			}
		}
	}
	return Value{}, false
}

// TestEvaluatorCore exercises the Phase-01 "can the evaluator
// produce values at all?" contract. Each case is an independent
// subtest named for the behavior under test; the combined set
// covers literal folding, arithmetic, branches, const fn recursion,
// and bitmask/shift corners.
func TestEvaluatorCore(t *testing.T) {
	t.Run("integer-literal", func(t *testing.T) {
		prog := buildProgram(t, `const X: I32 = 42;`)
		res, diags := Evaluate(prog)
		if len(diags) != 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		v, ok := findConstValue(prog, res, "X")
		if !ok || v.Kind != VKInt || v.SignedInt(prog.Types) != 42 {
			t.Fatalf("X = %+v, want int 42", v)
		}
	})

	t.Run("arithmetic", func(t *testing.T) {
		prog := buildProgram(t, `const X: I32 = (2 + 3) * 4 - 1;`)
		res, _ := Evaluate(prog)
		v, _ := findConstValue(prog, res, "X")
		if v.SignedInt(prog.Types) != 19 {
			t.Fatalf("X = %d, want 19", v.SignedInt(prog.Types))
		}
	})

	t.Run("bool-logic", func(t *testing.T) {
		prog := buildProgram(t, `const X: Bool = true && (false || true);`)
		res, _ := Evaluate(prog)
		v, _ := findConstValue(prog, res, "X")
		if v.Kind != VKBool || !v.Bool {
			t.Fatalf("X = %+v, want bool true", v)
		}
	})

	t.Run("hex-binary-literals", func(t *testing.T) {
		prog := buildProgram(t, `const X: U32 = 0xFFu32 & 0b0101_1010u32;`)
		res, diags := Evaluate(prog)
		if len(diags) != 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		v, _ := findConstValue(prog, res, "X")
		if v.Int != 0x5A {
			t.Fatalf("X = %d, want 0x5A", v.Int)
		}
	})

	t.Run("const-fn-call", func(t *testing.T) {
		prog := buildProgram(t, `
const fn square(n: I32) -> I32 { return n * n; }
const NINE: I32 = square(3);
`)
		res, diags := Evaluate(prog)
		if len(diags) != 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		v, _ := findConstValue(prog, res, "NINE")
		if v.SignedInt(prog.Types) != 9 {
			t.Fatalf("NINE = %d, want 9", v.SignedInt(prog.Types))
		}
	})

	t.Run("const-fn-recursion", func(t *testing.T) {
		prog := buildProgram(t, `
const fn factorial(n: U64) -> U64 {
    if n == 0u64 { return 1u64; }
    return n * factorial(n - 1u64);
}
const FACT_10: U64 = factorial(10u64);
`)
		res, diags := Evaluate(prog)
		if len(diags) != 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		v, _ := findConstValue(prog, res, "FACT_10")
		if v.Int != 3628800 {
			t.Fatalf("FACT_10 = %d, want 3628800", v.Int)
		}
	})

	t.Run("shift-and-cast", func(t *testing.T) {
		prog := buildProgram(t, `const X: I32 = (1u32 << 10u32) as I32;`)
		res, diags := Evaluate(prog)
		if len(diags) != 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		v, _ := findConstValue(prog, res, "X")
		if v.SignedInt(prog.Types) != 1024 {
			t.Fatalf("X = %d, want 1024", v.SignedInt(prog.Types))
		}
	})
}

// TestEvaluatorDeterminism re-runs Evaluate three times on the same
// program and asserts the stringified value set is byte-identical
// every time. This is the spec's determinism exit criterion —
// the tests/e2e harness runs `go test -count=3` to exercise it
// further; running it in-test here catches regressions earlier.
func TestEvaluatorDeterminism(t *testing.T) {
	const src = `
const fn step(n: I32) -> I32 { return n + 1; }
const A: I32 = step(10);
const B: I32 = step(A);
const C: I32 = step(B);
`
	var first string
	for i := 0; i < 3; i++ {
		prog := buildProgram(t, src)
		res, diags := Evaluate(prog)
		if len(diags) != 0 {
			t.Fatalf("run %d: diags %v", i, diags)
		}
		var sb strings.Builder
		for _, sym := range res.SortedConstSymbols() {
			sb.WriteString(res.ConstValues[sym].String())
			sb.WriteByte('\n')
		}
		got := sb.String()
		if i == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("run %d diverged:\n--- first ---\n%s\n--- got ---\n%s", i, first, got)
		}
	}
}
