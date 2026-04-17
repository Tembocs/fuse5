package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// RunNoUnknownCheck walks every Typed node in prog and returns a
// slice of diagnostic strings for any node still carrying
// KindInfer after Check. The function is the enforcement point
// for the L013 "no Unknown after check" invariant (W06 exit
// criterion #1).
//
// A non-empty result is a compiler bug, not a user error: the
// checker should have either assigned a concrete TypeId or emitted
// a diagnostic that blocks compilation. The test
// `TestNoUnknownAfterCheck` consumes this output.
func RunNoUnknownCheck(prog *hir.Program) []string {
	inferID := prog.Types.Infer()
	var out []string
	for _, modPath := range prog.Order {
		m := prog.Modules[modPath]
		for _, it := range m.Items {
			out = append(out, walkItem(it, inferID)...)
		}
	}
	return out
}

func walkItem(it hir.Item, infer typetable.TypeId) []string {
	var out []string
	switch x := it.(type) {
	case *hir.FnDecl:
		for _, p := range x.Params {
			if p.TypeOf() == infer {
				out = append(out, fmt.Sprintf("param %q in fn %q still has KindInfer", p.Name, x.Name))
			}
		}
		if x.Body != nil {
			out = append(out, walkBlock(x.Body, infer, string(x.ID))...)
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			out = append(out, walkItem(sub, infer)...)
		}
	case *hir.TraitDecl:
		for _, sub := range x.Items {
			out = append(out, walkItem(sub, infer)...)
		}
	case *hir.ConstDecl:
		if x.Value != nil {
			out = append(out, walkExpr(x.Value, infer, string(x.ID))...)
		}
	case *hir.StaticDecl:
		if x.Value != nil {
			out = append(out, walkExpr(x.Value, infer, string(x.ID))...)
		}
	}
	return out
}

func walkBlock(b *hir.Block, infer typetable.TypeId, anchor string) []string {
	var out []string
	if b == nil {
		return out
	}
	if b.TypeOf() == infer {
		out = append(out, fmt.Sprintf("block at %s still has KindInfer", anchor))
	}
	for _, s := range b.Stmts {
		out = append(out, walkStmt(s, infer, anchor)...)
	}
	if b.Trailing != nil {
		out = append(out, walkExpr(b.Trailing, infer, anchor)...)
	}
	return out
}

func walkStmt(s hir.Stmt, infer typetable.TypeId, anchor string) []string {
	var out []string
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.DeclaredType == infer {
			out = append(out, fmt.Sprintf("let at %s still has KindInfer declared type", anchor))
		}
		if x.Value != nil {
			out = append(out, walkExpr(x.Value, infer, anchor)...)
		}
	case *hir.VarStmt:
		if x.DeclaredType == infer {
			out = append(out, fmt.Sprintf("var at %s still has KindInfer declared type", anchor))
		}
		if x.Value != nil {
			out = append(out, walkExpr(x.Value, infer, anchor)...)
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			out = append(out, walkExpr(x.Value, infer, anchor)...)
		}
	case *hir.ExprStmt:
		if x.Expr != nil {
			out = append(out, walkExpr(x.Expr, infer, anchor)...)
		}
	}
	return out
}

func walkExpr(e hir.Expr, infer typetable.TypeId, anchor string) []string {
	var out []string
	if e == nil {
		return out
	}
	if e.TypeOf() == infer {
		out = append(out, fmt.Sprintf("%T at %s still has KindInfer", e, anchor))
	}
	switch x := e.(type) {
	case *hir.BinaryExpr:
		out = append(out, walkExpr(x.Lhs, infer, anchor)...)
		out = append(out, walkExpr(x.Rhs, infer, anchor)...)
	case *hir.UnaryExpr:
		out = append(out, walkExpr(x.Operand, infer, anchor)...)
	case *hir.CallExpr:
		out = append(out, walkExpr(x.Callee, infer, anchor)...)
		for _, a := range x.Args {
			out = append(out, walkExpr(a, infer, anchor)...)
		}
	case *hir.Block:
		out = append(out, walkBlock(x, infer, anchor)...)
	case *hir.IfExpr:
		out = append(out, walkExpr(x.Cond, infer, anchor)...)
		out = append(out, walkBlock(x.Then, infer, anchor)...)
		if x.Else != nil {
			out = append(out, walkExpr(x.Else, infer, anchor)...)
		}
	case *hir.TupleExpr:
		for _, el := range x.Elements {
			out = append(out, walkExpr(el, infer, anchor)...)
		}
	case *hir.StructLitExpr:
		for _, f := range x.Fields {
			if f.Value != nil {
				out = append(out, walkExpr(f.Value, infer, anchor)...)
			}
		}
	}
	return out
}
