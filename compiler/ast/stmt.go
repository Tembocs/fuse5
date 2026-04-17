package ast

// LetStmt is `let PATTERN [: TYPE] [= EXPR];`.
type LetStmt struct {
	NodeBase
	Pattern Pat
	Type    Type // nil when omitted
	Value   Expr // nil when omitted (pattern-only binding)
}

// VarStmt is `var NAME [: TYPE] = EXPR;`.
type VarStmt struct {
	NodeBase
	Name  Ident
	Type  Type // nil when omitted
	Value Expr
}

// ReturnStmt is `return [EXPR];`.
type ReturnStmt struct {
	NodeBase
	Value Expr // nil for bare `return;`
}

// BreakStmt is `break [EXPR];`.
type BreakStmt struct {
	NodeBase
	Value Expr // nil for bare `break;`
}

// ContinueStmt is `continue;`.
type ContinueStmt struct {
	NodeBase
}

// ExprStmt is any expression used in statement position: `EXPR;`.
type ExprStmt struct {
	NodeBase
	Expr Expr
}

// ItemStmt wraps an item that appears inside a block (reference grammar
// `stmt = ... | item_decl`). Pulled into its own node so a block's statement
// list has a single element type.
type ItemStmt struct {
	NodeBase
	Item Item
}

// stmtNode markers.
func (*LetStmt) stmtNode()      {}
func (*VarStmt) stmtNode()      {}
func (*ReturnStmt) stmtNode()   {}
func (*BreakStmt) stmtNode()    {}
func (*ContinueStmt) stmtNode() {}
func (*ExprStmt) stmtNode()     {}
func (*ItemStmt) stmtNode()     {}
