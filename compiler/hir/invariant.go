package hir

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/typetable"
)

// InvariantViolation records one broken HIR invariant discovered by
// the walker. Violations are collected and returned en masse so a
// single pass reports every problem, not just the first.
type InvariantViolation struct {
	NodeID  NodeID
	NodeKind string
	Reason  string
}

func (v InvariantViolation) String() string {
	return fmt.Sprintf("[%s] %s: %s", v.NodeKind, v.NodeID, v.Reason)
}

// RunInvariantWalker walks every node in p and returns the violations
// it finds. An empty return value means all invariants hold.
//
// Enforced invariants (the W04 "invariant walkers" contract):
//
//   - Every Node has a non-empty NodeID.
//   - Every Typed node has a TypeId that is either a concrete
//     primitive/structural/nominal TypeId or the explicit KindInfer
//     TypeId. NoType is a violation (L013 defense).
//   - Every structural child that the grammar makes mandatory
//     (IfExpr.Then, Match arms' pattern and body, RangePat bounds) is
//     non-nil.
//   - Every ConstructorPat carries a non-NoType ConstructorType.
//
// The walker itself is also the "debug mode" invariant check: W18's
// incremental driver will call it between pipeline stages. For now
// callers (tests and the bridge's self-test) invoke it directly.
func RunInvariantWalker(p *Program) []InvariantViolation {
	w := &invariantWalker{}
	for _, modPath := range p.Order {
		m := p.Modules[modPath]
		w.checkModule(m)
	}
	return w.out
}

type invariantWalker struct {
	out []InvariantViolation
}

func (w *invariantWalker) record(id NodeID, kind, reason string) {
	w.out = append(w.out, InvariantViolation{NodeID: id, NodeKind: kind, Reason: reason})
}

func (w *invariantWalker) checkNodeID(n Node, kind string) {
	if n.NodeHirID() == EmptyNodeID {
		w.record(n.NodeHirID(), kind, "empty NodeID")
	}
}

func (w *invariantWalker) checkType(n Typed, kind string) {
	if n.TypeOf() == typetable.NoType {
		w.record(n.NodeHirID(), kind, "NoType; use KindInfer for pending inference")
	}
}

func (w *invariantWalker) checkModule(m *Module) {
	w.checkNodeID(m, "Module")
	for _, it := range m.Items {
		w.checkItem(it)
	}
}

func (w *invariantWalker) checkItem(it Item) {
	switch x := it.(type) {
	case *FnDecl:
		w.checkNodeID(x, "FnDecl")
		if x.TypeID == typetable.NoType {
			w.record(x.ID, "FnDecl", "missing TypeID")
		}
		for _, p := range x.Params {
			w.checkNodeID(p, "Param")
			w.checkType(p, "Param")
		}
		if x.Body != nil {
			w.checkBlock(x.Body)
		}
	case *StructDecl:
		w.checkNodeID(x, "StructDecl")
		if x.TypeID == typetable.NoType {
			w.record(x.ID, "StructDecl", "missing TypeID")
		}
		for _, f := range x.Fields {
			w.checkNodeID(f, "Field")
			w.checkType(f, "Field")
		}
	case *EnumDecl:
		w.checkNodeID(x, "EnumDecl")
		if x.TypeID == typetable.NoType {
			w.record(x.ID, "EnumDecl", "missing TypeID")
		}
	case *ConstDecl:
		w.checkNodeID(x, "ConstDecl")
		if x.Type == typetable.NoType {
			w.record(x.ID, "ConstDecl", "missing declared Type")
		}
		if x.Value != nil {
			w.checkExpr(x.Value)
		}
	case *StaticDecl:
		w.checkNodeID(x, "StaticDecl")
		if x.Type == typetable.NoType {
			w.record(x.ID, "StaticDecl", "missing declared Type")
		}
		if x.Value != nil {
			w.checkExpr(x.Value)
		}
	case *UnionDecl:
		w.checkNodeID(x, "UnionDecl")
		if x.TypeID == typetable.NoType {
			w.record(x.ID, "UnionDecl", "missing TypeID")
		}
	case *TraitDecl:
		w.checkNodeID(x, "TraitDecl")
		for _, sub := range x.Items {
			w.checkItem(sub)
		}
	case *ImplDecl:
		w.checkNodeID(x, "ImplDecl")
		for _, sub := range x.Items {
			w.checkItem(sub)
		}
	case *TypeAliasDecl:
		w.checkNodeID(x, "TypeAliasDecl")
	}
}

func (w *invariantWalker) checkBlock(b *Block) {
	if b == nil {
		return
	}
	w.checkNodeID(b, "Block")
	w.checkType(b, "Block")
	for _, s := range b.Stmts {
		w.checkStmt(s)
	}
	if b.Trailing != nil {
		w.checkExpr(b.Trailing)
	}
}

