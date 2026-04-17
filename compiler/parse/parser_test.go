package parse

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/ast"
)

// parseOK parses src and asserts the parser reported no diagnostics. Returns
// the parsed file for further inspection.
func parseOK(t *testing.T, src string) *ast.File {
	t.Helper()
	f, diags := Parse("test.fuse", []byte(src))
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics parsing %q:\n%v", src, diags)
	}
	if f == nil {
		t.Fatalf("parse returned nil file for %q", src)
	}
	return f
}

// TestItemParsing exercises every top-level item kind. The test asserts both
// that parsing succeeds without diagnostics and that the root node type
// matches the grammar production.
func TestItemParsing(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"fn-empty", "fn main() {}", "*ast.FnDecl"},
		{"fn-return", "fn id(x: I32) -> I32 { return x; }", "*ast.FnDecl"},
		{"const-fn", "const fn two() -> I32 { return 2; }", "*ast.FnDecl"},
		{"fn-generic", "fn id[T](x: T) -> T { return x; }", "*ast.FnDecl"},
		{"fn-where", "fn f[T](x: T) -> T where T: Copy { return x; }", "*ast.FnDecl"},
		{"struct-named", "struct Point { x: I32, y: I32 }", "*ast.StructDecl"},
		{"struct-tuple", "struct Pair(I32, I32);", "*ast.StructDecl"},
		{"struct-unit", "struct Marker;", "*ast.StructDecl"},
		{"struct-generic", "struct Box[T] { inner: T }", "*ast.StructDecl"},
		{"enum-basic", "enum Direction { North, South, East, West }", "*ast.EnumDecl"},
		{"enum-variants", "enum Msg { Quit, Move { x: I32, y: I32 }, Write(String) }", "*ast.EnumDecl"},
		{"enum-explicit", "enum Code { A = 1, B = 2 }", "*ast.EnumDecl"},
		{"trait", "trait Greet { fn hello(self) -> String; }", "*ast.TraitDecl"},
		{"trait-default", "trait Name { fn default() -> I32 { return 0; } }", "*ast.TraitDecl"},
		{"impl-inherent", "impl Point { fn new() -> Point { return Point { x: 0, y: 0 }; } }", "*ast.ImplDecl"},
		{"impl-trait", "impl Point : Greet { fn hello(self) -> String { return \"hi\"; } }", "*ast.ImplDecl"},
		{"const", "const MAX: I32 = 100;", "*ast.ConstDecl"},
		{"static", "static GREETING: String = \"hi\";", "*ast.StaticDecl"},
		{"type-alias", "type Pair = (I32, I32);", "*ast.TypeDecl"},
		{"extern-fn", "extern fn puts(s: Ptr[U8]) -> I32;", "*ast.ExternDecl"},
		{"extern-static", "extern static errno: I32;", "*ast.ExternDecl"},
		{"union", "union Repr { a: I32, b: F32 }", "*ast.UnionDecl"},
		{"import", "import std.io;", "__import__"},
		{"import-alias", "import std.collections as c;", "__import__"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := parseOK(t, tc.src)
			if tc.want == "__import__" {
				if len(f.Imports) != 1 {
					t.Fatalf("expected 1 import, got %d", len(f.Imports))
				}
				return
			}
			if len(f.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(f.Items))
			}
			got := fmt.Sprintf("%T", f.Items[0])
			if got != tc.want {
				t.Errorf("item type = %s, want %s", got, tc.want)
			}
		})
	}
}

