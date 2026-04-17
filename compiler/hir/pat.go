package hir

import "github.com/Tembocs/fuse5/compiler/typetable"

// Patterns are structured HIR nodes. Storing patterns as text at HIR is
// forbidden (reference §7.9; L007). Every variant below carries a
// TypeId so later passes can match exhaustively and lower deterministically.
//
// The variant set is fixed:
//
//   - LiteralPat: `42`, `"hello"`, `true`
//   - BindPat: a plain identifier that binds the scrutinee
//   - ConstructorPat: `Some(x)`, `Rect { w, h }`, `None`
//   - WildcardPat: `_`
//   - OrPat: `p1 | p2 | p3`
//   - RangePat: `lo..hi`, `lo..=hi`
//   - AtBindPat: `name @ pat`
//
// The bridge rejects any AST pattern it cannot translate into one of
// these, rather than falling back to a text placeholder (L007 defense).

// LiteralPat matches a specific literal value.
type LiteralPat struct {
	TypedBase
	Kind  LitKind
	Text  string // original source spelling; the checker normalizes at HIR→MIR
	Bool  bool   // valid only when Kind == LitBool
}

func (*LiteralPat) patNode() {}

// LitKind mirrors ast.LiteralKind but is owned by HIR so passes do not
// need to cross the AST/HIR boundary to dispatch on literals.
type LitKind int

const (
	LitInt LitKind = iota
	LitFloat
	LitString
	LitRawString
	LitCString
	LitChar
	LitBool
	LitNone
)

// BindPat binds the scrutinee to a name.
type BindPat struct {
	TypedBase
	Name string
}

func (*BindPat) patNode() {}

// ConstructorPat matches a variant or struct constructor. Path is
// the resolved nominal or variant name list kept as dotted segments
// so downstream passes don't re-parse identifiers.
type ConstructorPat struct {
	TypedBase
	// ConstructorType is the TypeId of the type being constructed
	// (for a variant pattern it is the parent enum's TypeId; for a
	// struct pattern it is the struct's TypeId).
	ConstructorType typetable.TypeId
	// VariantName is non-empty only for enum variant patterns and
	// names the specific variant matched.
	VariantName string
	// Path is the dotted spelling as it appeared in source, kept so
	// diagnostics can point back at the written name.
	Path []string
	// Tuple holds positional sub-patterns (nil for struct-shaped).
	Tuple []Pat
	// Fields holds struct-shaped sub-patterns. Mutually exclusive
	// with Tuple.
	Fields []*FieldPat
	// HasRest is true when the struct pattern ended with `..`.
	HasRest bool
}

func (*ConstructorPat) patNode() {}

// FieldPat is one field in a struct-shaped ConstructorPat.
type FieldPat struct {
	Base
	Name    string
	Pattern Pat // never nil — shorthand forms are expanded by the bridge into a BindPat
}

// FieldPat satisfies Node via Base; it is not a Pat in its own right
// (it only appears inside ConstructorPat.Fields).

// WildcardPat is `_`.
type WildcardPat struct {
	TypedBase
}

func (*WildcardPat) patNode() {}

// OrPat matches any alternative.
type OrPat struct {
	TypedBase
	Alts []Pat
}

func (*OrPat) patNode() {}

// RangePat matches a numeric range. Lo and Hi are expressions (almost
// always LiteralExpr or a resolved constant); Inclusive distinguishes
// `lo..hi` from `lo..=hi`.
type RangePat struct {
	TypedBase
	Lo        Expr
	Hi        Expr
	Inclusive bool
}

func (*RangePat) patNode() {}

// AtBindPat is `name @ pat` — bind the scrutinee to `name` while also
// matching its shape against `pat`.
type AtBindPat struct {
	TypedBase
	Name    string
	Pattern Pat
}

func (*AtBindPat) patNode() {}
