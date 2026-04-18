package codegen

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// TestSpawnEmission exercises the OpSpawn → fuse_rt_thread_spawn
// emission. The emitted C must:
//   - include fuse_rt.h (the header where fuse_rt_thread_spawn lives)
//   - cast the entry-fn to int64_t(*)(void*) so the runtime
//     trampoline's signature matches
//   - pass the argument as a void* via intptr_t coercion
func TestSpawnEmission(t *testing.T) {
	fn, b := mir.NewFunction("", "main")
	arg := b.ConstInt(21)
	handle := b.Spawn("fuse_worker", arg)
	result := b.ThreadJoin(handle)
	b.Return(result)
	if err := fn.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	mod := &mir.Module{Functions: []*mir.Function{fn}}
	out, err := EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	mustContain(t, out, `#include "fuse_rt.h"`)
	mustContain(t, out, "fuse_rt_thread_spawn(")
	mustContain(t, out, "(int64_t(*)(void*))&fuse_worker")
	mustContain(t, out, "(void*)(intptr_t)")
}

// TestJoinEmission covers OpThreadJoin → fuse_rt_thread_join.
func TestJoinEmission(t *testing.T) {
	fn, b := mir.NewFunction("", "main")
	arg := b.ConstInt(0)
	handle := b.Spawn("fuse_worker", arg)
	result := b.ThreadJoin(handle)
	b.Return(result)
	if err := fn.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	mod := &mir.Module{Functions: []*mir.Function{fn}}
	out, err := EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	mustContain(t, out, "fuse_rt_thread_join((void*)(intptr_t)r")

	// The join destination feeds main's return; confirm `(int)` main
	// narrowing still happens (preserves the spine contract).
	mustContain(t, out, "return (int)r")
}

// TestChannelOpsEmission covers OpChanNew / OpChanSend / OpChanRecv
// / OpChanClose → fuse_rt_chan_* calls.
func TestChannelOpsEmission(t *testing.T) {
	fn, b := mir.NewFunction("", "main")
	cap := b.ConstInt(4)
	elem := b.ConstInt(8)
	ch := b.ChanNew(cap, elem)
	val := b.ConstInt(42)
	sendRC := b.ChanSend(ch, val)
	recvRC := b.ChanRecv(ch, val)
	b.ChanClose(ch)
	// Return the sum of the two status codes so both paths stay
	// in the MIR. (No other semantics intended.)
	sum := b.Binary(mir.OpAdd, sendRC, recvRC)
	b.Return(sum)
	if err := fn.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	mod := &mir.Module{Functions: []*mir.Function{fn}}
	out, err := EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	mustContain(t, out, "fuse_rt_chan_new(")
	mustContain(t, out, "fuse_rt_chan_send(")
	mustContain(t, out, "fuse_rt_chan_recv(")
	mustContain(t, out, "fuse_rt_chan_close(")
}

// TestPanicEmission confirms OpPanic emits a string-literal fuse_rt_panic call.
func TestPanicEmission(t *testing.T) {
	fn, b := mir.NewFunction("", "main")
	b.Panic("something went wrong")
	b.Unreachable()
	// main still needs a real terminator for the emitter's main-
	// narrowing path; add a second block with a Return so the
	// function has at least one Return terminator. (This matches
	// the shape the lowerer emits after a panic guard.)
	sink := b.BeginBlock()
	b.UseBlock(sink)
	r := b.ConstInt(0)
	b.Return(r)
	if err := fn.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	mod := &mir.Module{Functions: []*mir.Function{fn}}
	out, err := EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	mustContain(t, out, `fuse_rt_panic("something went wrong");`)
}

