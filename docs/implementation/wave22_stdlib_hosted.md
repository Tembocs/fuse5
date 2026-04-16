# Wave 22: Stdlib Hosted

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: implement hosted stdlib on top of core; preserve the core/hosted
boundary.

Entry criterion: W21 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- IO, fs, os, time, thread (with `ThreadHandle`), sync, channel modules
  ship
- concurrency surface passes threaded tests
- hosted modules do not leak back into core

Proof of completion:

```
fuse build stdlib/full/...
go test ./tests/... -run TestHostedStdlib -v
go test ./tests/... -run TestConcurrency -v
```

## Phase 00: Stub Audit [W22-P00-STUB-AUDIT]

- Task 01: Hosted stdlib audit [W22-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W22 -phase P00`

## Phase 01: IO and OS [W22-P01-IO-OS]

- Task 01: IO modules [W22-P01-T01-IO]
  Verify: `fuse build stdlib/full/io/...`
- Task 02: OS modules [W22-P01-T02-OS]
  Verify: `fuse build stdlib/full/os/...`

## Phase 02: Threads, Sync, Channels [W22-P02-CONCURRENCY]

- Task 01: `ThreadHandle` module [W22-P02-T01-THREAD]
  DoD: `ThreadHandle[T]`, `spawn`, `join()`, detach on drop.
  Verify: `fuse build stdlib/full/thread/...`
- Task 02: Sync modules [W22-P02-T02-SYNC]
  DoD: `Mutex`, `RwLock`, `Cond`, `Once`, `Shared`, `@rank` enforcement
  already live in checker.
  Verify: `fuse build stdlib/full/sync/...`
- Task 03: Channels [W22-P02-T03-CHAN]
  Verify: `fuse build stdlib/full/chan/...`

## Wave Closure Phase [W22-PCL-WAVE-CLOSURE]

- Task 01: Retire hosted stubs [W22-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W22`
- Task 02: WC022 entry [W22-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC022" docs/learning-log.md`

