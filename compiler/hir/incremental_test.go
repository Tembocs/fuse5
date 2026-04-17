package hir

import (
	"reflect"
	"testing"
)

// TestInvariantWalkers exercises the invariant walker. Two programs
// are constructed: one clean via the bridge, one hand-rolled with a
// deliberate violation (an expression with NoType). The walker must
// produce zero violations for the first and at least one for the
// second.
func TestInvariantWalkers(t *testing.T) {
	t.Run("clean-bridge-output", func(t *testing.T) {
		prog, _ := bridgeTest(t, "m", "m.fuse", `
fn f(x: I32) -> I32 {
	return x;
}
`)
		if out := RunInvariantWalker(prog); len(out) != 0 {
			t.Fatalf("clean program produced violations: %v", out)
		}
	})

	t.Run("synthetic-violation-detected", func(t *testing.T) {
		prog, tab := bridgeTest(t, "m", "m.fuse", `fn f(x: I32) -> I32 { return x; }`)
		fn := prog.Modules["m"].Items[0].(*FnDecl)
		// Deliberately corrupt: clear the TypeId on one expression.
		visitExpr(fn.Body.Stmts[0].(*ReturnStmt).Value, func(e Expr) {
			if p, ok := e.(*PathExpr); ok {
				p.Type = 0 // NoType — should trigger a violation
			}
		})
		out := RunInvariantWalker(prog)
		if len(out) == 0 {
			t.Fatalf("invariant walker missed a synthetic NoType violation")
		}
		_ = tab // kept in scope so future sub-tests can extend this one
	})
}

// TestStableNodeIdentity asserts that NodeIDs depend on (module,
// item, local-path), not allocation order. Editing function `g`
// must not shift the identity of nodes inside function `f`.
func TestStableNodeIdentity(t *testing.T) {
	progA, _ := bridgeTest(t, "m", "m.fuse", `
fn f(x: I32) -> I32 { return x; }
fn g() -> I32 { return 1; }
`)
	progB, _ := bridgeTest(t, "m", "m.fuse", `
fn f(x: I32) -> I32 { return x; }
fn g() -> I32 { return 1 + 2; }
`)
	fA := findFn(t, progA, "f")
	fB := findFn(t, progB, "f")
	if fA.ID != fB.ID {
		t.Fatalf("fn f NodeID changed across unrelated edit: %q vs %q", fA.ID, fB.ID)
	}
	if len(fA.Params) != 1 || len(fB.Params) != 1 {
		t.Fatalf("fn f must have 1 param in both programs")
	}
	if fA.Params[0].ID != fB.Params[0].ID {
		t.Fatalf("param NodeID changed across unrelated edit: %q vs %q",
			fA.Params[0].ID, fB.Params[0].ID)
	}
	// And the body's internal NodeIDs too — walk down to the path expr.
	retA := fA.Body.Stmts[0].(*ReturnStmt).Value.(*PathExpr)
	retB := fB.Body.Stmts[0].(*ReturnStmt).Value.(*PathExpr)
	if retA.ID != retB.ID {
		t.Fatalf("return-value NodeID changed: %q vs %q", retA.ID, retB.ID)
	}
}

// TestIncrementalSubstitutable proves that when one function's HIR
// is invalidated, only dependent passes re-run; passes with no
// dependency on that function's output reuse their cached result.
//
// This is modelled as a two-function pipeline where two late passes
// depend on different early-pass outputs.
func TestIncrementalSubstitutable(t *testing.T) {
	m := NewManifest()
	m.Register(&simplePass{name: "parse", outputKey: "ast"})
	m.Register(&simplePass{name: "resolve", inputs: []string{"parse"}, outputKey: "resolved"})
	m.Register(&simplePass{name: "bridge-f", inputs: []string{"resolve"}, outputKey: "hir-f"})
	m.Register(&simplePass{name: "bridge-g", inputs: []string{"resolve"}, outputKey: "hir-g"})
	m.Register(&simplePass{name: "check-f", inputs: []string{"bridge-f"}, outputKey: "check-f"})
	m.Register(&simplePass{name: "check-g", inputs: []string{"bridge-g"}, outputKey: "check-g"})
	if err := m.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// Only bridge-f's output was invalidated (e.g. we edited
	// function f's source). check-f depends on bridge-f, so it
	// re-runs. bridge-g and check-g do not share any transitively
	// dirty input; they must reuse.
	plan := IncrementalPlan(m, map[string]bool{"bridge-f": true})
	wantRerun := []string{"bridge-f", "check-f"}
	wantReuse := []string{"bridge-g", "check-g", "parse", "resolve"}
	if !reflect.DeepEqual(plan.Rerun, wantRerun) {
		t.Fatalf("Rerun = %v, want %v", plan.Rerun, wantRerun)
	}
	if !reflect.DeepEqual(plan.Reuse, wantReuse) {
		t.Fatalf("Reuse = %v, want %v", plan.Reuse, wantReuse)
	}

	// Source-level edit of parse should cascade to everything.
	plan2 := IncrementalPlan(m, map[string]bool{"parse": true})
	wantAll := []string{"bridge-f", "bridge-g", "check-f", "check-g", "parse", "resolve"}
	if !reflect.DeepEqual(plan2.Rerun, wantAll) {
		t.Fatalf("Full-recompute Rerun = %v, want %v", plan2.Rerun, wantAll)
	}
	if len(plan2.Reuse) != 0 {
		t.Fatalf("expected empty Reuse on root-level dirty, got %v", plan2.Reuse)
	}
}

// findFn returns the FnDecl with the given name in the given program's
// first module, failing the test if it is not found.
func findFn(t *testing.T, p *Program, name string) *FnDecl {
	t.Helper()
	for _, m := range p.Modules {
		for _, it := range m.Items {
			if fn, ok := it.(*FnDecl); ok && fn.Name == name {
				return fn
			}
		}
	}
	t.Fatalf("fn %q not found", name)
	return nil
}
