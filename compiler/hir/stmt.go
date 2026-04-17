package hir

import "github.com/Tembocs/fuse5/compiler/typetable"

// LetStmt is `let pat [: T] [= expr];`. The pattern is always
// structured (L007). DeclaredType records the annotated TypeId or
// Infer when the source omitted an annotation; the bridge never
// records NoType here (L013 defense).
type LetStmt struct {
	Base
	Pattern      Pat
	DeclaredType typetable.TypeId
	Value        Expr // nil when the source omitted the initializer
}

func (*LetStmt) stmtNode() {}

// VarStmt is `var NAME [: T] = EXPR;` — mutable binding. VarStmt
// always has an initializer per the grammar; the bridge rejects
// malformed AST forms.
type VarStmt struct {
	Base
	Name         string
	DeclaredType typetable.TypeId
	Value        Expr
}

func (*VarStmt) stmtNode() {}

// ReturnStmt is `return [expr];`.
type ReturnStmt struct {
	Base
	Value Expr // nil for bare `return;`
}

func (*ReturnStmt) stmtNode() {}

// BreakStmt is `break [expr];`.
type BreakStmt struct {
	Base
	Value Expr // nil for bare `break;`
}

func (*BreakStmt) stmtNode() {}

// ContinueStmt is `continue;`.
type ContinueStmt struct {
	Base
}

func (*ContinueStmt) stmtNode() {}

// ExprStmt wraps an expression used in statement position.
type ExprStmt struct {
	Base
	Expr Expr
}

func (*ExprStmt) stmtNode() {}

// ItemStmt lifts an inner item declaration (`fn nested() {}` inside a
// block) into statement position. The bridge registers the nested
// item with the parent Module's Items in the same index order the
// source used, so identity remains stable.
type ItemStmt struct {
	Base
	Item Item
}

func (*ItemStmt) stmtNode() {}
