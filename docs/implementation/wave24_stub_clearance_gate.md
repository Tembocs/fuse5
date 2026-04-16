# Wave 24: Stub Clearance Gate

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: single-purpose wave whose sole exit criterion is an empty Active
stubs table. The compiler does not proceed to Stage 2 self-hosting with
any unimplemented feature remaining.

This wave exists specifically because every past attempt at Fuse has
reached self-hosting with silent stubs still in place, and those stubs
later surfaced as missing features (L013, L015). The clearance wave is
the forcing function that guarantees they cannot slip through.

Entry criterion: W23 done. Phase 00 confirms no overdue stubs.

State on entry: the Active stubs table may or may not be empty. Any
remaining entries represent features whose retirement was delayed past
their originally scheduled wave, or features whose scheduled wave
completed but produced a new stub.

Exit criteria:

- Active stubs table in STUBS.md is empty
- every feature documented in the language reference has a `DONE — Wxx`
  status tag (Rule 2.5)
- every reference section has a corresponding e2e proof program or
  checker regression test listed in `tests/e2e/README.md`
- the `tests/e2e/` suite and all unit tests pass on Linux, macOS, Windows
- `tools/checkstubs -require-empty-active` passes
- `tools/checkref -all-done` passes (verifies every reference section is
  tagged DONE)

Proof of completion:

```
go run tools/checkstubs/main.go -require-empty-active
go run tools/checkref/main.go -all-done
go test ./... -v
go test ./tests/e2e/... -v
```

## Phase 00: Stub Audit [W24-P00-STUB-AUDIT]

- Task 01: Enumerate remaining stubs [W24-P00-T01-ENUMERATE]
  DoD: the Phase 00 audit produces a prioritized list of every remaining
  stub with its retirement path. A stub without a clear retirement path
  is escalated to the user.
  Verify: `go run tools/checkstubs/main.go -wave W24 -phase P00 -enumerate`

## Phase 01: Retire Remaining Stubs [W24-P01-RETIRE]

This phase has one task per remaining stub. Stubs are retired in reverse
dependency order: leaf features first, cross-cutting features last. Each
retirement follows the normal pattern: retire the stub in the code, add a
proof program that would fail if the stub were reinstated, update
STUBS.md.

- Task 01..N: one task per stub, each with its own `[W24-P01-Txx-...]`
  identifier.
  DoD: every stub retired with a corresponding proof program committed to
  `tests/e2e/` or a checker regression test. Each retirement records a
  line in the Stub history log.
  Verify: `go run tools/checkstubs/main.go -wave W24 -retired <stub-name>`

## Phase 02: Reference Status Audit [W24-P02-REF-AUDIT]

- Task 01: Verify every reference section is `DONE` [W24-P02-T01-REF]
  DoD: `tools/checkref -all-done` reports every feature section tagged
  `DONE — Wxx` pointing at the wave that retired it.
  Verify: `go run tools/checkref/main.go -all-done`
- Task 02: Verify every feature has a committed proof or regression
  [W24-P02-T02-PROOFS]
  DoD: `tests/e2e/README.md` lists a program or test for every reference
  feature. Orphan references are a CI failure.
  Verify: `go run tools/checkref/main.go -proof-coverage`

## Phase 03: Clean Build Gate [W24-P03-CLEAN-BUILD]

- Task 01: Full test suite passes on all hosts [W24-P03-T01-FULL-TEST]
  Verify: CI green on Linux, macOS, Windows for the full suite
  (unit + e2e + property + bootstrap harness).
- Task 02: Empty Active stubs [W24-P03-T02-EMPTY]
  Verify: `go run tools/checkstubs/main.go -require-empty-active`

## Wave Closure Phase [W24-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W24-PCL-T01-HISTORY]
  DoD: `## W24` block in STUBS.md Stub history lists every stub retired
  this wave, with the proof program that confirmed it.
  Verify: `go run tools/checkstubs/main.go -wave W24`
- Task 02: Write WC024 learning-log entry [W24-PCL-T02-CLOSURE-LOG]
  DoD: WC024 records which stubs remained until this late (and why), what
  the clearance wave surfaced, and the assertion that no stub remains.
  Verify: `grep "WC024" docs/learning-log.md`