// TestExprPrecedence verifies the precedence ladder from reference Appendix B
// produces the expected tree shape. Rather than rendering full AST, we walk
// the tree and compare operator placement.
func TestExprPrecedence(t *testing.T) {
	cases := []struct {
		src    string
		render string // S-expression render of the parsed expression
	}{
		// Multiplicative binds tighter than additive.
		{"a + b * c", "(+ a (* b c))"},
		{"a * b + c", "(+ (* a b) c)"},
		// Left-associative arithmetic.
		{"a - b - c", "(- (- a b) c)"},
		{"a + b + c", "(+ (+ a b) c)"},
		// Comparison binds looser than shift.
		{"a << 2 == b", "(== (<< a 2) b)"},
		// Logical short-circuit precedence: && binds tighter than ||.
		{"a || b && c", "(|| a (&& b c))"},
		// Bitwise: & tighter than ^ tighter than |.
		{"a | b ^ c & d", "(| a (^ b (& c d)))"},
		// Unary binds tighter than multiplicative.
		{"-a * b", "(* (- a) b)"},
		{"!a || b", "(|| (! a) b)"},
		// Assignment is right-associative and looser than everything else.
		{"a = b = c + d", "(= a (= b (+ c d)))"},
		{"a += b * c", "(+= a (* b c))"},
		// `as` cast sits between unary and multiplicative.
		{"a as I32 * b", "(* (as a I32) b)"},
		// Postfix tightest: `x?.y + 1` is `(?. x y) + 1`.
		{"x?.y + 1", "(+ (?. x y) 1)"},
		// Parenthesized expressions override precedence.
		{"(a + b) * c", "(* (paren (+ a b)) c)"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			src := "fn f() { " + tc.src + "; }"
			f := parseOK(t, src)
			fn := f.Items[0].(*ast.FnDecl)
			stmt := fn.Body.Stmts[0].(*ast.ExprStmt)
			got := renderSexpr(stmt.Expr)
			if got != tc.render {
				t.Errorf("src=%q\n got:  %s\n want: %s", tc.src, got, tc.render)
			}
		})
	}
}

// TestTypeExprs covers each type_expr branch.
func TestTypeExprs(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"named", "fn f(x: I32) {}", "*ast.PathType"},
		{"qualified", "fn f(x: std.io.Handle) {}", "*ast.PathType"},
		{"generic-args", "fn f(x: Vec[I32]) {}", "*ast.PathType"},
		{"tuple", "fn f(x: (I32, I32)) {}", "*ast.TupleType"},
		{"unit", "fn f(x: ()) {}", "*ast.UnitType"},
		{"array", "fn f(x: [I32; 4]) {}", "*ast.ArrayType"},
		{"slice", "fn f(x: [I32]) {}", "*ast.SliceType"},
		{"ptr", "fn f(x: Ptr[U8]) {}", "*ast.PtrType"},
		{"fntype", "fn f(x: fn(I32) -> I32) {}", "*ast.FnType"},
		{"dyn", "fn f(x: dyn Greet) {}", "*ast.DynType"},
		{"dyn-plus", "fn f(x: dyn Greet + Send) {}", "*ast.DynType"},
		{"impl", "fn f() -> impl Greet {}", "*ast.ImplType"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := parseOK(t, tc.src)
			fn := f.Items[0].(*ast.FnDecl)
			var ty ast.Type
			if len(fn.Params) > 0 {
				ty = fn.Params[0].Type
			} else {
				ty = fn.Return
			}
			got := fmt.Sprintf("%T", ty)
			if got != tc.want {
				t.Errorf("type = %s, want %s (src=%q)", got, tc.want, tc.src)
			}
		})
	}
}

// TestPatternParsing covers each pattern production.
func TestPatternParsing(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"literal", "match x { 42 => 1 }", "*ast.LiteralPat"},
		{"wildcard", "match x { _ => 1 }", "*ast.WildcardPat"},
		{"bind", "match x { y => y }", "*ast.BindPat"},
		{"ctor-unit", "match x { None => 0 }", "*ast.CtorPat"},
		{"ctor-tuple", "match x { Some(v) => v }", "*ast.CtorPat"},
		{"ctor-struct", "match x { Point { x, y } => x }", "*ast.CtorPat"},
		{"tuple", "match x { (a, b) => a }", "*ast.TuplePat"},
		{"or", "match x { 1 | 2 | 3 => 0 }", "*ast.OrPat"},
		{"range", "match x { 1..10 => 0 }", "*ast.RangePat"},
		{"range-inclusive", "match x { 1..=10 => 0 }", "*ast.RangePat"},
		{"at", "match x { v @ 1..=5 => v }", "*ast.AtPat"},
		{"struct-rest", "match x { Point { x, .. } => x }", "*ast.CtorPat"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "fn f() { " + tc.src + "; }"
			f := parseOK(t, src)
			fn := f.Items[0].(*ast.FnDecl)
			exprStmt := fn.Body.Stmts[0].(*ast.ExprStmt)
			m := exprStmt.Expr.(*ast.MatchExpr)
			if len(m.Arms) == 0 {
				t.Fatalf("no arms in match")
			}
			got := fmt.Sprintf("%T", m.Arms[0].Pattern)
			if got != tc.want {
				t.Errorf("pattern = %s, want %s (src=%q)", got, tc.want, tc.src)
			}
		})
	}
}

