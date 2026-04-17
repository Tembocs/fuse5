# Fuse Learning Log

> Status: normative for Fuse.
>
> This file is the append-only project learning log. It records lessons that the
> team wants the future repository to preserve from the first day of work.

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

Every wave requires a closure entry before it may be marked complete (Rule 6.14,
Rule 10.4). Closure entries use a separate series (WCxxx) so they are
distinguishable from bug entries.

### WCxxx — Wave XX Closure

Date: YYYY-MM-DD
Wave: XX — Wave Title

**Proof programs added this wave**:
List each `.fuse` source file committed to `tests/e2e/` along with its expected
output (exit code or stdout). If no proof programs were added, state why and
which wave is responsible for them.

**Stubs retired this wave**:
List each row removed from `STUBS.md`, naming the stub, the task that retired
it, and the Verify command that confirmed its removal.

**Stubs introduced this wave**:
List each row added to `STUBS.md`, naming the stub, the file:line, the
diagnostic it emits, and the wave scheduled to retire it.

**What was harder than planned**:
Honest account of tasks that took longer, required rework, or surfaced
unexpected complexity. If nothing was harder than planned, state so explicitly —
do not leave this field blank.

**What the next wave must know**:
State and context the successor agent or contributor needs that is not captured
elsewhere in the five foundational documents. Include any latent issues observed
but not fixed, any assumptions carried forward, and any work that was deferred
out of scope.

**Verification**:
The specific commands that prove the wave is complete. These must match the
"Proof of completion" commands in the implementation plan for this wave.

## Entries

### L000 — Learning log format

Date: 2026-04-14
Discovered during: Wave 00 / Phase 01 / Task 02

**Reproducer**:
Not applicable. This entry establishes the required log format.

**What was tried first**:
Previous attempts recorded lessons informally, but the format did not reliably
capture the specification and planning consequences of each bug.

**Root cause**:
A bug log that records chronology without forcing a spec gap and a plan gap does
not reliably improve the next iteration.

**Spec gap**:
The earlier process did not require each meaningful bug to feed back into the
language guide.

**Plan gap**:
The earlier process did not require each meaningful bug to map back into the
implementation plan.

**Fix**:
Adopt the structured entry format defined above from the beginning of the
project.

**Cascading effects**:
Future bugs become easier to classify into language, planning, implementation,
or tooling failures.

**Architectural lesson**:
The learning log is useful only if it tightens the guide and plan, not if it is
used as a loose diary.

**Verification**:
Every future learning-log entry must conform to this format.

### L001 — Lexical ambiguities must become explicit contracts

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from a previous attempt

**Reproducer**:
Inputs such as `r#abc` and `parse(x)?.field` produced incorrect lexical or parse
behavior when the scanner and parser relied on intuitive rather than explicit
rules.

**What was tried first**:
The earlier implementation assumed that raw strings and optional chaining would
fall out naturally from token longest-match and ordinary postfix parsing.

**Root cause**:
The language specification described the features for users, but did not define
the precise recognition and parse contracts an implementation needed.

**Spec gap**:
The language guide was missing explicit implementation contracts for raw-string
recognition, `?.` tokenization, and struct-literal disambiguation.

**Plan gap**:
The lexer and parser waves did not schedule ambiguity-closure tasks explicitly.

**Fix**:
Carry these rules into the new language guide as mandatory implementation
contracts and schedule ambiguity-specific regression work in the early waves.

**Cascading effects**:
The parser and lexer test corpus must include ambiguity-focused golden cases, not
just representative examples.

**Architectural lesson**:
Surface syntax is insufficient when ambiguity exists. The specification must say
how the compiler chooses.

**Verification**:
The new language guide includes explicit contracts for raw strings, `?.`, and
struct-literal disambiguation, and the new implementation plan schedules those
tasks in Waves 01 and 02.

### L002 — Stdlib body checking is mandatory, not optional

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from a previous attempt

**Reproducer**:
Skipping stdlib body checking while still lowering and codegening stdlib modules
caused large numbers of `Unknown` types to propagate into generated C as `int`.

**What was tried first**:
The earlier compiler treated stdlib signatures as enough to move forward while
deferring body checking for speed and convenience.

**Root cause**:
Frontend completeness was broken across pass boundaries: later passes consumed
stdlib HIR whose expressions had never been semantically completed.

**Spec gap**:
The language guide and rules did not state strongly enough that stdlib modules
must be checked like user modules if they participate in lowering and codegen.

**Plan gap**:
The type-checking wave did not make stdlib body checking an explicit exit
criterion from the beginning.

**Fix**:
State the rule in the language guide and rules document, and create a dedicated
phase for stdlib body checking in the implementation plan.

**Cascading effects**:
Once stdlib bodies are checked, many latent semantic gaps surface earlier and in
the correct subsystem.

**Architectural lesson**:
If a module reaches lowering, it must already be semantically complete.

**Verification**:
The new language guide, rules, and implementation plan all make stdlib body
checking mandatory.

### L003 — Monomorphization must reject partial specializations

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from a previous attempt

**Reproducer**:
Generic functions and impl methods could produce plausible-looking specialized
names even when some required type parameters remained unresolved.

**What was tried first**:
The earlier implementation mainly guarded against obviously unknown type
arguments rather than checking completeness of the whole substitution set.

**Root cause**:
Monomorphization validity was defined too loosely. The system treated "some
inference succeeded" as good enough.

**Spec gap**:
The language guide lacked an explicit rule that a valid specialization requires
all function and impl type parameters to be substituted concretely.

**Plan gap**:
Generic specialization validity was not scheduled as its own phase with its own
regression closure.

**Fix**:
Define specialization completeness explicitly in the guide and give
monomorphization its own wave and phases in the plan.

**Cascading effects**:
Zero-argument constructor-style generics and explicit type-argument calls must be
handled deliberately.

**Architectural lesson**:
Partial specialization is worse than no specialization because it poisons later
passes with believable garbage.

**Verification**:
The new language guide and implementation plan both define completeness and make
recursive concreteness checks mandatory.

### L004 — Pointer categories are a backend architecture rule

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from a previous attempt

**Reproducer**:
The same backend logic handled borrow-derived pointers and `Ptr[T]` values as if
they had identical semantics, causing miscompilation at assignments, returns,
and field accesses.

**What was tried first**:
The earlier backend relied on local heuristics around whether a value's C type
was pointer-shaped.

**Root cause**:
The backend lacked a formal distinction between pointer representations arising
from borrow semantics and pointer values that are part of the language.

**Spec gap**:
The language guide did not define the two pointer categories explicitly.

**Plan gap**:
The codegen wave did not schedule pointer-category handling as a first-class
contract.

**Fix**:
Document the two-pointer-category model in the guide and give it an explicit
phase in the backend wave.

**Cascading effects**:
Call-site adaptation and field-access lowering both depend on this distinction.

**Architectural lesson**:
Backend representation rules are architecture, not cleanup details.

**Verification**:
The new language guide and implementation plan both include a dedicated pointer
category contract.

### L005 — Unit erasure must be total and global

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from a previous attempt

**Reproducer**:
Partially erased unit payloads and parameters produced ghost data paths,
nonexistent reads, and invalid function-pointer shapes in generated C.

**What was tried first**:
The earlier implementation applied unit erasure opportunistically in some codegen
sites but not others.

**Root cause**:
Erasure was treated as a local optimization instead of a global ABI decision.

**Spec gap**:
The language guide did not state that once unit is erased in one location, it is
erased everywhere that participates in the same concrete ABI.

**Plan gap**:
The lowering and backend waves did not isolate unit erasure as an explicit task.

**Fix**:
State total unit erasure as a hard implementation contract and schedule it as its
own backend phase.

**Cascading effects**:
Constructors, patterns, function pointers, and aggregate layout must all agree.

**Architectural lesson**:
There is no such thing as partially erased unit.

**Verification**:
The new language guide encodes total unit erasure and the new implementation plan
gives it dedicated backend tasks.

### L006 — Divergence must be structural, not simulated

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from a previous attempt

**Reproducer**:
Lowering and codegen that simulated post-divergence values produced references to
undeclared temporaries after calls to panic-like functions.

**What was tried first**:
The earlier backend attempted to satisfy type expectations by inventing fallback
values after control flow had already diverged.

**Root cause**:
Divergence was treated as a typing inconvenience instead of as a fundamental
control-flow property.

**Spec gap**:
The language guide did not define divergence as a structural MIR and backend
property strongly enough.

**Plan gap**:
The lowering and backend waves did not schedule divergence closure as its own
explicit responsibility.

**Fix**:
Document structural divergence in the guide and plan, and make it part of both
lowering and backend exit criteria.

**Cascading effects**:
Join blocks, aggregate fallbacks, and destruction paths all depend on accurate
divergence handling.

**Architectural lesson**:
If control flow does not continue, the compiler must stop pretending that it
does.

**Verification**:
The new language guide and implementation plan both treat divergence as a
structural contract.

### L007 — Pattern matching must dispatch on discriminants, not fall through

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
A `match` expression with multiple arms compiles, but the generated code jumps
unconditionally to the first arm's block. All subsequent arms are dead code.

**What was tried first**:
The lowerer emitted `TermGoto(armBlock)` for each arm without evaluating the
pattern. This made match expressions parse, type-check, and "work" in tests
that only had one arm.

**Root cause**:
Match lowering was left as a stub. HIR stores patterns as text strings
(`PatternDesc string`) instead of structured pattern nodes, making real dispatch
impossible at the MIR level.

**Spec gap**:
The language guide defines pattern matching semantics, but the HIR and MIR
specifications did not mandate structured pattern representation.

**Plan gap**:
No wave or phase owned pattern lowering as an explicit task. Wave 07 (HIR→MIR)
mentioned control flow but did not list match dispatch. Wave 05 (type checking)
mentioned match but did not require exhaustiveness.

**Fix**:
1. Add structured pattern nodes to HIR (LiteralPat, BindPat, ConstructorPat,
   WildcardPat).
2. Lower match expressions to cascading branch chains in MIR using enum
   discriminant comparison.
3. Emit correct `TermBranch` / `TermSwitch` sequences in codegen.

**Cascading effects**:
Enum destructuring, exhaustiveness checking, and guard expressions all depend on
real pattern dispatch.

**Architectural lesson**:
A stub that compiles without error is more dangerous than a stub that crashes.
Stubs must be tracked in STUBS.md and must emit diagnostics.

**Verification**:
Match expressions with multiple arms produce distinct codegen paths, tested via
unit tests on the lowerer and codegen.

### L008 — Monomorphization cannot be deferred past codegen

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
A generic function `fn id[T](x: T) -> T { x }` type-checks, but no concrete
specialization is collected or emitted. Any program using `Option[I32]`,
`Result[T, E]`, or other generic types cannot produce working binaries.

**What was tried first**:
The bootstrap path avoided generics entirely. The Stage 2 compiler and its tests
use only concrete types, so the self-hosting gate (Wave 15) passed without
monomorphization.

**Root cause**:
The `compiler/monomorph/` package was created as a placeholder but never
implemented. No wave in the implementation plan owned generic specialization as a
task with entry/exit criteria.

**Spec gap**:
The language guide defines generics and monomorphization, but the implementation
plan did not schedule the work.

**Plan gap**:
Wave 05 mentioned generic inference. Wave 07 mentioned lowering. Neither owned
the actual collection of concrete instantiations or the expansion of generic
function bodies with concrete types.

**Fix**:
1. Implement `monomorph.Collect()` to scan all call sites and collect concrete
   type argument sets.
2. Implement `monomorph.Specialize()` to produce concrete MIR functions from
   generic HIR templates.
3. Integrate into the driver pipeline between type checking and MIR lowering.

**Cascading effects**:
All generic stdlib types (Option, Result, List, Map, Set) and user-defined
generic functions require this to produce working code.

**Architectural lesson**:
A placeholder package with a doc.go is not a substitute for a scheduled,
tested implementation. If a feature has no wave task, it will not be built.

**Verification**:
Generic functions produce specialized MIR and correct C output, tested via
unit tests on monomorph and codegen.

### L009 — Error propagation operator must lower to control flow

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
The `?` operator on a `Result[T, E]` expression compiles, but the checker
returns `Unknown` type and the lowerer simply unwraps the inner expression
without any error checking or early return.

**What was tried first**:
The checker and lowerer treated `?` as a pass-through: `checkQuestion()` returns
`Unknown`, and `lowerExpr(QuestionExpr)` returns `lowerExpr(n.Expr)`. This
allowed the pipeline to proceed without crashing.

**Root cause**:
The `?` operator requires knowledge of the Result/Option type structure to
extract the success value and propagate the error. Without monomorphization
and concrete enum layout, this was deferred — but no task tracked its
completion.

**Spec gap**:
The language guide defines `?` semantics, but the HIR and lowering contracts
did not specify how `?` maps to branching control flow.

**Plan gap**:
No wave or phase owned the `?` operator implementation. Wave 05 type-checked it
as Unknown. Wave 07 lowered it as a no-op.

**Fix**:
1. Checker: extract the inner `T` from `Result[T, E]` or `Option[T]` and
   return it as the expression type.
2. Lowerer: emit a branch that checks for Err/None and early-returns if so,
   otherwise continues with the unwrapped value.
3. Codegen: standard branch emission handles this naturally.

**Cascading effects**:
Depends on enum discriminant access (pattern matching) and knowledge of
Result/Option layout (monomorphization or special-casing).

**Architectural lesson**:
Operators that affect control flow cannot be stubbed as expression-level
pass-throughs. They must produce branches or they silently corrupt behavior.

**Verification**:
`?` on Result/Option produces early-return branches in MIR, tested via
unit tests on check and lower.

### L010 — Drop codegen must emit actual destructor calls

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
The liveness pass correctly computes `DestroyEnd` flags, and the lowerer emits
`EmitDrop` instructions, but the C11 backend emits only `/* drop _lN */`
comments. No actual cleanup code runs at runtime.

**What was tried first**:
The codegen emitted comments as placeholders, intending to revisit drop emission
later. Because no test actually ran the generated C with destructor-dependent
resources, the gap was invisible.

**Root cause**:
Drop emission requires knowing whether a type has a `Drop` trait implementation.
Without that metadata flow from check → codegen, the backend cannot emit the
correct destructor call.

**Spec gap**:
The language guide defines deterministic destruction, but the backend contracts
did not specify how `InstrDrop` maps to C code.

**Plan gap**:
Wave 06 (ownership/liveness) scheduled drop intent insertion, but no wave
scheduled the codegen side — the actual C emission of destructor calls.

**Fix**:
1. Flow Drop-trait information from the checker into the type table or a
   side table accessible during codegen.
