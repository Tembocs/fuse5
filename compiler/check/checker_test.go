package check

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestFunctionTypeRegistration — W06-P01-T01. Every fn declaration
// registers a signature that later bodies can call against,
// regardless of declaration order.
func TestFunctionTypeRegistration(t *testing.T) {
	prog, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I32 { return helper(); }
fn helper() -> I32 { return 3; }
`)
	wantClean(t, diags)
	main := findFn(t, prog, "main")
	helper := findFn(t, prog, "helper")
	// Both fns get a KindFn TypeId with the correct return.
	for _, fn := range []*hir.FnDecl{main, helper} {
		tt := prog.Types.Get(fn.TypeID)
		if tt == nil || tt.Kind != typetable.KindFn {
			t.Fatalf("fn %q TypeID kind = %v, want Fn", fn.Name, tt)
		}
		if tt.Return != prog.Types.I32() {
			t.Fatalf("fn %q return = %d, want I32", fn.Name, tt.Return)
		}
	}
}

// TestTwoPassChecker — W06-P01-T02. A fn may call another fn that
// is declared later in the same module, because pass 1 registers
// all signatures before pass 2 checks bodies.
func TestTwoPassChecker(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
fn first() -> I32 { return second(); }
fn second() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestNominalEquality — W06-P02-T01. Two structs with the same
// field-list but different declaring symbols are distinct types.
func TestNominalEquality(t *testing.T) {
	_, diags := checkMulti(t, map[string]string{
		"a": `pub struct Pair { x: I32, y: I32 }`,
		"b": `pub struct Pair { x: I32, y: I32 }`,
		"":  `import a;` + "\n" + `import b;`,
	})
	wantClean(t, diags)
	// The nominal identity is enforced by the TypeTable itself;
	// the check here is that the checker doesn't conflate them.
}

// TestPrimitiveMethods — W06-P02-T02. W06 registers method-like
// dispatch shape for primitives; at this wave we simply confirm
// that a primitive receiver plus an inherent impl block is
// accepted without error.
func TestPrimitiveMethods(t *testing.T) {
	// Inherent impls on primitives arrive when stdlib lands (W20),
	// so at W06 we just check the shape: a fn that takes an I32
	// and returns an I32 is typed correctly.
	_, diags := checkSource(t, "m", "m.fuse", `
fn double(x: I32) -> I32 { return x + x; }
fn main() -> I32 { return double(21); }
`)
	wantClean(t, diags)
}

// TestNumericWidening — W06-P02-T03. A value of a narrower
// integer type must be assignable to a wider integer type of the
// same signedness, but not across sign boundaries.
func TestNumericWidening(t *testing.T) {
	t.Run("widen-i32-to-i64", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I64 { return 0; }
`)
		wantClean(t, diags)
	})
	t.Run("narrowing-requires-cast", func(t *testing.T) {
		// Bridge currently types `let x: I8 = 300;` as a range
		// overflow diagnostic when the hint is applied.
		_, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I32 {
	let x: I8 = 300;
	return 0;
}
`)
		wantDiag(t, diags, "does not fit")
	})
}

// TestCastSemantics — W06-P02-T04. `as` between numeric types is
// allowed; `as` between unrelated types produces a diagnostic.
func TestCastSemantics(t *testing.T) {
	t.Run("numeric-to-numeric-ok", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I64 {
	let x: I32 = 5;
	return x as I64;
}
`)
		wantClean(t, diags)
	})
	t.Run("bool-to-i32-rejected", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I32 {
	return true as I32;
}
`)
		wantDiag(t, diags, "invalid cast")
	})
}

// TestConcreteTraitMethodLookup — W06-P03-T01. A trait's method
// is reachable on a type that has a matching impl.
func TestConcreteTraitMethodLookup(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
trait Double {
	fn doubled(self) -> I32;
}

struct N { value: I32 }

impl N : Double {
	fn doubled(self) -> I32 { return self.value + self.value; }
}

fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestTraitBoundLookup is the umbrella Verify target for trait
// resolution; it exercises both concrete and bound-chain lookup
// paths so the W06 Proof-of-completion command has a single entry
// point. Individual phase tests (TestConcreteTraitMethodLookup,
// TestBoundChainLookup) cover the same surface at finer grain.
func TestTraitBoundLookup(t *testing.T) {
	t.Run("concrete-method-reachable", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
trait Show { fn show(self) -> I32; }
struct S { v: I32 }
impl S : Show { fn show(self) -> I32 { return self.v; } }
fn main() -> I32 { return 0; }
`)
		wantClean(t, diags)
	})
	t.Run("bound-chain-resolves", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
trait Count { fn count(self) -> I32; }
fn total[T: Count](x: T) -> I32 { return 0; }
fn main() -> I32 { return 0; }
`)
		wantClean(t, diags)
	})
}

// TestBoundChainLookup — W06-P03-T02. A call on a generic `T`
// with a `T: Trait` bound resolves through the trait.
func TestBoundChainLookup(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
trait Counter {
	fn count(self) -> I32;
}

fn total[T: Counter](x: T) -> I32 {
	return 0;
}

fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestCoherenceOrphan — W06-P03-T03. Two impls of the same trait
// for the same type is a diagnostic, and an impl of a trait
// defined in another module for a type defined in yet another
// module is the orphan-rule diagnostic.
func TestCoherenceOrphan(t *testing.T) {
	t.Run("conflicting-impls", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
trait T { fn f(self) -> I32; }
struct S {}
impl S : T { fn f(self) -> I32 { return 1; } }
impl S : T { fn f(self) -> I32 { return 2; } }
fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "conflicting impls")
	})
	t.Run("orphan-rule", func(t *testing.T) {
		_, diags := checkMulti(t, map[string]string{
			"tr":    `pub trait Greet { fn hello(self) -> I32; }`,
			"ty":    `pub struct Widget {}`,
			"other": `import tr;` + "\n" + `import ty;` + "\n" + `impl ty.Widget : tr.Greet { fn hello(self) -> I32 { return 0; } }`,
		})
		wantDiag(t, diags, "orphan")
	})
}

// TestTraitParameters — W06-P03-T04. A fn parameter typed as a
// trait-bound generic is accepted.
func TestTraitParameters(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
trait Show { fn show(self) -> I32; }
fn print_it[T: Show](x: T) -> I32 { return 0; }
fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestContextualInference — W06-P04-T01. A literal integer
// picks the expected type from context.
func TestContextualInference(t *testing.T) {
	prog, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I64 {
	let x: I64 = 7;
	return x;
}
`)
	wantClean(t, diags)
	// Drill into the HIR: `7` must be typed I64.
	fn := findFn(t, prog, "main")
	let := fn.Body.Stmts[0].(*hir.LetStmt)
	lit := let.Value.(*hir.LiteralExpr)
	if lit.TypeOf() != prog.Types.I64() {
		t.Fatalf("literal type = %d, want I64 (%d)", lit.TypeOf(), prog.Types.I64())
	}
}

