# Fuse Phase Model

> Status: normative for the staged Fuse implementation tree.
>
> This document describes the phase model used throughout the
> implementation plan. It is a reference for contributors and agents working
> in the staged documentation tree. The canonical rule set lives in
> [../rules.md](../rules.md) and [../implementation-plan.md](../implementation-plan.md).

## Identifiers

Every unit of work carries a globally unique identifier.

- **Wave**: `Wxx` — a thematic chunk of work with explicit entry and exit
  criteria. Examples: `W01` (Lexer), `W19` (Language Server).
- **Phase**: `Pxx` within a wave — `W06-P03-TRAIT-RESOLUTION`. Phase `00`
  is always the Stub Audit; phase `PCL` is always the Wave Closure.
- **Task**: `Txx` within a phase — `W06-P01-T01-FN-TYPE-REGISTRATION`.
- **Wave closure log entry**: `WCxxx` — the learning-log entry that
  documents wave completion. `WC006` for Wave 06.
- **Learning-log bug entry**: `Lxxx` — bug entries are numbered
  independently of waves.

All numeric components are zero-padded.

## Phase model per wave

Every wave has the same three-part phase shape.

## Wave prologue requirements

Every wave document must open with enough structure that a contributor can tell
what the wave is responsible for without reading source code or guessing.

Each wave prologue must therefore name:

- the wave goal in user-visible terms
- the entry criterion and state on entry
- the exit criteria as concrete deliverables, not umbrella labels
- the proof of completion commands
- the pillar alignment for the wave: memory safety, concurrency safety,
  developer experience

If a wave owns standard-library modules, the prologue must name those public
modules explicitly. It is not acceptable to say "misc hosted libraries" or
"remaining utilities" when the language reference already names the modules.

### Phase 00 — Stub Audit (mandatory, first phase)

Rule 6.12 requires every wave to begin with a stub audit. Phase 00:

1. Enumerates every stub in scope for the wave, naming the file:line,
   current behavior, emitted diagnostic, and the task that retires it.
2. Checks `STUBS.md` for overdue rows (Rule 6.15). A stub is overdue when
  its retiring wave is numerically less than the current wave and it has not
   yet been retired. Any overdue stub blocks wave entry until retired or
   explicitly rescheduled.
3. Commits the audit to `STUBS.md` before any implementation phase
   begins.

A wave that proceeds past Phase 00 without a committed audit is in
violation regardless of the quality of subsequent work.

### Phases P01..Pnn — Implementation phases

Implementation phases contain the actual tasks. Each phase is scoped
narrowly enough to be reviewable. Each task within a phase carries:

- a short goal
- a `Currently:` line describing the starting state
- an `Expected deliveries:` line or equivalent wording that names the public
  behavior, modules, files, or proofs that phase must leave behind
- a DoD (definition of done) clause
- a `Verify:` command that proves the DoD is met
- required regression coverage

A task is not done until its `Verify:` command runs to completion and
emits the declared passing output (Rule 4.6). "It looks correct" is not
equivalent to "Verify passes."

Phase names must expose the semantic or public-surface family being delivered.
Names such as "misc", "remaining work", or "cleanup" are not valid for a
normative implementation phase.

### Phase PCL — Wave Closure (mandatory, last phase)

Rule 6.14 requires every wave to end with a wave closure phase. PCL:

1. Retires every stub scheduled for this wave in `STUBS.md`.
2. Adds any new stubs introduced by this wave, each with a concrete
   retiring wave.
3. Appends a wave history block to `STUBS.md` — append-only, per
   Rule 6.16.
4. Writes a `WCxxx` entry in `docs/learning-log.md` naming proof
   programs added, stubs retired and introduced, what was harder than
   planned, and what the next wave must know.
5. Confirms CI is green on the full suite, not just the wave's local
   tests.

An agent that cannot fill a WCxxx entry honestly has not completed the
wave.

## Wave-level gates

Four waves are *gating* — their entry requires an extra invariant:

- **W24 Stub Clearance Gate**: no overdue stubs and no silent stubs
  before Stage 2 self-hosting begins; remaining Active rows must belong
  to later waves explicitly.
- **W25 Stage 2 and Self-Hosting**: Phase 00 re-runs the Rule 6.15 stub
  audit so self-hosting does not begin with overdue residue.
- **W26 Native Backend Transition**: Phase 00 re-runs the Rule 6.15 stub
  audit so native-backend work does not inherit overdue residue.
- **W27 Performance Gate**: runtime ratios, compile-time budgets,
  code-size ceilings, and memory footprint all green before W28 retires
  Go and C from the active path.

## Evidence of completion

The burden of proof that a task or wave is done is cumulative:

| Level | Evidence |
|---|---|
| Task | `Verify:` command output recorded (PR body, commit message, or task log) |
| User-visible feature | Proof program committed under `tests/e2e/` and green in CI on that SHA |
| Wave | All tasks green + `STUBS.md` updated + `WCxxx` learning-log entry appended + CI green on full suite |
| Project | No overdue or silent stubs before self-hosting; perf gate passes before retiring Go/C |

"The agent ran it locally and it passed" is not evidence at the
user-visible-feature level or above (Rule 6.11).
