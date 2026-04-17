package mir

import "testing"

// TestMinimalMir exercises the W05 instruction set: a single block
// that constructs a constant, runs two arithmetic ops on it, and
// returns the result. Validate must accept the result.
func TestMinimalMir(t *testing.T) {
	t.Run("const-then-return", func(t *testing.T) {
		fn, b := NewFunction("m", "main")
		r := b.ConstInt(42)
		b.Return(r)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(fn.Blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(fn.Blocks))
		}
		if fn.Blocks[0].Term != TermReturn {
			t.Fatalf("expected TermReturn, got %s", fn.Blocks[0].Term)
		}
	})

	t.Run("add-and-return", func(t *testing.T) {
		fn, b := NewFunction("m", "main")
		a := b.ConstInt(20)
		c := b.ConstInt(22)
		sum := b.Binary(OpAdd, a, c)
		b.Return(sum)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	})

	t.Run("validate-rejects-undefined-register", func(t *testing.T) {
		fn, b := NewFunction("m", "main")
		a := b.ConstInt(1)
		// Reach into the function and emit an Inst referring to a
		// register number that was never allocated. This simulates
		// a bridge bug that Validate must catch.
		blk := b.CurrentBlock()
		blk.Insts = append(blk.Insts, Inst{Op: OpAdd, Dst: Reg(99), Lhs: a, Rhs: Reg(500)})
		blk.Term = TermReturn
		blk.ReturnReg = Reg(99)
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted use of undefined register 500")
		}
	})

	t.Run("validate-rejects-missing-terminator", func(t *testing.T) {
		fn, _ := NewFunction("m", "main")
		// Default block has TermInvalid and no return reg.
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted block with no terminator")
		}
	})

	t.Run("binary-op-guard", func(t *testing.T) {
		_, b := NewFunction("m", "main")
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("Binary(OpConstInt, ...) should panic (not a binary op)")
			}
		}()
		a := b.ConstInt(1)
		b.Binary(OpConstInt, a, a)
	})

	t.Run("op-string-stable", func(t *testing.T) {
		cases := map[Op]string{
			OpConstInt: "const_int",
			OpAdd:      "add",
			OpSub:      "sub",
			OpMul:      "mul",
			OpDiv:      "div",
			OpMod:      "mod",
		}
		for op, want := range cases {
			if got := op.String(); got != want {
				t.Errorf("Op(%d).String() = %q, want %q", op, got, want)
			}
		}
	})
}