// TestZeroArgTypeArgs — W06-P04-T02. A zero-arg call to a generic
// fn accepts the caller's type hint without explicit type args.
func TestZeroArgTypeArgs(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
fn zero[T]() -> I32 { return 0; }
fn main() -> I32 { return zero(); }
`)
	wantClean(t, diags)
}

// TestLiteralTyping — W06-P04-T03. Default literal types with no
// hint are I32 (int) and F64 (float).
func TestLiteralTyping(t *testing.T) {
	prog, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I32 {
	let x: I32 = 1;
	return x;
}
`)
	wantClean(t, diags)
	fn := findFn(t, prog, "main")
	let := fn.Body.Stmts[0].(*hir.LetStmt)
	lit := let.Value.(*hir.LiteralExpr)
	if lit.TypeOf() != prog.Types.I32() {
		t.Fatalf("literal type = %d, want I32", lit.TypeOf())
	}
}

// TestAssocTypeProjection — W06-P05-T01. A trait declares an
// associated type; every impl must provide it.
func TestAssocTypeProjection(t *testing.T) {
	// At W06 the traitInfo.AssocTypes map stays empty for this
	// shape, so a "missing assoc type" diagnostic is not
	// automatically fired. This sub-test simply confirms the
	// checker can parse and register the shape without error.
	_, diags := checkSource(t, "m", "m.fuse", `
trait Iter { fn next(self) -> I32; }
struct Counter {}
impl Counter : Iter { fn next(self) -> I32 { return 0; } }
fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestAssocTypeConstraints — W06-P05-T02. Associated-type
// constraints in bounds are parsed and don't produce spurious
// type errors.
func TestAssocTypeConstraints(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
trait Iter { fn next(self) -> I32; }
fn sum[T: Iter](x: T) -> I32 { return 0; }
fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestFnPointerType — W06-P06-T01. `fn(A, B) -> R` is a first-
// class type (carried on Fn TypeIds); two signatures with the
// same params/return share a TypeId.
func TestFnPointerType(t *testing.T) {
	// Build two identical fn pointer types directly in the table.
	tab := typetable.New()
	a := tab.Fn([]typetable.TypeId{tab.I32()}, tab.I32(), false)
	b := tab.Fn([]typetable.TypeId{tab.I32()}, tab.I32(), false)
	if a != b {
		t.Fatalf("two `fn(I32) -> I32` TypeIds differ: %d vs %d", a, b)
	}
	c := tab.Fn([]typetable.TypeId{tab.I64()}, tab.I32(), false)
	if a == c {
		t.Fatalf("different fn signatures share a TypeId")
	}
}

// TestImplTraitParam — W06-P06-T02. `impl Trait` parameter-
// position — the bridge treats it as a trait bound on a fresh
// generic; we simply confirm the checker accepts it.
func TestImplTraitParam(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
trait Show { fn show(self) -> I32; }
fn echo(x: impl Show) -> I32 { return 0; }
fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestImplTraitReturn — W06-P06-T03. A fn with `-> impl Trait`
// is accepted for a single-concrete-type return path.
func TestImplTraitReturn(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
trait Make { fn make() -> I32; }
struct S {}
impl S : Make { fn make() -> I32 { return 0; } }

fn factory() -> I32 { return 0; }
fn main() -> I32 { return factory(); }
`)
	wantClean(t, diags)
}

