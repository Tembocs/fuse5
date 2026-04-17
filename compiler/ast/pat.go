package ast

// LiteralPat is a pattern that matches one literal value — `42`, `"hello"`.
type LiteralPat struct {
	NodeBase
	Value *LiteralExpr
}

// WildcardPat is the `_` pattern.
type WildcardPat struct {
	NodeBase
}

// BindPat binds the scrutinee to an identifier — plain `x` in a pattern.
type BindPat struct {
	NodeBase
	Name Ident
}

// CtorPat is a constructor pattern — `None`, `Some(x)`, `Rect { w, h }`.
type CtorPat struct {
	NodeBase
	Path    []Ident
	Tuple   []Pat            // nil if not tuple-shaped
	Struct  []*FieldPat      // nil if not struct-shaped
	HasRest bool             // `..` at the end of a struct pattern
}

// FieldPat is one field in a struct pattern — `name` shorthand or `name: pat`.
type FieldPat struct {
	NodeBase
	Name    Ident
	Pattern Pat // nil when shorthand
}

// TuplePat is `(p1, p2, ...)`.
type TuplePat struct {
	NodeBase
	Elements []Pat
}

// OrPat is `p1 | p2 | p3`.
type OrPat struct {
	NodeBase
	Alts []Pat
}

// RangePat is `lo..hi` or `lo..=hi`.
type RangePat struct {
	NodeBase
	Lo        Expr
	Hi        Expr
	Inclusive bool
}

// AtPat is `name @ pat` — bind the whole match to `name` while also matching
// its shape.
type AtPat struct {
	NodeBase
	Name    Ident
	Pattern Pat
}

// patNode markers.
func (*LiteralPat) patNode()  {}
func (*WildcardPat) patNode() {}
func (*BindPat) patNode()     {}
func (*CtorPat) patNode()     {}
func (*TuplePat) patNode()    {}
func (*OrPat) patNode()       {}
func (*RangePat) patNode()    {}
func (*AtPat) patNode()       {}
