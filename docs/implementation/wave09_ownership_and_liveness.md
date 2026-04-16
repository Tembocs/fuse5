# Wave 09: Ownership and Liveness

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: compute ownership and liveness once, enforce the full borrow-rule set
(no-borrow-in-struct-field, return-borrow rule, aliasing exclusion,
use-after-move rejection), and emit actual destructor calls for types with
`Drop` impls.

Entry criterion: W08 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- ownership metadata complete on HIR
- liveness computed exactly once per function
- borrow lowering uses `InstrBorrow` with precise kind
- no borrows in struct fields (reference §54.1)
- borrows returned from a function must point into a borrowed parameter
  (reference §54.6); returning a borrow to a local is a diagnostic
- `mutref` aliasing rejected: no coexisting `mutref` + `ref`, no two `mutref`
  to overlapping memory
- use-after-move rejected: moved locals and moved-from fields may not be
  read again on any path
- drop metadata flows from checker to codegen
- `InstrDrop` emits `TypeName_drop(&_lN);` for types with `Drop`
- proof program: type with `Drop` impl demonstrates observable cleanup
- rejection proofs: borrow-in-field, return-local-borrow, aliased-mutref,
  and use-after-move each have a committed `.fuse` fixture that must fail
  to compile with the declared diagnostic

Proof of completion:

```
go test ./compiler/liveness/... -v
go test ./compiler/liveness/... -run TestSingleLiveness -v
go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v
go test ./compiler/liveness/... -run TestReturnBorrowRule -v
go test ./compiler/liveness/... -run TestMutrefAliasing -v
go test ./compiler/liveness/... -run TestUseAfterMove -v
go test ./compiler/codegen/... -run TestDestructorCallEmitted -v
go test ./tests/e2e/... -run TestDropObservable -v
go test ./tests/e2e/... -run TestBorrowRejections -v
```

## Phase 00: Stub Audit [W09-P00-STUB-AUDIT]

- Task 01: Liveness audit [W09-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W09 -phase P00`

## Phase 01: Ownership Semantics [W09-P01-OWNERSHIP]

- Task 01: Ownership contexts [W09-P01-T01-CONTEXTS]
  Verify: `go test ./compiler/liveness/... -run TestOwnershipContexts -v`
- Task 02: No-borrow-in-struct-field [W09-P01-T02-NO-BORROW-IN-FIELD]
  DoD: struct declarations whose field types contain `ref T` or `mutref T`
  at any nesting depth are rejected (reference §54.1). Rejection includes a
  specific diagnostic naming the offending field and citing §54.
  Verify: `go test ./compiler/liveness/... -run TestNoBorrowInField -v`
- Task 03: Return-borrow rule [W09-P01-T03-RETURN-BORROW]
  DoD: a function returning `ref T` or `mutref T` must be returning a
  borrow that points into a parameter that was itself borrowed (reference
  §54.6). Returning a borrow to a local binding or to a temporary is
  rejected with a specific diagnostic. This rule is enforced structurally
  by the ownership checker, without lifetime variables.
  Verify: `go test ./compiler/liveness/... -run TestReturnBorrowRule -v`
- Task 04: Aliasing exclusion for `mutref` [W09-P01-T04-MUTREF-ALIASING]
  DoD: the checker rejects (a) a `mutref` that coexists with any other
  borrow to the same memory, (b) two `mutref` borrows whose memory
  footprints overlap. Overlap is determined structurally, not heuristically.
  Verify: `go test ./compiler/liveness/... -run TestMutrefAliasing -v`
- Task 05: Use-after-move rejection [W09-P01-T05-USE-AFTER-MOVE]
  DoD: any read of a local after its ownership has been transferred by
  `move`, `owned`, or function-call move semantics is a compile error on
  every control-flow path that could observe the read.
  Verify: `go test ./compiler/liveness/... -run TestUseAfterMove -v`

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

## Phase 05: Drop and Borrow Rejection Proofs [W09-P05-PROOF]

- Task 01: `drop_observable.fuse` [W09-P05-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestDropObservable -v`
- Task 02: Borrow rejection fixtures [W09-P05-T02-BORROW-REJECTIONS]
  DoD: `tests/e2e/` contains four committed `.fuse` fixtures, each of which
  must fail to compile with a specific named diagnostic:
  `reject_borrow_in_field.fuse`, `reject_return_local_borrow.fuse`,
  `reject_aliased_mutref.fuse`, `reject_use_after_move.fuse`. The e2e
  runner confirms each one fails and that the diagnostic text matches the
  golden.
  Verify: `go test ./tests/e2e/... -run TestBorrowRejections -v`

## Wave Closure Phase [W09-PCL-WAVE-CLOSURE]

- Task 01: Retire liveness and drop stubs [W09-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W09`
- Task 02: WC009 entry [W09-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC009" docs/learning-log.md`