// TestDecoratorParsing covers every decorator from the wave plan's list.
func TestDecoratorParsing(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"value", "@value struct P { x: I32 }"},
		{"rank", "@rank(1) static LOCK: I32 = 0;"},
		{"repr-c", "@repr(C) struct X { x: I32 }"},
		{"repr-packed", "@repr(packed) struct X { x: I32 }"},
		{"repr-u8", "@repr(U8) enum E { A, B }"},
		{"repr-i32", "@repr(I32) enum E { A, B }"},
		{"align", "@align(16) struct A { x: I32 }"},
		{"inline", "@inline fn f() {}"},
		{"inline-always", "@inline(always) fn f() {}"},
		{"inline-never", "@inline(never) fn f() {}"},
		{"cold", "@cold fn slow() {}"},
		{"cfg", "@cfg(target = linux) fn f() {}"},
		{"stacked", "@repr(C) @align(8) struct X { x: I32 }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := parseOK(t, tc.src)
			if len(f.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(f.Items))
			}
			decs := itemDecorators(f.Items[0])
			if len(decs) == 0 {
				t.Errorf("no decorators attached to %T", f.Items[0])
			}
		})
	}
}

// TestStructLiteralDisambig covers reference §10.7: `IDENT {` is only a
// struct literal when the brace body looks like a field list. In
// `if`/`while`/`for`/`match` headers, `Foo { ... }` is forbidden — the `Foo`
// is a plain path expression and the `{` opens the block.
func TestStructLiteralDisambig(t *testing.T) {
	// Helper: return the last expression in a function body, whether it
	// lives in Stmts as an ExprStmt or is the block's Trailing value. A
	// block-expression that is the final item of the body lands in Trailing.
	bodyExpr := func(fn *ast.FnDecl) ast.Expr {
		if fn.Body.Trailing != nil {
			return fn.Body.Trailing
		}
		if len(fn.Body.Stmts) == 0 {
			t.Fatalf("empty body for %s", fn.Name.Name)
		}
		es, ok := fn.Body.Stmts[len(fn.Body.Stmts)-1].(*ast.ExprStmt)
		if !ok {
			t.Fatalf("last stmt is not ExprStmt, got %T", fn.Body.Stmts[len(fn.Body.Stmts)-1])
		}
		return es.Expr
	}

	// The forcing case: `if Foo { x }` is a path + block, NOT
	// `if (Foo { x }) { ... }`.
	f := parseOK(t, `fn f() { if Foo { x } }`)
	fn := f.Items[0].(*ast.FnDecl)
	ifx := bodyExpr(fn).(*ast.IfExpr)
	if _, ok := ifx.Cond.(*ast.PathExpr); !ok {
		t.Errorf("if cond should be PathExpr (not StructLitExpr), got %T", ifx.Cond)
	}
	if ifx.Then == nil {
		t.Errorf("if must have a then-block")
	}

	// Positive case: outside a no-struct context, `Foo { x: 1 }` IS a struct
	// literal.
	f2 := parseOK(t, `fn f() -> Foo { return Foo { x: 1 }; }`)
	fn2 := f2.Items[0].(*ast.FnDecl)
	ret := fn2.Body.Stmts[0].(*ast.ReturnStmt)
	if _, ok := ret.Value.(*ast.StructLitExpr); !ok {
		t.Errorf("return value should be StructLitExpr, got %T", ret.Value)
	}

	// Empty braces `Foo {}` is also a struct literal in normal position.
	f3 := parseOK(t, `fn f() -> Foo { return Foo {}; }`)
	fn3 := f3.Items[0].(*ast.FnDecl)
	ret3 := fn3.Body.Stmts[0].(*ast.ReturnStmt)
	if _, ok := ret3.Value.(*ast.StructLitExpr); !ok {
		t.Errorf("empty-body return value should be StructLitExpr, got %T", ret3.Value)
	}

	// `while Foo { }` — `Foo` must be a path expression.
	f4 := parseOK(t, `fn f() { while Foo { x } }`)
	fn4 := f4.Items[0].(*ast.FnDecl)
	wh := bodyExpr(fn4).(*ast.WhileExpr)
	if _, ok := wh.Cond.(*ast.PathExpr); !ok {
		t.Errorf("while cond should be PathExpr, got %T", wh.Cond)
	}

	// `match Foo { ... }` — scrutinee is PathExpr, not StructLitExpr.
	f5 := parseOK(t, `fn f() { match Foo { _ => 1 } }`)
	fn5 := f5.Items[0].(*ast.FnDecl)
	m := bodyExpr(fn5).(*ast.MatchExpr)
	if _, ok := m.Scrutinee.(*ast.PathExpr); !ok {
		t.Errorf("match scrutinee should be PathExpr, got %T", m.Scrutinee)
	}
}

