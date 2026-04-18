package lower

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// TestSealedBlocks is the lower-layer half of the W15 sealed-block
// invariant. A block that has been terminated by Return, Jump, IfEq,
// or Unreachable must not accept further instructions. Callers
// detect the seal via `Builder.EmitAfterSeal()` — bypassing the
// seal would either panic on the nil current block or emit dead
// instructions that Validate rejects.
//
// Reference §6.7: "Lowering of `return`, `break`, and `continue`
// must seal the current basic block."
//
// Bound by the wave-doc Verify command:
//
//	go test ./compiler/lower/... -run TestSealedBlocks -v
func TestSealedBlocks(t *testing.T) {
	t.Run("return-seals", func(t *testing.T) {
		_, b := mir.NewFunction("m", "f")
		r := b.ConstInt(1)
		b.Return(r)
		if !b.EmitAfterSeal() {
			t.Fatalf("Return did not seal the current block")
		}
	})
	t.Run("unreachable-seals", func(t *testing.T) {
		_, b := mir.NewFunction("m", "f")
		b.Unreachable()
		if !b.EmitAfterSeal() {
			t.Fatalf("Unreachable did not seal the current block")
		}
	})
	t.Run("jump-then-unseal-via-useblock", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		next := b.BeginBlock()
		b.UseBlock(fn.Blocks[0].ID)
		b.Jump(next)
		if !b.EmitAfterSeal() {
			t.Fatalf("Jump did not seal the first block")
		}
		// UseBlock on the jump target is the only way to re-open.
		b.UseBlock(next)
		if b.EmitAfterSeal() {
			t.Fatalf("UseBlock(next) did not un-seal; current block should be active")
		}
		r := b.ConstInt(7)
		b.Return(r)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate after seal-unseal cycle: %v", err)
		}
	})
}

// TestStructuralDivergence verifies the reference §57.4 rule:
// TermUnreachable blocks have no successors, carry no ReturnReg,
// and must not contribute joined values downstream. The wave-doc
// Verify command binds here.
func TestStructuralDivergence(t *testing.T) {
	t.Run("unreachable-block-valid", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		// Simulate a panic-like call whose control flow never returns.
		_ = b.Call("fuse_rt_panic", nil)
		b.Unreachable()
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		if err := PassInvariants(mod); err != nil {
			t.Fatalf("PassInvariants on divergent function: %v", err)
		}
		if fn.Blocks[0].Term != mir.TermUnreachable {
			t.Fatalf("expected TermUnreachable, got %s", fn.Blocks[0].Term)
		}
	})
	t.Run("unreachable-rejects-value-output", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		r := b.ConstInt(42)
		// Manually set TermUnreachable *with* ReturnReg — the
		// structural-divergence invariant must reject this shape.
		b.CurrentBlock().Term = mir.TermUnreachable
		b.CurrentBlock().ReturnReg = r
		if err := fn.Validate(); err == nil {
			t.Fatalf("Validate accepted an Unreachable block carrying ReturnReg")
		}
	})
}

// TestNoMoveAfterMove is the lower-layer binding for the wave-doc
// Verify command `go test ./compiler/lower/... -run TestNoMoveAfterMove`.
// It exercises the MIR-level invariant walker (CheckNoMoveAfterMove)
// through the PassInvariants entry point, so any future lowerer that
// emits MIR is covered by the same structural gate.
func TestNoMoveAfterMove(t *testing.T) {
	t.Run("drop-then-read-rejected", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		r := b.ConstInt(1)
		b.Drop("Box_drop", r)
		s := b.Binary(mir.OpAdd, r, r) // illegal use after drop
		b.Return(s)
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		if err := PassInvariants(mod); err == nil {
			t.Fatalf("PassInvariants accepted a read of register after drop")
		}
	})
	t.Run("drop-without-follow-up-read-passes", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		r := b.ConstInt(1)
		keep := b.ConstInt(42)
		b.Drop("Box_drop", r)
		b.Return(keep)
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		if err := PassInvariants(mod); err != nil {
			t.Fatalf("PassInvariants rejected a clean drop: %v", err)
		}
	})
}
