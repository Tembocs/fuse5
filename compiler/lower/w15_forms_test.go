package lower

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/mir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// w15TestLowerer builds a minimal *lowerer + *mir.Builder the
// tests invoke directly. Keeps the test surface independent of
// parse/resolve/bridge while still exercising the production
// lowering helpers (lowerReference, lowerFieldAccess, etc.).
func w15TestLowerer(t *testing.T) (*lowerer, *mir.Function, *mir.Builder, *typetable.Table) {
	t.Helper()
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	m := &hir.Module{Path: ""}
	prog.RegisterModule(m)
	l := &lowerer{prog: prog, module: &mir.Module{}}
	fn, b := mir.NewFunction("", "f")
	return l, fn, b, tab
}

// mkLitInt makes a hir.LiteralExpr of integer kind with the given
// TypeId, so lowerExpr can process it through the LitInt arm.
func mkLitInt(tab *typetable.Table, text string, tid typetable.TypeId) *hir.LiteralExpr {
	return &hir.LiteralExpr{
		TypedBase: hir.TypedBase{Base: hir.Base{Span: lex.Span{}}, Type: tid},
		Kind:      hir.LitInt,
		Text:      text,
	}
}

// mkLitBool makes a hir.LiteralExpr of bool kind.
func mkLitBool(tab *typetable.Table, v bool) *hir.LiteralExpr {
	return &hir.LiteralExpr{
		TypedBase: hir.TypedBase{Base: hir.Base{Span: lex.Span{}}, Type: tab.Bool()},
		Kind:      hir.LitBool,
		Bool:      v,
	}
}

// findInst returns the first inst in fn with the given Op, or nil
// if no such inst exists. Test-local helper.
func findInst(fn *mir.Function, op mir.Op) *mir.Inst {
	for _, blk := range fn.Blocks {
		for i, in := range blk.Insts {
			if in.Op == op {
				return &blk.Insts[i]
			}
		}
	}
	return nil
}

// countInsts returns the number of instructions in fn whose Op
// equals target. Used to assert "exactly one of these was emitted".
func countInsts(fn *mir.Function, target mir.Op) int {
	n := 0
	for _, blk := range fn.Blocks {
		for _, in := range blk.Insts {
			if in.Op == target {
				n++
			}
		}
	}
	return n
}

