# Wave 11: Error Propagation

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: implement `?` as a real branch-and-early-return on `Result[T, E]`
and `Option[T]`.

Entry criterion: W10 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- `?` on `Result[T, E]` extracts `T` or returns `Err(e)`
- `?` on `Option[T]` extracts `T` or returns `None`
- lowered MIR contains discriminant read, success extraction, error
  early-return
- proof: `run(false)` exits 43; `run(true)` exits 0

Proof of completion:

```
go test ./compiler/check/... -run TestQuestionTypecheck -v
go test ./compiler/lower/... -run TestQuestionBranch -v
go test ./tests/e2e/... -run TestErrorPropagation -v
```

## Phase 00: Stub Audit [W11-P00-STUB-AUDIT]

- Task 01: `?` audit [W11-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W11 -phase P00`

## Phase 01: Type Checking `?` [W11-P01-CHECK]

- Task 01: `?` on Result [W11-P01-T01-RESULT]
  Verify: `go test ./compiler/check/... -run TestQuestionTypecheck -v`
- Task 02: `?` on Option [W11-P01-T02-OPTION]
  Verify: `go test ./compiler/check/... -run TestQuestionOptionTypecheck -v`

## Phase 02: Lowering `?` [W11-P02-LOWER]

- Task 01: Branch-and-early-return [W11-P02-T01-BRANCH]
  Verify: `go test ./compiler/lower/... -run TestQuestionBranch -v`

## Phase 03: Proof Program [W11-P03-PROOF]

- Task 01: `error_propagation.fuse` [W11-P03-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestErrorPropagation -v`

## Wave Closure Phase [W11-PCL-WAVE-CLOSURE]

- Task 01: Retire `?` stubs [W11-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W11`
- Task 02: WC011 entry [W11-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC011" docs/learning-log.md`

