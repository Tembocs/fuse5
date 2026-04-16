# Wave 12: Closures and Callable Traits

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: capture analysis, environment-struct generation, closure lifting,
automatic `Fn`/`FnMut`/`FnOnce` implementation, call desugaring.

Entry criterion: W11 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- capture analysis scans closure bodies for outer-variable references
- environment struct type generated per closure
- closure body lifted to standalone MIR function
- closure expression emits struct init + function pointer pair
- `Fn`/`FnMut`/`FnOnce` declared as intrinsic traits; stdlib core re-exports
- closures auto-implement the tightest matching trait
- function pointers auto-implement all three
- call desugaring: `f(args)` → `f.call(args)` / `call_mut` / `call_once`
  based on tightest bound

Proof of completion:

```
go test ./compiler/lower/... -run TestCaptureAnalysis -v
go test ./compiler/lower/... -run TestClosureLifting -v
go test ./compiler/lower/... -run TestClosureConstruction -v
go test ./compiler/check/... -run TestCallableAutoImpl -v
go test ./tests/e2e/... -run TestClosureCaptureRuns -v
```

## Phase 00: Stub Audit [W12-P00-STUB-AUDIT]

- Task 01: Closure audit [W12-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W12 -phase P00`

## Phase 01: Capture and Lift [W12-P01-CAPTURE-LIFT]

- Task 01: Capture analysis [W12-P01-T01-CAPTURE]
  Verify: `go test ./compiler/lower/... -run TestCaptureAnalysis -v`
- Task 02: Environment struct + lifted body [W12-P01-T02-LIFT]
  Verify: `go test ./compiler/lower/... -run TestClosureLifting -v`
- Task 03: Closure construction [W12-P01-T03-CONSTRUCT]
  Verify: `go test ./compiler/lower/... -run TestClosureConstruction -v`

## Phase 02: Callable Traits [W12-P02-CALLABLE]

- Task 01: Intrinsic `Fn`/`FnMut`/`FnOnce` [W12-P02-T01-DECLARE]
  Verify: `go test ./compiler/check/... -run TestCallableTraitDeclaration -v`
- Task 02: Auto-impl [W12-P02-T02-AUTO]
  Verify: `go test ./compiler/check/... -run TestCallableAutoImpl -v`
- Task 03: Call desugaring [W12-P02-T03-DESUGAR]
  Verify: `go test ./compiler/lower/... -run TestCallDesugar -v`

## Phase 03: Closure Proof Program [W12-P03-PROOF]

- Task 01: `closure_capture.fuse` [W12-P03-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestClosureCaptureRuns -v`

## Wave Closure Phase [W12-PCL-WAVE-CLOSURE]

- Task 01: Retire closure stubs [W12-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W12`
- Task 02: WC012 entry [W12-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC012" docs/learning-log.md`

