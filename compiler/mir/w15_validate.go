package mir

import "fmt"

// validateW15Inst enforces the structural invariants for the W15
// consolidation op set. Called by Function.Validate when the default
// arm sees an op in the W15 numeric range (>= OpCast).
//
// Each op has a fixed shape. Violating the shape is a compiler bug
// (a lowerer or pass producing malformed MIR), not a user error, so
// the diagnostics identify the offending function, block, and inst
// index for direct triage.
func validateW15Inst(f *Function, blk *Block, i int, in Inst, defined map[Reg]bool) error {
	prefix := fmt.Sprintf("mir.Validate: %s/block %d inst %d", f.Name, blk.ID, i)

	switch in.Op {
	case OpCast:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: cast without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: cast uses undefined source register %d", prefix, in.Lhs)
		}
		if CastMode(in.Mode) == CastInvalid {
			return fmt.Errorf("%s: cast has invalid CastMode (0); classify as widen/narrow/reinterpret/etc.", prefix)
		}
		if CastMode(in.Mode) > CastIntToPtr {
			return fmt.Errorf("%s: cast has unknown CastMode %d", prefix, in.Mode)
		}
		defined[in.Dst] = true

	case OpBorrow:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: borrow without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: borrow target register %d is undefined", prefix, in.Lhs)
		}
		defined[in.Dst] = true

	case OpFnPtr:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: fn_ptr without destination", prefix)
		}
		if in.CallName == "" {
			return fmt.Errorf("%s: fn_ptr has empty target name", prefix)
		}
		defined[in.Dst] = true

	case OpCallIndirect:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: call_indirect without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: call_indirect uses undefined fn-ptr register %d", prefix, in.Lhs)
		}
		for j, a := range in.CallArgs {
			if !defined[a] {
				return fmt.Errorf("%s: call_indirect arg %d uses undefined register %d", prefix, j, a)
			}
		}
		defined[in.Dst] = true

	case OpSliceNew:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: slice_new without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: slice_new uses undefined base register %d", prefix, in.Lhs)
		}
		if in.Rhs != NoReg && !defined[in.Rhs] {
			return fmt.Errorf("%s: slice_new uses undefined low register %d", prefix, in.Rhs)
		}
		if in.Extra != NoReg && !defined[in.Extra] {
			return fmt.Errorf("%s: slice_new uses undefined high register %d", prefix, in.Extra)
		}
		defined[in.Dst] = true

	case OpFieldRead:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: field_read without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: field_read base register %d is undefined", prefix, in.Lhs)
		}
		if in.FieldName == "" {
			return fmt.Errorf("%s: field_read with empty field name", prefix)
		}
		defined[in.Dst] = true

	case OpStructNew:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: struct_new without destination", prefix)
		}
		if in.CallName == "" {
			return fmt.Errorf("%s: struct_new with empty type name", prefix)
		}
		for _, fw := range in.Fields {
			if fw.Value != NoReg && !defined[fw.Value] {
				return fmt.Errorf("%s: struct_new field %q uses undefined register %d", prefix, fw.Name, fw.Value)
			}
		}
		defined[in.Dst] = true

	case OpStructCopy:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: struct_copy without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: struct_copy source register %d is undefined", prefix, in.Lhs)
		}
		defined[in.Dst] = true

	case OpFieldWrite:
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: field_write target register %d is undefined", prefix, in.Lhs)
		}
		if in.Rhs == NoReg || !defined[in.Rhs] {
			return fmt.Errorf("%s: field_write value register %d is undefined", prefix, in.Rhs)
		}
		if in.FieldName == "" {
			return fmt.Errorf("%s: field_write with empty field name", prefix)
		}

	case OpMethodCall:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: method_call without destination", prefix)
		}
		if in.CallName == "" {
			return fmt.Errorf("%s: method_call with empty method name", prefix)
		}
		if len(in.CallArgs) == 0 {
			return fmt.Errorf("%s: method_call has no receiver (CallArgs[0] is required)", prefix)
		}
		for j, a := range in.CallArgs {
			if !defined[a] {
				return fmt.Errorf("%s: method_call arg %d uses undefined register %d", prefix, j, a)
			}
		}
		defined[in.Dst] = true

	case OpEqScalar:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: eq_scalar without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: eq_scalar lhs register %d is undefined", prefix, in.Lhs)
		}
		if in.Rhs == NoReg || !defined[in.Rhs] {
			return fmt.Errorf("%s: eq_scalar rhs register %d is undefined", prefix, in.Rhs)
		}
		defined[in.Dst] = true

	case OpEqCall:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: eq_call without destination", prefix)
		}
		if in.CallName == "" {
			return fmt.Errorf("%s: eq_call with empty equality fn name", prefix)
		}
		if len(in.CallArgs) != 2 {
			return fmt.Errorf("%s: eq_call requires exactly two arguments, got %d", prefix, len(in.CallArgs))
		}
		for j, a := range in.CallArgs {
			if !defined[a] {
				return fmt.Errorf("%s: eq_call arg %d uses undefined register %d", prefix, j, a)
			}
		}
		defined[in.Dst] = true

	case OpWrappingAdd, OpWrappingSub, OpWrappingMul,
		OpCheckedAdd, OpCheckedSub, OpCheckedMul,
		OpSaturatingAdd, OpSaturatingSub, OpSaturatingMul:
		if in.Dst == NoReg || in.Lhs == NoReg || in.Rhs == NoReg {
			return fmt.Errorf("%s: %s missing register", prefix, in.Op)
		}
		if !defined[in.Lhs] {
			return fmt.Errorf("%s: %s uses undefined lhs register %d", prefix, in.Op, in.Lhs)
		}
		if !defined[in.Rhs] {
			return fmt.Errorf("%s: %s uses undefined rhs register %d", prefix, in.Op, in.Rhs)
		}
		defined[in.Dst] = true

	default:
		return fmt.Errorf("%s: unknown W15 op %d", prefix, in.Op)
	}
	return nil
}

