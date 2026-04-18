package hir

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestBridgeSelfAndSelfTypeLowering pins the 2026-04-18 audit fix for
// impl/trait Self resolution at HIR-bridge time.
//
// Before the fix, two shortcuts in the bridge leaked Unit / Infer
// types into impl and trait method signatures:
//
//   - A parameter spelled bare `self` has no AST type (the parser drops
//     the `: Self` annotation per reference §9). lowerType(nil) then
//     returned Unit, so `self + other` in `impl I32` typed as
//     `Unit vs I32` during W06 checking.
//   - A parameter typed `Self` lowered via lowerPathType, which treated
//     `Self` as an unresolved path and returned Infer. Default methods
//     in `pub trait PartialOrd { fn lt(self, other: Self) -> Bool }`
//     then failed check with `parameter "other" has no declared type`.
//
// The bridge now tracks `selfType` across lowerImpl/lowerTrait frames
// and substitutes it both for unannotated `self` and for the `Self`
// type keyword. This test builds a mini impl + trait and asserts each
// parameter carries the expected TypeId.
func TestBridgeSelfAndSelfTypeLowering(t *testing.T) {
	prog, tab := bridgeTest(t, "m", "m.fuse", `
struct Point { x: I32, y: I32 }

impl Point {
    pub fn add(self, other: Point) -> I32 {
        return 0;
    }

    pub fn identity(self) -> Point {
        return self;
    }
}

pub trait Eq {
    fn eq(self, other: Self) -> Bool;
}
`)
	m := prog.Modules["m"]
	if m == nil {
		t.Fatalf("module m missing")
	}

	var impl *ImplDecl
	var trait *TraitDecl
	for _, it := range m.Items {
		switch x := it.(type) {
		case *ImplDecl:
			impl = x
		case *TraitDecl:
			trait = x
		}
	}
	if impl == nil {
		t.Fatalf("ImplDecl not lowered")
	}
	if trait == nil {
		t.Fatalf("TraitDecl not lowered")
	}

	// --- impl Point methods: self and Self both resolve to Point.
	want := impl.Target
	if want == typetable.NoType || want == tab.Infer() {
		t.Fatalf("impl.Target is not a concrete TypeId: %v", want)
	}
	for _, sub := range impl.Items {
		fn, ok := sub.(*FnDecl)
		if !ok {
			continue
		}
		for _, p := range fn.Params {
			if p.Name == "self" && p.TypeOf() != want {
				t.Errorf("impl Point::%s: self param TypeOf = %d, want %d (Target)",
					fn.Name, p.TypeOf(), want)
			}
			if p.Name == "other" && p.TypeOf() != want {
				t.Errorf("impl Point::%s: other:Point param TypeOf = %d, want %d",
					fn.Name, p.TypeOf(), want)
			}
		}
		if fn.Name == "identity" && fn.Return != want {
			t.Errorf("identity return TypeId = %d, want Point (%d)", fn.Return, want)
		}
	}

	// --- trait Eq: self and Self both resolve to trait.TypeID.
	traitSelf := trait.TypeID
	if traitSelf == typetable.NoType || traitSelf == tab.Infer() {
		t.Fatalf("trait.TypeID is not a concrete TypeId: %v", traitSelf)
	}
	for _, sub := range trait.Items {
		fn, ok := sub.(*FnDecl)
		if !ok || fn.Name != "eq" {
			continue
		}
		for _, p := range fn.Params {
			if p.TypeOf() != traitSelf {
				t.Errorf("trait Eq::eq: %s param TypeOf = %d, want %d (trait Self)",
					p.Name, p.TypeOf(), traitSelf)
			}
		}
	}
}