// TestOptionalChainParse covers reference §1.10 at the parser level: `?.` is
// a single postfix operator that produces OptFieldExpr.
func TestOptionalChainParse(t *testing.T) {
	// Simple `x?.y`.
	f := parseOK(t, `fn f() { x?.y; }`)
	fn := f.Items[0].(*ast.FnDecl)
	stmt := fn.Body.Stmts[0].(*ast.ExprStmt)
	of, ok := stmt.Expr.(*ast.OptFieldExpr)
	if !ok {
		t.Fatalf("expected OptFieldExpr, got %T", stmt.Expr)
	}
	if of.Name.Name != "y" {
		t.Errorf("opt-field name = %q, want %q", of.Name.Name, "y")
	}
	if _, ok := of.Receiver.(*ast.PathExpr); !ok {
		t.Errorf("opt-field receiver should be PathExpr, got %T", of.Receiver)
	}

	// Chain `x?.y?.z`.
	f2 := parseOK(t, `fn f() { x?.y?.z; }`)
	fn2 := f2.Items[0].(*ast.FnDecl)
	stmt2 := fn2.Body.Stmts[0].(*ast.ExprStmt)
	outer, ok := stmt2.Expr.(*ast.OptFieldExpr)
	if !ok {
		t.Fatalf("outer should be OptFieldExpr, got %T", stmt2.Expr)
	}
	if outer.Name.Name != "z" {
		t.Errorf("outer name = %q, want z", outer.Name.Name)
	}
	inner, ok := outer.Receiver.(*ast.OptFieldExpr)
	if !ok {
		t.Fatalf("inner should be OptFieldExpr, got %T", outer.Receiver)
	}
	if inner.Name.Name != "y" {
		t.Errorf("inner name = %q, want y", inner.Name.Name)
	}

	// Mixed `x.y?.z` — regular field then optional.
	f3 := parseOK(t, `fn f() { x.y?.z; }`)
	fn3 := f3.Items[0].(*ast.FnDecl)
	stmt3 := fn3.Body.Stmts[0].(*ast.ExprStmt)
	of3 := stmt3.Expr.(*ast.OptFieldExpr)
	if _, ok := of3.Receiver.(*ast.FieldExpr); !ok {
		t.Errorf("receiver should be FieldExpr, got %T", of3.Receiver)
	}

	// `x?` alone is the try operator, NOT optional chain.
	f4 := parseOK(t, `fn f() { x?; }`)
	fn4 := f4.Items[0].(*ast.FnDecl)
	stmt4 := fn4.Body.Stmts[0].(*ast.ExprStmt)
	if _, ok := stmt4.Expr.(*ast.TryExpr); !ok {
		t.Errorf("`x?` should be TryExpr, got %T", stmt4.Expr)
	}
}

// TestNopanicOnMalformed throws a corpus of malformed inputs at the parser
// and asserts that every call returns without panicking. Diagnostics are
// expected; panics are not (Rule 6.9 silent-default discipline).
func TestNopanicOnMalformed(t *testing.T) {
	cases := []string{
		"",
		"fn",
		"fn main",
		"fn main(",
		"fn main()",
		"fn main() {",
		"fn main() { let",
		"fn main() { let x",
		"fn main() { let x = ",
		"fn main() { let x = ; }",
		"fn main() { match { } }",
		"fn main() { if }",
		"fn main() { while }",
		"fn main() { for x in }",
		"struct",
		"struct S",
		"struct S { x:",
		"struct S { x: I32,",
		"enum",
		"enum E { , }",
		"trait T { fn",
		"impl",
		"impl T for",
		"const",
		"const X",
		"const X =",
		"type",
		"type T =",
		"extern",
		"extern fn",
		"@",
		"@rank(",
		"@rank)",
		"@@@",
		"}}}",
		"))) ]]] [[[ (((",
		"fn f[T, , ](x: T) {}",
		// Deeply nested parens — must not blow the stack.
		strings.Repeat("(", 300) + "x" + strings.Repeat(")", 300),
		// Garbage characters.
		"!@#$%^&*() fn main() {}",
		// Just unclosed strings handled by the lexer; the parser sees the
		// tokens before any EOF.
		"fn f() { \"hello }",
	}
	for i, src := range cases {
		t.Run(fmt.Sprintf("case-%02d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("parser panicked on %q: %v", src, r)
				}
			}()
			f, diags := Parse("malformed.fuse", []byte(src))
			if f == nil {
				t.Errorf("parser returned nil file for %q", src)
			}
			// Diagnostics are allowed; the test's only invariant is no panic.
			_ = diags
		})
	}
}