// CheckNoMoveAfterMove is the W15 structural invariant that rejects
// reads of a register after it has been consumed (dropped or moved).
// Liveness (W09) already prevents the HIR-level violation; this pass
// catches any regression at the MIR boundary before codegen sees it.
//
// Rule: once a register R appears as the Lhs of OpDrop within a
// block, it must not appear as an argument, source, or return
// register in any later instruction of the same block. Cross-block
// move tracking is deferred until a full dataflow pass lands; the
// single-block invariant is sufficient to catch the common bug class
// where an over-eager optimizer emits a use after drop on the same
// straight-line sequence.
func (f *Function) CheckNoMoveAfterMove() error {
	for _, blk := range f.Blocks {
		consumed := map[Reg]int{} // Reg -> inst index where consumed
		for i, in := range blk.Insts {
			if in.Op == OpDrop && in.Lhs != NoReg {
				consumed[in.Lhs] = i
				continue
			}
			// Every inst that reads a register must not read a
			// consumed one.
			check := func(r Reg, label string) error {
				if r == NoReg {
					return nil
				}
				if idx, isConsumed := consumed[r]; isConsumed {
					return fmt.Errorf("mir.CheckNoMoveAfterMove: %s/block %d inst %d: %s reads register %d consumed at inst %d",
						f.Name, blk.ID, i, label, r, idx)
				}
				return nil
			}
			if err := check(in.Lhs, "lhs"); err != nil {
				return err
			}
			if err := check(in.Rhs, "rhs"); err != nil {
				return err
			}
			if err := check(in.Extra, "extra"); err != nil {
				return err
			}
			for j, a := range in.CallArgs {
				if err := check(a, fmt.Sprintf("call-arg-%d", j)); err != nil {
					return err
				}
			}
			for _, fw := range in.Fields {
				if err := check(fw.Value, fmt.Sprintf("field-%s", fw.Name)); err != nil {
					return err
				}
			}
		}
		// Terminator reads.
		switch blk.Term {
		case TermReturn:
			if idx, isConsumed := consumed[blk.ReturnReg]; isConsumed {
				return fmt.Errorf("mir.CheckNoMoveAfterMove: %s/block %d: return reads register %d consumed at inst %d",
					f.Name, blk.ID, blk.ReturnReg, idx)
			}
		case TermIfEq:
			if idx, isConsumed := consumed[blk.BranchReg]; isConsumed {
				return fmt.Errorf("mir.CheckNoMoveAfterMove: %s/block %d: if_eq reads register %d consumed at inst %d",
					f.Name, blk.ID, blk.BranchReg, idx)
			}
		}
	}
	return nil
}
