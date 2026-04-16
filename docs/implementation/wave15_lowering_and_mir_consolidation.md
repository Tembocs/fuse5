# Wave 15: Lowering and MIR Consolidation

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: consolidate lowering across all features introduced so far. Add
cast lowering, function pointer lowering, slice range indexing, struct
update syntax, and overflow-aware arithmetic lowering. Enforce every
MIR invariant at pass boundaries.

Entry criterion: W14 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- MIR blocks terminate structurally (reference §57.4)
- sealed blocks stay sealed after `return`/`break`/`continue`
- no move-after-move violations
- method calls lower distinctly from field reads
- equality lowers through type-specific semantics for non-scalars
- optional chaining lowers to conditional read
- `as` cast lowers per reference §28.1
- function pointer values lower per reference §29.1
- slice range `arr[a..b]` lowers to slice descriptor (reference §32.1)
- struct update `{ ..base }` lowers with explicit-field precedence
  (reference §45.1)
- overflow-aware method calls (`wrapping_add`, `checked_add`, etc.)
  lower to the right MIR ops (reference §33.1)
- invariant walkers green on every pass boundary
- property tests cover MIR transformations

Proof of completion:

```
go test ./compiler/lower/... -v
go test ./compiler/mir/... -v
go test ./compiler/lower/... -run TestInvariantWalkersPass -v
go test ./compiler/lower/... -run TestCastLowering -v
go test ./compiler/lower/... -run TestFnPointerLowering -v
go test ./compiler/lower/... -run TestSliceRangeLowering -v
go test ./compiler/lower/... -run TestStructUpdateLowering -v
go test ./compiler/lower/... -run TestOverflowArithmeticLowering -v
go test ./tests/property/... -run TestMirTransforms -v
```

## Phase 00: Stub Audit [W15-P00-STUB-AUDIT]

- Task 01: MIR audit [W15-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W15 -phase P00`

## Phase 01: Structural Invariants [W15-P01-INVARIANTS]

- Task 01: Sealed blocks [W15-P01-T01-SEALED]
  Verify: `go test ./compiler/lower/... -run TestSealedBlocks -v`
- Task 02: Structural divergence [W15-P01-T02-DIV]
  Verify: `go test ./compiler/lower/... -run TestStructuralDivergence -v`
- Task 03: No-move-after-move [W15-P01-T03-NO-MOVE-AFTER]
  Verify: `go test ./compiler/lower/... -run TestNoMoveAfterMove -v`

## Phase 02: Operator and Method Lowering [W15-P02-OPS]

- Task 01: Borrow instruction [W15-P02-T01-BORROW]
  Verify: `go test ./compiler/lower/... -run TestBorrowInstr -v`
- Task 02: Method vs field [W15-P02-T02-METHOD-FIELD]
  Verify: `go test ./compiler/lower/... -run TestMethodVsField -v`
- Task 03: Semantic equality [W15-P02-T03-SEMANTIC-EQ]
  Verify: `go test ./compiler/lower/... -run TestSemanticEquality -v`
- Task 04: Optional chaining [W15-P02-T04-OPTIONAL-CHAIN]
  Verify: `go test ./compiler/lower/... -run TestOptionalChainLowering -v`

## Phase 03: Cast, FnPointer, Slice Range, Struct Update [W15-P03-EXPR-FORMS]

- Task 01: Cast lowering [W15-P03-T01-CAST]
  Verify: `go test ./compiler/lower/... -run TestCastLowering -v`
- Task 02: Function pointer lowering [W15-P03-T02-FN-POINTER]
  Verify: `go test ./compiler/lower/... -run TestFnPointerLowering -v`
- Task 03: Slice range indexing [W15-P03-T03-SLICE-RANGE]
  Verify: `go test ./compiler/lower/... -run TestSliceRangeLowering -v`
- Task 04: Struct update syntax [W15-P03-T04-STRUCT-UPDATE]
  Verify: `go test ./compiler/lower/... -run TestStructUpdateLowering -v`

## Phase 04: Overflow-Aware Arithmetic Lowering [W15-P04-OVERFLOW]

- Task 01: `wrapping_*`, `checked_*`, `saturating_*` lower correctly
  [W15-P04-T01-OVERFLOW]
  DoD: each method lowers to distinct MIR op tags that codegen (W17) picks
  up for policy-appropriate emission.
  Verify: `go test ./compiler/lower/... -run TestOverflowArithmeticLowering -v`

## Phase 05: Property Tests [W15-P05-PROPERTY]

- Task 01: MIR transform property tests [W15-P05-T01-PROPERTY]
  Verify: `go test ./tests/property/... -run TestMirTransforms -v`

## Wave Closure Phase [W15-PCL-WAVE-CLOSURE]

- Task 01: Retire lower/mir stubs [W15-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W15`
- Task 02: WC015 entry [W15-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC015" docs/learning-log.md`

