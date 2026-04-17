package lower

import (
	"sort"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// W13 `dyn Trait` lowering primitives. The wave's runtime
// representation (reference §57.8) is a fat pointer whose first
// word is the erased data pointer and whose second word is a
// pointer to the vtable. Vtables have a fixed-layout header
// (`size`, `align`, `drop_fn`) followed by method pointers in
// trait-declaration order. For `dyn A + B` the vtables are
// concatenated in alphabetical trait order so the layout is
// deterministic across runs.
//
// This file owns the shape helpers; codegen emits the real C
// structs and static tables. Keeping the shape computation in
// `lower` lets W13 test the layout logic without threading
// through the full pipeline.

// FatPointer is the lowered representation of a `dyn Trait`
// value. Field ordering is stable and matches §57.8: `data`
// (the erased receiver pointer) first, then `vtable` (the
// concrete-impl's method table).
type FatPointer struct {
	DataField   string // canonical field name for codegen ("data")
	VtableField string // canonical field name ("vtable")
	// DynType is the TypeTable-level `dyn Trait` TypeId the
	// FatPointer represents; downstream passes key off it.
	DynType typetable.TypeId
}

// FatPointerShape computes the fat-pointer description for a
// `dyn Trait` type. Accepts the TypeTable-level KindTraitObject
// TypeId. The W13 shape is the same for every trait-object
// flavour — the difference between `dyn A` and `dyn A + B`
// shows up only in the vtable layout (see `VtableLayout`), not
// in the fat-pointer itself.
func FatPointerShape(dynType typetable.TypeId) FatPointer {
	return FatPointer{
		DataField:   "data",
		VtableField: "vtable",
		DynType:     dynType,
	}
}

// VtableEntry is one slot in a vtable. The first three slots
// are always `size`, `align`, and `drop_fn` in that order;
// method slots follow in the trait's declaration order.
type VtableEntry struct {
	Name string // spelled as it appears in the emitted C table
	Kind VtableSlotKind
}

// VtableSlotKind tags what an entry holds so codegen can pick
// the right C type.
type VtableSlotKind int

const (
	SlotSize VtableSlotKind = iota
	SlotAlign
	SlotDropFn
	SlotMethod
)

// String returns the slot's C-level role name.
func (k VtableSlotKind) String() string {
	switch k {
	case SlotSize:
		return "size"
	case SlotAlign:
		return "align"
	case SlotDropFn:
		return "drop_fn"
	case SlotMethod:
		return "method"
	}
	return "unknown"
}

// VtableLayout describes the vtable shape for one (trait,
// concrete impl) pair. The Entries slice is ordered: header
// three (size/align/drop_fn) followed by method entries in
// the trait's declaration order. TraitName and ConcreteName
// together name the static C table ("Vtable_<ConcreteName>_for_<TraitName>").
type VtableLayout struct {
	TraitName    string
	ConcreteName string
	Entries      []VtableEntry
}

// VtableName returns the deterministic symbol under which the
// codegen emits this vtable in the generated C. Format:
// "Vtable_<ConcreteName>_for_<TraitName>" — alphanumeric-only
// so no extra escaping is needed.
func (l VtableLayout) VtableName() string {
	return "Vtable_" + safeIdent(l.ConcreteName) + "_for_" + safeIdent(l.TraitName)
}

// BuildVtableLayout returns the vtable layout for one
// (trait, impl) pair. The method entries are taken from the
// trait's declaration in declaration order — the W13 contract
// requires this deterministic ordering so two builds of the
// same program emit identical vtables.
func BuildVtableLayout(trait *hir.TraitDecl, concreteName string) VtableLayout {
	entries := []VtableEntry{
		{Name: "size", Kind: SlotSize},
		{Name: "align", Kind: SlotAlign},
		{Name: "drop_fn", Kind: SlotDropFn},
	}
	if trait != nil {
		for _, it := range trait.Items {
			if fn, ok := it.(*hir.FnDecl); ok {
				entries = append(entries, VtableEntry{
					Name: fn.Name,
					Kind: SlotMethod,
				})
			}
		}
	}
	traitName := ""
	if trait != nil {
		traitName = trait.Name
	}
	return VtableLayout{
		TraitName:    traitName,
		ConcreteName: concreteName,
		Entries:      entries,
	}
}

// CombinedVtable joins multiple per-trait vtable layouts into
// one concatenated vtable for a `dyn A + B` use site. The
// header (size/align/drop_fn) appears once; each trait's
// method list follows in alphabetical trait order (by
// TraitName) so the layout is deterministic.
//
// Returns a single VtableLayout whose Entries are the header
// followed by each trait's methods, with a synthetic
// TraitName like "A__B" combining the trait names.
func CombinedVtable(concreteName string, parts []VtableLayout) VtableLayout {
	// Sort the per-trait parts alphabetically by TraitName.
	sorted := make([]VtableLayout, len(parts))
	copy(sorted, parts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TraitName < sorted[j].TraitName
	})
	entries := []VtableEntry{
		{Name: "size", Kind: SlotSize},
		{Name: "align", Kind: SlotAlign},
		{Name: "drop_fn", Kind: SlotDropFn},
	}
	traitNames := make([]string, 0, len(sorted))
	for _, v := range sorted {
		traitNames = append(traitNames, v.TraitName)
		// Skip the first three (header) entries from each
		// part — they're only counted once.
		for _, e := range v.Entries {
			if e.Kind == SlotMethod {
				entries = append(entries, e)
			}
		}
	}
	combinedName := joinTraitNames(traitNames)
	return VtableLayout{
		TraitName:    combinedName,
		ConcreteName: concreteName,
		Entries:      entries,
	}
}

// joinTraitNames collapses a sorted trait-name list into the
// combined name used for `dyn A + B` vtables. Kept as a
// helper so the format is testable without import tricks.
func joinTraitNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += "__"
		}
		out += n
	}
	return out
}

// safeIdent strips any character that isn't a legal C
// identifier constituent.
func safeIdent(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_' {
			b = append(b, c)
		} else {
			b = append(b, '_')
		}
	}
	return string(b)
}
