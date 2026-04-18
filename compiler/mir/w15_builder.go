package mir

import "fmt"

// W15 Builder methods for the consolidation-wave ops. Each method
// emits exactly one Inst, returns the destination register (or nothing
// for lvalue-style ops like OpFieldWrite and TermUnreachable), and
// maintains the Builder's "current block sealed" invariant after
// terminators.

// Cast emits an OpCast instruction that converts src with the given
// mode. Panics on CastInvalid — the caller must classify before
// emitting (the lowerer's cast-helper is the canonical classifier).
func (b *Builder) Cast(src Reg, mode CastMode) Reg {
	if mode == CastInvalid {
		panic("mir.Builder.Cast: CastInvalid — classify before emitting")
	}
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpCast, Dst: dst, Lhs: src, Mode: int(mode),
	})
	return dst
}

// Borrow emits `Dst = &Lhs` (or `&mut Lhs` when mutable is true).
func (b *Builder) Borrow(target Reg, mutable bool) Reg {
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpBorrow, Dst: dst, Lhs: target, Flag: mutable,
	})
	return dst
}

// FnPtr loads the address of the named function into a fresh
// register. The returned register is callable via CallIndirect.
func (b *Builder) FnPtr(callName string) Reg {
	if callName == "" {
		panic("mir.Builder.FnPtr: empty callName")
	}
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpFnPtr, Dst: dst, CallName: callName,
	})
	return dst
}

// CallIndirect invokes the function whose address is held in fnReg,
// passing args and writing the result to a fresh Dst.
func (b *Builder) CallIndirect(fnReg Reg, args []Reg) Reg {
	dst := b.NewReg()
	cloned := make([]Reg, len(args))
	copy(cloned, args)
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpCallIndirect, Dst: dst, Lhs: fnReg, CallArgs: cloned,
	})
	return dst
}

// SliceNew constructs a slice descriptor from `base[low..high]`
// (inclusive when the last arg is true). All three register args
// may be NoReg for open-ended endpoints; the validator permits that.
func (b *Builder) SliceNew(base, low, high Reg, inclusive bool) Reg {
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpSliceNew, Dst: dst, Lhs: base, Rhs: low, Extra: high, Flag: inclusive,
	})
	return dst
}

// FieldRead reads `base.name` into a fresh register.
func (b *Builder) FieldRead(base Reg, name string) Reg {
	if name == "" {
		panic("mir.Builder.FieldRead: empty field name")
	}
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpFieldRead, Dst: dst, Lhs: base, FieldName: name,
	})
	return dst
}

// StructNew constructs a fresh struct of the named type and records
// the field-value pairs for downstream introspection. Field stores
// are emitted as OpFieldWrite instructions — StructNew is only the
// allocation + bookkeeping op.
func (b *Builder) StructNew(typeName string, fields []StructField) Reg {
	if typeName == "" {
		panic("mir.Builder.StructNew: empty type name")
	}
	dst := b.NewReg()
	cloned := make([]StructField, len(fields))
	copy(cloned, fields)
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpStructNew, Dst: dst, CallName: typeName, Fields: cloned,
	})
	return dst
}

// StructCopy copies the base struct value into a fresh register so
// the lowerer can then overwrite selected fields with FieldWrite
// (the `..base` update form, reference §45.1).
func (b *Builder) StructCopy(base Reg) Reg {
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpStructCopy, Dst: dst, Lhs: base,
	})
	return dst
}

// FieldWrite stores value into `target.name`. No Dst.
func (b *Builder) FieldWrite(target Reg, name string, value Reg) {
	if name == "" {
		panic("mir.Builder.FieldWrite: empty field name")
	}
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpFieldWrite, Lhs: target, Rhs: value, FieldName: name,
	})
}

// MethodCall invokes a method whose dispatch form is distinct from
// a plain call. receiver is placed at CallArgs[0]; additional args
// follow. Introduced for reference §9.4 method-vs-field disambig.
func (b *Builder) MethodCall(methodName string, receiver Reg, args []Reg) Reg {
	if methodName == "" {
		panic("mir.Builder.MethodCall: empty method name")
	}
	dst := b.NewReg()
	combined := make([]Reg, 0, len(args)+1)
	combined = append(combined, receiver)
	combined = append(combined, args...)
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpMethodCall, Dst: dst, CallName: methodName, CallArgs: combined,
	})
	return dst
}

// EqScalar computes `Dst = (Lhs == Rhs)` for primitive operands.
func (b *Builder) EqScalar(lhs, rhs Reg) Reg {
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpEqScalar, Dst: dst, Lhs: lhs, Rhs: rhs,
	})
	return dst
}

// EqCall dispatches to the named equality fn (a `PartialEq::eq`
// impl for the operand type) with lhs and rhs as the two args.
func (b *Builder) EqCall(eqName string, lhs, rhs Reg) Reg {
	if eqName == "" {
		panic("mir.Builder.EqCall: empty eqName")
	}
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpEqCall, Dst: dst, CallName: eqName, CallArgs: []Reg{lhs, rhs},
	})
	return dst
}

// OverflowArith emits one of the W15 overflow-policy arithmetic ops.
// op must be one of OpWrappingAdd/Sub/Mul, OpCheckedAdd/Sub/Mul, or
// OpSaturatingAdd/Sub/Mul — any other op panics as a bridge bug.
func (b *Builder) OverflowArith(op Op, lhs, rhs Reg) Reg {
	switch op {
	case OpWrappingAdd, OpWrappingSub, OpWrappingMul,
		OpCheckedAdd, OpCheckedSub, OpCheckedMul,
		OpSaturatingAdd, OpSaturatingSub, OpSaturatingMul:
	default:
		panic(fmt.Sprintf("mir.Builder.OverflowArith: %s is not an overflow-policy op", op))
	}
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: op, Dst: dst, Lhs: lhs, Rhs: rhs,
	})
	return dst
}

// Unreachable terminates the current block with TermUnreachable —
// the structural-divergence signal from reference §57.4. After
// Unreachable the block is sealed and further emits are rejected
// by the "emit after termination" guard (see EmitAfterSeal).
func (b *Builder) Unreachable() {
	b.current.Term = TermUnreachable
	b.current = nil
}

// EmitAfterSeal reports whether the builder's current block has
// already been terminated. Downstream lowerers use this to guard
// against a seal-violating emit (the sealed-block invariant from
// reference §6.7). Returns true when the builder has no active block.
func (b *Builder) EmitAfterSeal() bool { return b.current == nil }