// TestBorrowInstr exercises reference lowering: `&x` → OpBorrow
// with Flag=false, `&mut x` → OpBorrow with Flag=true. Verify
// the destination register feeds into Validate without error.
//
// Bound by the wave-doc Verify command:
//
//	go test ./compiler/lower/... -run TestBorrowInstr -v
func TestBorrowInstr(t *testing.T) {
	t.Run("shared-ref", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		inner := mkLitInt(tab, "42", tab.I32())
		ref := &hir.ReferenceExpr{
			TypedBase: hir.TypedBase{Type: tab.Ref(tab.I32())},
			Mutable:   false,
			Inner:     inner,
		}
		reg, ok := l.lowerReference("", b, ref, nil)
		if !ok {
			t.Fatalf("lowerReference returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		borrow := findInst(fn, mir.OpBorrow)
		if borrow == nil {
			t.Fatalf("no OpBorrow in function: %+v", fn.Blocks[0].Insts)
		}
		if borrow.Flag {
			t.Fatalf("shared ref produced Flag=true; expected &-not-&mut")
		}
	})
	t.Run("mutable-ref", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		inner := mkLitInt(tab, "42", tab.I32())
		ref := &hir.ReferenceExpr{
			TypedBase: hir.TypedBase{Type: tab.Mutref(tab.I32())},
			Mutable:   true,
			Inner:     inner,
		}
		reg, ok := l.lowerReference("", b, ref, nil)
		if !ok {
			t.Fatalf("lowerReference returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		borrow := findInst(fn, mir.OpBorrow)
		if borrow == nil {
			t.Fatalf("no OpBorrow in function")
		}
		if !borrow.Flag {
			t.Fatalf("mutable ref produced Flag=false; expected &mut")
		}
	})
}

// TestMethodVsField verifies the §9.4 disambiguation rule:
//
//   - `obj.name` used as a value lowers to OpFieldRead.
//   - `obj.name(...)` used in a call position lowers to OpMethodCall.
//
// Both forms share a FieldExpr HIR node; the lowerer's dispatch
// decides which MIR op to emit based on whether the FieldExpr is
// a CallExpr.Callee.
func TestMethodVsField(t *testing.T) {
	t.Run("field-read-emits-field-read-op", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		// A struct type `Counter { value: I32 }` — nominal identity
		// via Symbol 1; Name is "Counter" for method mangling.
		counterType := tab.Nominal(typetable.KindStruct, 1, "", "Counter", nil)
		recv := mkLitInt(tab, "0", counterType)
		field := &hir.FieldExpr{
			TypedBase: hir.TypedBase{Type: tab.I32()},
			Receiver:  recv,
			Name:      "value",
		}
		reg, ok := l.lowerFieldAccess("", b, field, nil)
		if !ok {
			t.Fatalf("lowerFieldAccess returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		read := findInst(fn, mir.OpFieldRead)
		if read == nil {
			t.Fatalf("no OpFieldRead in function: %+v", fn.Blocks[0].Insts)
		}
		if read.FieldName != "value" {
			t.Fatalf("OpFieldRead.FieldName = %q, want %q", read.FieldName, "value")
		}
		// Must NOT emit OpMethodCall for a plain field access.
		if findInst(fn, mir.OpMethodCall) != nil {
			t.Fatalf("field read leaked into OpMethodCall")
		}
	})
	t.Run("method-call-emits-method-call-op", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		counterType := tab.Nominal(typetable.KindStruct, 2, "", "Counter", nil)
		recv := mkLitInt(tab, "0", counterType)
		field := &hir.FieldExpr{
			TypedBase: hir.TypedBase{Type: tab.I32()},
			Receiver:  recv,
			Name:      "get",
		}
		call := &hir.CallExpr{
			TypedBase: hir.TypedBase{Type: tab.I32()},
			Callee:    field,
			Args:      nil,
		}
		reg, ok := l.lowerMethodCall("", b, call, field, nil)
		if !ok {
			t.Fatalf("lowerMethodCall returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		mc := findInst(fn, mir.OpMethodCall)
		if mc == nil {
			t.Fatalf("no OpMethodCall in function")
		}
		if mc.CallName != "Counter__get" {
			t.Fatalf("OpMethodCall.CallName = %q, want %q", mc.CallName, "Counter__get")
		}
		if len(mc.CallArgs) != 1 {
			t.Fatalf("OpMethodCall.CallArgs len = %d, want 1 (receiver)", len(mc.CallArgs))
		}
		if findInst(fn, mir.OpFieldRead) != nil {
			t.Fatalf("method call leaked into OpFieldRead")
		}
	})
}

// TestSemanticEquality exercises reference §5.8 type-aware
// equality dispatch: scalar operands emit OpEqScalar, nominal
// operands emit OpEqCall with a mangled `PartialEq::eq` name.
// Inequality (`!=`) wraps the equality with a 0/1 flip.
func TestSemanticEquality(t *testing.T) {
	t.Run("scalar-equality-emits-eq-scalar", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		a := mkLitInt(tab, "1", tab.I32())
		c := mkLitInt(tab, "2", tab.I32())
		eq := &hir.BinaryExpr{
			TypedBase: hir.TypedBase{Type: tab.Bool()},
			Op:        hir.BinEq,
			Lhs:       a,
			Rhs:       c,
		}
		reg, ok := l.lowerSemanticEquality("", b, eq, nil)
		if !ok {
			t.Fatalf("lowerSemanticEquality returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if findInst(fn, mir.OpEqScalar) == nil {
			t.Fatalf("scalar equality did not emit OpEqScalar")
		}
		if findInst(fn, mir.OpEqCall) != nil {
			t.Fatalf("scalar equality leaked into OpEqCall")
		}
	})
	t.Run("nominal-equality-emits-eq-call", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		// Nominal struct type — no scalar kind.
		pairType := tab.Nominal(typetable.KindStruct, 3, "", "Pair", nil)
		a := mkLitInt(tab, "0", pairType)
		c := mkLitInt(tab, "0", pairType)
		eq := &hir.BinaryExpr{
			TypedBase: hir.TypedBase{Type: tab.Bool()},
			Op:        hir.BinEq,
			Lhs:       a,
			Rhs:       c,
		}
		reg, ok := l.lowerSemanticEquality("", b, eq, nil)
		if !ok {
			t.Fatalf("lowerSemanticEquality returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		eqCall := findInst(fn, mir.OpEqCall)
		if eqCall == nil {
			t.Fatalf("nominal equality did not emit OpEqCall")
		}
		if eqCall.CallName != "Pair__eq" {
			t.Fatalf("OpEqCall.CallName = %q, want %q", eqCall.CallName, "Pair__eq")
		}
	})
	t.Run("ne-inverts-eq", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		a := mkLitInt(tab, "1", tab.I32())
		c := mkLitInt(tab, "2", tab.I32())
		ne := &hir.BinaryExpr{
			TypedBase: hir.TypedBase{Type: tab.Bool()},
			Op:        hir.BinNe,
			Lhs:       a,
			Rhs:       c,
		}
		reg, ok := l.lowerSemanticEquality("", b, ne, nil)
		if !ok {
			t.Fatalf("lowerSemanticEquality(BinNe) returned ok=false")
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if findInst(fn, mir.OpEqScalar) == nil {
			t.Fatalf("BinNe did not first emit OpEqScalar")
		}
		// The inverted result is an OpSub(1, eq) — look for it.
		if countInsts(fn, mir.OpSub) == 0 {
			t.Fatalf("BinNe did not emit an inverting OpSub")
		}
	})
}

// TestOptionalChainLowering verifies that `receiver?.field` emits
// the conditional-read sequence: compute receiver, branch on its
// error/none discriminant, propagate via early return on the err
// arm, read the field on the ok arm. Reference §5.8 explicitly
// forbids lowering `?.` as `?` followed by field access.
func TestOptionalChainLowering(t *testing.T) {
	l, fn, b, tab := w15TestLowerer(t)
	// Build a minimal option-shaped enum (`Option { Some, None }`)
	// and register it on the program so errorVariantIndex can find it.
	optType := tab.Nominal(typetable.KindEnum, 10, "", "Option", nil)
	optEnum := &hir.EnumDecl{
		Base:   hir.Base{Span: lex.Span{}},
		Name:   "Option",
		TypeID: optType,
		Variants: []*hir.Variant{
			{Base: hir.Base{Span: lex.Span{}}, Name: "Some"},
			{Base: hir.Base{Span: lex.Span{}}, Name: "None"},
		},
	}
	l.prog.Modules[""].Items = append(l.prog.Modules[""].Items, optEnum)

	recv := mkLitInt(tab, "0", optType)
	chain := &hir.OptFieldExpr{
		TypedBase: hir.TypedBase{Type: tab.I32()},
		Receiver:  recv,
		Name:      "value",
	}
	reg, ok := l.lowerOptChain("", b, chain, nil)
	if !ok {
		t.Fatalf("lowerOptChain returned ok=false; diags=%v", l.diags)
	}
	// Close the ok-block with a return so Validate passes.
	b.Return(reg)
	if err := fn.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	// Must have at least one TermIfEq (the error-vs-ok branch).
	foundBranch := false
	for _, blk := range fn.Blocks {
		if blk.Term == mir.TermIfEq {
			foundBranch = true
			break
		}
	}
	if !foundBranch {
		t.Fatalf("optional chain did not emit a TermIfEq branch")
	}
	// Must read the field on the ok arm.
	fr := findInst(fn, mir.OpFieldRead)
	if fr == nil {
		t.Fatalf("optional chain did not emit OpFieldRead for the success arm")
	}
	if fr.FieldName != "value" {
		t.Fatalf("OpFieldRead.FieldName = %q, want %q", fr.FieldName, "value")
	}
}
