package hir

import "github.com/Tembocs/fuse5/compiler/typetable"

// LiteralExpr is any lexical literal lowered to HIR. Text preserves
// the source spelling so diagnostics can quote the original; the
// HIR→MIR boundary normalizes content (reference §1.10).
type LiteralExpr struct {
	TypedBase
	Kind LitKind
	Text string
	Bool bool // valid only when Kind == LitBool
}

func (*LiteralExpr) exprNode() {}

// PathExpr is a resolved path reference. Unlike the AST form, HIR
// PathExpr carries the resolved target: Symbol is the resolve.SymbolID
// of what this path names, and Segments is preserved for diagnostics
// and stable identity. TypeArgs holds the explicit turbofish type
// arguments (e.g. `identity[I32]`); W08 monomorphization consumes
// them to drive specialization.
type PathExpr struct {
	TypedBase
	Symbol   int // resolve.SymbolID (kept as int to avoid import cycle)
	Segments []string
	TypeArgs []typetable.TypeId
}

func (*PathExpr) exprNode() {}

// BinaryExpr is any infix operator except assignment and cast.
type BinaryExpr struct {
	TypedBase
	Op       BinaryOp
	Lhs, Rhs Expr
}

func (*BinaryExpr) exprNode() {}

// BinaryOp mirrors ast.BinaryOp; HIR carries its own enum so passes
// do not cross the AST boundary (Rule 3.1).
type BinaryOp int

const (
	BinAdd BinaryOp = iota
	BinSub
	BinMul
	BinDiv
	BinMod
	BinShl
	BinShr
	BinAnd
	BinOr
	BinXor
	BinLogAnd
	BinLogOr
	BinEq
	BinNe
	BinLt
	BinLe
	BinGt
	BinGe
)

// AssignExpr is `lhs <op>= rhs`.
type AssignExpr struct {
	TypedBase
	Op       AssignOp
	Lhs, Rhs Expr
}

func (*AssignExpr) exprNode() {}

// AssignOp mirrors ast.AssignOp.
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

// UnaryExpr is a prefix-unary operator application.
type UnaryExpr struct {
	TypedBase
	Op      UnaryOp
	Operand Expr
}

func (*UnaryExpr) exprNode() {}

// UnaryOp mirrors ast.UnaryOp.
type UnaryOp int

const (
	UnNot UnaryOp = iota
	UnNeg
	UnDeref
	UnAddr
)

// CastExpr is `expr as T`.
type CastExpr struct {
	TypedBase
	Expr Expr
	// Target is the TypeId of the cast destination, set by the
	// bridge from the annotated AST type.
}

func (*CastExpr) exprNode() {}

// CallExpr is `callee(args...)`.
type CallExpr struct {
	TypedBase
	Callee Expr
	Args   []Expr
}

func (*CallExpr) exprNode() {}

// FieldExpr is `receiver.name`.
type FieldExpr struct {
	TypedBase
	Receiver Expr
	Name     string
}

func (*FieldExpr) exprNode() {}

// OptFieldExpr is `receiver?.name` (reference §5.6). A dedicated node
// avoids conflation with plain field access; W11 lowers it to an
// Option-aware branch.
type OptFieldExpr struct {
	TypedBase
	Receiver Expr
	Name     string
}

func (*OptFieldExpr) exprNode() {}

// TryExpr is `receiver?` — error propagation (reference §5.7, §16).
type TryExpr struct {
	TypedBase
	Receiver Expr
}

func (*TryExpr) exprNode() {}

// IndexExpr is `receiver[idx]`.
type IndexExpr struct {
	TypedBase
	Receiver Expr
	Index    Expr
}

func (*IndexExpr) exprNode() {}

// IndexRangeExpr is `receiver[a..b]` or similar.
type IndexRangeExpr struct {
	TypedBase
	Receiver  Expr
	Low       Expr // nil for open-low
	High      Expr // nil for open-high
	Inclusive bool
}

func (*IndexRangeExpr) exprNode() {}

// Block is the HIR counterpart of AST BlockExpr. Stmts is the
// statement list; Trailing is the optional final expression whose
// type becomes the block's type.
type Block struct {
	TypedBase
	Stmts    []Stmt
	Trailing Expr
}

func (*Block) exprNode() {}

// IfExpr is `if cond { then } else { else }`.
type IfExpr struct {
	TypedBase
	Cond Expr
	Then *Block
	// Else is *Block, *IfExpr, or nil.
	Else Expr
}

func (*IfExpr) exprNode() {}

// MatchExpr is `match scrutinee { arms }`. Each arm carries a
// structured Pat (L007).
type MatchExpr struct {
	TypedBase
	Scrutinee Expr
	Arms      []*MatchArm
}

func (*MatchExpr) exprNode() {}

// MatchArm is one `pat [if guard] => body` arm.
type MatchArm struct {
	Base
	Pattern Pat
	Guard   Expr // nil when no guard
	Body    *Block
}

// MatchArm satisfies Node via Base; it is not a Pat or Expr on its own.

// LoopExpr is `loop { body }`.
type LoopExpr struct {
	TypedBase
	Body *Block
}

func (*LoopExpr) exprNode() {}

// WhileExpr is `while cond { body }`.
type WhileExpr struct {
	TypedBase
	Cond Expr
	Body *Block
}

func (*WhileExpr) exprNode() {}

// ForExpr is `for pat in iter { body }`.
type ForExpr struct {
	TypedBase
	Pattern Pat
	Iter    Expr
	Body    *Block
}

func (*ForExpr) exprNode() {}

// TupleExpr is `(a, b, ...)`.
type TupleExpr struct {
	TypedBase
	Elements []Expr
}

func (*TupleExpr) exprNode() {}

// StructLitField is one `name: expr` or shorthand `name` inside a
// struct literal. The bridge expands shorthand into `name: name`.
type StructLitField struct {
	Base
	Name  string
	Value Expr
}

// StructLitExpr is `Path { fields [..base] }`.
type StructLitExpr struct {
	TypedBase
	StructType typetable.TypeId
	Fields     []*StructLitField
	Base       Expr // `..base` expression; nil when absent
}

func (*StructLitExpr) exprNode() {}

// ClosureExpr is `fn(params) [-> T] { body }`.
type ClosureExpr struct {
	TypedBase
	IsMove bool
	Params []*Param
	Return typetable.TypeId
	Body   *Block
}

func (*ClosureExpr) exprNode() {}

// SpawnExpr is `spawn closure` (reference §17).
type SpawnExpr struct {
	TypedBase
	Closure *ClosureExpr
}

func (*SpawnExpr) exprNode() {}

// UnsafeExpr is `unsafe { body }`.
type UnsafeExpr struct {
	TypedBase
	Body *Block
}

func (*UnsafeExpr) exprNode() {}

// ReferenceExpr adapts `&expr` / `&mut expr` at HIR level. Unlike the
// AST's UnaryExpr(UnAddr), HIR distinguishes the mutability at
// construction so later passes do not re-parse the operator position.
type ReferenceExpr struct {
	TypedBase
	Mutable bool
	Inner   Expr
}

func (*ReferenceExpr) exprNode() {}