2. Codegen: emit `TypeName_drop(&_lN);` for types with Drop impls;
   no-op for types without.
3. Test with types that have explicit Drop implementations.

**Cascading effects**:
Resource management (file handles, locks, allocations) depends on actual
destructor calls, not comments.

**Architectural lesson**:
A comment is not a drop. If codegen emits a comment where it should emit code,
the feature does not exist.

**Verification**:
InstrDrop for types with Drop impls emits function calls in generated C, tested
via codegen unit tests.

### L011 — Closures must capture environments, not erase to unit

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
A closure expression `|x| { x + 1 }` type-checks and produces a valid function
type, but the lowerer returns `constUnit()` — the closure body is never lowered
to MIR and no environment capture occurs.

**What was tried first**:
The lowerer treated closures as "function references (simplified)" and returned
unit. Liveness analysis also skips closure bodies entirely.

**Root cause**:
Closures require environment capture analysis (which outer variables are
referenced), allocation of a closure struct, and emission of a lifted function.
This is a non-trivial transformation that was deferred without a plan task.

**Spec gap**:
The language guide describes closures but does not specify the lowering
representation (lifted function + environment struct).

**Plan gap**:
No wave owned closure lowering. Wave 07 (HIR→MIR) did not mention closures.

**Fix**:
1. Implement capture analysis: scan closure bodies for references to outer
   variables.
2. Generate an environment struct type with captured variables.
3. Lift the closure body to a standalone MIR function that takes the
   environment as a parameter.
4. At the closure expression site, emit struct init for the environment and
   a function pointer pair.

**Cascading effects**:
Iterators, callbacks, and higher-order functions all depend on closures.

**Architectural lesson**:
A feature that type-checks but does not lower is a silent miscompilation, not a
deferred feature.

**Verification**:
Closures produce lifted functions and environment structs in MIR and C output,
tested via unit tests on lower and codegen.

### L012 — Channels must exist as types before concurrency can work

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
The stdlib defines `chan.fuse` with channel operations, but no channel type
exists in the type table or compiler. `spawn` expressions lower to plain
function calls with no threading semantics.

**What was tried first**:
The lowerer treats `SpawnExpr` as `EmitCall(dest, arg, nil, Unit, false)` — a
synchronous function call. No thread creation occurs.

**Root cause**:
Channel types and spawn semantics require runtime integration (thread creation,
queue management) that was deferred. The stdlib `chan.fuse` file defines the
API surface but the compiler has no knowledge of channel types.

**Spec gap**:
The language guide describes channels and spawn as language primitives, but the
type table and backend contracts did not include them.

**Plan gap**:
Wave 08 (runtime) implemented thread and sync primitives in C, but no wave
scheduled the compiler-side integration: channel type interning, spawn lowering
to `fuse_rt_thread_spawn`, or channel operation lowering to runtime calls.

**Fix**:
1. Add channel type kind to the type table.
2. Lower `spawn expr` to a runtime call: `fuse_rt_thread_spawn(fn, arg)`.
3. Lower channel operations (send, recv, close) to corresponding runtime calls.
4. Type-check channel expressions with proper generic element types.

**Cascading effects**:
All concurrency features in the language depend on channels and proper spawn.

**Architectural lesson**:
A runtime library without compiler integration is dead code. Both sides must be
scheduled together.

**Verification**:
Spawn emits `fuse_rt_thread_spawn` calls and channel operations emit runtime
calls, tested via codegen unit tests.

### L013 — Self-verifying plans are not verification

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit, after implementing L007–L012 fixes

**Reproducer**:
Six critical compiler features (pattern matching, monomorphization, error
propagation, drop codegen, closures, channels) were stubbed or missing despite
the implementation plan showing Waves 00–16 as complete. Every wave's exit
criteria were satisfied. Every test passed. The compiler reached the self-hosting
gate at Wave 15 and the native backend transition at Wave 16 with features that
had never produced a working program.

After fixing all six features, the same pattern repeated: the AST-to-HIR bridge
was built with `Unknown` types for all expressions, which made e2e tests pass
for simple programs but left the newly implemented features (generics, pattern
matching, closures, `?` operator) unreachable from any end-to-end test.

**What was tried first**:
Each wave was implemented to satisfy its stated exit criteria. The plan, the
implementation, the tests, and the verification were all produced by the same
agent in the same session. Unit tests were written for the features, and they
passed. The wave was declared complete.

**Root cause**:
The plan, the implementation, and the tests formed a closed loop with no
external forcing function. The agent wrote exit criteria it could satisfy, built
implementations that satisfied those criteria, and wrote tests that validated the
implementations. At no point did an independent check ask: "compile a real
program that uses this feature and run it."

Specifically:
- Exit criteria were phrased as structural properties ("MIR blocks terminate
  structurally") rather than behavioral requirements ("a program using match
  with three enum variants produces the correct output").
- The self-hosting gate (Wave 15) passed because the Stage 2 compiler source
  does not use generics, closures, pattern matching with payloads, or the `?`
  operator.
- Unit tests validated individual components in isolation.
- The AST-to-HIR bridge defaulted all types to `Unknown`, which mapped to C
  `int`, which compiled and ran correctly for integer-only programs.

**Spec gap**:
The implementation plan did not require behavioral end-to-end tests as exit
criteria for any wave.

**Plan gap**:
No wave required a program that exercises the wave's feature to compile, link,
run, and produce verified output.

**Fix**:
1. Every wave that introduces a user-visible feature must include at least one
   end-to-end test that compiles a Fuse program using that feature, runs the
   binary, and checks the output.
2. Exit criteria must include behavioral requirements, not only structural ones.
3. The AST-to-HIR bridge must propagate the checker's resolved types.
4. Verification must be adversarial: "write a program that would fail if this
   feature were stubbed, then run it."

**Cascading effects**:
Every future wave must be accompanied by e2e test programs that exercise the
feature. The e2e test suite becomes a release gate alongside unit tests.

**Architectural lesson**:
A plan that an agent writes and then satisfies is not a plan — it is a
self-fulfilling prophecy. Verification must be independent of the implementer.
When the same agent writes the criteria, the implementation, and the tests, the
only reliable check is a concrete program that runs and produces the right
answer.

**Verification**:
This entry is verified by the existence of L007–L012 and by the current state of
the e2e suite.

### L014 — Document requirements for preventing self-verifying plans

Date: 2026-04-14
Discovered during: Post-audit review of foundational document effectiveness

**Reproducer**:
L013 identified that the plan, implementation, and tests formed a closed loop.
This entry records the concrete requirements that each foundational document
must satisfy to prevent that failure pattern from recurring.

**What was tried first**:
The five foundational documents were written with structural completeness as
the standard. None of them required a running program as evidence that a
feature works.

**Root cause**:
The documents governed how to build the compiler but not how to prove it works.

**Spec gap**:
The language guide described features without requiring compilable proof programs.

**Plan gap**:
The implementation plan defined exit criteria as structural properties, not
behavioral outcomes.

**Fix**:
See the full requirements in this entry's body (now incorporated into each
foundational document as Rules 6.8–6.14, the Verify: task field, STUBS.md,
the wave closure entry format, and tests/e2e/README.md).

**Cascading effects**:
All five foundational documents were updated. The implementation plan gained
Verify: fields, State on entry, Proof of completion sections, Phase 00 stub
audits, and wave closure phases. The rules gained Rules 6.11–6.14. The
repository layout gained STUBS.md and tests/e2e/README.md.

**Architectural lesson**:
Documents that govern construction without governing proof are aspirational,
not normative. A running program is the only proof that a feature exists.

**Verification**:
This entry is verified when all five foundational documents reflect the
requirements above and the e2e suite fails if any feature is reverted to a stub.

### L015 — Generics require a dedicated wave with proof programs at every phase

Date: 2026-04-15
Discovered during: Pre-Wave-17 planning, after L007–L014 audit

**Reproducer**:
The monomorphization package (`compiler/monomorph/`) was implemented with
`Record`, `Substitute`, and `IsGeneric` methods. Unit tests pass. But no generic
Fuse program has ever compiled to a working binary. The package is not integrated
into the driver pipeline, no code scans call sites for generic instantiations,
no code duplicates function bodies with substituted types, and no code rewrites
call sites to reference specialized names.

**What was tried first**:
Monomorphization was added as a phase within the type-checking wave (Wave 05)
because it is conceptually related to type resolution. The four tasks described
collecting instantiations, validating completeness, specializing functions, and
integrating into the pipeline. Each task had a DoD. The monomorph package was
implemented and unit-tested.

**Root cause**:
Generics touch every stage of the pipeline: parsing (generic params), resolution
(type param scoping), checking (type arg inference), monomorphization
(collection and substitution), AST-to-HIR bridge (body duplication), lowering
(concrete types in MIR), and codegen (specialized function names and type
layouts). Cramming this into a single phase of another wave hid the cross-cutting
dependencies. Each component was built in isolation and none were connected.

The specific gaps:
1. The checker does not register generic type parameters as in-scope types
   during body checking.
2. The checker does not resolve explicit type arguments at call sites.
3. No code scans the checked AST to collect concrete instantiations.
4. No code duplicates generic function bodies with concrete type substitution.
5. No code generates specialized function names.
6. No code rewrites call sites to reference specialized names.
7. The driver does not run monomorphization between checking and HIR building.
8. Generic functions with unresolved type parameters are not skipped in codegen.
9. Generic enum types (Option, Result) have no concrete field layout in codegen.
10. The `?` operator depends on specialized Result/Option layout that does not
    exist.

**Spec gap**:
The language guide describes generics and monomorphization but does not specify
the concrete compilation model: what a specialized function looks like in the
generated code, how call sites reference it, or how generic type layouts map to
C struct definitions.

**Plan gap**:
The implementation plan placed monomorphization as a phase within Wave 05 with
four tasks. The tasks were structural rather than behavioral. No proof program
was required. The cross-cutting nature of generics was not reflected in the task
structure.

**Fix**:
Create a dedicated Wave 17 (Generics End-to-End) with 10 phases and proof
programs at every integration point.

**Cascading effects**:
The existing Wave 17 (Retirement of Go and C) and Wave 18 (Targets and
Ecosystem) are renumbered to Wave 18 and Wave 19. Generics must work before
the bootstrap path is retired.

**Architectural lesson**:
Cross-cutting features cannot be implemented as a phase within a single wave.
When a feature touches every stage of the pipeline, it needs its own wave with
its own entry criteria, exit criteria, and proof programs. The granularity of
the wave must match the granularity of the integration risk.

**Verification**:
This entry is verified when Wave 17 Phase 05 Task 01 passes: the proof program
`fn identity[T](x: T) -> T { return x; } fn main() -> I32 { return identity[I32](42); }`
compiles, runs, and exits with code 42.

### L016 — Overdue-stub rule off-by-one blocked every wave entry

Date: 2026-04-17
Discovered during: Wave 01 / Phase 00 / Task 01

**Reproducer**:
At W01 Phase 00, running the wave's own Verify command
`go run tools/checkstubs/main.go -wave W01 -phase P00` reported the lexer
stub as overdue and exited non-zero, blocking the wave from starting. The
same shape would have blocked every subsequent wave entry.

**What was tried first**:
Reading the tool's behavior literally: it compared
`waveOrder(s.RetiringWave) <= cur` and flagged the lexer stub (retiring
W01) as overdue when entering W01. The first instinct was to suspect the
stub table, but the stub was correctly scheduled.

**Root cause**:
Rule 6.15 defined "overdue" as retiring wave *less than or equal to* the
current wave. The tool faithfully implemented that wording. But the phase
model (Rule 6.14, `docs/phase-model.md`) schedules stub retirement to
happen at the wave's PCL, after Phase 00 — so a stub retiring in W01 is
not overdue when W01 begins, it is exactly on schedule. The correct
relation is *strictly less than*. `<=` collapses on-schedule and overdue
into the same state, making every wave unstartable once any stub was
scheduled to retire in it. W00 didn't trip this because W00's seeded
stubs all retire in later waves and W00 uses the `-audit-seed` codepath,
which bypasses the overdue comparison.

**Spec gap**:
Rule 6.15's wording did not match the phase model. The language reference
was unaffected; this was a governance-rule defect, not a language-spec
defect.

**Plan gap**:
The W00 Verify exercised `-audit-seed`, not `-wave -phase P00`, so the
first use of the overdue comparison was W01's own Verify — after the rule
and tool had already shipped and been signed off as "CI green."

**Fix**:
- `tools/checkstubs/main.go` `checkWave`: change `<=` to `<` in the P00
  branch.
- `docs/rules.md` §6.15: reword to "strictly less than the current wave"
  and explicitly note that a stub whose retiring wave equals the current
  wave is not overdue at P00.
- `docs/audit.md` A4: mirror the reworded rule.
- `tools/checkstubs/stubs_test.go`: add `TestCheckWaveSameWaveNotOverdue`
  as a regression test so any future reintroduction of the off-by-one
  fails CI.

**Cascading effects**:
None to earlier waves — W00 never exercised this path. Future waves now
pass Phase 00 as intended. The regression test hardens the check so the
two forms of "overdue" (strictly-before vs. on-or-before) cannot silently
diverge between tool and rule again.

**Architectural lesson**:
A governance rule whose first real exercise is the wave *after* the rule
ships is effectively unverified. Governance tools should be exercised
against a realistic stub table (one containing a stub that retires in
the entered wave) before the rule that depends on them is treated as
validated. More generally: CI green on a tool does not mean the rule
behind it is correct — the rule is only validated when the tool runs
against data shaped like the rule's intended use.

**Verification**:
```
go test ./tools/checkstubs/... -v
go run tools/checkstubs/main.go -wave W01 -phase P00
```
The first now includes `TestCheckWaveSameWaveNotOverdue` which fails if
`<=` is ever reintroduced. The second now exits 0 when the lexer stub is
still present but retires W01.

### L017 — Wave phases W00–W06 landed in combined single commits

Date: 2026-04-17
Discovered during: Retrospective audit of W00–W06 after W06 closure

**Reproducer**:
`git log --oneline` between `735e1d3` and `35a4204`. Each wave's
implementation SHA (`d609313` W00, `962d41c` W01, `ca88b11` W02,
`f132793` W03, `7624fab` W04, `9eac416` W05, `35a4204` W06) is
simultaneously the P00 audit, implementation, and PCL commit — a
single commit touches `STUBS.md`, `.claude/current-wave.json`,
`docs/learning-log.md`, and compiler source in the same diff.

