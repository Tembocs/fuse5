package consteval

import (
	"strconv"

	"github.com/Tembocs/fuse5/compiler/hir"
)

// Substitute walks the program and replaces every PathExpr that
// names an evaluated const or static with an equivalent LiteralExpr
// carrying the evaluated value. Downstream passes (lower, codegen)
// therefore never have to resolve const references themselves — the
// spine contract remains a plain "expression tree of literals and
// arithmetic" after W14 runs.
//
// Only VKInt and VKBool values are substituted: the spine consumes
// integer and boolean literals today. Struct / tuple / array
// constants remain as path references so later waves can specialize
// their lowering with richer support.
//
// Substitute also strips pure `const fn` declarations from the HIR
// after substitution: their bodies have no runtime call sites once
// every const initializer has been reduced to a literal, and the
// spine lowerer rejects the if / loop / match shapes common in
// const fn bodies. Removing them here keeps the driver pipeline
// compatible with existing W05–W13 lowering rules.
//
// Substitute is in-place: it mutates the nodes it touches but never
// replaces whole statements. The replacement LiteralExpr inherits
// the original PathExpr's NodeID and Span so fingerprinting and
// diagnostics remain anchored to the source.
func Substitute(prog *hir.Program, res *Result) {
	if prog == nil || res == nil {
		return
	}
	s := &substituter{prog: prog, res: res}
	for _, modPath := range prog.Order {
		mod := prog.Modules[modPath]
		for _, it := range mod.Items {
			s.walkItem(it)
		}
	}
	stripConstFns(prog)
}

// stripConstFns removes every `const fn` item from each module.
// Const fns only participate in compile-time evaluation; after
// substitution their bodies have no runtime references and keeping
// them would force the lowerer to accept const-fn-specific shapes
// (early returns, recursion, if-else chains) that the spine does
// not otherwise support.
func stripConstFns(prog *hir.Program) {
	for _, modPath := range prog.Order {
		mod := prog.Modules[modPath]
		kept := mod.Items[:0]
		for _, it := range mod.Items {
			if fn, ok := it.(*hir.FnDecl); ok && fn.IsConst {
				continue
			}
			kept = append(kept, it)
		}
		mod.Items = kept
	}
}

type substituter struct {
	prog *hir.Program
	res  *Result
}

func (s *substituter) walkItem(it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		if x.Body != nil {
			s.walkBlock(x.Body)
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			s.walkItem(sub)
		}
	case *hir.ConstDecl:
		if x.Value != nil {
			x.Value = s.replace(x.Value)
		}
	case *hir.StaticDecl:
		if x.Value != nil {
			x.Value = s.replace(x.Value)
		}
	}
}

func (s *substituter) walkBlock(b *hir.Block) {
	if b == nil {
		return
	}
	for _, st := range b.Stmts {
		s.walkStmt(st)
	}
	if b.Trailing != nil {
		b.Trailing = s.replace(b.Trailing)
	}
}

func (s *substituter) walkStmt(st hir.Stmt) {
	switch x := st.(type) {
	case *hir.LetStmt:
		if x.Value != nil {
			x.Value = s.replace(x.Value)
		}
	case *hir.VarStmt:
		if x.Value != nil {
			x.Value = s.replace(x.Value)
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			x.Value = s.replace(x.Value)
		}
	case *hir.BreakStmt:
		if x.Value != nil {
			x.Value = s.replace(x.Value)
		}
	case *hir.ExprStmt:
		x.Expr = s.replace(x.Expr)
	}
}

