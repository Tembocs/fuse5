# Wave 26: Native Backend Transition

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: remove the bootstrap C11 backend from the compiler implementation
path by standing up a native backend on the same semantic contracts.

Entry criterion: W25 done. Phase 00 re-verifies that no overdue stubs
block native-backend entry.

Exit criteria:

- native backend exists and passes correctness gates
- backend contracts §57.1–§57.10 hold for native as well
- stage2 builds through native path
- every committed e2e proof program passes with `--backend=native`
- native backend emits DWARF debug info: `gdb` / `lldb` can break on a
  Fuse source line, read a local by its Fuse-level name, and walk a
  backtrace of Fuse-level frames — matching the W17-P12 bootstrap
  experience, now through the native path

Proof of completion:

```
fuse build --backend=native stage2/src/... -o stage2_out/fusec2_native
stage2_out/fusec2_native --version
go test ./tests/e2e/... -run TestNativeBackendAllProofs -v
go test ./compiler/codegen/native/... -run TestDwarfEmission -v
go test ./tests/e2e/... -run TestNativeDebugBreakpoint -v
```

## Phase 00: Stub Audit [W26-P00-STUB-AUDIT]

- Task 01: Native backend audit [W26-P00-T01-ZERO]
  DoD: no Active row retires before W26. Native-backend work may inherit
  active W26+ tracker rows, but not overdue residue from earlier waves.
  Verify: `go run tools/checkstubs/main.go -wave W26 -phase P00`

## Phase 01: Native Backend Foundation [W26-P01-FOUNDATION]

- Task 01: Backend interface [W26-P01-T01-INTERFACE]
  Verify: `go build ./compiler/codegen/native/...`
- Task 02: Reuse backend contracts [W26-P01-T02-CONTRACTS]
  Verify: `go test ./compiler/codegen/native/... -run TestBackendContracts -v`

## Phase 02: Native Backend Proof [W26-P02-PROOF]

- Task 01: All e2e proofs through native [W26-P02-T01-ALL-PROOFS]
  Verify: `go test ./tests/e2e/... -run TestNativeBackendAllProofs -v`
- Task 02: stage2 through native [W26-P02-T02-STAGE2-NATIVE]
  Verify: `fuse build --backend=native stage2/src/... && stage2_out/fusec2_native --version`

## Phase 03: DWARF Debug Info [W26-P03-DWARF]

- Task 01: DWARF emission [W26-P03-T01-DWARF-EMIT]
  DoD: the native backend emits DWARF 5 (or the closest supported
  version on each target) for every function, parameter, local, and
  source line. Compile units reference original `.fuse` paths, not
  intermediate artifacts.
  Verify: `go test ./compiler/codegen/native/... -run TestDwarfEmission -v`
- Task 02: Fuse-named local visibility under native [W26-P03-T02-LOCAL-NAMES]
  DoD: local variable names in emitted DWARF preserve their Fuse-level
  identifiers, so `gdb` / `lldb` print them by the name the user wrote.
  Verify: `go test ./compiler/codegen/native/... -run TestDwarfLocalNames -v`
- Task 03: End-to-end native debugger proof [W26-P03-T03-PROOF]
  DoD: `tests/e2e/debug_breakpoint.fuse`, compiled with
  `--backend=native --debug`, runs under a scripted debugger session
  that breaks on a Fuse source line, reads a local, and walks the
  backtrace, matching expected goldens.
  Verify: `go test ./tests/e2e/... -run TestNativeDebugBreakpoint -v`

## Wave Closure Phase [W26-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W26-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W26`
- Task 02: WC026 entry [W26-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC026" docs/learning-log.md`

