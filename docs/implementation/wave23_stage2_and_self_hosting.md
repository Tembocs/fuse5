# Wave 23: Stage 2 and Self-Hosting

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: bring up the self-hosted Fuse compiler and prove it compiles itself
reproducibly.

Entry criterion: W22 done. Phase 00 re-verifies the Active stubs table
is empty.

State on entry: `stage2/src/` is empty. All compiler infrastructure
complete. STUBS.md Active table is empty.

Exit criteria:

- stage1 compiles stage2 end-to-end
- stage2 recompiles itself
- reproducibility checks pass

Proof of completion:

```
fuse build stage2/src/... -o stage2_out/fusec2
stage2_out/fusec2 build stage2/src/... -o stage2_out/fusec2_gen2
make repro
go test ./tests/bootstrap/... -v
```

## Phase 00: Stub Audit [W23-P00-STUB-AUDIT]

- Task 01: Self-hosting audit [W23-P00-T01-ZERO-STUBS]
  DoD: Active stubs table empty. If not empty, the wave cannot begin and
  the missing retirements go back to W22.
  Verify: `go run tools/checkstubs/main.go -wave W23 -phase P00 -require-empty-active`

## Phase 01: Port Stage 2 [W23-P01-PORT]

- Task 01: Port frontend [W23-P01-T01-FRONTEND]
  Verify: `fuse check stage2/src/compiler/lex/...`
- Task 02: Port driver [W23-P01-T02-DRIVER]
  Verify: `fuse build stage2/src/... -o stage2_out/fusec2`

## Phase 02: First Self-Compilation [W23-P02-SELF-COMPILE]

- Task 01: stage1 compiles stage2 [W23-P02-T01-STAGE1-COMPILES-STAGE2]
  Verify: `fuse build stage2/src/... -o stage2_out/fusec2`
- Task 02: stage2 compiles itself [W23-P02-T02-STAGE2-COMPILES-ITSELF]
  Verify: `stage2_out/fusec2 build stage2/src/... -o stage2_out/fusec2_gen2`

## Phase 03: Reproducibility [W23-P03-REPRO]

- Task 01: Bootstrap reproducibility [W23-P03-T01-REPRO]
  Verify: `make repro`
- Task 02: Gate merges on bootstrap health [W23-P03-T02-GATE]
  Verify: `go run tools/checkci/main.go -bootstrap-gate`

## Wave Closure Phase [W23-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W23-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W23`
- Task 02: WC023 entry [W23-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC023" docs/learning-log.md`

