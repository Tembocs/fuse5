# Wave 09: Ownership and Liveness

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: compute ownership and liveness once, enforce no-borrow-in-struct-field,
and emit actual destructor calls for types with `Drop` impls.

Entry criterion: W08 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- ownership metadata complete on HIR
- liveness computed exactly once per function
- borrow lowering uses `InstrBorrow` with precise kind
- no borrows in struct fields (reference §54.1)
- drop metadata flows from checker to codegen
- `InstrDrop` emits `TypeName_drop(&_lN);` for types with `Drop`
- proof program: type with `Drop` impl demonstrates observable cleanup

Proof of completion:

```
go test ./compiler/liveness/... -v
go test ./compiler/liveness/... -run TestSingleLiveness -v
go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v
go test ./compiler/codegen/... -run TestDestructorCallEmitted -v
go test ./tests/e2e/... -run TestDropObservable -v
```

## Phase 00: Stub Audit [W09-P00-STUB-AUDIT]

- Task 01: Liveness audit [W09-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W09 -phase P00`

## Phase 01: Ownership Semantics [W09-P01-OWNERSHIP]

- Task 01: Ownership contexts [W09-P01-T01-CONTEXTS]
  Verify: `go test ./compiler/liveness/... -run TestOwnershipContexts -v`
- Task 02: Borrow rules and no-borrow-in-field [W09-P01-T02-BORROW]
  Verify: `go test ./compiler/liveness/... -run TestBorrowRules -v`

## Phase 02: Single Liveness Computation [W09-P02-LIVENESS]

- Task 01: Live-after data [W09-P02-T01-LIVE-AFTER]
  Verify: `go test ./compiler/liveness/... -run TestLiveAfter -v`
- Task 02: Last-use and destroy-after [W09-P02-T02-LAST-USE]
  Verify: `go test ./compiler/liveness/... -run TestLastUse -v`

## Phase 03: Drop Intent and Codegen [W09-P03-DROP]

- Task 01: Insert drop intent [W09-P03-T01-DROP-INTENT]
  Verify: `go test ./compiler/liveness/... -run TestDropIntent -v`
- Task 02: Drop trait metadata [W09-P03-T02-DROP-METADATA]
  Verify: `go test ./compiler/codegen/... -run TestDropTraitMetadata -v`
- Task 03: Emit destructor calls [W09-P03-T03-EMIT]
  Verify: `go test ./compiler/codegen/... -run TestDestructorCallEmitted -v`

## Phase 04: Control-Flow Destruction [W09-P04-CFLOW-DROP]

- Task 01: Loops, breaks, early returns [W09-P04-T01-CFLOW]
  Verify: `go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v`

## Phase 05: Drop Proof Program [W09-P05-PROOF]

- Task 01: `drop_observable.fuse` [W09-P05-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestDropObservable -v`

## Wave Closure Phase [W09-PCL-WAVE-CLOSURE]

- Task 01: Retire liveness and drop stubs [W09-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W09`
- Task 02: WC009 entry [W09-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC009" docs/learning-log.md`

