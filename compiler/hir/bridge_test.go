package hir

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// bridgeTest builds a complete pipeline (parse → resolve → bridge)
// from one in-memory source file and returns the resulting Program
// and TypeTable. Parse and resolve diagnostics fail the test fast so
// bridge-level assertions don't spend time on garbage inputs.
func bridgeTest(t *testing.T, modPath, filename, src string) (*Program, *typetable.Table) {
	t.Helper()
	f, pdiags := parse.Parse(filename, []byte(src))
	if len(pdiags) != 0 {
		t.Fatalf("parse failed: %v", pdiags)
	}
	srcs := []*resolve.SourceFile{{ModulePath: modPath, File: f}}
	resolved, rdiags := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rdiags) != 0 {
		t.Fatalf("resolve failed: %v", rdiags)
	}
	tab := typetable.New()
	prog, bdiags := NewBridge(tab, resolved, srcs).Run()
	if len(bdiags) != 0 {
		t.Fatalf("bridge diagnostics: %v", bdiags)
	}
	return prog, tab
}

// TestAstToHirTypePreservation exercises the L013 defense contract:
// every expression the bridge emits carries a concrete TypeId or an
// explicit KindInfer TypeId — never NoType, never Unknown.
func TestAstToHirTypePreservation(t *testing.T) {
	t.Run("fn-signature-preserves-annotations", func(t *testing.T) {
		prog, tab := bridgeTest(t, "m", "m.fuse", `
fn id(x: I32) -> I32 { return x; }
`)
		m := prog.Modules["m"]
		if m == nil || len(m.Items) != 1 {
			t.Fatalf("expected 1 module item, got %d", len(m.Items))
		}
		fn := m.Items[0].(*FnDecl)
		// Parameter x has TypeId I32.
		if len(fn.Params) != 1 {
			t.Fatalf("expected 1 param, got %d", len(fn.Params))
		}
		if fn.Params[0].TypeOf() != tab.I32() {
			t.Fatalf("param x: TypeOf = %d, want I32 (%d)", fn.Params[0].TypeOf(), tab.I32())
		}
		// Return type is I32.
		if fn.Return != tab.I32() {
			t.Fatalf("return TypeId = %d, want I32", fn.Return)
		}
		// TypeID is Fn(I32) -> I32.
		fnTy := tab.Get(fn.TypeID)
		if fnTy == nil || fnTy.Kind != typetable.KindFn {
			t.Fatalf("FnDecl.TypeID is not KindFn: %+v", fnTy)
		}
		if fnTy.Return != tab.I32() {
			t.Fatalf("Fn return in TypeTable = %d, want I32", fnTy.Return)
		}
		if len(fnTy.Children) != 1 || fnTy.Children[0] != tab.I32() {
			t.Fatalf("Fn param types = %v, want [I32]", fnTy.Children)
		}
	})

	t.Run("literal-type-hint-from-declared", func(t *testing.T) {
		prog, tab := bridgeTest(t, "m", "m.fuse", `
fn f() -> I64 {
	let x: I64 = 7;
	return x;
}
`)
		fn := prog.Modules["m"].Items[0].(*FnDecl)
		body := fn.Body
		if body == nil || len(body.Stmts) < 1 {
			t.Fatalf("expected at least 1 stmt, got %d", len(body.Stmts))
		}
		let := body.Stmts[0].(*LetStmt)
		if let.DeclaredType != tab.I64() {
			t.Fatalf("let declared = %d, want I64", let.DeclaredType)
		}
		// The literal 7 should have been hinted to I64.
		val, ok := let.Value.(*LiteralExpr)
		if !ok {
			t.Fatalf("let value is not LiteralExpr: %T", let.Value)
		}
		if val.TypeOf() != tab.I64() {
			t.Fatalf("literal `7` type = %d, want I64 (hinted)", val.TypeOf())
		}
	})

	t.Run("bool-literal-is-bool", func(t *testing.T) {
		prog, tab := bridgeTest(t, "m", "m.fuse", `
fn f() -> Bool {
	return true;
}
`)
		fn := prog.Modules["m"].Items[0].(*FnDecl)
		ret := fn.Body.Stmts[0].(*ReturnStmt)
		val := ret.Value.(*LiteralExpr)
		if val.TypeOf() != tab.Bool() {
			t.Fatalf("bool literal type = %d, want Bool", val.TypeOf())
		}
	})

	t.Run("struct-nominal-identity-propagates", func(t *testing.T) {
		prog, tab := bridgeTest(t, "m", "m.fuse", `
struct Point { x: I32, y: I32 }

fn origin() -> Point {
	return Point { x: 0, y: 0 };
}
`)
		items := prog.Modules["m"].Items
		var structDecl *StructDecl
		var fn *FnDecl
		for _, it := range items {
			switch x := it.(type) {
			case *StructDecl:
				structDecl = x
			case *FnDecl:
				fn = x
			}
		}
		if structDecl == nil || fn == nil {
			t.Fatalf("expected struct and fn; got items=%v", items)
		}
		// Struct TypeID is a KindStruct.
		st := tab.Get(structDecl.TypeID)
		if st == nil || st.Kind != typetable.KindStruct {
			t.Fatalf("struct TypeID kind = %v, want Struct", st)
		}
		// Fn return type is the same struct TypeId (nominal identity).
		if fn.Return != structDecl.TypeID {
			t.Fatalf("fn return TypeId = %d, want struct TypeId %d", fn.Return, structDecl.TypeID)
		}
		// Struct literal's StructType matches.
		body := fn.Body
		ret := body.Stmts[0].(*ReturnStmt)
		lit := ret.Value.(*StructLitExpr)
		if lit.StructType != structDecl.TypeID {
			t.Fatalf("struct literal type = %d, want %d", lit.StructType, structDecl.TypeID)
		}
	})

	t.Run("no-expression-has-NoType", func(t *testing.T) {
		prog, _ := bridgeTest(t, "m", "m.fuse", `
fn f() -> I32 {
	let a: I32 = 1;
	let b: I32 = a + 2;
	if a > b {
		return a;
	}
	return b;
}
`)
		// Walk every expression and assert TypeOf != NoType.
		// Explicit Infer is allowed (that is the bridge's pending
		// inference marker, not an Unknown default).
		fn := prog.Modules["m"].Items[0].(*FnDecl)
		walkExprs(fn.Body, func(e Expr) {
			if e.TypeOf() == typetable.NoType {
				t.Errorf("expression %T at %s has NoType", e, e.NodeSpan())
			}
		})
	})

	t.Run("enum-variant-ctor-type-set", func(t *testing.T) {
		prog, tab := bridgeTest(t, "m", "m.fuse", `
enum Dir { North, South }

fn go() -> Dir {
	return Dir.North;
}
`)
		items := prog.Modules["m"].Items
		var enumDecl *EnumDecl
		var fn *FnDecl
		for _, it := range items {
			switch x := it.(type) {
			case *EnumDecl:
				enumDecl = x
			case *FnDecl:
				fn = x
			}
		}
		if enumDecl == nil {
			t.Fatalf("enum Dir not lowered")
		}
		et := tab.Get(enumDecl.TypeID)
		if et == nil || et.Kind != typetable.KindEnum {
			t.Fatalf("enum TypeID kind = %v, want Enum", et)
		}
		// fn return is Dir (nominal identity).
		if fn.Return != enumDecl.TypeID {
			t.Fatalf("fn return = %d, want enum %d", fn.Return, enumDecl.TypeID)
		}
	})
}

