# Wave 25: Retirement of Go and C

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: complete the transition to a Fuse-owned compiler implementation path.

Entry criterion: W24 done.

Exit criteria:

- Fuse owns the compiler implementation path
- Go not required to build the compiler in the active path
- C not required as a backend or runtime dependency in the active path

Proof of completion:

```
go run tools/checkgoc/main.go -retirement
fuse build stage2/src/... -o stage2_out/fusec2_final
stage2_out/fusec2_final --version
```

## Phase 00: Stub Audit [W25-P00-STUB-AUDIT]

- Task 01: Bootstrap dependency audit [W25-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W25 -phase P00`

## Phase 01: Retire Go [W25-P01-RETIRE-GO]

- Task 01: Freeze Stage 1 [W25-P01-T01-FREEZE]
  Verify: `go run tools/checkgoc/main.go -freeze-stage1`
- Task 02: Remove Go from active workflow [W25-P01-T02-REMOVE-GO]
  Verify: `go run tools/checkgoc/main.go -active-path-no-go`

## Phase 02: Retire C [W25-P02-RETIRE-C]

- Task 01: Replace C runtime dependencies [W25-P02-T01-REPLACE-C]
  Verify: `go run tools/checkgoc/main.go -active-path-no-c`
- Task 02: Remove C from bootstrap assumptions [W25-P02-T02-C-FREE]
  Verify: `fuse build --backend=native tests/e2e/hello_exit.fuse -o /tmp/he && /tmp/he`

## Wave Closure Phase [W25-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W25-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W25`
- Task 02: WC025 entry [W25-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC025" docs/learning-log.md`

