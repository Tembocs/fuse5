package ast

import "github.com/Tembocs/fuse5/compiler/lex"

// LiteralKind identifies which lexical literal category a LiteralExpr holds.
type LiteralKind int

const (
	LitInt LiteralKind = iota
	LitFloat
	LitString
	LitRawString
	LitCString
	LitChar
	LitBool
	LitNone // the `None` keyword used as literal-form in patterns/expressions
)

// LiteralExpr is any lexical literal — integer, float, string, char,
// boolean, or the keyword `None`. Text is the original source spelling; the
// checker normalizes at the HIR/MIR boundary per reference §1.10.
type LiteralExpr struct {
	NodeBase
	Kind  LiteralKind
	Text  string
	Value bool // filled only when Kind == LitBool
}

// PathExpr is an identifier or qualified path like `a.b.c`. When TypeArgs is
// non-nil, the path carries turbofish-style generic arguments `a.b[T]`.
type PathExpr struct {
	NodeBase
	Segments []Ident
	TypeArgs []Type
}

// BinaryOp enumerates operators used by BinaryExpr. The set matches reference
// Appendix B; assignment variants live in AssignOp.
type BinaryOp int

const (
	BinAdd BinaryOp = iota
	BinSub
	BinMul
	BinDiv
	BinMod
	BinShl
	BinShr
	BinAnd    // bitwise &
	BinOr     // bitwise |
	BinXor    // bitwise ^
	BinLogAnd // &&
	BinLogOr  // ||
	BinEq
	BinNe
	BinLt
	BinLe
	BinGt
	BinGe
)

// BinaryExpr is any infix operator expression except assignment and cast.
type BinaryExpr struct {
	NodeBase
	Op       BinaryOp
	OpSpan   lex.Span
	Lhs, Rhs Expr
}

// AssignOp enumerates assignment forms.
type AssignOp int

const (
	AssignEq AssignOp = iota
	AssignAdd
	AssignSub
	AssignMul
	AssignDiv
	AssignMod
	AssignAnd
	AssignOr
	AssignXor
	AssignShl
	AssignShr
)

// AssignExpr is `lhs <op>= rhs` (reference grammar `assignment_expr`).
// Expressed as an expression rather than a statement to match the grammar
// (assignments have expression position in some nesting contexts).
type AssignExpr struct {
	NodeBase
	Op       AssignOp
	OpSpan   lex.Span
	Lhs, Rhs Expr
}

// UnaryOp enumerates prefix-unary operators from reference grammar
// `unary_op`: `!`, `-`, `*` (deref), `&` (ref).
type UnaryOp int

const (
	UnNot UnaryOp = iota
	UnNeg
	UnDeref
	UnAddr
)

// UnaryExpr is a prefix-unary operator application.
type UnaryExpr struct {
	NodeBase
	Op     UnaryOp
	OpSpan lex.Span
	Operand Expr
}

// CastExpr is `EXPR as TYPE` (reference grammar `cast_expr`).
type CastExpr struct {
	NodeBase
	Expr Expr
	Type Type
}

// CallExpr is `callee(args...)`.
type CallExpr struct {
	NodeBase
	Callee Expr
	Args   []Expr
}

// FieldExpr is `receiver.field`.
type FieldExpr struct {
	NodeBase
	Receiver Expr
	Name     Ident
}

// OptFieldExpr is `receiver?.field` (reference §1.10, §5.6). A dedicated node
// avoids conflating it with plain field access — later waves lower it to an
// Option-aware branch.
type OptFieldExpr struct {
	NodeBase
	Receiver Expr
	Name     Ident
}

// TryExpr is `receiver?` — the error-propagation operator (reference §5.7).
type TryExpr struct {
	NodeBase
	Receiver Expr
}

// IndexExpr is `expr[idx]` for single-index access.
type IndexExpr struct {
	NodeBase
	Receiver Expr
	Index    Expr
}

// IndexRangeExpr is slice-range indexing `expr[a..b]`, `expr[a..]`, `expr[..b]`,
// or `expr[a..=b]` (reference grammar `postfix_op` range variants).
type IndexRangeExpr struct {
	NodeBase
	Receiver Expr
	Low      Expr // nil means open-low
	High     Expr // nil means open-high
	Inclusive bool // true for `..=`
}

