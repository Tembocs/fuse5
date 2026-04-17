package lower

import (
	"reflect"
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestDynTraitFatPointer — W13-P02-T01. The fat-pointer shape
// is always `{data, vtable}` regardless of the trait; the
// DynType field tracks which `dyn Trait` TypeId it belongs to.
func TestDynTraitFatPointer(t *testing.T) {
	tab := typetable.New()
	traitID := tab.Nominal(typetable.KindTrait, 100, "m", "Greet", nil)
	shape := FatPointerShape(traitID)
	if shape.DataField != "data" {
		t.Errorf("DataField = %q, want %q", shape.DataField, "data")
	}
	if shape.VtableField != "vtable" {
		t.Errorf("VtableField = %q, want %q", shape.VtableField, "vtable")
	}
	if shape.DynType != traitID {
		t.Errorf("DynType not preserved: got %v, want %v", shape.DynType, traitID)
	}
}

// TestDynOwnershipForms — W13-P02-T02. The fat-pointer shape
// is stable across `ref dyn Trait`, `mutref dyn Trait`, and
// `owned dyn Trait`. Ownership differs only in how the
// receiving param is typed; the layout is the same. This
// test constructs three TypeIds using the W04 TypeTable
// helpers and confirms all three produce a valid FatPointer
// when the DynType is the trait-object TypeId itself.
func TestDynOwnershipForms(t *testing.T) {
	tab := typetable.New()
	traitID := tab.Nominal(typetable.KindTrait, 101, "m", "Show", nil)
	dyn := tab.TraitObject([]typetable.TypeId{traitID})
	refDyn := tab.Ref(dyn)
	mutrefDyn := tab.Mutref(dyn)

	for _, tid := range []typetable.TypeId{dyn, refDyn, mutrefDyn} {
		shape := FatPointerShape(tid)
		if shape.DataField == "" || shape.VtableField == "" {
			t.Errorf("FatPointerShape returned empty field(s) for TypeId %v", tid)
		}
	}
}

// mkTrait creates a minimal synthetic TraitDecl with the
// given method names; each method has a `self` receiver and
// returns Unit.
func mkTrait(tab *typetable.Table, name string, methodNames ...string) *hir.TraitDecl {
	items := make([]hir.Item, 0, len(methodNames))
	for _, m := range methodNames {
		items = append(items, &hir.FnDecl{
			Base:   hir.Base{ID: hir.ItemID("m", m)},
			Name:   m,
			TypeID: tab.Fn([]typetable.TypeId{tab.Unit()}, tab.Unit(), false),
			Params: []*hir.Param{{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: hir.NodeID("m::" + m + "::self")}, Type: tab.Unit()},
				Name:      "self",
			}},
			Return: tab.Unit(),
		})
	}
	return &hir.TraitDecl{
		Base:   hir.Base{ID: hir.ItemID("m", name)},
		Name:   name,
		TypeID: tab.Nominal(typetable.KindTrait, 200, "m", name, nil),
		Items:  items,
	}
}

// TestVtableLayoutShape — companion to W13-P03-T01 codegen
// test. The layout always begins with size/align/drop_fn and
// then lists methods in declared order.
func TestVtableLayoutShape(t *testing.T) {
	tab := typetable.New()
	trait := mkTrait(tab, "Draw", "draw", "clear")
	layout := BuildVtableLayout(trait, "Circle")
	want := []VtableEntry{
		{Name: "size", Kind: SlotSize},
		{Name: "align", Kind: SlotAlign},
		{Name: "drop_fn", Kind: SlotDropFn},
		{Name: "draw", Kind: SlotMethod},
		{Name: "clear", Kind: SlotMethod},
	}
	if !reflect.DeepEqual(layout.Entries, want) {
		t.Fatalf("vtable entries:\n got  %+v\n want %+v", layout.Entries, want)
	}
	if layout.VtableName() != "Vtable_Circle_for_Draw" {
		t.Errorf("VtableName = %q, want Vtable_Circle_for_Draw", layout.VtableName())
	}
}

// TestCombinedVtableOrdering — companion to W13-P03-T02. `dyn
// A + B` combines A's and B's method lists in alphabetical
// trait order regardless of how they were passed in.
func TestCombinedVtableOrdering(t *testing.T) {
	tab := typetable.New()
	traitB := mkTrait(tab, "B", "bb")
	traitA := mkTrait(tab, "A", "aa")
	partA := BuildVtableLayout(traitA, "S")
	partB := BuildVtableLayout(traitB, "S")

	combined := CombinedVtable("S", []VtableLayout{partB, partA}) // reversed
	// Expected: header + A's method + B's method (alphabetical).
	var gotMethods []string
	for _, e := range combined.Entries {
		if e.Kind == SlotMethod {
			gotMethods = append(gotMethods, e.Name)
		}
	}
	want := []string{"aa", "bb"}
	if !reflect.DeepEqual(gotMethods, want) {
		t.Errorf("combined method order = %v, want %v (alphabetical by trait)", gotMethods, want)
	}
	if combined.TraitName != "A__B" {
		t.Errorf("combined TraitName = %q, want A__B", combined.TraitName)
	}
}