// renderSexpr produces a compact s-expression string for an AST expression.
// Used by precedence tests to compare tree shape.
func renderSexpr(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.LiteralExpr:
		return v.Text
	case *ast.PathExpr:
		names := make([]string, len(v.Segments))
		for i, s := range v.Segments {
			names[i] = s.Name
		}
		return strings.Join(names, ".")
	case *ast.BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", binOpSym(v.Op), renderSexpr(v.Lhs), renderSexpr(v.Rhs))
	case *ast.AssignExpr:
		return fmt.Sprintf("(%s %s %s)", assignOpSym(v.Op), renderSexpr(v.Lhs), renderSexpr(v.Rhs))
	case *ast.UnaryExpr:
		return fmt.Sprintf("(%s %s)", unaryOpSym(v.Op), renderSexpr(v.Operand))
	case *ast.CastExpr:
		return fmt.Sprintf("(as %s %s)", renderSexpr(v.Expr), renderType(v.Type))
	case *ast.ParenExpr:
		return fmt.Sprintf("(paren %s)", renderSexpr(v.Inner))
	case *ast.OptFieldExpr:
		return fmt.Sprintf("(?. %s %s)", renderSexpr(v.Receiver), v.Name.Name)
	case *ast.FieldExpr:
		return fmt.Sprintf("(. %s %s)", renderSexpr(v.Receiver), v.Name.Name)
	}
	return fmt.Sprintf("?%T", e)
}

func renderType(t ast.Type) string {
	switch v := t.(type) {
	case *ast.PathType:
		names := make([]string, len(v.Segments))
		for i, s := range v.Segments {
			names[i] = s.Name
		}
		return strings.Join(names, ".")
	}
	return fmt.Sprintf("?%T", t)
}

func binOpSym(op ast.BinaryOp) string {
	switch op {
	case ast.BinAdd:
		return "+"
	case ast.BinSub:
		return "-"
	case ast.BinMul:
		return "*"
	case ast.BinDiv:
		return "/"
	case ast.BinMod:
		return "%"
	case ast.BinShl:
		return "<<"
	case ast.BinShr:
		return ">>"
	case ast.BinAnd:
		return "&"
	case ast.BinOr:
		return "|"
	case ast.BinXor:
		return "^"
	case ast.BinLogAnd:
		return "&&"
	case ast.BinLogOr:
		return "||"
	case ast.BinEq:
		return "=="
	case ast.BinNe:
		return "!="
	case ast.BinLt:
		return "<"
	case ast.BinLe:
		return "<="
	case ast.BinGt:
		return ">"
	case ast.BinGe:
		return ">="
	}
	return "?"
}

func assignOpSym(op ast.AssignOp) string {
	switch op {
	case ast.AssignEq:
		return "="
	case ast.AssignAdd:
		return "+="
	case ast.AssignSub:
		return "-="
	case ast.AssignMul:
		return "*="
	case ast.AssignDiv:
		return "/="
	case ast.AssignMod:
		return "%="
	case ast.AssignAnd:
		return "&="
	case ast.AssignOr:
		return "|="
	case ast.AssignXor:
		return "^="
	case ast.AssignShl:
		return "<<="
	case ast.AssignShr:
		return ">>="
	}
	return "?="
}

func unaryOpSym(op ast.UnaryOp) string {
	switch op {
	case ast.UnNot:
		return "!"
	case ast.UnNeg:
		return "-"
	case ast.UnDeref:
		return "*"
	case ast.UnAddr:
		return "&"
	}
	return "?"
}

// itemDecorators returns the decorator list attached to a top-level item, or
// nil if the item type does not carry decorators.
func itemDecorators(it ast.Item) []*ast.Decorator {
	switch v := it.(type) {
	case *ast.FnDecl:
		return v.Decorators
	case *ast.StructDecl:
		return v.Decorators
	case *ast.EnumDecl:
		return v.Decorators
	case *ast.UnionDecl:
		return v.Decorators
	case *ast.StaticDecl:
		return v.Decorators
	case *ast.ConstDecl:
		return v.Decorators
	}
	return nil
}
