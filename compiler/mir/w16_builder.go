package mir

// W16 Builder methods — one per runtime-ABI op.
//
// These methods live in a dedicated file so the W16 surface is easy
// to locate and extend. Each method follows the same contract as
// the W05/W15 builders: emit one Inst, return the destination
// register (or nothing for ops that have no Dst), and preserve the
// sealed-block invariant after terminators.

// Spawn emits an OpSpawn that launches a thread running `entryName`
// with `arg` as its sole argument. Returns the destination register
// that receives the ThreadHandle pointer.
func (b *Builder) Spawn(entryName string, arg Reg) Reg {
	if entryName == "" {
		panic("mir.Builder.Spawn: empty entry name")
	}
	dst := b.NewReg()
	args := []Reg{}
	if arg != NoReg {
		args = append(args, arg)
	}
	b.appendInst(Inst{
		Op: OpSpawn, Dst: dst, CallName: entryName, CallArgs: args,
	})
	return dst
}

// ThreadJoin emits an OpThreadJoin for the ThreadHandle in handleReg.
// Returns the register that receives the thread's int64_t return.
func (b *Builder) ThreadJoin(handleReg Reg) Reg {
	dst := b.NewReg()
	b.appendInst(Inst{
		Op: OpThreadJoin, Dst: dst, Lhs: handleReg,
	})
	return dst
}

// ChanNew emits an OpChanNew with the given capacity and element-
// size registers. Returns the register that receives the opaque
// channel pointer.
func (b *Builder) ChanNew(capacityReg, elemBytesReg Reg) Reg {
	dst := b.NewReg()
	b.appendInst(Inst{
		Op: OpChanNew, Dst: dst, Lhs: capacityReg, Rhs: elemBytesReg,
	})
	return dst
}

// ChanSend emits an OpChanSend on the channel in chanReg using the
// value pointer in valuePtrReg. Returns the status-code register.
func (b *Builder) ChanSend(chanReg, valuePtrReg Reg) Reg {
	dst := b.NewReg()
	b.appendInst(Inst{
		Op: OpChanSend, Dst: dst, Lhs: chanReg, Rhs: valuePtrReg,
	})
	return dst
}

// ChanRecv emits an OpChanRecv on the channel in chanReg that
// writes to outPtrReg. Returns the status-code register.
func (b *Builder) ChanRecv(chanReg, outPtrReg Reg) Reg {
	dst := b.NewReg()
	b.appendInst(Inst{
		Op: OpChanRecv, Dst: dst, Lhs: chanReg, Rhs: outPtrReg,
	})
	return dst
}

// ChanClose emits an OpChanClose on the channel in chanReg. No Dst.
func (b *Builder) ChanClose(chanReg Reg) {
	b.appendInst(Inst{
		Op: OpChanClose, Lhs: chanReg,
	})
}

// Panic emits an OpPanic with the given literal message. No Dst
// (panic never returns to the caller; the block should terminate
// with Unreachable immediately after).
func (b *Builder) Panic(message string) {
	b.appendInst(Inst{
		Op: OpPanic, CallName: message,
	})
}