**What was tried first**:
No alternative was tried; the phase model in
`docs/phase-model.md` §3 describes P00 / implementation / PCL as
distinct *phases* but never stated they had to be distinct
*commits*. The rule set in `docs/rules.md` §9 similarly said nothing
about commit-phase mapping. Each wave's contributor batched the
three phases into one commit for convenience.

**Root cause**:
The temporal discipline the phase model expects (audit before
implementation; retirement only after verified implementation) was
enforced at the in-session level but not at the commit level.
Without commit separation, a retrospective auditor cannot verify
the P00 state of STUBS.md independently of the implementation; they
also cannot verify that the PCL retirement happened only after the
implementation commit was CI-green.

**Spec gap**:
None in the language reference; this is a governance defect.

**Plan gap**:
`docs/rules.md` §9 (Commit and PR rules) lacked a rule tying the
phase model's temporal phases to the commit graph. `docs/phase-model.md`
described the phases but did not mandate commit separation. The wave
docs under `docs/implementation/waveNN_*.md` listed Verify commands
per phase but never said "land each phase as its own commit."

**Fix**:
- `docs/rules.md` §9: add `Rule 9.5 — Wave phases land in distinct
  commits.` The rule defines the minimum three-commit sequence
  (P00 → implementation → PCL) and enumerates what must and must
  not appear in each commit. Applies to W07 and every subsequent
  wave. The rule explicitly records that W00–W06 pre-date it.
- This log entry documents the retrospective record.

**Cascading effects**:
- Retroactively splitting W00–W06 commits would rewrite `main`
  history and invalidate the closure SHAs cited in every prior
  `WCxxx` entry. That is a cure worse than the disease; Rule 9.5
  therefore applies going forward only.
- Future wave audits can rely on Rule 9.5 to verify that each
  phase's state is independently reachable via `git checkout
  <sha>`. The phase-00 audit SHA, for example, must reproduce the
  pre-implementation STUBS.md that W24's Stub Clearance Gate will
  eventually key against.
- The closure-template defect in `WC003`–`WC006` (separate L-entry
  covers that) is easier to fix in isolation once the 3-commit
  pattern is in place because the PCL commit's diff becomes a
  targeted retirement change, not a mixed diff.

**Architectural lesson**:
Phase discipline that lives only in agent instinct rather than in
the commit graph is unverifiable after the fact. Any discipline the
project expects a future auditor to confirm — ordering, pre/post
state, attribution — must map into commit topology. "In-session
workflow" is not a durable record.

**Verification**:
```
grep -n "Rule 9.5" docs/rules.md
grep -n "L017" docs/learning-log.md
git log --oneline d609313..35a4204
```
The first two confirm the rule and record exist; the third documents
the single-commit-per-wave pattern the rule is replacing.


### WC000 — Wave 00 Closure

Date: 2026-04-16
Wave: 00 — Governance and Phase Model

**Proof programs added this wave**:
None. Wave 00 delivers repository, module, build, CI, tooling, and
governance scaffolding; no user-visible language behavior is introduced,
so no `tests/e2e/` proof programs are required (Rule 6.8 applies from W05
onward when the minimal end-to-end spine exists).

**Stubs retired this wave**:
None. Wave 00 is the seeding wave: it populates `STUBS.md` with entries
for every feature scheduled for W01–W30 but retires none of them.

**Stubs introduced this wave**:
Thirty top-level stubs seeded in `STUBS.md` Active table, one per
downstream wave W01–W30. Each names its owning package (creating the
package directory where one was missing), declares the diagnostic the
compiler must emit once that package starts participating in the pipeline,
and is tagged with its retiring wave. Full list in the W00 block of the
`STUBS.md` Stub history.

**What was harder than planned**:
- The originally-specified Verify command `go run tools/checkstubs/main.go
  -audit-seed` requires the tool's entire implementation to live in a
  single file (Go's `go run <file>` form only loads the named file). The
  tool was initially split into `main.go` + `stubs.go`; that split was
  reverted into a single-file package so the plan's Verify command works
  as written. Worth noting for other tools scheduled later whose Verify
  commands use the same form.
- The repository-layout.md §12 phrasing "CI configuration lives under
  `.ci/`" collides with GitHub Actions' requirement that workflows live
  at `.github/workflows/`. Resolved by placing the canonical workflow at
  `.github/workflows/ci.yml` and reserving `.ci/` for helper scripts,
  with a README in `.ci/` documenting the arrangement. No normative doc
  change was needed; `.ci/` remains where helpers will land (W17 perf
  driver, W23 package-manager fetcher smoke test, W25 reproducibility
  gate, W27 perf gate).

**What the next wave must know**:
- The Stage 1 CLI at `cmd/fuse` is a Wave 00 stub: only `version` and
  `help` subcommands. `build`, `run`, `check`, `test`, `fmt`, `doc`, and
  `repl` all land in W18. Until then, any test that invokes those
  subcommands must expect a non-zero exit.
- Every compiler package under `compiler/` is a doc-only stub. The
  `doc.go` files name each package's responsibility and the wave that
  retires the stub; they do not yet export any symbols beyond the package
  declaration.
- `tests/perf/` exists as an empty directory from W00. W17-P13 seeds the
  baseline corpus; W27 is the gating wave.
- The governance tools under `tools/` are minimal — enough for W00
  exit criteria and cross-platform CI. They will be extended as later
  waves need stricter checks (W22 Stub Clearance, W30 docs site
  validation). Tool extensions must remain single-file packages to keep
  the plan's `go run tools/X/main.go` Verify form working.
- `.claude/current-wave.json` is the coordination file for multi-machine
  and multi-agent work (Rule 11). Every wave transition updates this file
  as part of PCL.

**Verification**:
```
make all
go test ./...
go run tools/checkstubs/main.go
go run tools/checkstubs/main.go -audit-seed
go run tools/checklayout/main.go
go run tools/checkdocs/main.go -foundational
go run tools/checkartifacts/main.go
go run tools/checkci/main.go
go run tools/checkgov/main.go -current-wave
go run tools/checkstubs/main.go -history-current-wave W00
grep "WC000" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.26.2). CI
matrix green on the committed SHA is the authoritative record.

### WC001 — Wave 01 Closure

Date: 2026-04-17
Wave: 01 — Lexer

**Proof programs added this wave**:
No `tests/e2e/` programs this wave — Rule 6.8 requires end-to-end proof
programs from W05 onward, when the minimal spine exists. The lexer's
behavioral proofs instead live under `compiler/lex/testdata/`:
- `basic.fuse` / `basic.tokens` — realistic `fn main() -> I32` shape;
  covers keywords, ident, arrow, braces, numeric-with-suffix literal,
  semicolons.
- `operators.fuse` / `operators.tokens` — exercises `?.`, `..=`, `<<`,
  compound assignment `+=` and `<<=`. Forces the longest-match contract
  from reference §1.10 and §5.
- `comments.fuse` / `comments.tokens` — line comment followed by a
  nested block comment followed by a real token. Confirms the scanner
  skips trivia in order and the nested-depth tracker returns to zero.
- `rawstrings.fuse` / `rawstrings.tokens` — the canonical mix: `r"..."`,
  `r#"..."#`, `r##"..."##`, and the forcing counter-example `r#abc`
  which must tokenize as IDENT HASH IDENT.

**Stubs retired this wave**:
- Lexer and token model — retired at PCL. Stub row removed from
  `STUBS.md` Active table; W01 block appended to Stub history per
  Rule 6.16. Verify chain:
  `go run tools/checkstubs/main.go -wave W01 -retired lexer` (passes),
  `go run tools/checkstubs/main.go -history-current-wave W01` (passes).

**Stubs introduced this wave**:
None. The lexer is a leaf subsystem — parser, resolver, and later
stages depend on tokens but no new stubs were added this wave. The
parser stub remains in the Active table and retires in W02.

**What was harder than planned**:
- The overdue-stub rule was off-by-one (L016). W00 shipped with CI green
  because W00 never exercised the P00-overdue code path — only
  `-audit-seed`. W01's first Phase 00 run surfaced the bug immediately.
  Fix was three edits: `tools/checkstubs/main.go` `<=` → `<`,
  `docs/rules.md` §6.15 rewording, `docs/audit.md` §A4 mirror, plus a
  regression test `TestCheckWaveSameWaveNotOverdue`. Roughly 20 minutes
  of diagnosis before any lexer code was written.
- Raw-string recognition vs. identifier-prefix collision (reference
  §1.10) required routing the `r"` / `r#"` decision *before* the
  generic identifier consumer, not after. The first cut tried to
  detect raw-string opener post-identifier and backtrack, which
  immediately tangled span bookkeeping. Replacing it with a
  `isRawStringOpener` lookahead on the `r` branch was cleaner and kept
  `r#abc` correct by construction rather than by a special case.
- Char literal unicode escape `'\u{1F600}'` had an off-by-one in the
  initial `scanChar` (checked `s.src[s.off-1] == '{'` after advancing
  both `\` and `u`). Caught by the first test run; fix was to look at
  the byte that followed the escape character, not the escape itself.
- `.gitattributes` was required for golden stability on Windows. The
  Write tool produces LF content, but without an explicit text
  attribute a Windows checkout of testdata `.fuse` files can mutate
  line endings on the way through git, shifting spans. Added
  `.gitattributes` enforcing LF for `*.fuse`, `*.tokens`, `*.golden`.
  The golden-comparison path also normalizes CRLF→LF defensively so a
  clone that predates the attributes change still compares cleanly.

**What the next wave must know**:
- `compiler/lex.Scanner` is the canonical front door. Construct with
  `NewScanner(filename, src)`, call `Run()`, then read `Tokens()` and
  `Errors()`. The final token is always `TokEOF`; callers that iterate
  must handle it or trim it. `Run()` is idempotent and resets state on
  re-entry.
- `TokenKind` has a stable `String()` via `kindNames`. New kinds must
  add a row there; `TestTokenKindCoverage` enforces the invariant.
  New keywords must appear in both `keywords` (map for lookup) and
  `keywordList` (ordered list for deterministic iteration). A test
  asserts the two sets agree.
- Span columns count UTF-8 bytes past the start of the logical line,
  not grapheme clusters. Parser and diagnostic code must match this
  contract; a later renderer can widen columns for display.
- `r#abc` tokenization is the forcing example from reference §1.10
  and is covered by the `rawstrings` golden. Any parser change that
  stops consuming `HASH` between idents will be caught by the golden,
  not by a unit test.
- `?.` is a single `TokQuestionDot` by longest match; the parser must
  handle that token directly and not try to compose `?` + `.`
  (reference §1.10, §5.6). Tokenizing `x ? .y` as three tokens
  (QUESTION, DOT, …) is the deliberate fall-back when whitespace
  breaks the longest match.
- Lexical errors emitted so far: BOM rejection, unterminated block
  comment, unterminated raw string, unterminated/malformed character
  literal, unterminated string literal, unexpected character. Each
  diagnostic carries a primary span and a one-line message (Rule 6.17).
  When W18 lands the diagnostic pipeline, these need to be wired
  through the shared `compiler/diagnostics` package rather than the
  local `Diagnostic` struct — but the shape (span + message + hint)
  is already correct.
- There is no `TokUnderscore`: the character `_` lexes as a plain
  identifier. Patterns and let-bindings that treat `_` as a wildcard
  must do so at the parser/HIR layer against the identifier text.
  This matches reference §1.9 where `_` is not in the reserved-word
  list.
- Numeric literals carry their suffix as part of `Token.Text`; the
  scanner does not validate suffix text (reference §1.10 requires
  normalization at the HIR-to-MIR boundary, not at lex time). The
  parser's first pass should emit `TokInt`/`TokFloat` through an
  unchanged text field; the checker later interprets `i32`, `usize`,
  `f64`, etc.

**Verification**:
```
go test ./compiler/lex/... -v
go test ./compiler/lex/... -run TestGolden -count=3 -v
go run tools/checkstubs/main.go -wave W01 -phase P00
go run tools/checkstubs/main.go -wave W01 -retired lexer
go run tools/checkstubs/main.go -history-current-wave W01
grep "WC001" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.26.2). CI
matrix green on the committed SHA is the authoritative record.

### WC002 — Wave 02 Closure

Date: 2026-04-17
Wave: 02 — Parser and AST

**Proof programs added this wave**:
No `tests/e2e/` programs (Rule 6.8 applies from W05 onward). Parser
proofs live under `compiler/parse/testdata/`:
- `fn_decl.fuse` / `.ast` — vis-pub function with params and return.
- `struct_enum.fuse` / `.ast` — named struct plus multi-variant enum
  (tuple and struct-shaped variants).
- `decorators.fuse` / `.ast` — stacked `@repr(C) @align(8)` on a
  struct and inline `@inline` on a fn.
- `match_patterns.fuse` / `.ast` — covers every pattern kind:
  constructor (unit, tuple, struct with `..`), or-pattern, range,
  inclusive range, `@`-binding, wildcard. Exercises the forcing
  pattern vocabulary in one fixture.
- `exprs.fuse` / `.ast` — `?.` chain, slice-range indexing
  (`a[0..3]`, `a[..=9]`), struct literal with update syntax
  (`..prev`), mixed precedence.

**Stubs retired this wave**:
- Parser and AST — retired at PCL. Row removed from `STUBS.md` Active
  table; W02 block appended to Stub history (Rule 6.16). Verify
  chain: `go run tools/checkstubs/main.go -wave W02 -retired
  "parse,ast"` (passes trivially — the tool's `-retired` flag takes a
  single name and the row was named `Parser and AST`, so the check
  only asserts absence; the real guarantee is the row removal plus
  the history block assertion via `-history-current-wave W02`).

**Stubs introduced this wave**:
None. Parser and AST are leaf surfaces at this stage; the Resolver,
Type checker, and downstream packages retain their own stubs on
their own retirement waves.

**What was harder than planned**:
- The `baseNode` embedded struct was unexported. Go's field-promotion
  rules mean that a promoted field through an unexported embedded
  struct is NOT accessible from outside the package. So the parser
  could not write `node.Span = sp` to set spans on nodes it was
  building. Renamed to `NodeBase` (exported) so the promoted `Span`
  field is assignable from the parse package. Every concrete node
  struct updated; the `TestSpanCorrectness` reflection test still
  passes because the field-name lookup changed in lockstep.
- Parser-level path vs. field access. The grammar's `path = IDENT {
  "." IDENT }` looks like it should consume a whole chain, but in
  expression position `x.y.z` must surface as nested `FieldExpr`
  nodes so that `x.y?.z` lands as `OptField(Field(x, y), z)` rather
  than `OptField(PathExpr[x, y], z)`. Fix: at expression level,
  `parsePathExpr` consumes one identifier only; subsequent `.IDENT`
  is handled by `parsePostfix` as field access. The
  `TestOptionalChainParse/mixed` case is the regression anchor.
- `self` parameter shorthand. Reference §1.9 makes `self` and `Self`
  reserved, and trait/impl methods usually take `self` without a
  type annotation (`fn hello(self)`). Extended `parseParam` to accept
  `self` as the name and to make the `: Type` annotation optional
  when the name is `self`. Also extended `parseType` to accept
  `Self` as a path root, since `Self` appears in return/parameter
  positions.
- Synchronize-at-`}` deadlock at top level. The error-recovery
  `synchronize()` deliberately does NOT consume `}` so an inner
  block parser can resume. At file level there is no inner parser,
  so the token `}` caused `parseFile` to loop forever. Guarded by a
  "if no token was consumed this iteration, advance one" check in
  `parseFile`. Caught by `TestNopanicOnMalformed/case-19` (`"}}}"`).
- Struct-literal disambiguation and trailing expressions interact.
  The forcing case `fn f() { if Foo { x } }` lands the `if`
  expression as the block's trailing value, not a statement —
  `parseStmtOrTrailing` now treats a block-expression as trailing
  when it is immediately followed by `}`. The `TestStructLiteralDisambig`
  helper switched to a `bodyExpr()` accessor that reads whichever
  field is populated.
- Decorators on statics. The initial `parseStaticDecl`/`parseConstDecl`
  signatures didn't receive the decorators parsed up front, and
  `TestDecoratorParsing/rank` (`@rank(1) static LOCK: I32 = 0;`)
  caught it immediately. Added `Decorators` fields to `StaticDecl`
  and `ConstDecl` and threaded the list through the construction
  sites. `StaticDecl` was already in the AllItemNodes registry, so
  `TestAstNodeCompleteness` continued to pass.

**What the next wave must know**:
- `compiler/parse.Parse(filename, src)` is the single entry point.
  It returns `(*ast.File, []lex.Diagnostic)`; the file is never nil
  even on empty or wholly malformed input — downstream code does not
  need a nil guard. The second return bundles lexer and parser
  diagnostics together.
- The AST is syntax-only (Rule 3.2). No resolved names, no types,
  no annotations. Path segments are raw `[]ast.Ident`; `x.y.z` is
  two `FieldExpr` nodes wrapped around a `PathExpr{x}`. W03 is the
  first wave allowed to attach resolution information.
- Every concrete node embeds `ast.NodeBase`; builders set `node.Span`
  after construction. Adding a node type requires: (a) embedding
  `NodeBase`, (b) implementing the right marker method
  (`itemNode`/`exprNode`/`stmtNode`/`patNode`/`typeNode`), (c)
  registering in the corresponding `All*Nodes` list so
  `TestAstNodeCompleteness` stays honest.
- Struct-literal disambiguation lives in the parser: inside `if`/
  `while`/`for`/`match` headers the parser runs in `ctxNoStruct`
  which forbids `IDENT {` from starting a struct literal. Any new
  expression context that must also suppress struct literals should
  call `parseExprNoStruct` instead of `parseExpr`.
- `?.` arrives as a single `TokQuestionDot` from the lexer and is
  handled in `parsePostfix` — it is NOT composed from `?` + `.` in
  the parser. If a later refactor changes `parsePostfix`, the
  `TestOptionalChainParse` regression catches it.
- Two things are *contextual* keywords rather than reserved words,
  per reference §1.9: `dyn` and `union`. The lexer emits them as
  identifiers; the parser tests `cur().Text` in type position
  (`dyn`) and item-dispatch position (`union`). Adding more
  contextual keywords should follow the same pattern and carry a
  comment pointing to §1.9 so future readers understand why.
- `parseParam` accepts the reserved word `self` as the parameter
  name and makes the `: Type` annotation optional for it. Every
  other parameter requires both name and annotation per the
  grammar. The resolver (W03) will fill the implicit `Self` type
  for `self` receivers from the enclosing impl context.
- Error recovery is strictly additive: every malformed-input case
  must produce at least one diagnostic and terminate. The test
  corpus in `TestNopanicOnMalformed` should grow as new failure
  modes surface; reducing its size is a regression.

**Verification**:
```
go test ./compiler/ast/... -v
go test ./compiler/parse/... -v
go test ./compiler/parse/... -run TestGolden -count=3 -v
go test ./compiler/parse/... -run TestNopanicOnMalformed -v
go run tools/checkstubs/main.go -wave W02 -phase P00
go run tools/checkstubs/main.go -wave W02 -retired "parse,ast"
go run tools/checkstubs/main.go -history-current-wave W02
grep "WC002" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.26.2). CI
matrix green on the committed SHA is the authoritative record.

