package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// lookupTraitMethod resolves a method call `trait.method(target, ...)`
// by finding the impl block that ties `trait` to `target`. Returns
// the method's FnDecl and true on success; nil, false when no impl
// exists. A call-site is expected to have already resolved the
// trait TypeId either directly (`<T as Trait>.method(...)`) or via
// a bound on a generic parameter.
func (c *checker) lookupTraitMethod(trait, target typetable.TypeId, method string) (*hir.FnDecl, bool) {
	if blk, ok := c.implByPair[coherenceKey{Trait: trait, Target: target}]; ok {
		if fn, ok := blk.Methods[method]; ok {
			return fn, true
		}
	}
	// Bound-chain lookup: if the target is a generic parameter
	// with a bound, walk its bounds for a matching method.
	tt := c.tab.Get(target)
	if tt != nil && tt.Kind == typetable.KindGenericParam {
		for _, b := range c.boundsForGeneric(target) {
			if fn, ok := c.lookupTraitMethod(b, target, method); ok {
				return fn, true
			}
		}
	}
	return nil, false
}

// lookupInherentMethod resolves a method call on a concrete type
// via an inherent impl block (`impl T { fn method(...) }`).
func (c *checker) lookupInherentMethod(target typetable.TypeId, method string) (*hir.FnDecl, bool) {
	if blk, ok := c.implByPair[coherenceKey{Trait: typetable.NoType, Target: target}]; ok {
		if fn, ok := blk.Methods[method]; ok {
			return fn, true
		}
	}
	return nil, false
}

// boundsForGeneric returns the trait bounds that apply to a
// generic parameter TypeId. The bridge stores bounds on
// GenericParam nodes; at W06 we scan linearly since per-fn
// generic counts are small.
func (c *checker) boundsForGeneric(tid typetable.TypeId) []typetable.TypeId {
	var out []typetable.TypeId
	for _, modPath := range c.prog.Order {
		m := c.prog.Modules[modPath]
		for _, it := range m.Items {
			switch x := it.(type) {
			case *hir.FnDecl:
				for _, g := range x.Generics {
					if g.TypeID == tid {
						out = append(out, g.Bounds...)
					}
				}
			case *hir.ImplDecl:
				for _, g := range x.Generics {
					if g.TypeID == tid {
						out = append(out, g.Bounds...)
					}
				}
			}
		}
	}
	return out
}

// reportUnresolvedMethod emits the uniform "no matching method"
// diagnostic used by callers that failed both inherent and trait
// lookup.
func (c *checker) reportUnresolvedMethod(span lex.Span, target typetable.TypeId, method string) {
	c.diagnose(span,
		fmt.Sprintf("no method %q on type %s", method, c.typeName(target)),
		"check that the type implements a trait providing this method, or define an inherent impl")
}
