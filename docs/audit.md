# Wave Completion Audit Checklist

> Status: reference. This document is a plain-language companion to
> [docs/rules.md](rules.md) §6 and the [learning log](learning-log.md)
> entries L001–L015 and WC000+. The normative rules are in those files;
> this checklist is the practical walkthrough an auditor uses to confirm
> a wave-completion claim is real.

## Who this is for

You are an auditing agent (or human reviewer) who has been asked to
confirm that Wave `Wxx` is truly complete. You did **not** write the wave
and you did **not** write the exit criteria. Your job is to be the
outside check that Fuse's predecessors never had — the reason six critical
features (pattern matching, monomorphization, `?`, drop codegen,
closures, channels) were silently broken at the self-hosting gate in the
previous attempt ([L013](learning-log.md)).

You should assume the implementer:

- may have satisfied every exit criterion as literally written while
  missing the spirit,
- may have unit tests that pass but no program that actually runs,
- may have marked stubs as retired in `STUBS.md` without deleting the
  stub behavior from the code,
- may have written a `WCxxx` closure entry that sounds plausible but
  glosses over real gaps.

Your job is not to be polite. Your job is to find the gap before Wave
`Wxx+1` begins.

## Before you start

Read, in this order:

1. The wave's per-wave plan: `docs/implementation/waveXX_<topic>.md`
2. The overall plan at [docs/implementation-plan.md](implementation-plan.md)
3. The current state of [STUBS.md](../STUBS.md)
4. The `WCxxx` closure entry the implementer added to
   [docs/learning-log.md](learning-log.md)
5. `.claude/current-wave.json` — should name `Wxx` and phase `PCL` at
   the moment you audit

Then re-read the "Exit criteria" and "Proof of completion" sections of
`waveXX_<topic>.md`. Every checklist section below refers back to
specific Rules in [rules.md](rules.md) and specific `Lxxx` entries — if
you want to understand **why** a check exists, follow that link.

## Checklist A — Governance and structural hygiene

These are the non-negotiable shape checks. If any of them fail, stop and
report; no later section matters until these are clean.

- [ ] **A1. Stub audit was committed before implementation began.**
      The wave's first phase is always `P00 Stub Audit`. There should
      be a commit touching `STUBS.md` dated **before** any
      implementation commits in this wave. (Rule 6.12.)
- [ ] **A2. The `STUBS.md` Active table matches reality.**
      Every row the wave was scheduled to retire is gone from the
      Active table. Every new stub the wave introduced is in the
      Active table with a diagnostic and a concrete retiring wave of
      the form `Wxx`. (Rules 6.13, 2.5.)
- [ ] **A3. The `STUBS.md` Stub history has a new append-only block
      for `Wxx`.**
      The block lists Added/Retired/Rescheduled. No prior block was
      edited in place. (Rule 6.16.) Run
      `go run tools/checkstubs/main.go -history-current-wave Wxx`.
- [ ] **A4. No stub is overdue.**
      Run `go run tools/checkstubs/main.go -wave Wxx -phase P00`.
      A stub is overdue if its retiring wave < the current wave and
      it is still in the Active table. A stub whose retiring wave equals
      the current wave is not overdue at P00; the wave's PCL retires it.
      (Rule 6.15.)
- [ ] **A5. A `WCxxx` entry exists in `docs/learning-log.md`.**
      It has all five required fields: proof programs added, stubs
      retired, stubs introduced, what was harder than planned, what
      the next wave must know. "What was harder than planned" is not
      blank — it says "nothing" explicitly or lists real friction.
      (Rule 6.14.)
- [ ] **A6. `.claude/current-wave.json` is consistent.**
      Its `current_wave` is `Wxx`, `current_phase` is `PCL`, and
      `updated` is a recent date. Run
      `go run tools/checkgov/main.go -current-wave`.
- [ ] **A7. The full CI suite is green on the closure SHA.**
      Not just the local tests, not just "it passed on my machine."
      Look at CI on the commit that the closure claim points to.
      (Rule 6.11.)

## Checklist B — Behavioral proof (the L013 trap)

This is the section that existed to catch the failure mode from
[L013](learning-log.md): a wave where every structural exit criterion
passes, every unit test passes, but no program using the feature has
ever run end-to-end.

**From W05 onward**, every user-visible feature introduced in `Wxx`
must have at least one proof program in [tests/e2e/](../tests/e2e/).

- [ ] **B1. The wave's exit criteria include a behavioral requirement.**
      Open `waveXX_<topic>.md` and find the Exit criteria section.
      At least one bullet must be of the form "a program that uses
      `<feature>` compiles, runs, and exits N" — not just "HIR nodes
      carry metadata" or "MIR blocks terminate structurally."
      (Rule 6.10, L014.)