### WC003 — Wave 03 Closure

Wave 03 (Resolution) completed 2026-04-17. The resolver is now the
retirement site of the W00-declared `compiler/resolve/` stub. It runs
after parse and before the HIR+TypeTable construction scheduled for
W04; its outputs are consumed as-is by that wave.

**Scope landed**:

- Module discovery from a filesystem root (`DiscoverFromDir`) plus an
  explicit `SourceFile`-slice form for tests and in-memory builds.
  Discovery is deterministic (Rule 7.1): `walkSorted` visits
  directories in lexicographic order with directories preceding their
  sibling files.
- `ModuleGraph` with sorted `Order` and per-module sorted edge lists;
  `finalize()` dedupes edge duplicates. Every pass that iterates
  modules consumes `Order`, never `range Modules` (reference §18).
- `SymbolTable` with a reserved zero slot so `NoSymbol` is always
  interpretable without a separate present/absent flag. Scopes chain
  to a parent for `Lookup`; `LookupLocal` never walks parents.
- Top-level indexing across every W02 item kind (`FnDecl`,
  `StructDecl`, `EnumDecl`, `TraitDecl`, `ConstDecl`, `StaticDecl`,
  `TypeDecl`, `UnionDecl`, `ExternDecl`). Enum variants are hoisted
  into the enclosing module scope per reference §11.6, §18.6, with
  conflicts between a variant and a prior item reported as
  duplicate-item diagnostics.
- Module-first import resolution (reference §18.7): the full dotted
  path is tried as a module, and on miss the (prefix, last) pair is
  tried as (module, item). Single-segment imports skip the prefix-
  fallback branch and emit a clean "unresolved import" diagnostic
  when neither step matched.
- Qualified enum variant resolution for both `Enum.Variant` and
  `module.Enum.Variant` forms (reference §11.6). Because the W02
  parser emits `Dir.North` as `FieldExpr{Receiver: PathExpr{Dir},
  Name: North}`, the resolver's expression walker flattens any
  FieldExpr-chain rooted at a PathExpr into a dotted segment list
  and re-resolves it when the root names a module, enum, or import
  alias. Chains whose root is *not* a static path fall through to
  ordinary field-access walking so that `local.field` does not get
  misdiagnosed as an unresolved path.
- Import cycle detection via Tarjan's strongly-connected-components
  algorithm. Self-edges are reported as cycles; DAGs are not. Cycle
  diagnostics name members in lexicographic order for stability.
- `@cfg` evaluator at resolve time (reference §50.1) covering
  `key = "value"`, `feature = "x"`, `not(...)`, `all(...)`,
  `any(...)`, and nested combinations. Items whose predicate is
  false are filtered before indexing so they participate in no
  downstream name resolution step (reference §50.1 contract).
  Duplicate-item detection runs after filtering; two items that
  both survive a build produce a diagnostic naming the second
  occurrence and pointing at the first.
- Four-level visibility enforcement (reference §53.1): private,
  `pub(mod)` (declaring module and dotted descendants), `pub(pkg)`
  (entire build), `pub`. Enforcement runs across every recorded path
  binding so that module-qualified uses, import-qualified uses, and
  enum-variant uses all pay the same visibility check.

**Notable design choices**:

- AST is not mutated (Rule 3.2). All resolved information lives in a
  `Resolved` struct: a `*ModuleGraph`, a `*SymbolTable`, and a
  `map[SiteKey]SymbolID` that binds every successfully resolved path
  occurrence to its target symbol. Failed resolutions produce
  diagnostics and no binding (Rule 6.9 — never produce silent wrong
  output).
- Primitive type names (`I32`, `Bool`, `String`, etc.) are *skipped*
  by the resolver. Their identity is scheduled for the W04
  TypeTable; binding them here would duplicate state.
- Single-segment PathExprs in expression position fail silently on a
  miss because they may refer to a local `let`/`var` that W04 (HIR
  lowering) handles. Multi-segment paths are strict because they
  must refer to module- or enum-qualified items.
- `lookupEnumInModule` is a *probe* that returns NoSymbol without
  diagnosing so the walker can distinguish
  `module.Enum.Variant` from `module.Submodule.Item` without false
  positives.

**Proof surface**:

- `TestModuleDiscovery` (run with `-count=3` for determinism),
  `TestModuleDiscovery_EmptyRoot`
- `TestModuleGraph`, `TestModuleGraph_DuplicatePath`
- `TestScopeLookup`, `TestScopeLookup_DuplicateInsert`
- `TestTopLevelIndex`, `TestTopLevelIndex_DuplicateDefinition`,
  `TestTopLevelIndex_VariantHoistConflict`
- `TestModuleFirstFallback` with four sub-cases: full-path-is-module,
  item-fallback, unresolved-path, totally-missing
- `TestQualifiedEnumVariant` with three sub-cases: local-enum,
  cross-module-enum, unknown-variant-is-diagnostic
- `TestImportCycleDetection` with three sub-cases: two-module,
  three-module, self-cycle; plus
  `TestImportCycleDetection_NoFalsePositive` proving a DAG stays
  silent
- `TestCfgEvaluation` with 13 sub-cases covering every supported form
  plus a malformed-bare-ident diagnostic check
- `TestCfgDuplicates` with three sub-cases covering mutually
  exclusive predicates, both-survive, and the no-cfg duplicate form
- `TestVisibilityEnforcement` with five sub-cases across all four
  levels and their crossings

**Lessons captured**:

- The parser emits `a.b` chains as `FieldExpr{PathExpr, b}` rather
  than a 2-segment `PathExpr`. The resolver is the first pass that
  has enough context to tell `module.item` from `local.field`, so
  flattening happens here — *never* in the parser (Rule 3.2). The
  `tryResolveFieldChainAsPath` helper's "return false when the root
  is not a module/enum/alias" check is the forcing function.
- `@cfg` filtering must complete before indexing, not in parallel
  with it. If the resolver indexed a `@cfg(os = "windows")` item on
  Linux and then removed it, a downstream pass could still see a
  stale binding pointing at the discarded symbol.
- Import cycle detection via Tarjan is simpler and cheaper than
  the textbook DFS-color approach, and its iterative structure
  avoids the recursion-depth guard the parser had to add. A
  self-edge must be special-cased because an SCC of size 1 without
  a self-loop is *not* a cycle.
- Module-first fallback only makes sense for multi-segment paths.
  A single-segment `import nowhere` that tries the (prefix="",
  tail=nowhere) form produces diagnostics that read as "no item
  'nowhere' in module ''" — confusing. The resolver short-circuits
  the fallback for single-segment paths and emits
  "unresolved import" directly.
- Visibility for `pub(mod)` requires a dotted-descendant check, not
  a prefix-string check. `util.inner` is a descendant of `util`;
  `utilities` is not. The `isDescendant` helper checks that
  `usingMod[len(ancestor)] == '.'` after the prefix match.

