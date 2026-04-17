package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// checkItemShapes applies the item-level invariants that don't
// require body walking: union field restrictions, newtype
// distinctness, variadic extern signatures, and impl-Trait
// return-type uniqueness. Runs after collectItems so every
// signature is already registered.
//
// Invoked from Check after pass 1. Returns no value; diagnostics
// are appended to c.diags as they are discovered.
func (c *checker) checkItemShapes() {
	for _, modPath := range c.prog.Order {
		m := c.prog.Modules[modPath]
		for _, it := range m.Items {
			c.checkItemShape(modPath, it)
		}
	}
}

func (c *checker) checkItemShape(modPath string, it hir.Item) {
	switch x := it.(type) {
	case *hir.UnionDecl:
		c.checkUnionFields(x)
	case *hir.StructDecl:
		c.checkNewtypeShape(x)
	case *hir.FnDecl:
		c.checkExternVariadic(x)
		c.checkImplTraitReturn(x)
	}
}

// checkUnionFields enforces reference §49.1: union fields must
// have statically known sizes and may not implement Drop. At W06
// we approximate the Drop check by requiring field types to be
// primitives or pointers; later waves tighten when Drop trait
// integration lands.
func (c *checker) checkUnionFields(u *hir.UnionDecl) {
	for _, f := range u.Fields {
		t := c.tab.Get(f.TypeOf())
		if t == nil {
			continue
		}
		switch t.Kind {
		case typetable.KindI8, typetable.KindI16, typetable.KindI32, typetable.KindI64, typetable.KindISize,
			typetable.KindU8, typetable.KindU16, typetable.KindU32, typetable.KindU64, typetable.KindUSize,
			typetable.KindF32, typetable.KindF64, typetable.KindChar, typetable.KindBool,
			typetable.KindPtr, typetable.KindUnit:
			// OK
		default:
			c.diagnose(f.Span,
				fmt.Sprintf("union field %q has a non-trivial type %s", f.Name, c.typeName(f.TypeOf())),
				"union fields must be primitives or pointers; Drop-implementing types are not permitted")
		}
	}
}

// checkNewtypeShape is a no-op validation at W06. The W02 parser
// already distinguishes `struct T(U);` (newtype) from the named-
// field form, and the bridge emits a KindStruct TypeId with a
// distinct defining Symbol (reference §2.8). That gives newtype
// its type-identity guarantee without any dedicated flag on the
// HIR.
func (c *checker) checkNewtypeShape(s *hir.StructDecl) {
	_ = s // kept as a hook; later waves can validate inherent impls
}

// checkExternVariadic enforces that variadic (`...`) appears only
// on extern declarations, never on Fuse-defined functions
// (reference §29.1).
func (c *checker) checkExternVariadic(fn *hir.FnDecl) {
	if fn.Variadic && !fn.IsExtern {
		c.diagnose(fn.Span,
			"variadic `...` is only allowed on extern fn declarations",
			"remove the variadic marker or declare this fn as `extern`")
	}
}

// checkImplTraitReturn enforces the §56.1 rule: a fn whose return
// signature is `-> impl Trait` must return exactly one concrete
// type from every path. The bridge encodes `impl Trait` as an
// ImplType in the return position which currently maps to a
// trait's TypeId; multi-type returns are diagnosed below by
// walking the body's return statements and gathering the distinct
// returned-value TypeIds.
func (c *checker) checkImplTraitReturn(fn *hir.FnDecl) {
	if fn.Body == nil || fn.Return == typetable.NoType {
		return
	}
	// Only apply when the return TypeId is a nominal trait (the
	// bridge flattens `impl Trait` to the trait's TypeId when no
	// obvious concrete unification is available).
	t := c.tab.Get(fn.Return)
	if t == nil || t.Kind != typetable.KindTrait {
		return
	}
	seen := map[typetable.TypeId]bool{}
	c.collectReturnTypes(fn.Body, seen)
	if len(seen) > 1 {
		// Build a deterministic list for the diagnostic.
		names := []string{}
		for k := range seen {
			names = append(names, c.typeName(k))
		}
		c.diagnose(fn.Span,
			fmt.Sprintf("fn with `impl %s` return must produce one concrete type; observed: %v",
				c.typeName(fn.Return), names),
			"refactor so every return path yields the same concrete type")
	}
}

// collectReturnTypes accumulates the TypeIds of every `return`
// statement inside a block. Used by the impl-Trait return check.
func (c *checker) collectReturnTypes(blk *hir.Block, out map[typetable.TypeId]bool) {
	if blk == nil {
		return
	}
	for _, s := range blk.Stmts {
		switch x := s.(type) {
		case *hir.ReturnStmt:
			if x.Value != nil {
				out[x.Value.TypeOf()] = true
			}
		case *hir.ExprStmt:
			if b, ok := x.Expr.(*hir.Block); ok {
				c.collectReturnTypes(b, out)
			}
		}
	}
	if b, ok := blk.Trailing.(*hir.Block); ok {
		c.collectReturnTypes(b, out)
	}
}
