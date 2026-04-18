package lower

// W17 lowering forms. Completes the HIR → MIR bridge for the
// deferred W16 items (spawn closure lifting, ThreadHandle.join,
// Chan method calls) and wires the W15 lowering helpers into
// lowerExpr's default dispatch.
//
// Every helper lives on *lowerer so it shares the HIR Program and
// diagnostic channel with the rest of the lowering pass.

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/mir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// lowerRuntimeMethodCall recognises method calls on runtime-owned
// nominal types (`ThreadHandle[T]`, `Chan[T]`) and emits the
// matching W16 MIR op. Returns ok=false when the receiver's type
// is not one the runtime surfaces — the caller then falls back to
// the generic method-call lowering.
func (l *lowerer) lowerRuntimeMethodCall(modPath string, b *mir.Builder, call *hir.CallExpr, field *hir.FieldExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	if field.Receiver == nil {
		return mir.NoReg, false
	}
	recvTypeID := field.Receiver.TypeOf()
	t := l.prog.Types.Get(recvTypeID)
	if t == nil {
		return mir.NoReg, false
	}
	switch t.Kind {
	case typetable.KindThreadHandle:
		return l.lowerThreadHandleMethod(modPath, b, call, field, params)
	case typetable.KindChannel:
		return l.lowerChannelMethod(modPath, b, call, field, params)
	}
	return mir.NoReg, false
}

// lowerThreadHandleMethod lowers `handle.join()`. Any other method
// on ThreadHandle is diagnosed rather than silently falling back,
// per Rule 6.9.
func (l *lowerer) lowerThreadHandleMethod(modPath string, b *mir.Builder, call *hir.CallExpr, field *hir.FieldExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	if field.Name != "join" {
		l.diagnose(call.NodeSpan(),
			fmt.Sprintf("ThreadHandle has no method %q; the W16 surface supports `join()` only", field.Name),
			"call `.join()` to await the thread's return value")
		return mir.NoReg, false
	}
	if len(call.Args) != 0 {
		l.diagnose(call.NodeSpan(),
			"`ThreadHandle.join()` takes no arguments",
			"drop the argument list")
		return mir.NoReg, false
	}
	recv, ok := l.lowerExpr(modPath, b, field.Receiver, params)
	if !ok {
		return mir.NoReg, false
	}
	return b.ThreadJoin(recv), true
}

// lowerChannelMethod lowers `chan.send(v)`, `chan.recv()`,
// `chan.close()`. Other method names produce a diagnostic.
func (l *lowerer) lowerChannelMethod(modPath string, b *mir.Builder, call *hir.CallExpr, field *hir.FieldExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	recv, ok := l.lowerExpr(modPath, b, field.Receiver, params)
	if !ok {
		return mir.NoReg, false
	}
	switch field.Name {
	case "send":
		if len(call.Args) != 1 {
			l.diagnose(call.NodeSpan(),
				"`Chan.send(value)` takes exactly one argument",
				"pass the value to send")
			return mir.NoReg, false
		}
		valueReg, okVal := l.lowerExpr(modPath, b, call.Args[0], params)
		if !okVal {
			return mir.NoReg, false
		}
		return b.ChanSend(recv, valueReg), true
	case "recv":
		if len(call.Args) != 0 {
			l.diagnose(call.NodeSpan(),
				"`Chan.recv()` takes no arguments",
				"drop the argument list")
			return mir.NoReg, false
		}
		// Allocate a slot register the runtime writes through.
		// The slot's register number doubles as the observed
		// value; the emitted C uses `&r_slot` as the destination
		// buffer.
		slot := b.ConstInt(0)
		_ = b.ChanRecv(recv, slot)
		return slot, true
	case "close":
		if len(call.Args) != 0 {
			l.diagnose(call.NodeSpan(),
				"`Chan.close()` takes no arguments",
				"drop the argument list")
			return mir.NoReg, false
		}
		b.ChanClose(recv)
		// Method returns unit; at W17 we represent unit as a
		// zero const-int so the expression still has a value
		// register downstream code can read.
		return b.ConstInt(0), true
	}
	l.diagnose(call.NodeSpan(),
		fmt.Sprintf("Chan has no method %q; supported: send, recv, close", field.Name),
		"use send(value), recv(), or close()")
	return mir.NoReg, false
}

