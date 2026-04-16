# Wave 24: Native Backend Transition

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: remove the bootstrap C11 backend from the compiler implementation
path by standing up a native backend on the same semantic contracts.

Entry criterion: W23 done. Phase 00 re-verifies the Active stubs table
is empty.

Exit criteria:

- native backend exists and passes correctness gates
- backend contracts §57.1–§57.10 hold for native as well
- stage2 builds through native path
- every committed e2e proof program passes with `--backend=native`

Proof of completion:

```
fuse build --backend=native stage2/src/... -o stage2_out/fusec2_native
stage2_out/fusec2_native --version
go test ./tests/e2e/... -run TestNativeBackendAllProofs -v
```

## Phase 00: Stub Audit [W24-P00-STUB-AUDIT]

- Task 01: Native backend audit [W24-P00-T01-ZERO]
  Verify: `go run tools/checkstubs/main.go -wave W24 -phase P00 -require-empty-active`

## Phase 01: Native Backend Foundation [W24-P01-FOUNDATION]

- Task 01: Backend interface [W24-P01-T01-INTERFACE]
  Verify: `go build ./compiler/codegen/native/...`
- Task 02: Reuse backend contracts [W24-P01-T02-CONTRACTS]
  Verify: `go test ./compiler/codegen/native/... -run TestBackendContracts -v`

## Phase 02: Native Backend Proof [W24-P02-PROOF]

- Task 01: All e2e proofs through native [W24-P02-T01-ALL-PROOFS]
  Verify: `go test ./tests/e2e/... -run TestNativeBackendAllProofs -v`
- Task 02: stage2 through native [W24-P02-T02-STAGE2-NATIVE]
  Verify: `fuse build --backend=native stage2/src/... && stage2_out/fusec2_native --version`

## Wave Closure Phase [W24-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W24-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W24`
- Task 02: WC024 entry [W24-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC024" docs/learning-log.md`

