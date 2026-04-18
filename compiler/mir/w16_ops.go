package mir

// W16 MIR consolidation ops for the runtime-ABI bridge.
//
// These opcodes are the MIR-side surface the Fuse compiler emits to
// invoke the Wave 16 runtime primitives declared in
// runtime/include/fuse_rt.h. The scheme is one MIR op per runtime
// entry point the lowerer needs to reach: OpSpawn, OpThreadJoin,
// OpChanNew, OpChanSend, OpChanRecv, OpChanClose, OpPanic.
//
// The numeric values start at iota+2000 so the W15 gap (1000..)
// stays clean for future mid-wave additions. All eleven W16 ops live
// in this single block so adding another runtime surface (e.g.,
// Mutex-based sync) later means appending to the same block.

const (
	// OpSpawn calls fuse_rt_thread_spawn(entry, arg). CallName is
	// the mangled entry-fn name; CallArgs[0] (when present) is the
	// argument register; the spawned fn sees it as a `void *` cast
	// of a 64-bit value. Dst receives the opaque ThreadHandle
	// pointer. At W16 the entry signature is `int64_t (*)(void*)`.
	OpSpawn Op = iota + 2000

	// OpThreadJoin calls fuse_rt_thread_join(handle). Lhs holds the
	// ThreadHandle pointer; Dst receives the thread's int64_t
	// return value.
	OpThreadJoin

	// OpChanNew calls fuse_rt_chan_new(capacity, elem_bytes). Lhs
	// holds the capacity register; Rhs holds the elem_bytes
	// register. Dst receives the opaque channel pointer.
	OpChanNew

	// OpChanSend calls fuse_rt_chan_send(chan, value_ptr). Lhs
	// holds the channel pointer; Rhs holds a pointer register to
	// the value. Dst receives the int32 status code (0 on success,
	// FUSE_CHAN_CLOSED on closed).
	OpChanSend

	// OpChanRecv calls fuse_rt_chan_recv(chan, out_ptr). Lhs holds
	// the channel pointer; Rhs holds a pointer register to the
	// receive buffer. Dst receives the int32 status code.
	OpChanRecv

	// OpChanClose calls fuse_rt_chan_close(chan). Lhs holds the
	// channel pointer. No Dst.
	OpChanClose

	// OpPanic calls fuse_rt_panic(message). CallName holds a
	// string-literal register reference OR, when non-empty in
	// CallName, a direct string-constant name. For W16 simplicity
	// the message is passed as a string literal embedded into
	// codegen.
	OpPanic
)
