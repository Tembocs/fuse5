package resolve

import "github.com/Tembocs/fuse5/compiler/ast"

// walkStmtExprs is a small dispatcher that visits the expressions,
// types, and patterns a statement contains. It isolates the statement
// shape knowledge from the main walker in paths.go so the latter does
// not need to import every statement type name twice.
func walkStmtExprs(s ast.Stmt, onExpr func(ast.Expr), onPat func(ast.Pat), onType func(ast.Type)) {
	switch x := s.(type) {
	case *ast.LetStmt:
		if x.Pattern != nil {
			onPat(x.Pattern)
		}
		if x.Type != nil {
			onType(x.Type)
		}
		if x.Value != nil {
			onExpr(x.Value)
		}
	case *ast.VarStmt:
		if x.Type != nil {
			onType(x.Type)
		}
		if x.Value != nil {
			onExpr(x.Value)
		}
	case *ast.ReturnStmt:
		if x.Value != nil {
			onExpr(x.Value)
		}
	case *ast.BreakStmt:
		if x.Value != nil {
			onExpr(x.Value)
		}
	case *ast.ContinueStmt:
		// no sub-expression
	case *ast.ExprStmt:
		if x.Expr != nil {
			onExpr(x.Expr)
		}
	case *ast.ItemStmt:
		// Nested items inside a block do not participate in module-level
		// name indexing at W03; their bodies are walked by the normal
		// item visitor at the enclosing level. For now we skip them —
		// block-local item resolution is handled when HIR lowering
		// introduces its own scope in W04.
	}
}