// replace returns the substituted form of e, recursively descending
// into sub-expressions. The returned value replaces e in its parent.
func (s *substituter) replace(e hir.Expr) hir.Expr {
	switch x := e.(type) {
	case *hir.PathExpr:
		if x.Symbol == 0 {
			return e
		}
		v, ok := s.res.ConstValues[x.Symbol]
		if !ok {
			v, ok = s.res.StaticValues[x.Symbol]
		}
		if !ok {
			return e
		}
		lit := valueToLiteral(v)
		if lit == nil {
			return e
		}
		// Preserve the original PathExpr's NodeID, Span, and Type
		// on the synthesized literal so fingerprints line up.
		lit.Base = x.Base
		lit.Type = x.Type
		return lit
	case *hir.BinaryExpr:
		x.Lhs = s.replace(x.Lhs)
		x.Rhs = s.replace(x.Rhs)
	case *hir.UnaryExpr:
		x.Operand = s.replace(x.Operand)
	case *hir.CastExpr:
		x.Expr = s.replace(x.Expr)
	case *hir.CallExpr:
		x.Callee = s.replace(x.Callee)
		for i, a := range x.Args {
			x.Args[i] = s.replace(a)
		}
	case *hir.IfExpr:
		x.Cond = s.replace(x.Cond)
		s.walkBlock(x.Then)
		if x.Else != nil {
			x.Else = s.replace(x.Else)
		}
	case *hir.Block:
		s.walkBlock(x)
	case *hir.LoopExpr:
		s.walkBlock(x.Body)
	case *hir.WhileExpr:
		x.Cond = s.replace(x.Cond)
		s.walkBlock(x.Body)
	case *hir.ForExpr:
		x.Iter = s.replace(x.Iter)
		s.walkBlock(x.Body)
	case *hir.MatchExpr:
		x.Scrutinee = s.replace(x.Scrutinee)
		for _, arm := range x.Arms {
			if arm.Guard != nil {
				arm.Guard = s.replace(arm.Guard)
			}
			s.walkBlock(arm.Body)
		}
	case *hir.TupleExpr:
		for i, el := range x.Elements {
			x.Elements[i] = s.replace(el)
		}
	case *hir.StructLitExpr:
		for _, f := range x.Fields {
			f.Value = s.replace(f.Value)
		}
		if x.Base != nil {
			x.Base = s.replace(x.Base)
		}
	case *hir.FieldExpr:
		x.Receiver = s.replace(x.Receiver)
	case *hir.OptFieldExpr:
		x.Receiver = s.replace(x.Receiver)
	case *hir.TryExpr:
		x.Receiver = s.replace(x.Receiver)
	case *hir.IndexExpr:
		x.Receiver = s.replace(x.Receiver)
		x.Index = s.replace(x.Index)
	case *hir.IndexRangeExpr:
		x.Receiver = s.replace(x.Receiver)
		if x.Low != nil {
			x.Low = s.replace(x.Low)
		}
		if x.High != nil {
			x.High = s.replace(x.High)
		}
	case *hir.ReferenceExpr:
		x.Inner = s.replace(x.Inner)
	case *hir.UnsafeExpr:
		s.walkBlock(x.Body)
	case *hir.AssignExpr:
		x.Lhs = s.replace(x.Lhs)
		x.Rhs = s.replace(x.Rhs)
	}
	return e
}

// valueToLiteral turns an integer or boolean Value into a
// LiteralExpr. Returns nil for kinds that cannot be substituted
// inline (tuples, structs, arrays, unit).
func valueToLiteral(v Value) *hir.LiteralExpr {
	switch v.Kind {
	case VKInt:
		return &hir.LiteralExpr{
			TypedBase: hir.TypedBase{Type: v.Type},
			Kind:      hir.LitInt,
			Text:      strconv.FormatUint(v.Int, 10),
		}
	case VKBool:
		return &hir.LiteralExpr{
			TypedBase: hir.TypedBase{Type: v.Type},
			Kind:      hir.LitBool,
			Bool:      v.Bool,
		}
	case VKChar:
		return &hir.LiteralExpr{
			TypedBase: hir.TypedBase{Type: v.Type},
			Kind:      hir.LitInt, // chars narrow to int literals for the spine
			Text:      strconv.FormatUint(v.Int, 10),
		}
	}
	return nil
}