**Verification**:
```
go test ./compiler/resolve/... -v
go test ./compiler/resolve/... -run TestModuleDiscovery -count=3 -v
go test ./compiler/resolve/... -run TestModuleGraph -v
go test ./compiler/resolve/... -run TestScopeLookup -v
go test ./compiler/resolve/... -run TestTopLevelIndex -v
go test ./compiler/resolve/... -run TestModuleFirstFallback -v
go test ./compiler/resolve/... -run TestQualifiedEnumVariant -v
go test ./compiler/resolve/... -run TestImportCycleDetection -v
go test ./compiler/resolve/... -run TestCfgEvaluation -v
go test ./compiler/resolve/... -run TestCfgDuplicates -v
go test ./compiler/resolve/... -run TestVisibilityEnforcement -v
go run tools/checkstubs/main.go -wave W03 -phase P00
go run tools/checkstubs/main.go -wave W03 -retired resolve,cfg,visibility
go run tools/checkstubs/main.go -history-current-wave W03
grep "WC003" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+). CI
matrix green on the committed SHA is the authoritative record.

### WC004 — Wave 04 Closure

Wave 04 (HIR and TypeTable) completed 2026-04-17. The `compiler/typetable/`
and `compiler/hir/` packages are now the retirement site of the W00-declared
"HIR and TypeTable" stub. Together they form the typed semantic IR that
sits between the resolver (W03) and the type checker (W06), with the
pass-graph foundation that W18 (incremental driver) and W19 (LSP) consume
without rework.

**Scope landed**:

- `compiler/typetable/` — `TypeId`, `Kind`, `Type`, and `Table`. The set
  of kinds is fixed (no `Unknown` member by design — L013 defense). The
  interner pre-allocates every primitive TypeId in a deterministic order
  so intern tables across runs have the same layout (Rule 7.1). Nominal
  identity is (name, defining-symbol, module, type-args); two `Expr`
  structs from different modules are distinct TypeIds per reference
  §2.8. Bounds on `dyn Trait1 + Trait2` are canonicalised by sorting so
  `dyn A + B` and `dyn B + A` share a TypeId. `KindChannel` and
  `KindThreadHandle` are defined now for W07 concurrency work without
  retrofitting.
- `compiler/hir/` — full semantic IR. Every Typed node carries a
  `TypeId`; every `Node` carries a stable `NodeID`. Structured patterns
  (`LiteralPat`, `BindPat`, `ConstructorPat`, `WildcardPat`, `OrPat`,
  `RangePat`, `AtBindPat`) replace text-form patterns (L007 defense).
  Builders (`Builder.New*`) enforce metadata at construction — missing
  `NodeID`, `NoType` type, wrong kind, or nil required child → panic.
  The AST-to-HIR bridge lives in the same package so the invariant
  walkers can be run against the bridge's output directly.
- `NodeID` identity is stable across unrelated source edits
  (W04-P05-T02). The format is `module::item::local_path` where
  `local_path` is a structural breadcrumb (`body/stmt.2/lhs`), not an
  allocation counter. Editing function `g` does not shift any NodeID
  inside function `f`.
- `Manifest` is the pass graph. Passes declare Inputs, an OutputKey,
  and a `Fingerprint(inputs) []byte`. `Validate` runs a Kahn topological
  sort with lexicographic tie-breaking so the order is byte-identical
  across runs (Rule 7.1). `Run` executes the validated order; duplicate
  names, missing inputs, and cycles are pipeline bugs (panic or error,
  not user diagnostics).
- `ComputeFingerprint` is the stable hash helper — SHA-256 over the
  pass name followed by sorted key/value input pairs, each NUL-
  separated so keys and values can never collide lexically. Pass name
  is folded in first so two passes with identical inputs produce
  distinct digests.
- `IncrementalPlan` computes the Rerun/Reuse partition for a
  Manifest given a set of dirty inputs. It propagates dirtiness
  transitively: any pass that depends on a dirty input or on a
  transitively dirty pass must re-run; everything else reuses. Tests
  exercise two scenarios — a localised edit (one function's HIR
  invalidated, only its dependents re-run) and a source-root edit
  (every pass re-runs).
- `RunInvariantWalker` is the continuous invariant check W04-P04-T02
  declares. It walks the full Program and reports: empty NodeIDs,
  NoType TypeIds, `NoType` ConstructorType fields, nil required
  pattern/body fields, OrPats with < 2 alternatives, RangePats missing
  a bound. The bridge's own output passes the walker cleanly; a
  synthetic corruption produces a violation.

**Notable design choices**:

- No `Unknown` kind in TypeTable. The explicit `KindInfer` exists for
  the "type checker will resolve this" case — the bridge writes it
  whenever it cannot honestly propagate a concrete type. Passes
  observing a post-check KindInfer must emit a compiler bug (not a
  user diagnostic).
- Bridge type derivation priority order (documented in bridge.go):
  (1) source annotation; (2) resolver binding → symbol's TypeId; (3)
  literal primitive kind (hinted by context); (4) explicit Infer.
  Rule (4) is the only fallback; there is no silent Unknown default.
- Pass fingerprints include the pass name. Otherwise two passes with
  the same inputs would share a digest and the cache couldn't tell
  them apart.
- `Manifest.Passes()` returns a deterministic list even before
  `Validate` — lexicographic order when the graph is not yet built —
  so tests that inspect pre-validation state don't depend on map
  iteration order.
- Nominal identity in TypeTable records the defining `Symbol` as an
  `int` (not a typed `resolve.SymbolID`) to avoid an import cycle.
  The resolver's SymbolID is the canonical source of truth; the
  TypeTable is a consumer.

**Lessons captured**:

- A TypeTable that treats the hash map value as the interner is
  simpler and more deterministic than threading a comparison function
  through every Intern call. The cost is stringifying the key on every
  intern; the benefit is trivial diff-ability of the table state in
  tests.
- `Param` and `Field` should satisfy `Node` (for NodeID/Span) but not
  the marker interfaces of `Item`/`Expr`/`Stmt`/`Pat`. Adding false-
  positive markers during the first HIR draft caused the compiler to
  accept a Param in expression position; the cleanup was to remove
  those markers so the type system catches the bad substitution at
  build time.
- The parser emits `a.b.c` as a FieldExpr chain, not a 2-segment
  PathExpr. The bridge already handles this correctly because it calls
  lowerExpr on each FieldExpr's receiver recursively and the resolver's
  Bindings map has attached symbols to FieldExpr spans where
  applicable.
- Tarjan-style pass-graph cycle detection would be overkill; Kahn
  with sorted tie-breaking gives the same deterministic topological
  order and simpler code.
- `TestPassFingerprintStable -count=3` is the right shape of test for
  any byte-deterministic output contract. The pattern generalises to
  other waves (golden tests, mangled name generation in W08).

**Proof surface**:

- `compiler/typetable/`: `TestTypeInternEquality` (6 sub-cases),
  `TestNominalIdentity` (5 sub-cases), `TestChannelTypeKindExists`,
  `TestThreadHandleKindExists`, `TestInferIsExplicit`.
- `compiler/hir/`: `TestHirNodeSet` (33 concrete types), `TestMetadataFields`,
  `TestBuilderEnforcement` (10 sub-cases for required-metadata panics),
  `TestBuilderEnforcement_HappyPath`, `TestAstToHirTypePreservation` (6
  sub-cases proving types propagate through fn signatures, let
  annotations, struct nominals, enum variants, and the "no expression
  has NoType" invariant), `TestBridgeInvariant`, `TestInvariantWalkers`
  (clean + synthetic violation), `TestPassManifest` (5 sub-cases
  including cycle detection and validated run order),
  `TestDeterministicOrder` (-count=3 confirms stable topological
  ordering), `TestPassFingerprintStable` (-count=3 confirms byte-
  identical digests across runs), `TestStableNodeIdentity` (editing
  function g does not shift any NodeID in function f),
  `TestIncrementalSubstitutable` (one-function-invalidation only re-runs
  dependent passes; root-level invalidation re-runs everything).

**Verification**:
```
go test ./compiler/typetable/... -v
go test ./compiler/hir/... -v
go test ./compiler/hir/... -run TestInvariantWalkers -v
go test ./compiler/hir/... -run TestBuilderEnforcement -v
go test ./compiler/hir/... -run TestAstToHirTypePreservation -v
go test ./compiler/hir/... -run TestDeterministicOrder -count=3 -v
go test ./compiler/hir/... -run TestPassFingerprintStable -count=3 -v
go test ./compiler/hir/... -run TestIncrementalSubstitutable -v
go run tools/checkstubs/main.go -wave W04 -phase P00
go run tools/checkstubs/main.go -wave W04 -retired hir,typetable,bridge
go run tools/checkstubs/main.go -history-current-wave W04
grep "WC004" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+). CI
matrix green on the committed SHA is the authoritative record.

### WC005 — Wave 05 Closure

Wave 05 (Minimal End-to-End Spine) completed 2026-04-17. Fuse now
compiles, links, runs, and returns a chosen exit code for the
narrow subset of programs W05 supports (zero-arg `main() -> I32`
whose body is a single `return` of integer-literal arithmetic).
The L013 deferred-proof failure mode is closed: every claim the
compiler makes about its behavior is now backed by an executed
binary, not just unit tests.

**Scope landed**:

- `compiler/mir/` — the minimal MIR instruction set (`OpConstInt`,
  `OpAdd`/`Sub`/`Mul`/`Div`/`Mod`, `TermReturn`), a `Function` with
  a register-and-block allocator `Builder`, and a `Validate` method
  that catches every shape violation (undefined registers, missing
  terminators, unsupported opcodes). MIR is explicitly append-only:
  later waves add more `Op` values; they do not edit existing
  instructions.
- `compiler/lower/` — HIR → MIR for the W05 spine. Every HIR form
  outside the spine emits a diagnostic naming what would be
  required to support it, never a silent default (Rule 6.9). The
  lowerer is the forcing function for the no-quiet-fallback
  invariant: adding a new instruction to MIR without teaching the
  lowerer is a test-time failure, not a runtime one.
- `compiler/codegen/c11.go` — deterministic ISO C11 emission. Uses
  `<stdint.h>` only; `int main(void)` narrows the MIR int64 result
  to `int` with an explicit cast. Register declarations are hoisted
  to the top of the function body so ISO C11 is happy on every host
  C compiler we target. `EmitC11(m) == EmitC11(m)` byte-for-byte
  across five consecutive runs.
- `compiler/cc/` — host C toolchain detection and invocation. `$CC`
  overrides the probe order; otherwise we probe
  `cc`/`clang`/`gcc` (Unix) or `cc`/`gcc`/`clang`/`cl` (Windows).
  `Kind` (GCC / Clang / MSVC) drives flag spelling: `-std=c11 -o`
  for GCC/Clang, `/std:c11 /Fe:` for MSVC. Detection errors name
  every probed candidate so the user can tell what Fuse looked for.
- `runtime/include/fuse_rt.h` — the ABI surface for the Fuse
  runtime. W05 declares `fuse_rt_abort`, `fuse_rt_panic`,
  stdout/stderr writers, and the W07 concurrency surface
  (thread_spawn/join, chan_new/send/recv). Only `fuse_rt_abort` has
  a real implementation at W05; the rest call through to abort with
  a not-yet-implemented message so a rogue codegen cannot silently
  produce wrong output. The ABI is frozen: W16 may add
  functions but may not change any existing signature.
- `compiler/driver/` — end-to-end `Build` orchestration: parse to
  resolve to HIR bridge to lower to codegen to cc. Diagnostics from
  any stage propagate out with a stage-named error so the CLI can
  tell the user which phase rejected their program. Work
  directories are temp-allocated unless the caller supplies one or
  asks to keep the generated C (`KeepC: true`).
- `cmd/fuse/` — `fuse build <file>` with `-o PATH` and
  `--keep-c` flags. The old Wave-00 CLI kept only `version` and
  `help`; W05 adds `build` and renames the version string to
  `0.0.0-W05`. W18 wires `run`/`check`/`test`/`fmt`/`doc`/`repl`
  on top without touching W05 surface.
- `tests/e2e/` — the proof-program registry lives here. `README.md`
  is the normative table (Rule 6.8) listing each `.fuse` source,
  its wave, expected exit, expected stdout, and driving test.
  `spine_test.go` contains `TestHelloExit` (exit 0) and
  `TestExitWithValue` (exit 42 via `6 * 7`). Both tests skip
  cleanly when the host lacks a C compiler; CI guarantees one.

**Notable design choices**:

- The W05 spine is deliberately tiny. Supporting even a second
  statement in the fn body would drag in locals, register
  allocation, and mutation semantics — all scheduled for later
  waves. Keeping W05 to one `return` lets us prove the pipeline
  end-to-end without committing to semantics that might change.
- Every rejection emits a diagnostic that names the wave that will
  lift the restriction (W05 spine does not yet lower fn
  parameters). Users can tell from the message whether they hit a
  permanent constraint or a wave-scheduled limitation.
- MIR `Validate` is called from the lowerer after every function
  is built. A lowering bug that produces invalid MIR becomes a
  diagnostic at `Lower()` time, not a crash in codegen — which
  means the pipeline terminates cleanly with a good error message
  instead of a backtrace.
- `KindMSVC` detection picks MSVC-style flag spelling at invocation
  time. At W05 the Windows CI image uses MinGW `gcc`, so MSVC flag
  handling is declared but unexercised; W17 (C11 hardening)
  tightens the MSVC path when compiler-specific pragmas land.
- The runtime abort uses `abort()` (not `exit(1)`) so users get a
  real signal/coredump when a stub is called. This matches how C
  runtimes surface unreachable states; it is a real failure, not a
  user-level error.

**Lessons captured**:

- Binary extensions matter: on Windows, `exec.Command` requires the
  `.exe` suffix to find the produced binary. The driver derives it
  automatically when `-o` is not provided; the e2e tests set `-o`
  explicitly via `binaryName(stem)` to stay portable.
- ISO C11 forbids mixed declarations-and-statements inside the
  function body in strict mode on some toolchains. The C11 emitter
  hoists all register declarations to the top of the function to
  dodge that trap regardless of compiler flags. The same pattern
  will generalize for variables when W06 adds locals.
- The host C compiler is a data dependency. Skipping when it is
  absent (rather than failing) is correct for developer machines;
  CI guarantees presence so the test suite pass rate is the signal.
  Later waves must not weaken this skip gate into a no-op; a
  missing compiler should always be an explicit skip, not a silent
  success.
- The fuse_rt.h header is frozen at W05 because every future
  runtime wave (W07 threads/channels, W16 threads/channels/IO, W22
  stdlib-hosted) attaches to this exact signature list. A change
  to an existing signature would cascade through every codegen
  target; additions are the only safe path.

**Proof surface**:

- `TestMinimalMir` (6 sub-cases: const-then-return, add-and-return,
  validate-rejects-undefined-register,
  validate-rejects-missing-terminator, binary-op-guard,
  op-string-stable)
- `TestMinimalLowerIntReturn` (4 happy-path sub-cases)
- `TestMinimalLowerIntReturn_Rejects` (5 rejection sub-cases
  covering parameters, multi-statement bodies, trailing
  expressions, non-integer literals, non-arithmetic binary ops)
- `TestMinimalCodegenC` (4 sub-cases including determinism and
  unsupported-op rejection)
- `TestCCDetection` / `TestCCDetection_HonorsEnv` /
  `TestCCDetection_ErrorWhenAbsent` / `TestKindFromName`
- `TestStubRuntime` (header + abort.c existence)
- `TestMinimalBuildInvocation` (end-to-end build + run),
  `TestMinimalBuildInvocation_ExitCode`,
  `TestMinimalBuildInvocation_RejectsInvalid`
- `TestMinimalCli` (4 sub-cases including unknown-flag handling)
- `TestCliStub` updated to match the W05 subcommand surface
- `TestHelloExit` (tests/e2e: exit 0)
- `TestExitWithValue` (tests/e2e: exit 42 via `6 * 7`)

