package check

// CallableTrait enumerates the three intrinsic callable traits
// §15.6: Fn, FnMut, FnOnce. They form a hierarchy — every Fn is
// an FnMut, every FnMut is an FnOnce — so a value that
// auto-impls Fn is also usable where FnMut or FnOnce is
// expected.
//
// The checker exposes these as intrinsic (user code never
// declares them); the stdlib-core wave re-exports them for
// user-facing use. W12 uses them to classify closures and
// function pointers at use sites.
type CallableTrait int

const (
	CallableFn CallableTrait = iota
	CallableFnMut
	CallableFnOnce
)

// String returns the trait's declared name.
func (t CallableTrait) String() string {
	switch t {
	case CallableFn:
		return "Fn"
	case CallableFnMut:
		return "FnMut"
	case CallableFnOnce:
		return "FnOnce"
	}
	return "unknown"
}

// CallableTraitFor returns the set of traits a given shape
// auto-implements, using the same classification the lowerer's
// closure analysis uses. Function pointers auto-impl all three;
// closures depend on their capture set.
//
// The wave-level rule (reference §15.6): calling through the
// tightest bound the receiver satisfies selects the call method
// (`call`, `call_mut`, `call_once`). The checker runs this
// classification to attribute auto-impls; the lowerer's
// `DesugarCall` maps trait → method name.
type CallableShape int

const (
	// ShapeFnPointer — a `fn(A, B) -> R` value; auto-impls all
	// three callable traits because it carries no env.
	ShapeFnPointer CallableShape = iota
	// ShapeNoCaptureClosure — a closure with no captures; same
	// behavior as a fn pointer.
	ShapeNoCaptureClosure
	// ShapeReadOnlyClosure — closure whose captures are all
	// Copy or Ref; auto-impls Fn + FnMut + FnOnce.
	ShapeReadOnlyClosure
	// ShapeMutClosure — closure with at least one Mutref
	// capture and no Owned captures; auto-impls FnMut + FnOnce.
	ShapeMutClosure
	// ShapeOnceClosure — closure with at least one Owned
	// capture; auto-impls FnOnce only.
	ShapeOnceClosure
)

// CallableTraitFor returns the set of callable traits a given
// shape satisfies. The returned slice is sorted by the trait's
// hierarchical position (Fn first when present, then FnMut,
// then FnOnce) so dispatch picks the tightest via slice[0].
func CallableTraitFor(s CallableShape) []CallableTrait {
	switch s {
	case ShapeFnPointer, ShapeNoCaptureClosure, ShapeReadOnlyClosure:
		return []CallableTrait{CallableFn, CallableFnMut, CallableFnOnce}
	case ShapeMutClosure:
		return []CallableTrait{CallableFnMut, CallableFnOnce}
	case ShapeOnceClosure:
		return []CallableTrait{CallableFnOnce}
	}
	return nil
}

// TightestCallableTrait returns the single tightest trait a
// shape satisfies — the one the call-desugaring lowerer uses to
// pick the call method.
func TightestCallableTrait(s CallableShape) (CallableTrait, bool) {
	traits := CallableTraitFor(s)
	if len(traits) == 0 {
		return CallableFn, false
	}
	return traits[0], true
}