- [ ] **B2. A proof program exists in `tests/e2e/` for every
      user-visible feature this wave added.**
      Open [tests/e2e/README.md](../tests/e2e/README.md) and confirm
      each new program is registered with its expected output. The
      `WCxxx` entry's "Proof programs added this wave" list should
      match what's in the directory.
- [ ] **B3. The proof programs actually exercise the feature.**
      Read each `.fuse` source. Would the program still produce the
      expected output if the feature were reverted to a stub? If yes,
      the proof program is not adversarial — it is a ceremonial check
      ([L013](learning-log.md)). Flag it.
- [ ] **B4. The proof programs are green in CI on the closure SHA.**
      Not "the agent ran them locally." CI, on the committed SHA.
      (Rule 6.11.)
- [ ] **B5. No proof program leans on `Unknown` types.**
      If the AST-to-HIR bridge or the type checker is silently
      defaulting unknown types to a usable shape (e.g. everything
      becomes C `int`), integer-only programs will pass while the
      real feature is broken. This is the specific secondary failure
      mode in [L013](learning-log.md). Spot-check by grep-ing the
      checker for `Unknown` fallback branches.

## Checklist C — Silent-stub traps (prior-attempt specific)

Prior attempts reached the self-hosting gate while six features were
silent stubs. For each feature, you are specifically checking that
the code path was implemented rather than skipped. Only apply the
sub-checks relevant to the current wave's scope.

- [ ] **C1. Pattern matching ([L007](learning-log.md)).**
      If `Wxx` is W10 or later: does `match` with ≥ 2 arms produce
      distinct codegen paths? Are HIR patterns structured nodes
      (`LiteralPat`, `BindPat`, `ConstructorPat`, `WildcardPat`), not
      a `PatternDesc string`? Is the MIR a cascading branch chain on
      the discriminant? Red flag: lowerer emits `TermGoto(armBlock)`
      unconditionally.
- [ ] **C2. Monomorphization ([L008](learning-log.md), [L015](learning-log.md)).**
      If `Wxx` is W08 or later: does `compiler/monomorph/` run in the
      driver pipeline between checking and HIR lowering? Does it
      collect concrete instantiations, duplicate bodies with
      substitution, and rewrite call sites to specialized names? Red
      flag: monomorph package exists with `Record`/`Substitute` but
      isn't called from the driver.
- [ ] **C3. Error propagation (`?` operator, [L009](learning-log.md)).**
      If `Wxx` is W11 or later: does `?` on a `Result[T, E]` lower to
      a branch that inspects the discriminant and early-returns on
      `Err`? Red flag: lowerer returns `lowerExpr(n.Expr)` (pass-through)
      or checker returns `Unknown` for the `?` expression type.
- [ ] **C4. Drop codegen ([L010](learning-log.md)).**
      If `Wxx` is W09 or later: does generated C for a type with a
      `Drop` impl emit an actual `TypeName_drop(&_lN);` call? Red
      flag: generated C contains `/* drop _lN */` comments where a
      function call should be.
- [ ] **C5. Closures ([L011](learning-log.md)).**
      If `Wxx` is W12 or later: does a closure `|x| { x + 1 }` lower
      to a lifted function plus an environment struct? Red flag:
      lowerer returns `constUnit()` for closure expressions, or
      liveness skips closure bodies entirely.
- [ ] **C6. Channels and `spawn` ([L012](learning-log.md)).**
      If `Wxx` is W07 or later: does `spawn f(x)` lower to a runtime
      thread-creation call, not a synchronous function call? Does
      the type table have a channel type kind? Red flag: lowerer
      emits `EmitCall(dest, arg, nil, Unit, false)` for `SpawnExpr`.

## Checklist D — Architectural invariants (L001–L006)

These are the carried-forward architecture rules from earlier attempts.
They apply to every wave that touches the relevant subsystem.

- [ ] **D1. Lexical ambiguities are explicit contracts, not intuition
      ([L001](learning-log.md)).**
      If the wave touches the lexer or parser: raw strings, `?.`,
      and struct-literal disambiguation are decided by written rules
      in the language reference, and the golden corpus includes
      adversarial cases — not just representative examples.
- [ ] **D2. Stdlib bodies are checked, not just signatures
      ([L002](learning-log.md)).**
      If any stdlib module reaches lowering in this wave, the
      checker ran on its bodies. Red flag: stdlib signatures used
      but bodies skipped "for speed" — `Unknown` types propagate
      to codegen as C `int`.
- [ ] **D3. Monomorphization rejects partial specializations
      ([L003](learning-log.md)).**
      If the wave produces a specialized function, all function-level
      and impl-level type parameters are concretely substituted.
      "Some inference succeeded" is not enough. A specialization with
      any remaining type variable is a defect.
