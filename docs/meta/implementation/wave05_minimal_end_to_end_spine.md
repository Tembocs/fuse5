# Wave 05: Minimal End-to-End Spine

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: make a trivial Fuse program compile, link, run, and return a chosen
exit code. Closes the L013 deferred-proof failure mode.

Entry criterion: W04 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/lower/`, `compiler/mir/`, `compiler/codegen/`,
`compiler/cc/`, `compiler/driver/`, `runtime/` are empty stubs.

Exit criteria:

- `tests/e2e/hello_exit.fuse` compiles, links, runs, returns 0 on Linux,
  macOS, Windows
- `tests/e2e/exit_with_value.fuse` returns a chosen nonzero code (e.g. 42)
- spine MIR, codegen, runtime, driver packages alive with minimal impls
- every feature beyond int-returning main either emits a diagnostic or is
  in STUBS.md with a retiring wave
- `tests/e2e/README.md` created with the two proof programs recorded

Proof of completion:

```
go test ./tests/e2e/... -run TestHelloExit -v
go test ./tests/e2e/... -run TestExitWithValue -v
go run tools/checkstubs/main.go -wave W05
```

## Phase 00: Stub Audit [W05-P00-STUB-AUDIT]

- Task 01: Spine audit [W05-P00-T01-SPINE-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W05 -phase P00`

## Phase 01: Minimal MIR [W05-P01-MIR]

- Task 01: Minimal MIR instruction set [W05-P01-T01-MIR-MIN]
  DoD: `const-int`, `return`, `binary-add/sub/mul/div/mod`, basic block
  with terminator. Anything else produces a lowerer diagnostic.
  Verify: `go test ./compiler/mir/... -run TestMinimalMir -v`
- Task 02: Minimal lowerer [W05-P01-T02-LOWER-MIN]
  Verify: `go test ./compiler/lower/... -run TestMinimalLowerIntReturn -v`

## Phase 02: Minimal Codegen [W05-P02-CODEGEN]

- Task 01: Minimal C11 emitter [W05-P02-T01-CODEGEN-MIN]
  Verify: `go test ./compiler/codegen/... -run TestMinimalCodegenC -v`
- Task 02: C compiler detection [W05-P02-T02-CC]
  Verify: `go test ./compiler/cc/... -run TestCCDetection -v`

## Phase 03: Stub Runtime [W05-P03-RUNTIME]

- Task 01: Minimal runtime [W05-P03-T01-RT-MIN]
  DoD: `runtime/include/fuse_rt.h` declares the ABI surface with stubs
  for all functions except process-entry and `fuse_rt_abort`.
  Verify: `go test ./runtime/tests/... -run TestStubRuntime -v`

## Phase 04: Minimal Driver [W05-P04-DRIVER]

- Task 01: `fuse build` for int-returning main [W05-P04-T01-DRIVER-MIN]
  Verify: `go test ./compiler/driver/... -run TestMinimalBuildInvocation -v`
- Task 02: CLI subcommand minimum [W05-P04-T02-CLI-MIN]
  Verify: `go test ./cmd/fuse/... -run TestMinimalCli -v`

## Phase 05: Proof Programs [W05-P05-PROOF]

- Task 01: `hello_exit.fuse` [W05-P05-T01-HELLO]
  Verify: `go test ./tests/e2e/... -run TestHelloExit -v`
- Task 02: `exit_with_value.fuse` [W05-P05-T02-EXIT-VALUE]
  Verify: `go test ./tests/e2e/... -run TestExitWithValue -v`

## Wave Closure Phase [W05-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md stub history [W05-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W05`
- Task 02: WC005 entry [W05-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC005" docs/learning-log.md`

