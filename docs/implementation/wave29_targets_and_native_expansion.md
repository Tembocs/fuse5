# Wave 29: Targets and Native Expansion

> Part of the [Fuse implementation plan](../implementation-plan.md).

Goal: resume broader target and library work on top of the native
self-hosted compiler. Ecosystem documentation moves to its own wave
(W30).

Entry criterion: W28 done.

Exit criteria:

- target expansion proceeds without reintroducing bootstrap debt
- `stdlib/ext/` hosts optional libraries
- target matrix documented; each supported target is exercised by CI

Proof of completion:

```
go run tools/checktargets/main.go
go run tools/checkci/main.go -target-matrix
fuse build stdlib/ext/...
```

## Phase 00: Stub Audit [W29-P00-STUB-AUDIT]

- Task 01: Target audit [W29-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W29 -phase P00`

## Phase 01: Additional Targets [W29-P01-TARGETS]

- Task 01: Target descriptions [W29-P01-T01-DESCRIPTIONS]
  Verify: `go run tools/checktargets/main.go`
- Task 02: Target CI [W29-P01-T02-TARGET-CI]
  Verify: `go run tools/checkci/main.go -target-matrix`

## Phase 02: Extended Libraries [W29-P02-EXT]

- Task 01: Implement ext stdlib [W29-P02-T01-EXT-STDLIB]
  Verify: `fuse build stdlib/ext/...`

## Wave Closure Phase [W29-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W29-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W29`
- Task 02: WC029 entry [W29-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC029" docs/learning-log.md`