// lowerSpawn lowers a `spawn closure` expression to an OpSpawn.
//
// Strategy (W17 closure-to-fn-ptr lifting):
//
//  1. Allocate a new top-level fn `__fuse_spawn_<N>` whose
//     parameter list matches the closure's params and whose body
//     is the closure's body.
//  2. Register the lifted fn in the lowerer's module so codegen
//     emits it alongside the original program.
//  3. Emit OpSpawn with the lifted fn's mangled name; the
//     argument (when present) is passed through as the runtime
//     trampoline's void* arg.
//
// At W17 we support no-capture closures — the surface that the
// W16 spawn proof describes. Capturing closures need the full
// W12 environment-struct pipeline which is still deferred.
func (l *lowerer) lowerSpawn(modPath string, b *mir.Builder, spawn *hir.SpawnExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	if spawn.Closure == nil {
		l.diagnose(spawn.NodeSpan(),
			"`spawn` without a closure expression",
			"this is a bridge bug, not a user error")
		return mir.NoReg, false
	}
	clos := spawn.Closure
	// Lift the closure body to a new top-level MIR fn.
	lifted, ok := l.liftSpawnClosure(modPath, clos)
	if !ok {
		return mir.NoReg, false
	}
	l.module.Functions = append(l.module.Functions, lifted)

	// Emit the Spawn op. The trampoline expects
	// `int64_t(*)(void*)`; the lifted fn accepts a single int64
	// parameter (even when the closure is nullary, we pass 0 so
	// the ABI is uniform).
	var arg mir.Reg = mir.NoReg
	if len(clos.Params) > 0 {
		// No-capture closures with params: only lift-time
		// binding, not user args — we supply 0 as the arg
		// placeholder. A user-argument path lands with the
		// capturing-closure work.
		arg = b.ConstInt(0)
	}
	return b.Spawn(lifted.Name, arg), true
}

// liftSpawnClosure produces a top-level mir.Function for the
// closure body. The new fn's name is deterministic:
// `__fuse_spawn_<modPath>_<n>` where n is a per-lowerer counter.
// Parameter count: either zero (for no-arg closures, matching
// fuse_rt_thread_spawn's entry signature) or one (matching the
// int64-argument path the W16 proof uses).
func (l *lowerer) liftSpawnClosure(modPath string, clos *hir.ClosureExpr) (*mir.Function, bool) {
	name := fmt.Sprintf("__fuse_spawn_%s_%d", sanitizeCName(modPath), l.spawnCounter)
	l.spawnCounter++
	fn, b := mir.NewFunction(modPath, name)
	liftedParams := map[string]mir.Reg{}
	// Accept exactly one int64 param so the spawn ABI is uniform;
	// when the closure declares a param, bind it to that register.
	if len(clos.Params) > 0 {
		r := b.Param(0)
		liftedParams[clos.Params[0].Name] = r
	} else {
		// Still declare the slot so the trampoline's signature
		// matches; the unused param keeps the C-level fn type
		// stable across spawn targets.
		_ = b.Param(0)
	}
	// Lower the closure body.
	body := clos.Body
	if body == nil {
		l.diagnose(clos.NodeSpan(),
			"spawned closure has no body",
			"this is a bridge bug, not a user error")
		return nil, false
	}
	if !l.lowerBlockToReturn(modPath, b, body, liftedParams) {
		return nil, false
	}
	if err := fn.Validate(); err != nil {
		l.diagnose(clos.NodeSpan(),
			fmt.Sprintf("lifted spawn closure failed validation: %v", err),
			"W17 closure lifting accepts only no-capture closures with a single-statement body")
		return nil, false
	}
	return fn, true
}

// Use the existing W15 cast helper; W17 swaps the passthrough
// CastExpr lowering to the tagged classifier so codegen emits the
// real cast shape.
func (l *lowerer) lowerCastW17(modPath string, b *mir.Builder, x *hir.CastExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	return l.lowerCastTagged(modPath, b, x, params)
}

// referenced imports guard — keeps the lex import non-optional so
// future helpers that need spans compile without touching the
// import list.
var _ = lex.Span{}