func (w *invariantWalker) checkStmt(s Stmt) {
	if s == nil {
		return
	}
	switch x := s.(type) {
	case *LetStmt:
		w.checkNodeID(x, "LetStmt")
		if x.DeclaredType == typetable.NoType {
			w.record(x.ID, "LetStmt", "DeclaredType is NoType (use Infer if pending)")
		}
		if x.Pattern == nil {
			w.record(x.ID, "LetStmt", "missing pattern")
		} else {
			w.checkPat(x.Pattern)
		}
		if x.Value != nil {
			w.checkExpr(x.Value)
		}
	case *VarStmt:
		w.checkNodeID(x, "VarStmt")
		if x.DeclaredType == typetable.NoType {
			w.record(x.ID, "VarStmt", "DeclaredType is NoType")
		}
		if x.Value != nil {
			w.checkExpr(x.Value)
		}
	case *ReturnStmt:
		w.checkNodeID(x, "ReturnStmt")
		if x.Value != nil {
			w.checkExpr(x.Value)
		}
	case *BreakStmt:
		w.checkNodeID(x, "BreakStmt")
		if x.Value != nil {
			w.checkExpr(x.Value)
		}
	case *ContinueStmt:
		w.checkNodeID(x, "ContinueStmt")
	case *ExprStmt:
		w.checkNodeID(x, "ExprStmt")
		if x.Expr != nil {
			w.checkExpr(x.Expr)
		}
	case *ItemStmt:
		w.checkNodeID(x, "ItemStmt")
		if x.Item != nil {
			w.checkItem(x.Item)
		}
	}
}

func (w *invariantWalker) checkExpr(e Expr) {
	if e == nil {
		return
	}
	w.checkNodeID(e, fmt.Sprintf("%T", e))
	w.checkType(e, fmt.Sprintf("%T", e))
	switch x := e.(type) {
	case *BinaryExpr:
		w.checkExpr(x.Lhs)
		w.checkExpr(x.Rhs)
	case *AssignExpr:
		w.checkExpr(x.Lhs)
		w.checkExpr(x.Rhs)
	case *UnaryExpr:
		w.checkExpr(x.Operand)
	case *CastExpr:
		w.checkExpr(x.Expr)
	case *CallExpr:
		w.checkExpr(x.Callee)
		for _, a := range x.Args {
			w.checkExpr(a)
		}
	case *FieldExpr:
		w.checkExpr(x.Receiver)
	case *OptFieldExpr:
		w.checkExpr(x.Receiver)
	case *TryExpr:
		w.checkExpr(x.Receiver)
	case *IndexExpr:
		w.checkExpr(x.Receiver)
		w.checkExpr(x.Index)
	case *IndexRangeExpr:
		w.checkExpr(x.Receiver)
		if x.Low != nil {
			w.checkExpr(x.Low)
		}
		if x.High != nil {
			w.checkExpr(x.High)
		}
	case *Block:
		w.checkBlock(x)
	case *IfExpr:
		w.checkExpr(x.Cond)
		if x.Then == nil {
			w.record(x.ID, "IfExpr", "Then block is nil")
		} else {
			w.checkBlock(x.Then)
		}
		if x.Else != nil {
			w.checkExpr(x.Else)
		}
	case *MatchExpr:
		w.checkExpr(x.Scrutinee)
		for _, arm := range x.Arms {
			if arm == nil {
				continue
			}
			if arm.Pattern == nil {
				w.record(arm.ID, "MatchArm", "missing Pattern")
			} else {
				w.checkPat(arm.Pattern)
			}
			if arm.Guard != nil {
				w.checkExpr(arm.Guard)
			}
			if arm.Body == nil {
				w.record(arm.ID, "MatchArm", "missing Body")
			} else {
				w.checkBlock(arm.Body)
			}
		}
	case *TupleExpr:
		for _, el := range x.Elements {
			w.checkExpr(el)
		}
	case *StructLitExpr:
		if x.StructType == typetable.NoType {
			w.record(x.ID, "StructLitExpr", "StructType is NoType")
		}
		for _, f := range x.Fields {
			if f.Value != nil {
				w.checkExpr(f.Value)
			}
		}
		if x.Base != nil {
			w.checkExpr(x.Base)
		}
	case *ClosureExpr:
		if x.Body != nil {
			w.checkBlock(x.Body)
		}
	case *SpawnExpr:
		if x.Closure != nil && x.Closure.Body != nil {
			w.checkBlock(x.Closure.Body)
		}
	case *UnsafeExpr:
		w.checkBlock(x.Body)
	case *LoopExpr:
		w.checkBlock(x.Body)
	case *WhileExpr:
		w.checkExpr(x.Cond)
		w.checkBlock(x.Body)
	case *ForExpr:
		w.checkExpr(x.Iter)
		w.checkBlock(x.Body)
	case *ReferenceExpr:
		w.checkExpr(x.Inner)
	}
}

func (w *invariantWalker) checkPat(p Pat) {
	if p == nil {
		return
	}
	w.checkNodeID(p, fmt.Sprintf("%T", p))
	w.checkType(p, fmt.Sprintf("%T", p))
	switch x := p.(type) {
	case *ConstructorPat:
		if x.ConstructorType == typetable.NoType {
			w.record(x.ID, "ConstructorPat", "ConstructorType is NoType")
		}
		for _, sp := range x.Tuple {
			w.checkPat(sp)
		}
		for _, f := range x.Fields {
			if f.Pattern != nil {
				w.checkPat(f.Pattern)
			}
		}
	case *OrPat:
		if len(x.Alts) < 2 {
			w.record(x.ID, "OrPat", "fewer than 2 alternatives")
		}
		for _, a := range x.Alts {
			w.checkPat(a)
		}
	case *RangePat:
		if x.Lo == nil || x.Hi == nil {
			w.record(x.ID, "RangePat", "missing bound")
		} else {
			w.checkExpr(x.Lo)
			w.checkExpr(x.Hi)
		}
	case *AtBindPat:
		if x.Pattern == nil {
			w.record(x.ID, "AtBindPat", "missing inner pattern")
		} else {
			w.checkPat(x.Pattern)
		}
	}
}
