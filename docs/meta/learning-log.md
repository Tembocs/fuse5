# Fuse Learning Log

> Status: normative for Fuse.
>
> This file is the append-only project learning log. It is intentionally seeded
> without entries. Every meaningful issue, bug, obstacle, and wave closure is
> fixed or explicitly rescheduled first, then appended here.

## Bug entry format

Every bug entry must use the following structure.

### LNNN — Title

Date: YYYY-MM-DD
Discovered during: Wave / Phase / Task

**Reproducer**:
Minimal case that exposes the problem.

**What was tried first**:
The first failed approach and why it failed.

**Root cause**:
What was actually wrong.

**Spec gap**:
Which part of the language reference was silent, ambiguous, or incomplete.

**Plan gap**:
Which part of the implementation plan failed to schedule or constrain the work.

**Fix**:
What changed.

**Cascading effects**:
What other bugs or design consequences the fix exposed.

**Architectural lesson**:
What invariant or design principle should be carried forward.

**Verification**:
The commands, tests, or fixtures that proved the fix.

## Wave closure entry format

Every wave requires a closure entry before it may be marked complete.
Closure entries use a separate series (`WCxxx`) so they are distinguishable
from bug entries.

### WCxxx — Wave XX Closure

Date: YYYY-MM-DD
Wave: XX — Wave Title

**Proof programs added this wave**:
List each `.fuse` source file committed to `tests/e2e/` along with its
expected output (exit code or stdout). If no proof programs were added,
state why and which wave is responsible for them.

**Stubs retired this wave**:
List each row removed from `STUBS.md`, naming the stub, the task that retired
it, and the `Verify:` command that confirmed its removal.

**Stubs introduced this wave**:
List each row added to `STUBS.md`, naming the stub, the file:line, the
diagnostic it emits, and the wave scheduled to retire it.

**What was harder than planned**:
Honest account of tasks that took longer, required rework, or surfaced
unexpected complexity. If nothing was harder than planned, state so explicitly.

**What the next wave must know**:
State and context the successor contributor or agent needs that is not captured
elsewhere in the foundational documents. Include any latent issues observed but
not fixed, any assumptions carried forward, and any work that was deferred out
of scope.

**Verification**:
The specific commands that prove the wave is complete. These must match the
"Proof of completion" commands in the implementation plan for this wave.

## Entries