package mir

import "fmt"

// validateW16Inst is the structural validator for the Wave 16
// runtime-ABI opcodes. Called by Function.Validate when the default
// arm sees an op in the W16 numeric range (>= OpSpawn).
func validateW16Inst(f *Function, blk *Block, i int, in Inst, defined map[Reg]bool) error {
	prefix := fmt.Sprintf("mir.Validate: %s/block %d inst %d", f.Name, blk.ID, i)

	switch in.Op {
	case OpSpawn:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: spawn without destination", prefix)
		}
		if in.CallName == "" {
			return fmt.Errorf("%s: spawn with empty entry-fn name", prefix)
		}
		for j, a := range in.CallArgs {
			if !defined[a] {
				return fmt.Errorf("%s: spawn arg %d uses undefined register %d", prefix, j, a)
			}
		}
		defined[in.Dst] = true

	case OpThreadJoin:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: thread_join without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: thread_join uses undefined handle register %d", prefix, in.Lhs)
		}
		defined[in.Dst] = true

	case OpChanNew:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: chan_new without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: chan_new capacity register %d is undefined", prefix, in.Lhs)
		}
		if in.Rhs == NoReg || !defined[in.Rhs] {
			return fmt.Errorf("%s: chan_new elem_bytes register %d is undefined", prefix, in.Rhs)
		}
		defined[in.Dst] = true

	case OpChanSend:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: chan_send without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: chan_send channel register %d is undefined", prefix, in.Lhs)
		}
		if in.Rhs == NoReg || !defined[in.Rhs] {
			return fmt.Errorf("%s: chan_send value-ptr register %d is undefined", prefix, in.Rhs)
		}
		defined[in.Dst] = true

	case OpChanRecv:
		if in.Dst == NoReg {
			return fmt.Errorf("%s: chan_recv without destination", prefix)
		}
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: chan_recv channel register %d is undefined", prefix, in.Lhs)
		}
		if in.Rhs == NoReg || !defined[in.Rhs] {
			return fmt.Errorf("%s: chan_recv out-ptr register %d is undefined", prefix, in.Rhs)
		}
		defined[in.Dst] = true

	case OpChanClose:
		if in.Lhs == NoReg || !defined[in.Lhs] {
			return fmt.Errorf("%s: chan_close channel register %d is undefined", prefix, in.Lhs)
		}

	case OpPanic:
		if in.CallName == "" {
			return fmt.Errorf("%s: panic with empty message", prefix)
		}

	default:
		return fmt.Errorf("%s: unknown W16 op %d", prefix, in.Op)
	}
	return nil
}
