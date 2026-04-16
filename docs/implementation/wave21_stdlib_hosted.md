# Wave 21: Stdlib Hosted

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: implement hosted stdlib on top of core; preserve the core/hosted
boundary.

Entry criterion: W20 done. Phase 00 confirms no overdue stubs.

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

## Phase 00: Stub Audit [W21-P00-STUB-AUDIT]

- Task 01: Hosted stdlib audit [W21-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W21 -phase P00`

## Phase 01: IO and OS [W21-P01-IO-OS]

- Task 01: IO modules [W21-P01-T01-IO]
  Verify: `fuse build stdlib/full/io/...`
- Task 02: OS modules [W21-P01-T02-OS]
  Verify: `fuse build stdlib/full/os/...`

## Phase 02: Threads, Sync, Channels [W21-P02-CONCURRENCY]

- Task 01: `ThreadHandle` module [W21-P02-T01-THREAD]
  DoD: `ThreadHandle[T]`, `spawn`, `join()`, detach on drop.
  Verify: `fuse build stdlib/full/thread/...`
- Task 02: Sync modules [W21-P02-T02-SYNC]
  DoD: `Mutex`, `RwLock`, `Cond`, `Once`, `Shared`, `@rank` enforcement
  already live in checker.
  Verify: `fuse build stdlib/full/sync/...`
- Task 03: Channels [W21-P02-T03-CHAN]
  Verify: `fuse build stdlib/full/chan/...`

## Wave Closure Phase [W21-PCL-WAVE-CLOSURE]

- Task 01: Retire hosted stubs [W21-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W21`
- Task 02: WC021 entry [W21-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC021" docs/learning-log.md`

