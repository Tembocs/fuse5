package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Associated types (reference §30.1) let a trait parameterise
// over a type the implementer supplies. At W06 the checker:
//
//   - Records the set of declared associated-type names per trait
//     (populated in collectItems via traitInfo.AssocTypes).
//   - Validates that every `impl Trait for Type` provides a
//     concrete type for each associated-type declaration
//     (`ImplTypeItem` at AST level).
//   - Projects `Self::Item` inside trait bodies to the associated
//     name; the real concrete lookup happens when the call site
//     resolves through a specific impl.
//
// The bridge emits AssocType constraints on a trait's items list
// but the HIR item shape at W04 does not yet carry a dedicated
// node for them. For W06 we inspect the underlying AST via the
// trait's Items slice and treat any non-FnDecl item as an
// associated-type declaration by name. This is a narrow W06
// shortcut that W08 (monomorphization) will formalise.

// checkAssociatedTypesCoverage verifies that every impl of a
// trait provides a concrete type for each of the trait's
// associated-type declarations. Missing projections produce a
// diagnostic naming the trait, the impl, and the missing item.
func (c *checker) checkAssociatedTypesCoverage() {
	for _, blk := range c.impls {
		if blk.Trait == typetable.NoType {
			continue
		}
		ti := c.traitInfoForTypeId(blk.Trait)
		if ti == nil || len(ti.AssocTypes) == 0 {
			continue
		}
		// Gather the set of names the impl provides.
		provided := map[string]bool{}
		for _, sub := range blk.Decl.Items {
			if ta, ok := sub.(*hir.TypeAliasDecl); ok {
				provided[ta.Name] = true
			}
		}
		for name := range ti.AssocTypes {
			if !provided[name] {
				c.diagnose(blk.Decl.Span,
					fmt.Sprintf("impl of %s for %s is missing associated type %q",
						c.typeName(blk.Trait), c.typeName(blk.Target), name),
					"add `type "+name+" = ...;` inside the impl body")
			}
		}
	}
}

// traitInfoForTypeId finds the traitInfo record whose Decl's
// TypeID matches tid. Used by coverage and method-lookup paths.
func (c *checker) traitInfoForTypeId(tid typetable.TypeId) *traitInfo {
	for _, ti := range c.traits {
		if ti.Decl != nil && ti.Decl.TypeID == tid {
			return ti
		}
	}
	return nil
}