- [ ] **D4. Pointer categories are distinct
      ([L004](learning-log.md)).**
      If the wave touches codegen: borrow-derived pointers and
      `Ptr[T]` values are not handled by the same logic. Call-site
      adaptation and field-access lowering both respect the distinction.
- [ ] **D5. Unit erasure is total, not opportunistic
      ([L005](learning-log.md)).**
      If the wave touches ABI or backend: once a `Unit` payload is
      erased at one location, it is erased at every location
      participating in the same concrete ABI. Constructors, patterns,
      function pointers, and aggregate layouts all agree.
- [ ] **D6. Divergence is structural, not simulated
      ([L006](learning-log.md)).**
      If the wave touches MIR or codegen: post-panic and post-`return`
      paths do not invent fallback values to satisfy type expectations.
      Divergent blocks terminate; nothing reads from them.

## Checklist E — Cross-cutting features (the L015 trap)

- [ ] **E1. Cross-cutting features are not folded into one phase of
      another wave.**
      If the wave introduces a feature that touches parsing,
      resolution, checking, lowering, and codegen, it should be its
      own wave with multiple phases — not a single phase inside
      somebody else's wave. ([L015](learning-log.md) — generics was
      the canonical failure.)
- [ ] **E2. Each integration point has its own proof program.**
      For cross-cutting features, a single end-to-end test is not
      enough; the plan should require a proof program at each
      pipeline seam (e.g. checker, monomorph, lowerer, codegen).
- [ ] **E3. Exit criteria name pipeline integration explicitly.**
      "The feature works in isolation" is never sufficient for a
      cross-cutting feature. The exit criteria must say the
      feature is reachable from the driver end-to-end.

## Checklist F — Diagnostics quality

- [ ] **F1. Every new diagnostic has a primary span, a one-line
      explanation, and a suggestion where applicable.**
      (Rule 6.17.) If the wave added parse or check errors, pick a
      few at random and confirm they meet all three requirements.
- [ ] **F2. Every active stub emits its declared diagnostic.**
      Pick two or three rows from the `STUBS.md` Active table that
      are in scope for this wave's packages. Write a minimal `.fuse`
      program that would trigger each stub. Confirm the compiler
      emits the exact diagnostic text declared in the table.
      (Rule 6.9.)

## Checklist G — Workarounds and shortcuts

- [ ] **G1. No workarounds were introduced.**
      Grep the diff for `TODO`, `FIXME`, `HACK`, `XXX`, "for now",
      "temporary", and "revisit." Each one that remains must have a
      concrete retiring wave recorded in `STUBS.md` — or it is a
      Rule 4.2 violation.
- [ ] **G2. No silent defaults were added.**
      Grep the diff for `Unknown` fallbacks in the checker, `Unit`
      returns in the lowerer for new expression kinds, and
      `/* ... */` placeholder comments in codegen. Each is a
      potential silent stub. (Rule 6.9, L013.)
- [ ] **G3. Verify commands are portable.**
      Every new `Verify:` command in the wave's plan runs on Linux,
      macOS, and Windows. No bash-specific forms, no hardcoded
      `/tmp/...` paths, no process substitution. (Rule 6 quickstart.)

## Red flags — stop and escalate

If you see any of these, stop the audit and write them up; they are
strong signals that the wave is not actually complete, regardless of
what the exit criteria say:

- The `WCxxx` entry's "What was harder than planned" is blank or one
  vague sentence. Real waves always surface friction.
- Proof programs exist but only exercise integer arithmetic or
  single-arm `match`. The feature under test is never adversarially
  pressed.
- Unit tests in the wave's packages are all green, but running
  `bin/fuse build tests/e2e/<proof>.fuse && ./<binary>` is not part
  of CI.
- Stubs were struck from the Active table without appearing in the
  Stub history's Retired list with a proof program reference.
- The closure commit and the implementation commits are the same
  commit. Closure should be a separate, auditable commit after the
  implementer has already convinced themselves the work is done.
- The implementer and the auditor are the same agent in the same
  session. ([L013](learning-log.md) is the direct warning about
  closed-loop self-verification.)

## Writing up your audit

Produce a short report with:

1. **Wave:** `Wxx` — `<topic>`
2. **Closure SHA:** the commit the implementer named as the closure
   commit.
3. **Checklist results:** one line per section (A–G) — pass, fail,
   or N/A, with a one-sentence reason.
4. **Red flags hit:** list every red flag triggered, with a pointer
   to the file or commit that triggered it.
5. **Verdict:** complete / incomplete / needs-rework.
6. **If incomplete:** the smallest concrete thing the implementer
   must do to re-qualify for audit.

The audit report itself should be committed alongside the wave's
closure artifacts — future auditors need to see what their predecessor
saw and decided.