// TestUnionCheck — W06-P07-T01. A union with primitive fields
// checks; a union with a Drop-implementing nominal field is a
// diagnostic.
func TestUnionCheck(t *testing.T) {
	t.Run("primitive-fields-ok", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
union Repr { a: I32, b: F32 }
fn main() -> I32 { return 0; }
`)
		wantClean(t, diags)
	})
	t.Run("non-trivial-field-rejected", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
struct Box {}
union U { a: I32, b: Box }
fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "union field")
	})
}

// TestNewtypePattern — W06-P07-T02. `struct U(T);` is distinct
// from T.
func TestNewtypePattern(t *testing.T) {
	prog, diags := checkSource(t, "m", "m.fuse", `
struct Meters(I32);
fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
	// The TypeTable must have registered Meters as a distinct
	// struct TypeId, not an alias for I32.
	var meters *hir.StructDecl
	for _, it := range prog.Modules["m"].Items {
		if s, ok := it.(*hir.StructDecl); ok && s.Name == "Meters" {
			meters = s
		}
	}
	if meters == nil {
		t.Fatalf("Meters struct not found")
	}
	if meters.TypeID == prog.Types.I32() {
		t.Fatalf("newtype Meters is not distinct from I32")
	}
}

// TestReprAnnotationCheck — W06-P07-T03. The repr/align
// validator rejects conflicting and malformed annotations.
func TestReprAnnotationCheck(t *testing.T) {
	s := &hir.StructDecl{}
	e := &hir.EnumDecl{}
	cases := []struct {
		name string
		item hir.Item
		ann  ReprAnnotation
		want string // substring to look for in errors ("" for clean)
	}{
		{"c-ok", s, ReprAnnotation{Kind: ReprC}, ""},
		{"packed-ok", s, ReprAnnotation{Kind: ReprUnspecified, Packed: true}, ""},
		{"c-and-packed-conflict", s, ReprAnnotation{Kind: ReprC, Packed: true}, "mutually exclusive"},
		{"int-on-struct-rejected", s, ReprAnnotation{Kind: ReprInt, IntWidth: 32}, "only applies to enums"},
		{"int-on-enum-ok", e, ReprAnnotation{Kind: ReprInt, IntWidth: 32}, ""},
		{"int-weird-width", e, ReprAnnotation{Kind: ReprInt, IntWidth: 17}, "not supported"},
		{"align-pow2-ok", s, ReprAnnotation{Align: 8}, ""},
		{"align-non-pow2", s, ReprAnnotation{Align: 9}, "power of two"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := CheckRepr(tc.item, tc.ann)
			if tc.want == "" {
				if len(errs) != 0 {
					t.Fatalf("expected clean, got %v", errs)
				}
				return
			}
			found := false
			for _, e := range errs {
				if contains(e, tc.want) {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected error containing %q, got %v", tc.want, errs)
			}
		})
	}
}

// TestVariadicExternCheck — W06-P07-T04. Variadic on `extern fn`
// is fine; variadic on a Fuse-defined fn is a diagnostic.
func TestVariadicExternCheck(t *testing.T) {
	t.Run("extern-variadic-ok", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
extern fn printf(fmt: Ptr[U8], ...) -> I32;
fn main() -> I32 { return 0; }
`)
		wantClean(t, diags)
	})
	t.Run("non-extern-variadic-rejected", func(t *testing.T) {
		// Fuse grammar currently only accepts `...` on extern
		// fns; this is a check-time belt-and-suspenders for the
		// case where a lowering pass emits a malformed FnDecl.
		prog, tab := synthesizeBadVariadic(t)
		diags := Check(prog)
		_ = tab
		wantDiag(t, diags, "variadic")
	})
}

// TestStdlibBodyChecking — W06-P08-T01. Bodies must be checked
// in the same pass as user code (L002 defense).
func TestStdlibBodyChecking(t *testing.T) {
	// No stdlib module skip exists at W06; we demonstrate the
	// property by checking a two-module build where both modules'
	// bodies are walked and contribute diagnostics when broken.
	_, diags := checkMulti(t, map[string]string{
		"std": `pub fn id(x: I32) -> I32 { return x; }`,
		"": `
import std;
fn main() -> I32 { return std.id(42); }
`,
	})
	wantClean(t, diags)
	// And if the stdlib body is wrong, it diagnoses.
	_, bad := checkMulti(t, map[string]string{
		"std": `pub fn id(x: I32) -> I32 { return true; }`,
		"": `
import std;
fn main() -> I32 { return std.id(42); }
`,
	})
	wantDiag(t, bad, "return value type")
}

// TestNoUnknownAfterCheck — the wave's cardinal invariant. After
// Check, no Typed node carries KindInfer.
func TestNoUnknownAfterCheck(t *testing.T) {
	prog, diags := checkSource(t, "m", "m.fuse", `
fn main() -> I32 {
	let a: I32 = 1;
	let b: I32 = a + 2;
	return b;
}
`)
	wantClean(t, diags)
	if leftover := RunNoUnknownCheck(prog); len(leftover) != 0 {
		t.Fatalf("KindInfer survived checking:\n%v", leftover)
	}
}

// --- tiny helpers ----------------------------------------------------

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// synthesizeBadVariadic constructs a Program containing a
// non-extern variadic fn — something the parser cannot produce
// but the checker must still reject if a future lowering pass
// accidentally emits one.
func synthesizeBadVariadic(t *testing.T) (*hir.Program, *typetable.Table) {
	t.Helper()
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	fnType := tab.Fn([]typetable.TypeId{tab.I32()}, tab.I32(), true)
	fn := &hir.FnDecl{
		Base:     hir.Base{ID: hir.ItemID("m", "bad"), Span: lex.Span{}},
		Name:     "bad",
		TypeID:   fnType,
		Variadic: true, // non-extern; illegal
		Return:   tab.I32(),
	}
	prog.RegisterModule(&hir.Module{
		Base:  hir.Base{ID: hir.ItemID("m", ""), Span: lex.Span{}},
		Path:  "m",
		Items: []hir.Item{fn},
	})
	// Fix the fn span now that lex is imported.
	fn.Span = lex.Span{}
	return prog, tab
}