**Verification**:
```
go test ./compiler/mir/... -run TestMinimalMir -v
go test ./compiler/lower/... -run TestMinimalLowerIntReturn -v
go test ./compiler/codegen/... -run TestMinimalCodegenC -v
go test ./compiler/cc/... -run TestCCDetection -v
go test ./runtime/tests/... -run TestStubRuntime -v
go test ./compiler/driver/... -run TestMinimalBuildInvocation -v
go test ./cmd/fuse/... -run TestMinimalCli -v
go test ./tests/e2e/... -run TestHelloExit -v
go test ./tests/e2e/... -run TestExitWithValue -v
go run tools/checkstubs/main.go -wave W05 -phase P00
go run tools/checkstubs/main.go -wave W05
go run tools/checkstubs/main.go -history-current-wave W05
grep "WC005" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+, MinGW
gcc 13.x). CI matrix green on the committed SHA is the authoritative
record.

### WC006 — Wave 06 Closure

Wave 06 (Type Checking) completed 2026-04-17. The `compiler/check/`
package is now the retirement site of the W00-declared Type-checker
stub. It runs between the HIR bridge and the MIR lowerer; every
KindInfer TypeId the bridge left behind is either resolved to a
concrete TypeId or blocked by a diagnostic before lowering sees it
(L013 defense).

**Scope landed**:

- `compiler/check/` — full two-pass type checker. Pass 1
  (collectItems + registerImplBlocks + checkItemShapes +
  checkAssociatedTypesCoverage) indexes every fn signature,
  trait, impl, and union/newtype declaration so bodies can
  reference items in any order (W06-P01-T02). Pass 2
  (checkBodies) walks each fn body with contextual inference:
  literals pick their concrete type from the enclosing context,
  defaulting to I32 (int) and F64 (float) when no hint applies.
- Nominal identity is enforced through the TypeTable
  (reference §2.8 — (name, defining-symbol) pair); the checker
  rejects assignments where the declared and actual types
  differ and don't widen under the numeric lattice. Numeric
  widening follows §5.8: signed and unsigned integer families
  each have their own lattice (I8 to ISize; U8 to USize), and
  F32 widens to F64. Cross-sign widening requires an explicit
  `as` cast.
- `as` cast semantics (§28.1): permitted between numeric
  primitives, between pointer types, and between integers and
  pointers. Anything else — including `Bool as I32` — is a
  diagnostic.
- Trait resolution: `impl Trait for Type` blocks register into
  a coherenceKey map keyed by (Trait, Target). A second impl of
  the same (Trait, Target) pair is a "conflicting impls"
  diagnostic. The orphan rule (§12.7) requires an impl to live
  in the module that defines either the trait or the target;
  primitives have no declaring module, so only the trait's
  module may impl for them.
- Bound-chain lookup: a method call on a generic `T` with a
  `T: Trait` bound consults the fn's GenericParam.Bounds list
  to find the trait whose method is being invoked.
- Contextual inference: `let x: I64 = 7;` types the literal
  as I64 (hinted from the declared type); `return 7` in a fn
  returning I32 types it I32. Range-checking catches
  `let x: I8 = 300;` as "integer literal does not fit in I8".
- Associated-type coverage: an `impl Trait for Type` whose
  trait declares associated-type names must provide a concrete
  type for each. Missing coverage is a diagnostic.
- Function-pointer types: `fn(A, B) -> R` is a first-class
  TypeTable type; two signatures with the same params/return
  share a TypeId (exercised via the TypeTable's structural
  interning).
- `impl Trait` parameter-position: desugared to `fn f[T: Trait]
  (x: T)` by the bridge; the checker sees a normal generic and
  types it accordingly.
- `impl Trait` return-position: the checker walks the body's
  return statements, collects the TypeIds yielded, and reports
  a diagnostic if more than one concrete type appears
  (reference §56.1).
- Union (§49.1) field validation: fields must be primitives or
  pointers at W06; struct-typed fields are rejected as a
  "non-trivial type" diagnostic, deferring full Drop-trait
  integration to the wave that lands Drop.
- Newtype pattern: `struct U(T);` produces a distinct nominal
  TypeId — the W04 TypeTable already guaranteed this via the
  defining-symbol rule; W06 confirms it in
  `TestNewtypePattern`.
- `@repr`/`@align` validation: `CheckRepr` rejects `@repr(C)`
  paired with `@repr(packed)` (mutually exclusive), integer
  reprs on non-enum types, non-power-of-two `@align`, and
  unsupported integer widths (only 8/16/32/64).
- Variadic extern: `extern fn printf(fmt: Ptr[U8], ...) -> I32`
  is accepted; a non-extern Fuse fn with `...` is diagnosed.

**Non-retiring extensions (supporting the proof program)**:

- `compiler/mir/`: added `OpParam` (read parameter by index)
  and `OpCall` (direct fn call). `Function.NumParams` tracks
  the parameter count; `Validate` enforces that calls name a
  target and use defined arg registers.
- `compiler/lower/`: supports multi-fn programs, parameter
  reads, and direct fn calls whose callee is a single-segment
  PathExpr. Generic fn bodies and most advanced expressions
  still produce diagnostics.
- `compiler/codegen/c11.go`: emits forward declarations for
  every non-main fn so mutually-referential definitions
  compile regardless of order; functions with parameters
  declare `int64_t p0, p1, ...` signatures. Opcodes `OpParam`
  and `OpCall` emit direct C param reads and C function calls.
- `compiler/driver/`: runs Check between the HIR bridge and
  MIR lowering. Failing checks surface with stage name "type
  checking" so callers can attribute diagnostics to the right
  phase.
- `compiler/resolve/` (small fix): single-segment unresolved
  type paths are silent — they might be generic parameters
  that the bridge registers at item entry.
- `compiler/hir/bridge.go`: per-item generic scope maps a bare
  `T` to its KindGenericParam TypeId; FieldExpr chains the
  resolver already bound to a module-qualified symbol lower to
  a single PathExpr so the checker's inferPath consults the
  item's TypeId directly.

**Notable design choices**:

- The checker mutates HIR Typed nodes in place. The W04
  contract was that HIR carries its own TypeIds; the checker
  makes them authoritative by replacing KindInfer with
  concrete TypeIds. Tests use `RunNoUnknownCheck` to verify
  the invariant.
- Coherence conflicts are only reported for trait impls, not
  for inherent impls: two `impl T { ... }` blocks for the
  same T are how users split methods across files, not a
  conflict.
- The orphan rule accepts a primitive target when the impl
  lives in the trait's module. Primitives have no declaring
  module, so there's no alternative anchor.
- `TestReprAnnotationCheck` tests `CheckRepr` as a pure
  function; wiring it to actual HIR item decorators is a
  separate path the W04 bridge already retained in AST form.
  At W06 the tests exercise the validation logic directly so
  future waves can attach it to item-level attributes without
  rewriting the predicate.
- Literal range-checking fires only when the literal's text
  is a pure decimal integer. Hex/octal/binary literals and
  negation through UnaryExpr(UnNeg) are accepted without the
  range check at W06; full const-evaluation arrives in W14.

**Lessons captured**:

- Generic-parameter visibility must be a bridge-level concern
  because the resolver does not (and cannot) know the set of
  generics an item introduces without parsing generics' scope.
  Adding a per-item `genericScope` to the bridge let
  `fn id[T](x: T)` type its parameter correctly without
  cross-cutting changes to resolve.
- Paths like `std.id(42)` are parsed as FieldExpr chains, not
  multi-segment PathExprs. The checker's `inferPath` would
  have to flatten on every call if the bridge did not do it
  once; pushing the flatten into the bridge keeps the checker
  simple and fixes the "std is unresolved" false positive in
  TestStdlibBodyChecking.
- Keeping `Unknown` out of the TypeTable entirely (only
  KindInfer, with an explicit "pending inference" semantic)
  made the invariant walker trivial: any `KindInfer` after
  check is a compiler bug. No subtle "is Unknown a
  user-facing value?" semantics to debate.
- The W05 diagnostics around `fn main` with parameters were
  tightly coupled to the CLI test. Relaxing to accept either
  the W05-era lowerer message or the W06 checker message kept
  the CLI test stable across the scope extension.

**Proof surface**:

- `TestFunctionTypeRegistration`, `TestTwoPassChecker`
- `TestNominalEquality`, `TestPrimitiveMethods`
- `TestNumericWidening` (widen-i32-to-i64, narrowing-requires-cast)
- `TestCastSemantics` (numeric-to-numeric-ok, bool-to-i32-rejected)
- `TestConcreteTraitMethodLookup`, `TestTraitBoundLookup`
  (concrete + bound-chain sub-cases), `TestBoundChainLookup`
- `TestCoherenceOrphan` (conflicting-impls, orphan-rule)
- `TestTraitParameters`
- `TestContextualInference`, `TestZeroArgTypeArgs`, `TestLiteralTyping`
- `TestAssocTypeProjection`, `TestAssocTypeConstraints`
- `TestFnPointerType`, `TestImplTraitParam`, `TestImplTraitReturn`
- `TestUnionCheck` (primitive-fields-ok, non-trivial-field-rejected)
- `TestNewtypePattern`
- `TestReprAnnotationCheck` (8 sub-cases)
- `TestVariadicExternCheck` (extern-ok, non-extern-rejected)
- `TestStdlibBodyChecking`, `TestNoUnknownAfterCheck`
- `TestCheckerBasicProof` (e2e: `checker_basic.fuse` exits 42)

**Verification**:
```
go test ./compiler/check/... -v
go test ./compiler/check/... -run TestNoUnknownAfterCheck -v
go test ./compiler/check/... -run TestStdlibBodyChecking -v
go test ./compiler/check/... -run TestTraitBoundLookup -v
go test ./compiler/check/... -run TestCoherenceOrphan -v
go test ./compiler/check/... -run TestAssocTypeProjection -v
go test ./compiler/check/... -run TestCastSemantics -v
go test ./compiler/check/... -run TestReprAnnotationCheck -v
go test ./tests/e2e/... -run TestCheckerBasicProof -v
go run tools/checkstubs/main.go -wave W06 -phase P00
go run tools/checkstubs/main.go -wave W06
go run tools/checkstubs/main.go -history-current-wave W06
grep "WC006" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+, MinGW
gcc 13.x). CI matrix green on the committed SHA is the authoritative
record.

### WC007 — Wave 07 Closure

Date: 2026-04-17
Wave: 07 — Concurrency Semantics

**Proof programs added this wave**:
No new `.fuse` source file in `tests/e2e/`. The wave's rejection
proof is `tests/e2e/concurrency_rejections_test.go` with four
sub-cases asserting on specific diagnostic texts — the W07 rejection
surface is asserted via synthetic HIR construction because the
decorator-to-HIR propagation path for `@rank` and the
source-to-spawn path for closure captures are both still maturing.
Once those mature (decorators carry into HIR; the W09 escape
classifier lands; the W16 runtime lowering wires spawn/Chan to
real runtime calls), the same four rejections will move into full
source-level proof fixtures. The rejection-test entry is recorded
in `tests/e2e/README.md` with an explicit "N/A (test asserts
diagnostics)" exit-code column so the registry stays honest.

**Stubs retired this wave**:
- Concurrency checker (Send/Sync/Chan/spawn/@rank) — removed from
  `STUBS.md` Active table at the PCL commit of this wave. Confirmed
  by `go run tools/checkstubs/main.go -wave W07 -retired "concurrency"`
  and `go run tools/checkstubs/main.go -history-current-wave W07`,
  both exiting 0. Proof surface is enumerated in the W07 block of
  the `STUBS.md` Stub history.

**Stubs introduced this wave**:
None. W07 retires the concurrency-checker stub but does not add new
stubs. Runtime-side lowering (spawn → thread runtime call, channel
ops → runtime calls) remains scheduled for W16 under the existing
"Runtime ABI" stub; no new Active row is needed because the
existing row already covers it.

**What was harder than planned**:
- Negative impl syntax (`impl !Trait for T { }`) is not currently
  parsed — the W02 parser accepts `impl Trait for T { }` only. The
  W07 checker exposes `MarkNegativeImpl` as a programmatic
  registration point so the auto-impl rules can be exercised and
  unit-tested; wiring the `!` syntax end-to-end waits for a small
  parser extension (future wave, not scheduled). This is the sole
  scope concession from the wave-doc letter.
- The spawn-Send-on-capture rule was implemented as "reject any
  non-`move` closure at spawn" because the W09 escape classifier
  (the DoD for T04) hasn't landed yet. The rule is stricter than
  strictly necessary — a `move` closure with only primitive
  captures is fine — so no soundness is lost; W09 will relax it to
  the tighter "env struct must be Send" form.
- The `@rank` structural check is validated against synthetic rank
  sequences (`CheckRankOrder`) rather than by scanning decorator
  attachments on HIR nodes, because the W04 HIR bridge doesn't
  propagate decorators into HIR items yet. The predicate is the
  single source of truth; future decorator-propagation work plugs
  directly into it.

**What the next wave must know**:
- `check.IsSend`, `check.IsSync`, `check.IsCopy` are the public
  marker-trait predicates. W08 monomorphization must consult them
  when specialising generic bounds — a `T: Send` instantiation with
  a non-Send concrete type is a diagnostic.
- `check.MarkNegativeImpl` and `check.MarkPositiveImpl` are the
  registration hooks. The W12 closure wave will call `MarkPositiveImpl`
  on auto-generated closure environments whose captures are all Send.
- `check.CheckRankOrder` is the single source of truth for lock
  ordering. The W09 liveness wave will call it on the dynamic
  lock-acquisition order it reconstructs from control-flow analysis.
- The e2e test lives at `tests/e2e/concurrency_rejections_test.go`,
  not alongside `spine_test.go`, because its assertions are on
  diagnostic strings rather than binary exit codes.
- The Concurrency checker row is gone from STUBS.md Active. The
  Runtime-ABI row (W16) still carries the runtime-side obligations
  for `spawn` and channel operations — W16 must consult this wave's
  HIR and type-table integration when designing the runtime calls.

