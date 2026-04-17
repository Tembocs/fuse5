package ast

// Visibility marks an item's public exposure. ("pub", "pub(mod)", "pub(pkg)",
// or absent.)
type Visibility int

const (
	VisPrivate Visibility = iota
	VisPub                // plain `pub`
	VisPubMod             // `pub(mod)`
	VisPubPkg             // `pub(pkg)`
)

// FnDecl is `[pub] [const] [decorator] fn NAME [<generics>] (params) [-> T] [where ...] { body }`.
// If IsExtern is true, the body is absent (extern fn declaration).
type FnDecl struct {
	NodeBase
	Decorators []*Decorator
	Vis        Visibility
	IsConst    bool
	IsExtern   bool
	Variadic   bool // trailing `...` on an extern fn
	Name       Ident
	Generics   []*GenericParam
	Params     []*Param
	Return     Type // nil when omitted (unit return)
	Where      []*WherePred
	Body       *BlockExpr // nil for extern / trait-signature-only
}

// StructDecl is `struct NAME [<generics>] { fields } | ( tuple-fields ); | ;`.
type StructDecl struct {
	NodeBase
	Decorators []*Decorator
	Vis        Visibility
	Name       Ident
	Generics   []*GenericParam
	Fields     []*Field // nil for unit-struct
	Tuple      []Type   // set for tuple-struct; mutually exclusive with Fields
	IsUnit     bool
}

// EnumDecl is `enum NAME [<generics>] { variants }`.
type EnumDecl struct {
	NodeBase
	Decorators []*Decorator
	Vis        Visibility
	Name       Ident
	Generics   []*GenericParam
	Variants   []*Variant
}

// Variant is one enum variant — unit, tuple, or struct-shaped.
type Variant struct {
	NodeBase
	Name       Ident
	Explicit   Expr    // `= EXPR` for discriminants (unit-variant only)
	Tuple      []Type  // set for tuple-variant
	Fields     []*Field // set for struct-variant
}

// TraitDecl is `trait NAME [<generics>] [: supers] [where ...] { items }`.
type TraitDecl struct {
	NodeBase
	Vis       Visibility
	Name      Ident
	Generics  []*GenericParam
	Supertrs  []Type
	Where     []*WherePred
	Items     []Item
}

// ImplDecl is `impl [<generics>] TYPE [: TYPE] [where ...] { items }`.
// Target is the type being implemented; Trait is the trait being implemented
// for (nil for inherent impls).
type ImplDecl struct {
	NodeBase
	Generics []*GenericParam
	Target   Type
	Trait    Type
	Where    []*WherePred
	Items    []Item
}

// ConstDecl is `const NAME: TYPE = EXPR;`.
type ConstDecl struct {
	NodeBase
	Decorators []*Decorator
	Vis        Visibility
	Name       Ident
	Type       Type
	Value      Expr
}

// StaticDecl is `static NAME: TYPE = EXPR;` or an extern static (no Value).
type StaticDecl struct {
	NodeBase
	Decorators []*Decorator
	Vis        Visibility
	IsExtern   bool
	Name       Ident
	Type       Type
	Value      Expr // nil for extern static
}

// TypeDecl is `type NAME [<generics>] = TYPE;`.
type TypeDecl struct {
	NodeBase
	Vis      Visibility
	Name     Ident
	Generics []*GenericParam
	Target   Type
}

// ExternDecl is `extern { ... }` or `extern fn/static ...;`. At W02 we model
// individual extern items; a future `extern { ... }` block form would wrap
// these.
type ExternDecl struct {
	NodeBase
	Item Item // FnDecl with IsExtern, or StaticDecl with IsExtern
}

// UnionDecl is `union NAME [<generics>] { fields }`.
type UnionDecl struct {
	NodeBase
	Decorators []*Decorator
	Vis        Visibility
	Name       Ident
	Generics   []*GenericParam
	Fields     []*Field
}

// TraitTypeItem is `type NAME [: bounds];` inside a trait body.
type TraitTypeItem struct {
	NodeBase
	Name   Ident
	Bounds []Type
}

// TraitConstItem is `const NAME: TYPE [= EXPR];` inside a trait body.
type TraitConstItem struct {
	NodeBase
	Name    Ident
	Type    Type
	Default Expr // nil when the trait leaves the default unspecified
}

// ImplTypeItem is `type NAME = TYPE;` inside an impl body.
type ImplTypeItem struct {
	NodeBase
	Name   Ident
	Target Type
}

// Field is a named field in a struct, union, or struct-variant.
type Field struct {
	NodeBase
	Vis  Visibility
	Name Ident
	Type Type
}

// Param is one function parameter.
type Param struct {
	NodeBase
	Ownership Ownership
	Name      Ident
	Type      Type
}

// Ownership tags reference/mutref/owned on a parameter, or None if absent.
type Ownership int

const (
	OwnNone Ownership = iota
	OwnRef
	OwnMutref
	OwnOwned
)

// GenericParam is one entry in `<T>` / `<T: Bound>`.
type GenericParam struct {
	NodeBase
	Name   Ident
	Bounds []Type
}

// WherePred is one predicate in a `where` clause: either `T: bounds` or
// `T = U` (associated-type equality).
type WherePred struct {
	NodeBase
	Target Type
	IsEq   bool
	Bounds []Type // set when IsEq is false
	Eq     Type   // set when IsEq is true
}

// itemNode markers.
func (*FnDecl) itemNode()         {}
func (*StructDecl) itemNode()     {}
func (*EnumDecl) itemNode()       {}
func (*TraitDecl) itemNode()      {}
func (*ImplDecl) itemNode()       {}
func (*ConstDecl) itemNode()      {}
func (*StaticDecl) itemNode()     {}
func (*TypeDecl) itemNode()       {}
func (*ExternDecl) itemNode()     {}
func (*UnionDecl) itemNode()      {}
func (*TraitTypeItem) itemNode()  {}
func (*TraitConstItem) itemNode() {}
func (*ImplTypeItem) itemNode()   {}
