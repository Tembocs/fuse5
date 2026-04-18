package lower

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// TestInvariantWalkersPass runs PassInvariants across a small hand-
// built Module to confirm the walker accepts a correct MIR and
// rejects a structurally broken one. The wave-doc Verify command
// `go test ./compiler/lower/... -run TestInvariantWalkersPass -v`
// binds to this test.
func TestInvariantWalkersPass(t *testing.T) {
	t.Run("healthy-module-passes", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		r := b.ConstInt(42)
		b.Return(r)
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		if err := PassInvariants(mod); err != nil {
			t.Fatalf("PassInvariants rejected healthy module: %v", err)
		}
	})
	t.Run("nil-module-rejected", func(t *testing.T) {
		if err := PassInvariants(nil); err == nil {
			t.Fatalf("PassInvariants accepted nil module")
		}
	})
	t.Run("broken-terminator-rejected", func(t *testing.T) {
		// A function whose only block has TermInvalid.
		fn := &mir.Function{Name: "broken", NumRegs: 1}
		fn.Blocks = []*mir.Block{{ID: 1}}
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		if err := PassInvariants(mod); err == nil {
			t.Fatalf("PassInvariants accepted a fn whose block has no terminator")
		}
	})
	t.Run("use-after-drop-rejected", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		r := b.ConstInt(1)
		b.Drop("Foo_drop", r)
		// Illegal follow-up use of r.
		s := b.Binary(mir.OpAdd, r, r)
		b.Return(s)
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		if err := PassInvariants(mod); err == nil {
			t.Fatalf("PassInvariants accepted use-after-drop")
		}
	})
	t.Run("unreachable-block-passes", func(t *testing.T) {
		fn, b := mir.NewFunction("m", "f")
		_ = b.ConstInt(0)
		b.Unreachable()
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		if err := PassInvariants(mod); err != nil {
			t.Fatalf("PassInvariants rejected a structurally divergent block: %v", err)
		}
	})
}
