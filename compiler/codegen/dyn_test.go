package codegen

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/lower"
)

// TestVtableEmission — W13-P03-T01. EmitVtable produces a
// struct + static table with the expected header (size/align/
// drop_fn) and method slots. The table is named
// `Vtable_<Concrete>_for_<Trait>` and contains a `sizeof`/
// `_Alignof` initializer pair.
func TestVtableEmission(t *testing.T) {
	layout := lower.VtableLayout{
		TraitName:    "Draw",
		ConcreteName: "Circle",
		Entries: []lower.VtableEntry{
			{Name: "size", Kind: lower.SlotSize},
			{Name: "align", Kind: lower.SlotAlign},
			{Name: "drop_fn", Kind: lower.SlotDropFn},
			{Name: "draw", Kind: lower.SlotMethod},
		},
	}
	out := EmitVtable(layout)
	mustContain(t, out, "struct VtableLayout_Draw")
	mustContain(t, out, "Vtable_Circle_for_Draw")
	mustContain(t, out, "sizeof(Circle)")
	mustContain(t, out, "_Alignof(Circle)")
	mustContain(t, out, "Circle_drop")
	mustContain(t, out, "Circle_draw")
}

// TestDynTraitMulti — W13-P03-T02. Combined vtables for
// `dyn A + B` contain the methods of both traits in
// alphabetical trait order. The emitted C still renders as a
// single table under the combined-name scheme.
func TestDynTraitMulti(t *testing.T) {
	// Manually construct a combined layout (CombinedVtable
	// from lower) with a single header, A's method, then B's
	// method.
	combined := lower.VtableLayout{
		TraitName:    "A__B",
		ConcreteName: "S",
		Entries: []lower.VtableEntry{
			{Name: "size", Kind: lower.SlotSize},
			{Name: "align", Kind: lower.SlotAlign},
			{Name: "drop_fn", Kind: lower.SlotDropFn},
			{Name: "aa", Kind: lower.SlotMethod},
			{Name: "bb", Kind: lower.SlotMethod},
		},
	}
	out := EmitVtable(combined)
	mustContain(t, out, "VtableLayout_A__B")
	mustContain(t, out, "Vtable_S_for_A__B")
	// The methods from both traits must appear in the emitted
	// table in the same alphabetical order.
	idxAA := strings.Index(out, ".aa")
	idxBB := strings.Index(out, ".bb")
	if idxAA < 0 || idxBB < 0 {
		t.Fatalf("missing method initializers; got:\n%s", out)
	}
	if idxAA > idxBB {
		t.Errorf("expected `aa` before `bb` in combined vtable; got reversed order")
	}
}

// TestDynMethodDispatch — W13-P04-T01. EmitMethodDispatch
// renders the vtable-indirect call shape expected by §57.8.
func TestDynMethodDispatch(t *testing.T) {
	call := EmitMethodDispatch("rfat", "draw", "VtableLayout_Draw", []string{"r1"})
	mustContain(t, call, "VtableLayout_Draw")
	mustContain(t, call, "rfat.vtable")
	mustContain(t, call, "->draw")
	mustContain(t, call, "rfat.data")
	mustContain(t, call, "r1")
}

// TestFatPointerStruct — structural: the fat-pointer struct
// has exactly two fields (`data`, `vtable`) with the right
// types.
func TestFatPointerStruct(t *testing.T) {
	out := EmitFatPointerStruct("Draw")
	mustContain(t, out, "struct DynPtr_Draw")
	mustContain(t, out, "void *data")
	mustContain(t, out, "const void *vtable")
}
