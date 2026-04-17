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
