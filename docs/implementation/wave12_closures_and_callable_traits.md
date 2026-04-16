# Wave 12: Closures and Callable Traits

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: capture analysis, environment-struct generation, closure lifting,
automatic `Fn`/`FnMut`/`FnOnce` implementation, call desugaring.

Entry criterion: W11 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- capture analysis scans closure bodies for outer-variable references
- each captured variable classified as `Copy`, `ref`, `mutref`, or owned,
  per reference §15.7; the default follows ordinary borrow/move rules
- a `move` closure prefix reclassifies every used outer binding as owned
  (reference §15.3)
- environment struct type generated per closure; the struct field types
  match the classified captures
- each closure carries an escape classification (escaping / non-escaping)
  on its HIR node, per reference §15.5, consumed by W09-P01-T06 enforcement
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
go test ./compiler/lower/... -run TestMoveClosurePrefix -v
go test ./compiler/lower/... -run TestEscapeClassification -v
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
  DoD: for every closure, scan the body for references to outer
  bindings and classify each as `Copy`, `ref`, `mutref`, or owned using
  the tightest classification that satisfies the body's uses (reads →
  `ref` or `Copy`; writes → `mutref`; `move x` / `owned x` inside the
  body → owned). Classifications are recorded on the closure HIR node.
  Verify: `go test ./compiler/lower/... -run TestCaptureAnalysis -v`
- Task 02: `move` closure prefix [W12-P01-T02-MOVE-PREFIX]
  DoD: a `move` prefix on a closure overrides per-binding inference and
  reclassifies every used outer binding as owned. The environment
  struct field types and outer-scope ownership effects reflect the
  override (reference §15.3).
  Verify: `go test ./compiler/lower/... -run TestMoveClosurePrefix -v`
- Task 03: Escape classification [W12-P01-T03-ESCAPE-CLASS]
  DoD: each closure's HIR node carries an `escape_class` metadata field
  computed from its environment struct (reference §15.5). A struct
  containing any `ref T` / `mutref T` field → non-escaping; otherwise →
  escaping. This metadata is what W09-P01-T06 consumes at use sites.
  Verify: `go test ./compiler/lower/... -run TestEscapeClassification -v`
- Task 04: Environment struct + lifted body [W12-P01-T04-LIFT]
  Verify: `go test ./compiler/lower/... -run TestClosureLifting -v`
- Task 05: Closure construction [W12-P01-T05-CONSTRUCT]
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

