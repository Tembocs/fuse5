# Wave 16: Runtime ABI

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: replace the W05 stub runtime with a full implementation. Wire
`spawn`, channel ops, panic, `ThreadHandle.join()`, and all IO/process/
time/thread/sync surface to real runtime calls.

Entry criterion: W15 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- runtime provides memory, panic, IO, process, time, thread, sync, channel
  surface
- runtime tests pass on Linux, macOS, Windows
- `spawn` lowers to `fuse_rt_thread_spawn`
- `handle.join()` lowers to `fuse_rt_thread_join` returning
  `Result[T, ThreadError]`
- channel ops lower to `fuse_rt_chan_send/recv/close`
- panic lowers to `fuse_rt_panic`
- proof programs: spawn produces observable effect; channel round-trip
  works

Proof of completion:

```
make runtime
cd runtime && make test
go test ./compiler/codegen/... -run TestSpawnEmission -v
go test ./compiler/codegen/... -run TestChannelOpsEmission -v
go test ./compiler/codegen/... -run TestJoinEmission -v
go test ./tests/e2e/... -run TestSpawnObservable -v
go test ./tests/e2e/... -run TestChannelRoundTrip -v
```

## Phase 00: Stub Audit [W16-P00-STUB-AUDIT]

- Task 01: Runtime audit [W16-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W16 -phase P00`

## Phase 01: Memory and Panic [W16-P01-MEM-PANIC]

- Task 01: Runtime header [W16-P01-T01-HEADER]
  Verify: `go run tools/checkruntime/main.go -header-syntax`
- Task 02: Memory and panic impl [W16-P01-T02-MEM-PANIC]
  Verify: `cd runtime && make test`

## Phase 02: IO, Process, Time [W16-P02-IO-PROC-TIME]

- Task 01: IO [W16-P02-T01-IO]
  Verify: `cd runtime && ./tests/test_io`
- Task 02: Process and time [W16-P02-T02-PROC-TIME]
  Verify: `cd runtime && ./tests/test_process`

## Phase 03: Threads and Sync [W16-P03-THREAD-SYNC]

- Task 01: Thread spawn and TLS [W16-P03-T01-SPAWN]
  Verify: `cd runtime && ./tests/test_thread`
- Task 02: Sync primitives [W16-P03-T02-SYNC]
  Verify: `cd runtime && ./tests/test_sync`

## Phase 04: Compiler-Runtime Bridging [W16-P04-BRIDGE]

- Task 01: Spawn lowering [W16-P04-T01-SPAWN-LOWER]
  Verify: `go test ./compiler/codegen/... -run TestSpawnEmission -v`
- Task 02: Join lowering [W16-P04-T02-JOIN]
  Verify: `go test ./compiler/codegen/... -run TestJoinEmission -v`
- Task 03: Channel op lowering [W16-P04-T03-CHAN-LOWER]
  Verify: `go test ./compiler/codegen/... -run TestChannelOpsEmission -v`

## Phase 05: Concurrency Proof Programs [W16-P05-PROOF]

- Task 01: `spawn_observable.fuse` [W16-P05-T01-SPAWN-PROOF]
  Verify: `go test ./tests/e2e/... -run TestSpawnObservable -v`
- Task 02: `channel_round_trip.fuse` [W16-P05-T02-CHAN-PROOF]
  Verify: `go test ./tests/e2e/... -run TestChannelRoundTrip -v`

## Wave Closure Phase [W16-PCL-WAVE-CLOSURE]

- Task 01: Retire runtime stubs [W16-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W16`
- Task 02: WC016 entry [W16-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC016" docs/learning-log.md`