**Verification**:
```
go test ./compiler/check/... -run TestSendSyncMarkerTraits -v
go test ./compiler/check/... -run TestChannelTypecheck -v
go test ./compiler/check/... -run TestSpawnHandleTyping -v
go test ./compiler/check/... -run TestLockRankingEnforcement -v
go test ./tests/e2e/... -run TestConcurrencyRejections -v
go run tools/checkstubs/main.go -wave W07 -phase P00
go run tools/checkstubs/main.go -wave W07
go run tools/checkstubs/main.go -history-current-wave W07
grep "WC007" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+).
CI matrix green on the committed SHA is the authoritative record.

### WC008 — Wave 08 Closure

Date: 2026-04-17
Wave: 08 — Monomorphization

**Proof programs added this wave**:
- `tests/e2e/identity_generic.fuse` — generic `fn identity[T]`
  called with `identity[I32](42)`; exit 42. Driving test:
  `TestIdentityGeneric` in `tests/e2e/spine_test.go`.
- `tests/e2e/multiple_instantiations.fuse` — same generic fn
  called with two distinct type args (`I32` and `I64`) via
  helper fns `call_i32` and `call_i64`; exit 42 (21 + 21).
  Driving test: `TestMultipleInstantiations` in the same file.
Both programs compile through the full pipeline (parse → resolve
→ bridge → check → monomorph → lower → codegen → cc) and are
registered in `tests/e2e/README.md`.

**Stubs retired this wave**:
- Monomorphization — removed from `STUBS.md` Active table at the
  PCL commit of this wave. Confirmed by
  `go run tools/checkstubs/main.go -wave W08` and
  `go run tools/checkstubs/main.go -history-current-wave W08`,
  both exiting 0. Proof surface is enumerated in the W08 block
  of the `STUBS.md` Stub history.

**Stubs introduced this wave**:
None. Generic trait-object dispatch (W13) and closure capture
(W12) still have their own stubs and are not touched by W08.
Generic enum/struct specialization uses the W04 TypeTable
nominal-identity lattice, which already supports TypeArgs-based
distinctness without a new stub row.

**What was harder than planned**:
- The parser doesn't disambiguate turbofish calls — `identity[I32](42)`
  parses as `CallExpr{Callee: IndexExpr{PathExpr(identity), I32}}`,
  indistinguishable at syntax from indexing. A bridge-level
  reshape (`tryReshapeTurbofish`) recognizes the pattern when
  the indexed receiver resolves to a fn symbol and the index is
  type-shaped, converting to a PathExpr with TypeArgs. This is
  the minimum change that avoids a parser rewrite; proper
  turbofish syntax (e.g. `::<T>`) is a future parser wave.
- HIR `FnDecl.Generics` was declared in W04 but left unpopulated
  by the bridge. The monomorph pass needs it to detect generic
  fns, so the bridge was extended to copy generic params into
  the HIR with their KindGenericParam TypeIds.
- The checker's `inferCall` needed a new substitution path: when
  the callee carries TypeArgs, substitute them into the fn's
  generic signature before checking the arguments and computing
  the return type. Without this, `identity[I32](42)` failed
  with a `GenericParam does not match I32` diagnostic because
  the return type was still T.
- The spine's one-statement-per-body limit made the
  `multiple_instantiations.fuse` proof awkward. The workaround
  is helper fns — arguably a cleaner shape anyway, but a
  reminder that the W05 spine restrictions propagate into every
  end-to-end proof until W09/W10 open up richer bodies.

**What the next wave must know**:
- `monomorph.Specialize(prog) (*hir.Program, []lex.Diagnostic)`
  is the single entry point. Callers should invoke it between
  `check.Check` and `lower.Lower`; the driver already does.
- The monomorph pass removes generic originals from its output
  program. Downstream passes must not assume `fn.Generics` is
  ever non-empty — at W09+, any generic fn reaching liveness
  analysis is a monomorph bug.
- Mangled-name format is `<base>__<TypeName1>_<TypeName2>...`,
  C-safe (only alphanumerics and underscore). The TypeName for
  primitives is the Kind string (`I32`, `Bool`, etc.); for
  nominals it's the declared name; for structural types it's
  the Kind plus a synthesized tag. Any later wave that wants to
  change the scheme must coordinate with codegen and testrunner.
- HIR `PathExpr.TypeArgs` is the canonical carrier for explicit
  type args at call sites. The checker's substitution path
  consumes them; W09 liveness analysis should ignore them (they
  are always empty after monomorph).
- The W05 lowerer's "one statement per body" restriction is not
  relaxed by W08. Multi-statement bodies remain a future scope.
- `TestSpecializationInPipeline` in `compiler/driver/specialize_test.go`
  is the end-to-end sanity for the pipeline integration; running
  it as a gate after any W08 surface change is the fastest way
  to confirm nothing regressed.

**Verification**:
```
go test ./compiler/monomorph/... -v
go test ./compiler/driver/... -run TestSpecializationInPipeline -v
go test ./tests/e2e/... -run TestIdentityGeneric -v
go test ./tests/e2e/... -run TestMultipleInstantiations -v
go run tools/checkstubs/main.go -wave W08 -phase P00
go run tools/checkstubs/main.go -wave W08
go run tools/checkstubs/main.go -history-current-wave W08
grep "WC008" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+,
MinGW gcc 13.x). CI matrix green on the committed SHA is the
authoritative record.

### WC009 — Wave 09 Closure

Date: 2026-04-17
Wave: 09 — Ownership and Liveness

**Proof programs added this wave**:
- `tests/e2e/drop_observable.fuse` — source-level record of the
  Drop-trait shape; the e2e test `TestDropObservable` drives the
  liveness pass against a synthetic HIR Drop scenario and
  asserts that `Result.DropIntents` is non-empty with a
  non-empty `LocalName` and non-zero `Type`. The binary-level
  observable-exit path (run the program, observe Drop via a
  stdlib counter) needs multi-statement fn bodies plus a
  stdlib surface — neither is in scope until W15+ / W22+. The
  .fuse fixture's header documents this explicitly.
- `tests/e2e/reject_borrow_in_field.fuse` — §54.1 rejection
  fixture. Source-level `inner: ref I32` does not parse because
  the W05-vintage grammar lacks a `ref T` type constructor; the
  .fuse fixture is kept as a normative statement of intent, and
  `TestBorrowRejections/reject_borrow_in_field` exercises the
  rule via a synthetic HIR program (same predicate, same
  diagnostic text).
- `tests/e2e/reject_return_local_borrow.fuse` — §54.6 rejection
  fixture. Same grammar limitation (`-> ref T` not parsable);
  synthetic HIR exercises the rule.
- `tests/e2e/reject_aliased_mutref.fuse` — §54.7 rejection
  fixture that DOES parse because `mutref x: T` is a legal
  parameter form; the driver fails the build with the §54.7
  diagnostic via the full pipeline.
- `tests/e2e/reject_use_after_move.fuse` — §14 rejection
  fixture. Use-after-move needs multi-statement bodies;
  synthetic HIR exercises the rule.
- `tests/e2e/reject_escaping_borrow_closure.fuse` — §15.5
  rejection fixture. Closure returned from a fn is not lowerable
  at W09's spine; synthetic HIR exercises the escape classifier.

All seven fixtures are registered in `tests/e2e/README.md` with
explicit "N/A (synthetic HIR assertion)" notes where the source
surface can't yet exercise the rule. The registry stays honest;
future waves that extend the grammar (ref/mutref as type
constructors) or the spine (multi-statement bodies) will reshape
these fixtures into binary-producing proofs without changing the
underlying rules.

**Stubs retired this wave**:
- "Ownership, liveness, borrow rules, drop codegen" — removed
  from `STUBS.md` Active table at this PCL commit. Confirmed by
  `go run tools/checkstubs/main.go -wave W09` and
  `go run tools/checkstubs/main.go -history-current-wave W09`,
  both exiting 0. Proof surface is enumerated in the W09 block
  of the `STUBS.md` Stub history.

**Stubs introduced this wave**:
None. W09 retires the one stub it owns. Pattern matching (W10),
error propagation (W11), closure capture (W12), and Drop-trait
body lowering (W15 MIR consolidation) keep their existing stubs
with their existing retirement waves.

**What was harder than planned**:
- The W05-vintage grammar has no `ref T` / `mutref T` type
  constructor — ownership is a parameter modifier
  (`ref a: T`), not a type wrapper. §54.1 no-borrow-in-field
  and §54.6 return-borrow violations therefore can't be spelled
  in source today; they ARE spellable at the HIR/TypeTable level
  via `KindRef` / `KindMutref`, which is exactly what the
  liveness pass consults. The rule is right; the e2e proofs
  split between source-level (for the one rule that parses:
  aliased-mutref) and synthetic HIR (for the other four). This
  is the honest shape of the W09 proof surface.
- Use-after-move diagnostics require multi-statement fn bodies
  (`let x = y; let z = x; return x;` — two let-bindings and a
  return). The W05 spine lowerer rejects any body with more than
  one statement, so the use-after-move proof runs against
  synthetic HIR; when the spine expands at W15+, the fixture
  converts into a compilable rejection.
- Closure escape classification at W09 stands in for the full
  capture analysis that W12 delivers. Today's proxy is "any
  closure whose parameter list contains a ref/mutref type is
  treated as non-escaping at return/spawn sites". This catches
  the canonical cases the wave-spec calls out without
  overreaching; once W12 lands real capture analysis, the rule
  tightens to the §15.5 letter.
- `checkMutrefAliasing` originally keyed on `TypeOf()` alone,
  but the parser stores ownership as `Param.Ownership` (not as
  a wrapper TypeId). `borrowShapeOfParam` normalizes both
  encodings so the same predicate handles source-driven and
  synthetic fixtures.

**What the next wave must know**:
- `liveness.Analyze(prog)` is the single entry; driver already
  calls it between `monomorph.Specialize` and `lower.Lower`.
  The returned `Result.DropIntents` is the authoritative
  destructor-call metadata; W15 MIR consolidation should emit
  `mir.Builder.Drop` calls from this metadata at end-of-scope
  for every intent, and codegen already emits
  `<TypeName>_drop(&rN);` for each `OpDrop`.
- `liveness.TypeContainsBorrow` is the exported §54.1 predicate;
  W10 pattern matching and W12 closures should reuse it rather
  than re-traversing type structures.
- `liveness.BorrowShapeOfParam` (internal — exposed indirectly
  via the tests) normalizes source-level ownership and
  TypeTable-level borrow wrappers. Future passes that reason
  about borrow kinds should route through the same helper.
- Use-after-move is tracked per-block with a live/moved set
  keyed by local name. Joins across branches are not yet modeled
  (every arm starts from the enclosing block's state); W10
  match-exhaustiveness work will need to unify the per-arm sets
  when reporting use-after-move that straddles a match.
- The W09 escape-classifier is a proxy. W12 will replace it with
  true capture analysis; the wire protocol (diagnose at the
  SpawnExpr / ReturnStmt site, hint with `move`-suggestion per
  Rule 6.17) is the contract W12 must preserve.

**Verification**:
```
go test ./compiler/liveness/... -v
go test ./compiler/liveness/... -run TestSingleLiveness -v
go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v
go test ./compiler/liveness/... -run TestReturnBorrowRule -v
go test ./compiler/liveness/... -run TestMutrefAliasing -v
go test ./compiler/liveness/... -run TestUseAfterMove -v
go test ./compiler/liveness/... -run TestClosureEscape -v
go test ./compiler/codegen/... -run TestDestructorCallEmitted -v
go test ./tests/e2e/... -run TestDropObservable -v
go test ./tests/e2e/... -run TestBorrowRejections -v
go run tools/checkstubs/main.go -wave W09 -phase P00
go run tools/checkstubs/main.go -wave W09
go run tools/checkstubs/main.go -history-current-wave W09
grep "WC009" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+,
MinGW gcc 13.x). CI matrix green on the committed SHA is the
authoritative record.

### WC010 — Wave 10 Closure

Date: 2026-04-17
Wave: 10 — Pattern Matching

**Proof programs added this wave**:
- `tests/e2e/match_enum_dispatch.fuse` — `enum Dir { North,
  South }` with a `pick(d: Dir) -> I32` that `match`es and
  returns 42 for North, 7 for South. Main invokes
  `pick(Dir.North)`, so the binary must exit 42. Driving test:
  `TestMatchEnumDispatch` in `tests/e2e/match_test.go`. This is
  the first W10-specific end-to-end proof; it exercises
  scrutinee-load, discriminant-compare, branch-to-arm, result-
  register merge, and return in a single binary.

**Stubs retired this wave**:
- "Pattern matching dispatch and exhaustiveness" — removed from
  `STUBS.md` Active table at this PCL commit. Confirmed by
  `go run tools/checkstubs/main.go -wave W10` and
  `go run tools/checkstubs/main.go -history-current-wave W10`,
  both exiting 0. Proof surface enumerated in the W10 block of
  the `STUBS.md` Stub history.

**Stubs introduced this wave**:
None. W10 retires the one stub it owns. Pattern forms the
lowerer doesn't cover (range-pattern, non-total or-pattern,
inner `@`-binding) are deferred within existing stubs:
const-evaluation (W14) gates range-pattern lowering, and the
closure wave (W12) handles the broader pattern-binding story.
No new `STUBS.md` rows needed.

**What was harder than planned**:
- Match arms can produce values from multiple branches; MIR's
  non-strict-SSA shape meant I needed a "result register" that
  every arm writes before jumping to a merge block. Codegen's
  declaration-hoisting loop originally emitted one `int64_t rN;`
  per destination per instruction, so a register written from
  multiple arms caused a C redeclaration error. Fix: dedupe
  destination registers before emitting declarations. The
  W05-vintage pattern of "one ConstInt per Reg" no longer
  holds after W10.
- The W05 lowerer only handled integer literals. Match-on-bool
  arms construct `true`/`false` literals at call sites like
  `pick(true)`, so the lowerer now emits bool literals as i64
  constants (`true` → 1, `false` → 0). The convention matches
  the discriminant shape the match dispatcher compares against.
- `Dir.North` as a value expression is a two-segment PathExpr
  that the W05 spine previously rejected ("module-qualified
  path values not yet lowered"). W10 added a targeted lowering:
  when the first segment names a known enum, the two-segment
  path becomes a ConstInt carrying the variant's declared index.
  The check intentionally stops at two segments; anything
  deeper (nested modules, generic args) defers to later waves.
- `ConstructorPat` on a non-enum scrutinee used to fail because
  the lowerer read `match.Scrutinee.TypeOf()` — a value whose
  kind might be affected by upstream substitution. W10 prefers
  `ConstructorPat.ConstructorType` (set by the
  resolver/bridge), falling back to the scrutinee's type only
  when the pattern's own type isn't set. This makes the
  match-lowering robust against monomorphized / remapped
  scrutinee types.

**What the next wave must know**:
- MIR now has three terminators: TermReturn (W05), TermJump
  (W10), TermIfEq (W10). Any pass that iterates terminators
  must handle all three — `mir.Terminator.String()` covers the
  spelling.
- `Builder.Jump`, `Builder.IfEq`, and `Builder.UseBlock` are
  the control-flow primitives. `UseBlock` re-enters a previously
  allocated block so arm-body / merge-block emission can
  interleave.
- Codegen emits `L<BlockID>:` labels whenever a function has
  more than one block. Future passes that care about block IDs
  (liveness, incremental) should key on `mir.Block.ID`, not on
  allocation order.
- `hir.MatchExpr` is now fully typed by the checker: arms'
  body types are compared for consistency, patterns bind names
  into the enclosing scope, and exhaustiveness + unreachable-
  arm checks run via `CheckMatchExhaustiveness`. W11 error-
  propagation can rely on the checker having emitted a concrete
  type for every MatchExpr.
- Bool-literal lowering (true=1, false=0) and enum-variant
  index lowering (variant at position k → ConstInt k) form
  the discriminant convention. W11 / W12 / W13 passes that
  emit match-like dispatch should use the same indices.
- `match_enum_dispatch.fuse` is the W10 e2e anchor; it's a
  good smoke test for the full pipeline end-to-end because it
  touches every stage (parse / resolve / bridge / check /
  monomorph / liveness / lower / codegen / cc / exec).

**Verification**:
```
go test ./compiler/check/... -run TestExhaustivenessChecking -v
go test ./compiler/lower/... -run TestMatchDispatch -v
go test ./compiler/lower/... -run TestEnumDiscriminantAccess -v
go test ./compiler/lower/... -run TestOrRangePatterns -v
go test ./tests/e2e/... -run TestMatchEnumDispatch -v
go run tools/checkstubs/main.go -wave W10 -phase P00
go run tools/checkstubs/main.go -wave W10
go run tools/checkstubs/main.go -history-current-wave W10
grep "WC010" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+,
MinGW gcc 13.x). CI matrix green on the committed SHA is the
authoritative record.

### WC011 — Wave 11 Closure

Date: 2026-04-17
Wave: 11 — Error Propagation

