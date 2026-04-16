# Wave 10: Pattern Matching

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: structured pattern nodes, exhaustiveness, match dispatch, extended
pattern forms (or, range, `@`).

Entry criterion: W09 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- match with N arms produces at least N-1 conditional branches
- discriminant read exactly once
- payload extraction before entering arm body
- exhaustiveness checking rejects non-exhaustive matches on finite domains
- unreachable arms produce diagnostic
- or-patterns, range patterns, `@`-bindings lower correctly

Proof of completion:

```
go test ./compiler/check/... -run TestExhaustivenessChecking -v
go test ./compiler/lower/... -run TestMatchDispatch -v
go test ./compiler/lower/... -run TestEnumDiscriminantAccess -v
go test ./compiler/lower/... -run TestOrRangePatterns -v
go test ./tests/e2e/... -run TestMatchEnumDispatch -v
```

## Phase 00: Stub Audit [W10-P00-STUB-AUDIT]

- Task 01: Pattern matching audit [W10-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W10 -phase P00`

## Phase 01: Exhaustiveness [W10-P01-EXHAUSTIVENESS]

- Task 01: Exhaustiveness over enums and bools [W10-P01-T01-ENUMS]
  Verify: `go test ./compiler/check/... -run TestExhaustivenessChecking -v`
- Task 02: Unreachable arm detection [W10-P01-T02-UNREACHABLE]
  Verify: `go test ./compiler/check/... -run TestUnreachableArmDetection -v`

## Phase 02: Match Lowering [W10-P02-LOWER]

- Task 01: Cascading branches [W10-P02-T01-CASCADE]
  Verify: `go test ./compiler/lower/... -run TestMatchDispatch -v`
- Task 02: Discriminant access [W10-P02-T02-DISCRIM]
  Verify: `go test ./compiler/lower/... -run TestEnumDiscriminantAccess -v`
- Task 03: Payload extraction [W10-P02-T03-PAYLOAD]
  Verify: `go test ./compiler/lower/... -run TestPayloadExtraction -v`

## Phase 03: Extended Pattern Forms [W10-P03-EXT-PATTERNS]

- Task 01: Or-pattern lowering [W10-P03-T01-OR]
  Verify: `go test ./compiler/lower/... -run TestOrPattern -v`
- Task 02: Range pattern lowering [W10-P03-T02-RANGE]
  Verify: `go test ./compiler/lower/... -run TestRangePattern -v`
- Task 03: `@`-binding [W10-P03-T03-AT]
  Verify: `go test ./compiler/lower/... -run TestAtBinding -v`

## Phase 04: Match Proof Program [W10-P04-PROOF]

- Task 01: `match_enum_dispatch.fuse` [W10-P04-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestMatchEnumDispatch -v`

## Wave Closure Phase [W10-PCL-WAVE-CLOSURE]

- Task 01: Retire pattern matching stubs [W10-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W10`
- Task 02: WC010 entry [W10-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC010" docs/learning-log.md`

