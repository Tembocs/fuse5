package codegen

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// TestDestructorCallEmitted — W09-P03-T03. An OpDrop instruction
// emits a `TypeName_drop(&rN);` call in the generated C.
func TestDestructorCallEmitted(t *testing.T) {
	fn, b := mir.NewFunction("", "main")
	zero := b.ConstInt(0)
	b.Drop("D_drop", zero)
	b.Return(zero)
	mod := &mir.Module{Functions: []*mir.Function{fn}}
	out, err := EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	if !strings.Contains(out, "D_drop(&r") {
		t.Fatalf("emitted C missing destructor call; got:\n%s", out)
	}
}

// TestDropTraitMetadata — W09-P03-T02. The MIR instruction
// carries the drop target name and register so codegen has
// everything it needs without a separate side-table. Same
// emission path as above; the test explicitly inspects the MIR
// Inst fields after emission.
func TestDropTraitMetadata(t *testing.T) {
	fn, b := mir.NewFunction("", "f")
	r := b.ConstInt(7)
	b.Drop("Thing_drop", r)
	b.Return(r)
	blk := fn.Blocks[0]
	var drop mir.Inst
	for _, in := range blk.Insts {
		if in.Op == mir.OpDrop {
			drop = in
			break
		}
	}
	if drop.Op != mir.OpDrop {
		t.Fatalf("OpDrop not emitted")
	}
	if drop.CallName != "Thing_drop" {
		t.Fatalf("drop name = %q, want Thing_drop", drop.CallName)
	}
	if drop.Lhs != r {
		t.Fatalf("drop target register = %d, want %d", drop.Lhs, r)
	}
	if err := fn.Validate(); err != nil {
		t.Fatalf("Validate rejected a valid OpDrop: %v", err)
	}
}
