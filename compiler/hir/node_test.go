package hir

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestHirNodeSet enumerates every HIR concrete node and asserts that
// it implements the expected marker interface. Adding a new concrete
// type requires adding a row here (L007 defense: patterns are always
// structured).
func TestHirNodeSet(t *testing.T) {
	// Items.
	var _ Item = &Module{}
	var _ Item = &FnDecl{}
	var _ Item = &StructDecl{}
	var _ Item = &EnumDecl{}
	var _ Item = &TraitDecl{}
	var _ Item = &ImplDecl{}
	var _ Item = &ConstDecl{}
	var _ Item = &StaticDecl{}
	var _ Item = &TypeAliasDecl{}
	var _ Item = &UnionDecl{}

	// Expressions.
	var _ Expr = &LiteralExpr{}
	var _ Expr = &PathExpr{}
	var _ Expr = &BinaryExpr{}
	var _ Expr = &AssignExpr{}
	var _ Expr = &UnaryExpr{}
	var _ Expr = &CastExpr{}
	var _ Expr = &CallExpr{}
	var _ Expr = &FieldExpr{}
	var _ Expr = &OptFieldExpr{}
	var _ Expr = &TryExpr{}
	var _ Expr = &IndexExpr{}
	var _ Expr = &IndexRangeExpr{}
	var _ Expr = &Block{}
	var _ Expr = &IfExpr{}
	var _ Expr = &MatchExpr{}
	var _ Expr = &LoopExpr{}
	var _ Expr = &WhileExpr{}
	var _ Expr = &ForExpr{}
	var _ Expr = &TupleExpr{}
	var _ Expr = &StructLitExpr{}
	var _ Expr = &ClosureExpr{}
	var _ Expr = &SpawnExpr{}
	var _ Expr = &UnsafeExpr{}
	var _ Expr = &ReferenceExpr{}

	// Structured patterns — the L007 set.
	var _ Pat = &LiteralPat{}
	var _ Pat = &BindPat{}
	var _ Pat = &ConstructorPat{}
	var _ Pat = &WildcardPat{}
	var _ Pat = &OrPat{}
	var _ Pat = &RangePat{}
	var _ Pat = &AtBindPat{}

	// Statements.
	var _ Stmt = &LetStmt{}
	var _ Stmt = &VarStmt{}
	var _ Stmt = &ReturnStmt{}
	var _ Stmt = &BreakStmt{}
	var _ Stmt = &ContinueStmt{}
	var _ Stmt = &ExprStmt{}
	var _ Stmt = &ItemStmt{}
}

// TestMetadataFields confirms that every Typed HIR node exposes its
// TypeId through TypeOf() and that Node.NodeHirID() returns the
// embedded ID. This proves that the metadata fields the builders
// require are actually reachable through the interfaces (W04-P02-T02).
func TestMetadataFields(t *testing.T) {
	tab := typetable.New()
	ti := tab.I32()
	id := ItemID("m", "f")

	// Build a small set of typed nodes and check TypeOf / NodeHirID.
	lit := &LiteralExpr{
		TypedBase: TypedBase{Base: Base{ID: id, Span: lex.Span{}}, Type: ti},
		Kind:      LitInt,
		Text:      "7",
	}
	if lit.TypeOf() != ti {
		t.Fatalf("LiteralExpr.TypeOf = %d, want %d", lit.TypeOf(), ti)
	}
	if lit.NodeHirID() != id {
		t.Fatalf("LiteralExpr.NodeHirID = %q, want %q", lit.NodeHirID(), id)
	}

	pat := &BindPat{
		TypedBase: TypedBase{Base: Base{ID: id, Span: lex.Span{}}, Type: ti},
		Name:      "x",
	}
	if pat.TypeOf() != ti {
		t.Fatalf("BindPat.TypeOf = %d, want %d", pat.TypeOf(), ti)
	}

	// Wildcard still carries a TypeId so W10 has something to unify
	// against during exhaustiveness checking.
	w := &WildcardPat{TypedBase: TypedBase{Base: Base{ID: id, Span: lex.Span{}}, Type: ti}}
	if w.TypeOf() != ti {
		t.Fatalf("WildcardPat.TypeOf missing")
	}
}

// TestBuilderEnforcement confirms the builders panic on missing
// metadata (NodeID, type, name, required children). Each sub-test
// constructs a Builder and passes an invalid argument; the recover
// proves the builder refused.
func TestBuilderEnforcement(t *testing.T) {
	tab := typetable.New()
	b := NewBuilder(tab)

	mustPanic(t, "literal-missing-id", func() {
		b.NewLiteral(EmptyNodeID, lex.Span{}, LitInt, "1", tab.I32())
	})
	mustPanic(t, "literal-missing-type", func() {
		b.NewLiteral("m::f::lit", lex.Span{}, LitInt, "1", typetable.NoType)
	})
	mustPanic(t, "path-empty-segments", func() {
		b.NewPath("m::f::p", lex.Span{}, 1, nil, tab.I32())
	})
	mustPanic(t, "binary-nil-lhs", func() {
		one := b.NewLiteral("m::f::l", lex.Span{}, LitInt, "1", tab.I32())
		b.NewBinary("m::f::b", lex.Span{}, BinAdd, nil, one, tab.I32())
	})
	mustPanic(t, "fn-wrong-type-kind", func() {
		body := b.NewBlock("m::f::body", lex.Span{}, nil, nil, tab.Unit())
		b.NewFn("m::f", lex.Span{}, "f", tab.I32(), nil, tab.Unit(), body)
	})
	mustPanic(t, "struct-missing-type", func() {
		b.NewStruct("m::S", lex.Span{}, "S", typetable.NoType, nil, nil, false)
	})
	mustPanic(t, "or-pat-single-alt", func() {
		single := []Pat{b.NewBindPat("m::f::p", lex.Span{}, "x", tab.I32())}
		b.NewOrPat("m::f::or", lex.Span{}, single, tab.I32())
	})
	mustPanic(t, "constructor-missing-ctor-type", func() {
		b.NewConstructorPat("m::f::c", lex.Span{}, typetable.NoType, "Red",
			[]string{"Color", "Red"}, nil, nil, false, tab.I32())
	})
	mustPanic(t, "range-missing-hi", func() {
		lo := b.NewLiteral("m::f::lo", lex.Span{}, LitInt, "0", tab.I32())
		b.NewRangePat("m::f::r", lex.Span{}, lo, nil, false, tab.I32())
	})
	mustPanic(t, "fn-no-return-typeid", func() {
		body := b.NewBlock("m::f::body", lex.Span{}, nil, nil, tab.Unit())
		fnType := tab.Fn(nil, tab.Unit(), false)
		b.NewFn("m::f", lex.Span{}, "f", fnType, nil, typetable.NoType, body)
	})
}

// TestBuilderEnforcement_HappyPath confirms that well-formed calls
// succeed silently. Without this, a too-strict builder would pass
// the panic tests while rejecting legitimate input.
func TestBuilderEnforcement_HappyPath(t *testing.T) {
	tab := typetable.New()
	b := NewBuilder(tab)

	body := b.NewBlock("m::f::body", lex.Span{}, nil, nil, tab.Unit())
	fnType := tab.Fn(nil, tab.Unit(), false)
	fn := b.NewFn("m::f", lex.Span{}, "f", fnType, nil, tab.Unit(), body)
	if fn.Name != "f" {
		t.Fatalf("NewFn did not preserve Name")
	}
	if fn.Body != body {
		t.Fatalf("NewFn did not preserve Body pointer")
	}
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected builder panic, got none")
			}
		}()
		fn()
	})
}