// BlockExpr is `{ stmts... trailing? }`. Trailing is the optional final
// expression (reference grammar `block_expr`).
type BlockExpr struct {
	NodeBase
	Stmts    []Stmt
	Trailing Expr
}

// IfExpr is `if cond { then } else { else }` with optional else branch.
type IfExpr struct {
	NodeBase
	Cond Expr
	Then *BlockExpr
	Else Expr // *BlockExpr or *IfExpr or nil
}

// MatchExpr is `match scrutinee { arms... }`.
type MatchExpr struct {
	NodeBase
	Scrutinee Expr
	Arms      []*MatchArm
}

// MatchArm is one `pattern [if guard] { body }` arm.
type MatchArm struct {
	NodeBase
	Pattern Pat
	Guard   Expr // nil when no `if` guard
	Body    *BlockExpr
}

// LoopExpr is `loop { body }`.
type LoopExpr struct {
	NodeBase
	Body *BlockExpr
}

// WhileExpr is `while cond { body }`.
type WhileExpr struct {
	NodeBase
	Cond Expr
	Body *BlockExpr
}

// ForExpr is `for pat in iter { body }`.
type ForExpr struct {
	NodeBase
	Pattern Pat
	Iter    Expr
	Body    *BlockExpr
}

// TupleExpr is `(a, b, c)` with length >= 2, or `(a,)` (one-tuple).
type TupleExpr struct {
	NodeBase
	Elements []Expr
}

// StructLitField is one `name: expr` or shorthand `name` in a struct literal.
type StructLitField struct {
	NodeBase
	Name      Ident
	Value     Expr // nil for shorthand (resolver resolves `{ x }` = `{ x: x }`)
	Shorthand bool
}

// StructLitExpr is `Path { fields [..base] }` (reference grammar `struct_lit`).
type StructLitExpr struct {
	NodeBase
	Path   *PathExpr
	Fields []*StructLitField
	Base   Expr // `..base` expression (nil when absent)
}

// ClosureExpr is `fn(params) [-> T] { body }` — anonymous function literal.
// W12 will split FnDecl (items) from ClosureExpr (expressions) more
// aggressively; at W02 they share parameter/return shape.
type ClosureExpr struct {
	NodeBase
	IsMove bool // reserved; the `move` prefix is folded here when W12 lands it
	Params []*Param
	Return Type
	Body   *BlockExpr
}

// SpawnExpr is `spawn closure` (reference grammar `spawn_expr`).
type SpawnExpr struct {
	NodeBase
	Inner *ClosureExpr
}

// UnsafeExpr is `unsafe { ... }` (reference grammar `unsafe_block`).
type UnsafeExpr struct {
	NodeBase
	Body *BlockExpr
}

// ParenExpr is `(expr)` — parentheses preserved for span/pretty-print
// fidelity. Downstream passes may strip these.
type ParenExpr struct {
	NodeBase
	Inner Expr
}

// exprNode markers.
func (*LiteralExpr) exprNode()    {}
func (*PathExpr) exprNode()       {}
func (*BinaryExpr) exprNode()     {}
func (*AssignExpr) exprNode()     {}
func (*UnaryExpr) exprNode()      {}
func (*CastExpr) exprNode()       {}
func (*CallExpr) exprNode()       {}
func (*FieldExpr) exprNode()      {}
func (*OptFieldExpr) exprNode()   {}
func (*TryExpr) exprNode()        {}
func (*IndexExpr) exprNode()      {}
func (*IndexRangeExpr) exprNode() {}
func (*BlockExpr) exprNode()      {}
func (*IfExpr) exprNode()         {}
func (*MatchExpr) exprNode()      {}
func (*LoopExpr) exprNode()       {}
func (*WhileExpr) exprNode()      {}
func (*ForExpr) exprNode()        {}
func (*TupleExpr) exprNode()      {}
func (*StructLitExpr) exprNode()  {}
func (*ClosureExpr) exprNode()    {}
func (*SpawnExpr) exprNode()      {}
func (*UnsafeExpr) exprNode()     {}
func (*ParenExpr) exprNode()      {}
