package mir

import "testing"

// TestSealedBlocks_MIR is the MIR-layer half of the W15 sealed-block
// invariant (reference §6.7). After a terminator, the builder's
// current block is nil; any emit that needs the current block would
// deref nil. EmitAfterSeal is the observable predicate.
//
// The lower-layer counterpart (TestSealedBlocks) guards the same
// invariant through the lowerer's public surface.
func TestSealedBlocks_MIR(t *testing.T) {
	t.Run("return-seals-block", func(t *testing.T) {
		_, b := NewFunction("m", "f")
		r := b.ConstInt(7)
		b.Return(r)
		if !b.EmitAfterSeal() {
			t.Fatalf("Builder.EmitAfterSeal()=false after Return; block should be sealed")
		}
	})
	t.Run("jump-seals-block", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		next := b.BeginBlock()
		// Re-open the first block to jump out of it.
		b.UseBlock(fn.Blocks[0].ID)
		b.Jump(next)
		if !b.EmitAfterSeal() {
			t.Fatalf("Builder.EmitAfterSeal()=false after Jump")
		}
	})
	t.Run("if_eq-seals-block", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		cond := b.ConstInt(0)
		t1 := b.BeginBlock()
		t2 := b.BeginBlock()
		b.UseBlock(fn.Blocks[0].ID)
		b.IfEq(cond, 0, t1, t2)
		if !b.EmitAfterSeal() {
			t.Fatalf("Builder.EmitAfterSeal()=false after IfEq")
		}
	})
	t.Run("unreachable-seals-block", func(t *testing.T) {
		_, b := NewFunction("m", "f")
		b.Unreachable()
		if !b.EmitAfterSeal() {
			t.Fatalf("Builder.EmitAfterSeal()=false after Unreachable")
		}
	})
}

// TestStructuralDivergence_MIR verifies that a block terminated by
// TermUnreachable validates without a ReturnReg, jump target, or
// branch. Reference §57.4: "a basic block ending in a diverging call
// has no successors".
//
// Rejection case: a TermUnreachable block that carries a ReturnReg
// fails Validate because that would re-imply a successor.
func TestStructuralDivergence_MIR(t *testing.T) {
	t.Run("unreachable-block-validates", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		// Panic-like compute then seal with Unreachable.
		_ = b.ConstInt(0)
		b.Unreachable()
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if fn.Blocks[0].Term != TermUnreachable {
			t.Fatalf("expected TermUnreachable, got %s", fn.Blocks[0].Term)
		}
	})
	t.Run("unreachable-rejects-returnreg", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		r := b.ConstInt(42)
		b.current.Term = TermUnreachable
		b.current.ReturnReg = r // structurally wrong: unreachable must not carry a return value
		b.current = nil
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted an Unreachable block carrying ReturnReg %d", r)
		}
	})
}

// TestNoMoveAfterMove_MIR exercises the W15 structural pass that
// rejects reads of a register after it has been consumed by OpDrop.
// Liveness (W09) already catches the HIR violation; this invariant
// catches the same bug class at the MIR boundary so a rogue
// MIR-level transform cannot reintroduce a use-after-move between
// liveness and codegen.
func TestNoMoveAfterMove_MIR(t *testing.T) {
	t.Run("drop-then-read-rejected", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		r := b.ConstInt(1)
		b.Drop("Foo_drop", r)
		// Illegal second use of r after drop.
		_ = b.Binary(OpAdd, r, r)
		b.Return(r)
		if err := fn.CheckNoMoveAfterMove(); err == nil {
			t.Fatalf("CheckNoMoveAfterMove accepted a use-after-drop sequence")
		}
	})
	t.Run("drop-with-no-subsequent-use-ok", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		r := b.ConstInt(1)
		survivor := b.ConstInt(42)
		b.Drop("Foo_drop", r)
		b.Return(survivor)
		if err := fn.CheckNoMoveAfterMove(); err != nil {
			t.Fatalf("CheckNoMoveAfterMove unexpectedly rejected clean drop sequence: %v", err)
		}
	})
	t.Run("return-after-drop-rejected", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		r := b.ConstInt(1)
		b.Drop("Foo_drop", r)
		b.Return(r)
		if err := fn.CheckNoMoveAfterMove(); err == nil {
			t.Fatalf("CheckNoMoveAfterMove accepted return-after-drop")
		}
	})
}

// TestW15InstValidation exercises validate rejection for every W15
// op's primary failure mode. Keeps a single authoritative test of
// the W15 validator's coverage surface.
func TestW15InstValidation(t *testing.T) {
	t.Run("cast-without-mode-rejected", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		src := b.ConstInt(1)
		b.current.Insts = append(b.current.Insts, Inst{
			Op: OpCast, Dst: Reg(b.fn.NumRegs), Lhs: src, Mode: int(CastInvalid),
		})
		b.fn.NumRegs++
		b.Return(src)
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted OpCast with CastInvalid mode")
		}
	})
	t.Run("cast-widen-accepted", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		src := b.ConstInt(1)
		dst := b.Cast(src, CastWiden)
		b.Return(dst)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate rejected valid OpCast/CastWiden: %v", err)
		}
	})
	t.Run("slice-new-missing-base-rejected", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		// Emit SliceNew with an unallocated base register.
		dst := Reg(1) // invalid — nothing defined yet
		b.current.Insts = append(b.current.Insts, Inst{
			Op: OpSliceNew, Dst: dst, Lhs: Reg(99),
		})
		b.current.Term = TermReturn
		b.current.ReturnReg = dst
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted SliceNew with undefined base register")
		}
	})
	t.Run("field-write-without-name-rejected", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		tgt := b.StructNew("Foo", nil)
		val := b.ConstInt(0)
		b.current.Insts = append(b.current.Insts, Inst{
			Op: OpFieldWrite, Lhs: tgt, Rhs: val, FieldName: "",
		})
		b.Return(tgt)
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted OpFieldWrite with empty field name")
		}
	})
	t.Run("eq-call-wrong-arity-rejected", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		lhs := b.ConstInt(1)
		dst := b.NewReg()
		b.current.Insts = append(b.current.Insts, Inst{
			Op: OpEqCall, Dst: dst, CallName: "PartialEq__eq", CallArgs: []Reg{lhs}, // only one arg
		})
		b.Return(dst)
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted OpEqCall with only one argument")
		}
	})
	t.Run("method-call-needs-receiver", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		dst := b.NewReg()
		b.current.Insts = append(b.current.Insts, Inst{
			Op: OpMethodCall, Dst: dst, CallName: "Foo__bar", CallArgs: nil, // no receiver
		})
		b.Return(dst)
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted OpMethodCall with no receiver")
		}
	})
	t.Run("overflow-add-accepted", func(t *testing.T) {
		fn, b := NewFunction("m", "f")
		a := b.ConstInt(10)
		c := b.ConstInt(20)
		out := b.OverflowArith(OpWrappingAdd, a, c)
		b.Return(out)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate rejected wrapping_add: %v", err)
		}
	})
}