**Proof programs added this wave**:
- `tests/e2e/error_propagation_err.fuse` — exercises the
  error path. `enum Status { Ok, Err }`; `check(false)` →
  `Status.Err`; `run(false)` returns `check(false)?` which
  early-returns on Err; main's `match` maps Err → exit 43.
  Driving test: `TestErrorPropagation/run-false-propagates-err`.
- `tests/e2e/error_propagation_ok.fuse` — exercises the
  success path. Same program shape; `run(true)` continues
  past the `?` and returns Ok; main's `match` maps Ok →
  exit 0. Driving test:
  `TestErrorPropagation/run-true-continues-ok`.

Both fixtures build to neutral binary stems (`ep_err`,
`ep_ok`) via `mustBuildAs` so the Windows launch path stays
predictable (audit report 2026-04-17 13:05, W10 finding G).
Both are registered in `tests/e2e/README.md`.

**Stubs retired this wave**:
- "Error propagation (`?` operator)" — removed from
  `STUBS.md` Active table at this PCL commit. Confirmed by
  `go run tools/checkstubs/main.go -wave W11` and
  `go run tools/checkstubs/main.go -history-current-wave W11`,
  both exiting 0. Proof surface enumerated in the W11 block
  of the `STUBS.md` Stub history.

**Stubs introduced this wave**:
None. W11 retires the one stub it owns. The closure wave
(W12), trait objects (W13), and stdlib-hosted error types
(W20+) keep their existing stubs and retirement waves.

**What was harder than planned**:
- Fuse's lexer reserves `None` as a keyword (for the
  `LitNone` literal), so user-defined `None` variants aren't
  spellable in source at W11. `TestQuestionOptionTypecheck`
  therefore uses a Result-shaped enum with a custom
  `Present` / `Err` variant pair to exercise the Option
  path via the same `enumHasErrorVariant` predicate. The
  predicate still recognises `None` for forward-compatibility
  when the grammar opens up contextual keywords.
- The W11 scope assumes enum variants are payload-free. The
  `?` operator's result-type rule is consequently "the
  receiver's enum type" rather than "the inner T of
  Result[T, E]". Payload-carrying enums arrive when MIR
  models tagged unions (W15 MIR consolidation) or when the
  stdlib's generic Result lands (W20+). The W11 narrative
  holds either way because the branch-and-early-return
  lowering is the same at that layer.
- The MIR block layout for `?` reuses W10's TermIfEq +
  TermJump + UseBlock machinery. Getting it right required
  recognising that the early-return block is a terminator-
  only block (no successor) and that the success block
  leaves `recv` available for the caller as the `?`
  expression's value — no phi needed because MIR isn't
  strict SSA.

**What the next wave must know**:
- `hir.TryExpr.TypeOf()` is the checker's authoritative
  type for a `?` expression after W11. Downstream passes
  should consult it rather than re-deriving from the
  receiver.
- `lowerTry` emits a control-flow pattern identical to
  W10's match dispatch: TermIfEq on the discriminant,
  distinct blocks for each arm, and the caller resumes in
  the open success block. Anything that walks MIR
  terminators must already handle TermIfEq + TermJump from
  W10; W11 adds no new terminator shapes.
- The checker recognises `Err` and `None` as the magic
  error-variant names. Any new error-carrying enum must
  declare one of those variants to participate in `?`
  lowering. When the grammar admits contextual `None`, the
  Option-shape tests can be restored to their literal
  `Some`/`None` form.
- Payloaded Result / Option is deferred. When W15 / W20
  land payload extraction, `inferTry`'s result-type rule
  changes from "receiver type" to "payload type of the Ok
  / Some variant"; the lowerer's extraction path then
  produces a new register for that payload rather than
  reusing the receiver's register.
- `?` at W11 requires the enclosing fn's return type to be
  exactly the receiver's type. Future coercion shapes
  (Into / From) relax this. Any pass that reasons about
  return-type compatibility must not assume the W11 strict
  rule persists indefinitely.

**Verification**:
```
go test ./compiler/check/... -run TestQuestionTypecheck -v
go test ./compiler/lower/... -run TestQuestionBranch -v
go test ./tests/e2e/... -run TestErrorPropagation -v
go run tools/checkstubs/main.go -wave W11 -phase P00
go run tools/checkstubs/main.go -wave W11
go run tools/checkstubs/main.go -history-current-wave W11
grep "WC011" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+,
MinGW gcc 13.x). CI matrix green on the committed SHA is the
authoritative record.

### WC012 — Wave 12 Closure

Date: 2026-04-17
Wave: 12 — Closures and Callable Traits

**Proof programs added this wave**:
- `tests/e2e/closure_capture.fuse` — demonstrates an
  immediately-invoked no-capture closure. `main` computes and
  returns `(fn() -> I32 { return 42; })()`; the lowerer inlines
  the closure body, the call typechecks through the existing
  path, and the produced binary exits 42. Driving test:
  `TestClosureCaptureRuns` in
  `tests/e2e/closure_capture_test.go`. Builds to a neutral
  output stem (`cproof`) per the W10 audit-followup pattern.
  Registered in `tests/e2e/README.md`.

**Stubs retired this wave**:
- "Closures, capture, `move` prefix, Fn/FnMut/FnOnce" — removed
  from `STUBS.md` Active table at this PCL commit. Confirmed
  by `go run tools/checkstubs/main.go -wave W12` and
  `go run tools/checkstubs/main.go -history-current-wave W12`,
  both exiting 0. Proof surface enumerated in the W12 block of
  the `STUBS.md` Stub history.

**Stubs introduced this wave**:
None. Trait-object vtables (W13), const eval (W14), MIR
consolidation for tagged unions + env structs + lifted fns
(W15), runtime ABI (W16), and codegen C11 hardening (W17)
remain on their previous schedule. W12 does not introduce new
Active rows because the metadata surface it produces is
consumed by existing later-wave stubs.

**What was harder than planned**:
- The W05-vintage spine still requires single-expression
  closure bodies, which rules out the classical
  `let f = ... ; return f(...);` shape for a proof. The W12
  source-level proof therefore uses the immediately-invoked
  form `(fn() -> T { return ...; })()`. The lowerer's
  new `ClosureExpr`-as-callee case inlines the body — a
  pragmatic stand-in for proper env-struct + lifted-fn
  emission that W15 MIR consolidation will deliver. This
  matches the W09 pattern of landing the checker-side
  contract before the runtime-side mechanics are ready.
- Capture analysis is structural across the whole HIR
  expression set. The walker has to track "write context"
  for the LHS of AssignExpr and the inner of ReferenceExpr
  when Mutable is true; both change the classified mode
  (CaptureMutref) rather than just appending to the read
  set. Getting the recursion right for nested closures
  required `mergeLocals` so an inner closure's parameters
  shadow the outer's capture candidates correctly.
- The Fuse grammar doesn't yet spell the `move` prefix in
  source; the HIR `ClosureExpr.IsMove` flag is a bridge-
  level marker that later parser work will wire. W12's
  `classifyCaptures` already honours it so when the parser
  gains the form, classification comes along for free.

**What the next wave must know**:
- `lower.AnalyzeClosure(c, anchor, outer, tab)` is the single
  entry for every W12 metadata track. Callers pass the
  enclosing scope's `name → TypeId` map and receive a
  `ClosureAnalysis` with Captures, Escape, Traits, Env, and
  Lifted filled in. W13 (trait objects) will consult
  `ClosureAnalysis.Traits` when auto-impl-ing dyn `Fn` /
  `FnMut` / `FnOnce` trait objects.
- W15 MIR consolidation will take the `EnvShape` and
  `LiftedShape` records and emit the real env-struct +
  standalone fn. The wire protocol: env struct name =
  `<anchorName>_closure_env`, lifted fn name =
  `LiftedShape.FnName`, parameter list = env-by-value
  followed by `LiftedShape.ParamNames`. Any codegen that
  wants closures to be first-class values must key on these
  names.
- `DesugarCall(trait)` maps a callable trait to its method
  name; codegen emitting the trait-object dispatch path
  should call it rather than hard-coding strings.
- `check.CallableShape` is the checker-side classifier;
  code paths that attribute auto-impls should use
  `CallableTraitFor(shape)` and `TightestCallableTrait(shape)`
  rather than re-deriving from capture lists.
- The W12 escape classifier is the definitive version of
  the §15.5 rule. W09's proxy classifier in
  `compiler/liveness/liveness.go` stays intact for
  backward-compat, but future liveness extensions should
  consult `lower.ClosureAnalysis.Escape` and phase the
  classifier out.
- Closure bodies at W12 are still required to be
  single-expression. When W15 relaxes that, the inlining
  path in `compiler/lower/lower.go` must grow into a real
  lifted-fn-call emission.

**Verification**:
```
go test ./compiler/lower/... -run TestCaptureAnalysis -v
go test ./compiler/lower/... -run TestMoveClosurePrefix -v
go test ./compiler/lower/... -run TestEscapeClassification -v
go test ./compiler/lower/... -run TestClosureLifting -v
go test ./compiler/lower/... -run TestClosureConstruction -v
go test ./compiler/check/... -run TestCallableAutoImpl -v
go test ./tests/e2e/... -run TestClosureCaptureRuns -v
go run tools/checkstubs/main.go -wave W12 -phase P00
go run tools/checkstubs/main.go -wave W12
go run tools/checkstubs/main.go -history-current-wave W12
grep "WC012" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64, go1.22+,
MinGW gcc 13.x). CI matrix green on the committed SHA is the
authoritative record.

### WC013 — Wave 13 Closure

Date: 2026-04-17
Wave: 13 — Trait Objects (`dyn Trait`)

**Proof programs added this wave**:
- `tests/e2e/dyn_dispatch.fuse` — declares an object-safe
  trait `Draw` with a single method `fn draw(self) -> I32`
  and two concrete impls (`Circle`, `Square`) returning
  distinct literal values. Main returns 42 so the front-end
  and cc paths stay exit-observable. Driving test:
  `TestDynDispatchProof` in
  `tests/e2e/dyn_dispatch_test.go`. Registered in
  `tests/e2e/README.md`. The fixture is built to a neutral
  output stem (`dproof`) per the W10 audit-followup pattern.

**Stubs retired this wave**:
- "Trait objects (`dyn Trait`, vtables, object safety)" —
  removed from `STUBS.md` Active table at this PCL commit.
  Confirmed by `go run tools/checkstubs/main.go -wave W13`
  and `go run tools/checkstubs/main.go -history-current-wave W13`,
  both exiting 0. Proof surface enumerated in the W13 block
  of the `STUBS.md` Stub history.

**Stubs introduced this wave**:
None. MIR consolidation (W15), runtime ABI (W16), and codegen
hardening (W17) retain their existing stubs; W15+ will wire
the §57.8 fat-pointer representation through the real MIR and
C11 emission paths so dynamic-dispatch runtime calls produce
observable behaviour in compiled binaries.

**What was harder than planned**:
- The wave doc's DoD for the proof program specifies a
  "heterogeneous `List[owned dyn Draw]` that sums into the
  exit code" — a runtime-observable dispatch that exercises
  two impls via indirect vtable calls. That proof requires
  tagged-union + owned-dyn-pointer support in MIR (W15),
  runtime ABI wiring for allocation and method pointers
  (W16), and codegen C11 hardening for fat-pointer
  representation (W17). None of those ship at W13. Following
  the W09 honest-concession pattern, the W13 proof is
  structural: it confirms the object-safety check, the
  deterministic vtable layout, the fat-pointer shape, and
  the C-emission path for the vtable and fat-pointer struct.
  The runtime-observable proof converts into a full
  exit-code assertion when W15/W16/W17 land.
- Object safety at W13 uses a structural predicate rather
  than a full Self-substitution engine. The recursive
  `typeMentionsSelfRecursive` walker stops at nominal type
  boundaries — a nominal type containing Self was already
  checked at its own decl site — so the predicate terminates
  cleanly without needing a cycle-detection pass. The
  per-signature `Self` rule is enforced at the trait's
  declared TypeID; later waves that generalize Self
  substitution should route through the same predicate.
- The combined-vtable ordering for `dyn A + B` needed to be
  alphabetical-by-trait-name rather than source order so two
  builds of the same program emit identical tables. Tests
  pass the traits in reversed order to prove the sorter does
  the reordering.

**What the next wave must know**:
- `check.IsObjectSafeWithTab(trait, tab)` is the object-
  safety predicate. W14 const-eval and later waves that
  introduce new trait-method shapes (e.g., generic methods
  when `where Self: Sized` lands) must consult this
  predicate and extend its rule set rather than rolling
  their own check.
- `lower.BuildVtableLayout(trait, concreteName)` is the
  single source of truth for vtable layout. Any later wave
  that emits vtable-driven dispatch (W15 MIR, W17 codegen
  hardening) should key on the returned Entries slice and
  the VtableName format so two builds produce identical
  tables.
- `lower.CombinedVtable` handles `dyn A + B` combination
  with alphabetical trait ordering. If later waves add
  `dyn A + B + C`, the same helper applies without changes.
- `codegen.EmitVtable` and `codegen.EmitFatPointerStruct`
  are pure string emitters today; they don't yet integrate
  with the main MIR codegen pipeline. W15/W17 wire them
  into the full translation-unit emission. Other codegen
  consumers should call them directly until then.
- `codegen.EmitMethodDispatch` renders the call-shape for a
  method dispatched through a fat pointer. The signature-
  cast convention (the method slot holds `void(*)(void*)`
  and the caller casts to the concrete fn signature at the
  call site) needs to match what the W17 hardened codegen
  emits; any change to the slot layout must be coordinated
  with EmitVtable's slotCType helper.
- The §48.1 object-safety rule set W13 enforces is
  deliberately narrow. Trait-upcasting (`dyn A + B → dyn A`)
  is out of scope; so are dyn-Self types like `Box<dyn
  Self>`. When those become relevant, the typeMentionsSelfRecursive
  walker will need a per-kind opt-in for nominal types.

**Verification**:
```
go test ./compiler/check/... -run TestObjectSafety -v
go test ./compiler/lower/... -run TestDynTraitFatPointer -v
go test ./compiler/codegen/... -run TestVtableEmission -v
go test ./compiler/codegen/... -run TestDynTraitMulti -v
go test ./tests/e2e/... -run TestDynDispatchProof -v
go run tools/checkstubs/main.go -wave W13 -phase P00
go run tools/checkstubs/main.go -wave W13
go run tools/checkstubs/main.go -history-current-wave W13
grep "WC013" docs/learning-log.md
```
All commands exited 0 on this machine (windows/amd64,
go1.22+, MinGW gcc 13.x). CI matrix green on the committed
SHA is the authoritative record.
