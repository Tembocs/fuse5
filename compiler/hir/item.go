package hir

import (
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Module is one resolved-and-lowered module. One Module exists per
// dotted module path.
type Module struct {
	Base
	Path  string // dotted module path, empty for the crate root
	Items []Item
}

func (m *Module) itemNode() {}

// FnDecl is a lowered function declaration. Generics and where-clauses
// are preserved as metadata so later passes (W06 type checking, W08
// monomorphization) can specialise. The body is a Block; an extern
// declaration has Body == nil.
type FnDecl struct {
	Base
	Name     string
	Params   []*Param
	Return   typetable.TypeId // Unit when the source omitted a return type
	TypeID   typetable.TypeId // the function's own Fn TypeId (builder-enforced)
	// SymID is the resolve.SymbolID (stored as int to avoid import
	// cycles) that this fn declares. Populated by the bridge; zero
	// for synthesized fns that never entered the resolver. Passes
	// that need to correlate a PathExpr.Symbol back to its declaring
	// FnDecl consult this field directly.
	SymID    int
	Body     *Block // nil for extern
	Generics []*GenericParam
	IsExtern bool
	IsConst  bool
	Variadic bool
}

func (f *FnDecl) itemNode() {}

// Param is one function parameter.
type Param struct {
	TypedBase
	Name      string
	Ownership Ownership
}

// Param satisfies Node via the embedded TypedBase; it is intentionally
// neither an Item nor an Expr/Stmt/Pat.

// Ownership tags the ownership mode of a parameter or field reference.
type Ownership int

const (
	OwnNone Ownership = iota
	OwnRef
	OwnMutref
	OwnOwned
)

// StructDecl is a lowered struct declaration. Tuple-structs carry
// TupleFields, named structs carry Fields, unit structs have both
// slices empty and IsUnit true.
type StructDecl struct {
	Base
	Name        string
	TypeID      typetable.TypeId // nominal TypeId (builder-enforced)
	Fields      []*Field
	TupleFields []typetable.TypeId
	IsUnit      bool
	Generics    []*GenericParam
}

func (s *StructDecl) itemNode() {}

// Field is one named field in a struct, union, or struct-variant.
type Field struct {
	TypedBase
	Name      string
	Ownership Ownership
}

// Field satisfies Node via TypedBase; it is not an Item/Expr/Stmt/Pat.

// EnumDecl is a lowered enum declaration. Variants are hoisted into
// the enclosing module's name scope by the resolver (reference §18.6);
// the HIR preserves them as structured Variant nodes.
type EnumDecl struct {
	Base
	Name     string
	TypeID   typetable.TypeId
	Variants []*Variant
	Generics []*GenericParam
	// SymID is the resolve.SymbolID (stored as int to avoid import
	// cycles) that this enum declares. Populated by the bridge.
	SymID int
}

func (e *EnumDecl) itemNode() {}

// Variant is one enum variant — unit, tuple, or struct-shaped.
type Variant struct {
	Base
	Name   string
	TypeID typetable.TypeId // nominal TypeId of the enum
	Tuple  []typetable.TypeId
	Fields []*Field
	IsUnit bool
	// SymID is the variant's own resolve.SymbolID (stored as int to
	// avoid import cycles). Populated by the bridge.
	SymID int
}

// Variant satisfies Node via its embedded Base; it is a child of
// EnumDecl rather than a free-standing Item.

// TraitDecl is a lowered trait declaration.
type TraitDecl struct {
	Base
	Name     string
	TypeID   typetable.TypeId
	Supertrs []typetable.TypeId
	Items    []Item
	Generics []*GenericParam
}

func (t *TraitDecl) itemNode() {}

// ImplDecl is an inherent or trait impl block.
type ImplDecl struct {
	Base
	Target typetable.TypeId // the type being implemented
	Trait  typetable.TypeId // the trait being implemented (NoType for inherent)
	Items  []Item
	Generics []*GenericParam
}

func (i *ImplDecl) itemNode() {}

// ConstDecl is a lowered `const NAME: TYPE = EXPR;`.
type ConstDecl struct {
	Base
	Name  string
	Type  typetable.TypeId
	Value Expr
	// SymID is the resolve.SymbolID (stored as int to avoid import
	// cycles) that this const declares. Populated by the bridge.
	SymID int
}

func (c *ConstDecl) itemNode() {}

// StaticDecl is a lowered `static NAME: TYPE = EXPR;` (or extern).
type StaticDecl struct {
	Base
	Name     string
	Type     typetable.TypeId
	Value    Expr // nil for extern
	IsExtern bool
	// SymID is the resolve.SymbolID (stored as int to avoid import
	// cycles) that this static declares. Populated by the bridge.
	SymID int
}

func (s *StaticDecl) itemNode() {}

// TypeAliasDecl is `type NAME[..] = TYPE;`.
type TypeAliasDecl struct {
	Base
	Name     string
	TypeID   typetable.TypeId
	Target   typetable.TypeId
	Generics []*GenericParam
}

func (t *TypeAliasDecl) itemNode() {}

// UnionDecl is a lowered union.
type UnionDecl struct {
	Base
	Name     string
	TypeID   typetable.TypeId
	Fields   []*Field
	Generics []*GenericParam
}

func (u *UnionDecl) itemNode() {}

// GenericParam is one entry in `<T>` / `<T: Bound>`.
type GenericParam struct {
	Base
	Name   string
	TypeID typetable.TypeId // always KindGenericParam
	Bounds []typetable.TypeId
}

// GenericParam satisfies Node via its embedded Base.
