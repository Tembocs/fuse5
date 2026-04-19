# Wave 24: Stub Clearance Gate

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: single-purpose policy wave whose exit criterion is that Stage 2
self-hosting begins with no overdue stubs and no silent stubs. W24 does
not require the Active stubs table to be literally empty while later
waves still own explicit tracker rows.

This wave exists specifically because earlier Fuse work has
reached self-hosting with silent stubs still in place, and those stubs
later surfaced as missing features (L013, L015). The clearance wave is
the forcing function that guarantees they cannot slip through, while
still allowing future-wave tracker rows to remain active until their own
retirement wave.

Entry criterion: W23 done. Phase 00 confirms no overdue stubs.

State on entry: the Active stubs table may or may not be empty. Any
remaining entries must be classified as either overdue residue that W24
must retire or reschedule honestly, or legitimate W25+ tracker rows
whose retirement belongs to a later wave.

Exit criteria:

- no overdue row remains in STUBS.md after W24 closes
- every remaining Active row retires in W25 or later and still declares
  a concrete diagnostic
- every feature section in the language reference carries a valid status
  tag, and every section retired through W24 is marked `DONE — Wxx`
- every `DONE` reference section has a corresponding e2e proof program or
  checker regression test listed in `tests/e2e/README.md`
- the `tests/e2e/` suite and all unit tests pass on Linux, macOS, Windows
- `go run tools/checkstubs/main.go -wave W24 -phase P00` passes
- `go run tools/checkref/main.go` passes
- `go run tools/checkref/main.go -proof-coverage` passes

Proof of completion:

```
go run tools/checkstubs/main.go -wave W24 -phase P00
go run tools/checkref/main.go
go run tools/checkref/main.go -proof-coverage
go test ./... -v
go test ./tests/e2e/... -v
```

## Phase 00: Stub Audit [W24-P00-STUB-AUDIT]

- Task 01: Enumerate remaining stubs [W24-P00-T01-ENUMERATE]
  Currently: W24 can enter with a mix of real late stubs and legitimate
  future-wave tracker rows.
  Expected deliveries: a classified list of every remaining Active row,
  separating overdue residue from W25+ tracker rows.
  DoD: the Phase 00 audit produces a prioritized list of every remaining
  stub with its retirement path. A stub without a clear retirement path
  is escalated to the user. Any row that remains active after W24 is
  explicitly confirmed as W25+ work rather than unscheduled residue.
  Verify: `go run tools/checkstubs/main.go -wave W24 -phase P00 -enumerate`

## Phase 01: Retire Remaining Stubs [W24-P01-RETIRE]

This phase has one task per remaining stub. Stubs are retired in reverse
dependency order: leaf features first, cross-cutting features last. Each
retirement follows the normal pattern: retire the stub in the code, add a
proof program that would fail if the stub were reinstated, update
STUBS.md.

Rows that honestly belong to W25 or later are not force-retired here.
They remain active only if their diagnostic contract is still true and
their retiring wave remains concrete.

- Task 01..N: one task per stub, each with its own `[W24-P01-Txx-...]`
  identifier.
  DoD: every W24-owned stub is retired with a corresponding proof
  program committed to `tests/e2e/` or a checker regression test. Each
  retirement records a line in the Stub history log.
  Verify: `go run tools/checkstubs/main.go -wave W24 -retired <stub-name>`

## Phase 02: Reference Status Audit [W24-P02-REF-AUDIT]

- Task 01: Verify every reference section is status-tagged
  [W24-P02-T01-REF]
  Currently: W24 needs the reference status audit, but future-wave
  sections are not supposed to be `DONE` yet.
  Expected deliveries: a status-tagged language reference with no missing
  or malformed implementation-status lines.
  DoD: `tools/checkref` reports that every numbered feature section has a
  valid Rule 2.5 status tag.
  Verify: `go run tools/checkref/main.go`
- Task 02: Verify every feature has a committed proof or regression
  [W24-P02-T02-PROOFS]
  Currently: proof coverage only applies to sections already marked
  `DONE`, not to future-wave sections that remain specified.
  Expected deliveries: proof-registry coverage for every reference
  section already marked `DONE`.
  DoD: `tests/e2e/README.md` lists a program or test for every `DONE`
  reference feature. Orphan DONE sections are a CI failure.
  Verify: `go run tools/checkref/main.go -proof-coverage`

## Phase 03: Clean Build Gate [W24-P03-CLEAN-BUILD]

- Task 01: Full test suite passes on all hosts [W24-P03-T01-FULL-TEST]
  Verify: CI green on Linux, macOS, Windows for the full suite
  (unit + e2e + property + bootstrap harness).
- Task 02: No overdue or silent stubs remain [W24-P03-T02-POLICY-GATE]
  Currently: literal Active-table emptiness is not the real W24 safety
  property because later-wave tracker rows may still exist.
  Expected deliveries: proof that W24 closes with no overdue rows and no
  undeclared stub behavior.
  DoD: every remaining Active row retires in W25 or later, carries a
  concrete diagnostic, and corresponds to real future-wave work rather
  than unscheduled residue.
  Verify: `go run tools/checkstubs/main.go -wave W24 -phase P00`

## Wave Closure Phase [W24-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W24-PCL-T01-HISTORY]
  DoD: `## W24` block in STUBS.md Stub history lists every stub retired
  this wave, with the proof program that confirmed it.
  Verify: `go run tools/checkstubs/main.go -wave W24`
- Task 02: Write WC024 learning-log entry [W24-PCL-T02-CLOSURE-LOG]
  DoD: WC024 records which stubs remained until this late (and why), what
  the clearance wave surfaced, and which W25+ rows legitimately remained
  active after the gate.
  Verify: `grep "WC024" docs/learning-log.md`

