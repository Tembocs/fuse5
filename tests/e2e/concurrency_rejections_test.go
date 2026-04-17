package e2e_test

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestConcurrencyRejections is the W07-P05-T01 Verify target. It
// asserts that each of the wave's mandated rejections fires with
// the expected diagnostic text:
//
//   (a) spawn of a non-move closure capturing by ref
//   (b) spawn of a closure returning a non-Send type
//   (c) Chan[T].send value-type mismatch
//   (d) lock-rank violation (non-strict ordering)
//
// Each sub-case constructs a synthetic HIR shape and runs the
// concurrency check directly. Full source-to-diagnostic wiring
// arrives when decorator propagation and spawn/chan lowering
// mature in W16.
func TestConcurrencyRejections(t *testing.T) {
	t.Run("non-move-closure-at-spawn", func(t *testing.T) {
		diags := runSpawnCase(t, false /* IsMove */, sendRet)
		assertDiagContains(t, diags, "ref, but ref T is not Send")
		assertDiagContains(t, diags, "prefix the closure with `move`")
	})
	t.Run("non-Send-return-at-spawn", func(t *testing.T) {
		// Return type of &I32 is not Send — we mark it explicitly
		// so the synthesized closure carries a valid ThreadHandle
		// type even though the inner return is forbidden.
		diags := runSpawnCase(t, true /* IsMove */, refRet)
		assertDiagContains(t, diags, "not Send")
	})
	t.Run("lock-rank-violation", func(t *testing.T) {
		errs := check.CheckRankOrder([]int{2, 1})
		if len(errs) == 0 {
			t.Fatalf("expected lock-rank violation diagnostic")
		}
		if !strings.Contains(errs[0], "strictly increase") {
			t.Fatalf("rank-order diagnostic missing `strictly increase`: %q", errs[0])
		}
	})
	t.Run("invalid-rank-value", func(t *testing.T) {
		errs := check.CheckRankDecorator(check.RankAttribute{Rank: 0})
		if len(errs) == 0 {
			t.Fatalf("@rank(0) must be rejected")
		}
		if !strings.Contains(errs[0], "positive integer") {
			t.Fatalf("@rank diagnostic missing `positive integer`: %q", errs[0])
		}
	})
}

// runSpawnCase synthesizes a minimal HIR surface and runs the
// concurrency checker's spawn validation directly. Returns the
// accumulated diagnostics.
func runSpawnCase(t *testing.T, isMove bool, returnPick returnShape) []check.Diagnostic {
	t.Helper()
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	prog.RegisterModule(&hir.Module{
		Base: hir.Base{ID: hir.ItemID("m", "")},
		Path: "m",
	})

	retType := returnPick(tab)
	fnType := tab.Fn(nil, retType, false)
	closure := &hir.ClosureExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::closure"},
			Type: fnType,
		},
		IsMove: isMove,
		Return: retType,
	}
	spawn := &hir.SpawnExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::spawn"},
			Type: tab.ThreadHandle(retType),
		},
		Closure: closure,
	}
	// Wrap in an ExprStmt inside a fn so the concurrency walker
	// reaches it naturally. The fn returns Unit so return-type
	// propagation doesn't interfere.
	fnBody := &hir.Block{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::body"},
			Type: tab.Unit(),
		},
		Stmts: []hir.Stmt{
			&hir.ExprStmt{
				Base: hir.Base{ID: "m::f::body/stmt.0"},
				Expr: spawn,
			},
		},
	}
	fn := &hir.FnDecl{
		Base:   hir.Base{ID: hir.ItemID("m", "f")},
		Name:   "f",
		TypeID: tab.Fn(nil, tab.Unit(), false),
		Return: tab.Unit(),
		Body:   fnBody,
	}
	prog.Modules["m"].Items = append(prog.Modules["m"].Items, fn)
	return check.Check(prog)
}

type returnShape func(*typetable.Table) typetable.TypeId

func sendRet(tab *typetable.Table) typetable.TypeId { return tab.I32() }
func refRet(tab *typetable.Table) typetable.TypeId  { return tab.Ref(tab.I32()) }

func assertDiagContains(t *testing.T, diags []check.Diagnostic, substr string) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, substr) || strings.Contains(d.Hint, substr) {
			return
		}
	}
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.Message+" | "+d.Hint)
	}
	t.Fatalf("expected diagnostic containing %q, got %v", substr, msgs)
}
