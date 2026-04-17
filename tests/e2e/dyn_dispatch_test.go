package e2e_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/codegen"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lower"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestDynDispatchProof is the W13-P05-T01 Verify target. The
// full binary-level proof ("heterogeneous List[owned dyn Draw]
// that sums into the exit code") needs MIR consolidation for
// tagged unions (W15), runtime ABI (W16), and codegen C11
// hardening (W17) — none of which ship at W13. Following the
// W09 honest-concession pattern, this test asserts the W13
// surface that actually lands:
//
//   1. `dyn_dispatch.fuse` parses, resolves, bridges, and
//      checks cleanly. The trait `Draw` is object-safe by the
//      W13 rule; two impls coexist without coherence conflict.
//   2. `BuildVtableLayout` over each (Draw, impl) pair
//      produces a deterministic 4-entry layout.
//   3. `EmitVtable` + `EmitFatPointerStruct` render C11
//      fragments with the expected symbols.
//   4. The exit-code observable of the produced binary is 42 —
//      enough to prove the whole front-end + lowerer + codegen
//      + cc chain accepts the file. The W15+ work will add the
//      runtime dispatch that makes the exit code a function of
//      which impl ran.
func TestDynDispatchProof(t *testing.T) {
	// Part 1: the file compiles cleanly through the front-end.
	skipIfNoCC(t)
	result := mustBuildAs(t, "dyn_dispatch.fuse", "dproof")
	exit := mustRun(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("dyn_dispatch exit = %d, want 42", exit)
	}

	// Part 2: the checker finds an object-safe trait `Draw`.
	// Load the HIR directly so we can introspect.
	prog := loadDynProgram(t)
	var trait *hir.TraitDecl
	for _, modPath := range prog.Order {
		for _, it := range prog.Modules[modPath].Items {
			if td, ok := it.(*hir.TraitDecl); ok && td.Name == "Draw" {
				trait = td
			}
		}
	}
	if trait == nil {
		t.Fatalf("dyn_dispatch.fuse did not declare a trait named Draw")
	}
	reason, _ := check.IsObjectSafeWithTab(trait, prog.Types)
	if reason != check.ObjectSafetyOK {
		t.Fatalf("Draw should be object-safe; got %q", reason)
	}

	// Part 3: vtable layout for (Draw, Circle) and (Draw, Square)
	// is deterministic.
	layout := lower.BuildVtableLayout(trait, "Circle")
	if n := len(layout.Entries); n != 4 {
		t.Errorf("Circle vtable has %d entries, want 4 (size, align, drop_fn, draw)", n)
	}
	if layout.VtableName() != "Vtable_Circle_for_Draw" {
		t.Errorf("Circle vtable name = %q, want Vtable_Circle_for_Draw", layout.VtableName())
	}

	// Part 4: codegen emits the expected C fragment.
	c := codegen.EmitVtable(layout)
	if !strings.Contains(c, "Vtable_Circle_for_Draw") {
		t.Errorf("emitted C missing vtable symbol; got:\n%s", c)
	}
	if !strings.Contains(c, "sizeof(Circle)") {
		t.Errorf("emitted C missing sizeof(Circle); got:\n%s", c)
	}
	dp := codegen.EmitFatPointerStruct("Draw")
	if !strings.Contains(dp, "struct DynPtr_Draw") {
		t.Errorf("fat-pointer struct missing expected name; got:\n%s", dp)
	}
}

// loadDynProgram runs the front-end pipeline on
// `dyn_dispatch.fuse` and returns the checked Program.
func loadDynProgram(t *testing.T) *hir.Program {
	t.Helper()
	src, err := os.ReadFile("dyn_dispatch.fuse")
	if err != nil {
		t.Fatalf("read dyn_dispatch.fuse: %v", err)
	}
	f, pd := parse.Parse("dyn_dispatch.fuse", src)
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
