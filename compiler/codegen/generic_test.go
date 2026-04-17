package codegen

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lower"
	"github.com/Tembocs/fuse5/compiler/monomorph"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// emitFromSource drives the full pipeline up to codegen and
// returns the emitted C11 text for the given Fuse source. Test
// helpers reuse this so each test body stays focused on one
// assertion.
func emitFromSource(t *testing.T, src string) string {
	t.Helper()
	f, pd := parse.Parse("m.fuse", []byte(src))
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
	mono, md := monomorph.Specialize(prog)
	if len(md) != 0 {
		t.Fatalf("monomorph: %v", md)
	}
	mir, ld := lower.Lower(mono)
	if len(ld) != 0 {
		t.Fatalf("lower: %v", ld)
	}
	out, err := EmitC11(mir)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	return out
}

// TestGenericOriginalsSkipped — W08-P04-T02. After monomorph, the
// generic original `identity` is not in the emitted C; only the
// specialized `identity__I32` (and `main`) appear.
func TestGenericOriginalsSkipped(t *testing.T) {
	out := emitFromSource(t, `
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity[I32](42); }
`)
	// The specialized name must be present.
	if !strings.Contains(out, "identity__I32") {
		t.Errorf("emitted C missing identity__I32 specialization; got:\n%s", out)
	}
	// The generic original must NOT appear. Lowerer drops it from
	// the MIR Module, so codegen never sees it.
	if strings.Contains(out, "fuse_identity(") {
		t.Errorf("generic original `identity` reached codegen; got:\n%s", out)
	}
}

// TestDistinctSpecializations — W08-P06-T03. Two specializations
// of the same generic fn produce two distinct C function
// definitions with distinct names. The spine's one-statement
// body limit is worked around by doing both call sites inside
// helper fns so main remains single-statement.
func TestDistinctSpecializations(t *testing.T) {
	out := emitFromSource(t, `
fn first[T](x: T) -> I32 { return 0; }
fn call_i32() -> I32 { return first[I32](1); }
fn call_bool() -> I32 { return first[Bool](2); }
fn main() -> I32 { return call_i32() + call_bool(); }
`)
	if !strings.Contains(out, "first__I32") {
		t.Errorf("missing first__I32 specialization; got:\n%s", out)
	}
	if !strings.Contains(out, "first__Bool") {
		t.Errorf("missing first__Bool specialization; got:\n%s", out)
	}
}

// TestSpecializedEnumTypes — W08-P05-T01. Generic enum types
// specialize through the TypeTable's nominal-identity lattice.
// At W08 the user-facing proof is that two references to the
// same generic enum with different type args produce two
// distinct TypeIds. The enum's C emission path matures in W13
// (trait objects) / W15 (MIR consolidation); at W08 we assert
// the TypeId-level distinction.
func TestSpecializedEnumTypes(t *testing.T) {
	tab := typetable.New()
	// Simulate Option[T] as a nominal enum; the checker uses
	// TypeTable.Nominal(KindEnum, Symbol, Module, Name, TypeArgs).
	optI32 := tab.Nominal(typetable.KindEnum, 42, "m", "Option", []typetable.TypeId{tab.I32()})
	optBool := tab.Nominal(typetable.KindEnum, 42, "m", "Option", []typetable.TypeId{tab.Bool()})
	if optI32 == optBool {
		t.Fatalf("Option[I32] and Option[Bool] must be distinct TypeIds")
	}
	// Repeated intern of the same instantiation returns the same TypeId.
	optI32b := tab.Nominal(typetable.KindEnum, 42, "m", "Option", []typetable.TypeId{tab.I32()})
	if optI32 != optI32b {
		t.Fatalf("repeated Option[I32] intern produced different TypeIds")
	}
}

// TestGenericStructLayout — W08-P05-T02. Generic struct
// instantiations are distinct nominal TypeIds (same
// Symbol/Module/Name, different TypeArgs).
func TestGenericStructLayout(t *testing.T) {
	tab := typetable.New()
	vecI32 := tab.Nominal(typetable.KindStruct, 7, "m", "Vec", []typetable.TypeId{tab.I32()})
	vecF64 := tab.Nominal(typetable.KindStruct, 7, "m", "Vec", []typetable.TypeId{tab.F64()})
	if vecI32 == vecF64 {
		t.Fatalf("Vec[I32] and Vec[F64] must be distinct TypeIds")
	}
	// Unspecialized Vec is distinct from both specializations.
	vecUnspec := tab.Nominal(typetable.KindStruct, 7, "m", "Vec", nil)
	if vecUnspec == vecI32 || vecUnspec == vecF64 {
		t.Fatalf("unspecialized Vec must be distinct from its specializations")
	}
}