// TestBridgeInvariant walks every HIR node produced by the bridge and
// confirms: (a) no NodeID is empty; (b) every Typed node has a
// non-NoType TypeId. This is the continuous invariant check that
// W04-P03-T02 declares.
func TestBridgeInvariant(t *testing.T) {
	prog, _ := bridgeTest(t, "m", "m.fuse", `
struct Point { x: I32, y: I32 }

fn area(p: Point) -> I32 {
	let w: I32 = p.x;
	let h: I32 = p.y;
	return w * h;
}
`)
	report := RunInvariantWalker(prog)
	if len(report) != 0 {
		t.Fatalf("invariant walker produced violations:\n%v", report)
	}
}

// walkExprs is a test-only traversal that visits every Expr reachable
// from a Block. It deliberately lives in the test file so that the
// production code path only has one walker (the invariant walker).
func walkExprs(b *Block, visit func(Expr)) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		walkStmtExprs(s, visit)
	}
	if b.Trailing != nil {
		visitExpr(b.Trailing, visit)
	}
}

func walkStmtExprs(s Stmt, visit func(Expr)) {
	switch x := s.(type) {
	case *LetStmt:
		if x.Value != nil {
			visitExpr(x.Value, visit)
		}
	case *VarStmt:
		if x.Value != nil {
			visitExpr(x.Value, visit)
		}
	case *ReturnStmt:
		if x.Value != nil {
			visitExpr(x.Value, visit)
		}
	case *BreakStmt:
		if x.Value != nil {
			visitExpr(x.Value, visit)
		}
	case *ExprStmt:
		if x.Expr != nil {
			visitExpr(x.Expr, visit)
		}
	}
}

func visitExpr(e Expr, visit func(Expr)) {
	visit(e)
	switch x := e.(type) {
	case *BinaryExpr:
		visitExpr(x.Lhs, visit)
		visitExpr(x.Rhs, visit)
	case *AssignExpr:
		visitExpr(x.Lhs, visit)
		visitExpr(x.Rhs, visit)
	case *UnaryExpr:
		visitExpr(x.Operand, visit)
	case *CastExpr:
		visitExpr(x.Expr, visit)
	case *CallExpr:
		visitExpr(x.Callee, visit)
		for _, a := range x.Args {
			visitExpr(a, visit)
		}
	case *FieldExpr:
		visitExpr(x.Receiver, visit)
	case *OptFieldExpr:
		visitExpr(x.Receiver, visit)
	case *TryExpr:
		visitExpr(x.Receiver, visit)
	case *IndexExpr:
		visitExpr(x.Receiver, visit)
		visitExpr(x.Index, visit)
	case *IndexRangeExpr:
		visitExpr(x.Receiver, visit)
		if x.Low != nil {
			visitExpr(x.Low, visit)
		}
		if x.High != nil {
			visitExpr(x.High, visit)
		}
	case *Block:
		walkExprs(x, visit)
	case *IfExpr:
		visitExpr(x.Cond, visit)
		walkExprs(x.Then, visit)
		if x.Else != nil {
			visitExpr(x.Else, visit)
		}
	case *MatchExpr:
		visitExpr(x.Scrutinee, visit)
		for _, arm := range x.Arms {
			if arm.Guard != nil {
				visitExpr(arm.Guard, visit)
			}
			walkExprs(arm.Body, visit)
		}
	case *TupleExpr:
		for _, el := range x.Elements {
			visitExpr(el, visit)
		}
	case *StructLitExpr:
		for _, f := range x.Fields {
			if f.Value != nil {
				visitExpr(f.Value, visit)
			}
		}
		if x.Base != nil {
			visitExpr(x.Base, visit)
		}
	}
}
