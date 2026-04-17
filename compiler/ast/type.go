package ast

// PathType is a named type, optionally with generic arguments — `Foo`,
// `std.collections.Vec[Int]`.
type PathType struct {
	NodeBase
	Segments []Ident
	Args     []Type // generic args, nil when no `[T, U]` clause
}

// TupleType is `(T1, T2, ...)`. The single-element form `(T,)` is written by
// the parser when the grammar allows it.
type TupleType struct {
	NodeBase
	Elements []Type
}

// ArrayType is `[T; N]` — fixed-size array.
type ArrayType struct {
	NodeBase
	Element Type
	Length  Expr
}

// SliceType is `[T]` — unbounded slice.
type SliceType struct {
	NodeBase
	Element Type
}

// PtrType is `Ptr[T]` — raw pointer.
type PtrType struct {
	NodeBase
	Pointee Type
}

// FnType is `fn(T1, T2) -> R`.
type FnType struct {
	NodeBase
	Params []Type
	Return Type // nil for unit return
}

// DynType is `dyn Trait1 + Trait2 + ...` — trait object.
type DynType struct {
	NodeBase
	Traits []Type
}

// ImplType is `impl Trait` — existential type.
type ImplType struct {
	NodeBase
	Trait Type
}

// UnitType is `()`.
type UnitType struct {
	NodeBase
}

// typeNode markers.
func (*PathType) typeNode()  {}
func (*TupleType) typeNode() {}
func (*ArrayType) typeNode() {}
func (*SliceType) typeNode() {}
func (*PtrType) typeNode()   {}
func (*FnType) typeNode()    {}
func (*DynType) typeNode()   {}
func (*ImplType) typeNode()  {}
func (*UnitType) typeNode()  {}
