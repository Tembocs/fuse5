package lower

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// PassInvariants runs every W15 structural invariant across a MIR
// Module and reports the first violation. It is the consolidation
// wave's answer to "invariant walkers green on every pass boundary"
// — callers (driver, tests) invoke it after lowering to confirm the
// MIR is structurally sound before handing off to codegen.
//
// Invariants enforced:
//
//  1. `Function.Validate()` — every Inst is shape-valid, terminators
//     are well-formed, register uses follow defs (W05 + W15 extensions).
//  2. `Function.CheckNoMoveAfterMove()` — no use of a register after
//     OpDrop within the same block (W15 structural guard).
//  3. Blocks ending in `TermUnreachable` have no observable outputs
//     (reference §57.4). This is a subset of (1) but explicit here
//     to document the structural-divergence contract.
//  4. Sealed blocks: every block has exactly one terminator, and
//     Validate's terminator check rejects TermInvalid (covered by
//     (1)).
func PassInvariants(m *mir.Module) error {
	if m == nil {
		return fmt.Errorf("lower.PassInvariants: nil module")
	}
	for _, fn := range m.Functions {
		if err := fn.Validate(); err != nil {
			return err
		}
		if err := fn.CheckNoMoveAfterMove(); err != nil {
			return err
		}
	}
	return nil
}
