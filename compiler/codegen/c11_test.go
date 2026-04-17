package codegen

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// TestMinimalCodegenC covers the codegen contract for the W05 spine:
// literal-only main, arithmetic main, and determinism (same input
// yields byte-identical output).
func TestMinimalCodegenC(t *testing.T) {
	t.Run("literal-main", func(t *testing.T) {
		mod := buildMain(func(b *mir.Builder) {
			r := b.ConstInt(42)
			b.Return(r)
		})
		out, err := EmitC11(mod)
		if err != nil {
			t.Fatalf("EmitC11: %v", err)
		}
		mustContain(t, out, "#include <stdint.h>")
		mustContain(t, out, "int main(void) {")
		mustContain(t, out, "INT64_C(42)")
		mustContain(t, out, "return (int)r")
	})

	t.Run("arithmetic-main", func(t *testing.T) {
		mod := buildMain(func(b *mir.Builder) {
			a := b.ConstInt(20)
			c := b.ConstInt(22)
			sum := b.Binary(mir.OpAdd, a, c)
			b.Return(sum)
		})
		out, err := EmitC11(mod)
		if err != nil {
			t.Fatalf("EmitC11: %v", err)
		}
		mustContain(t, out, "INT64_C(20)")
		mustContain(t, out, "INT64_C(22)")
		mustContain(t, out, "= r1 + r2;") // reg 1 = 20, reg 2 = 22, reg 3 = sum
	})

	t.Run("deterministic", func(t *testing.T) {
		mod := buildMain(func(b *mir.Builder) {
			a := b.ConstInt(2)
			c := b.ConstInt(3)
			prod := b.Binary(mir.OpMul, a, c)
			b.Return(prod)
		})
		first, err := EmitC11(mod)
		if err != nil {
			t.Fatalf("EmitC11: %v", err)
		}
		for i := 0; i < 5; i++ {
			got, err := EmitC11(mod)
			if err != nil {
				t.Fatalf("EmitC11 run %d: %v", i+1, err)
			}
			if got != first {
				t.Fatalf("codegen output differs on run %d", i+1)
			}
		}
	})

	t.Run("rejects-unsupported-op", func(t *testing.T) {
		fn, b := mir.NewFunction("", "main")
		// Inject an unsupported opcode directly into the block.
		a := b.ConstInt(1)
		blk := b.CurrentBlock()
		blk.Insts = append(blk.Insts, mir.Inst{Op: mir.Op(99), Dst: mir.Reg(99), Lhs: a, Rhs: a})
		b.Return(a)
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		_, err := EmitC11(mod)
		if err == nil {
			t.Fatalf("expected error for unsupported op, got none")
		}
	})
}

// buildMain is a small helper that constructs a `main` function with
// the given body.
func buildMain(build func(b *mir.Builder)) *mir.Module {
	fn, b := mir.NewFunction("", "main")
	build(b)
	return &mir.Module{Functions: []*mir.Function{fn}}
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q; got:\n%s", needle, haystack)
	}
}
