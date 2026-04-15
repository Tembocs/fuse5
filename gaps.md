# Fuse Gaps Register

This document is a critical planning-phase audit of every tracked project file at current `HEAD`.

Reviewed files:

- `README.md`
- `docs/rules.md`
- `docs/repository-layout.md`
- `docs/language-guide.md`
- `docs/implementation-plan.md`
- `docs/learning-log.md`
- `docs/fuse-language-reference.md`

The purpose of this register is not to praise the plan. It is to identify every visible gap that can materially reduce the odds of implementation success, increase ambiguity, or allow false progress.

Severity scale:

- `Critical`: implementation should not begin until this is resolved
- `High`: likely to cause major rework, drift, or false completion
- `Medium`: important but can be resolved in parallel with implementation

## 1. Critical Cross-Project Gaps

### G001 — The docs do not clearly distinguish planning-phase obligations from execution-phase obligations
Severity: `Critical`

Affected files:

- `README.md`
- `docs/rules.md`
- `docs/repository-layout.md`
- `docs/implementation-plan.md`
- `docs/learning-log.md`

Gap:

- The docs mix two different layers without explicitly separating them:
  - planning-time source-of-truth documents
  - execution-time operational artifacts such as `STUBS.md`, `tests/e2e/README.md`, CI gates, and wave closure mechanics
- The intended activation point for those execution-time artifacts is not clearly defined.

Why this matters:

- Contributors can read the same docs and come away with different beliefs about what is mandatory now versus what becomes mandatory once implementation begins.
- Planning-phase review becomes noisier because execution-phase workflow rules are interleaved with design-time rules.
- This weakens document clarity before any code exists.

Required fix:

- Add a clear phase model to the docs:
  - planning phase
  - bootstrap execution phase
  - post-bootstrap implementation phase
- For each major artifact or rule set, say when it becomes active and who owns creating it.

### G002 — Authority conflict between the language guide and the language reference
Severity: `Critical`

Affected files:

- `docs/rules.md`
- `docs/language-guide.md`
- `docs/fuse-language-reference.md`
- `docs/repository-layout.md`

Gap:

- `docs/rules.md` says the language guide is normative and silence means absence.
- `docs/fuse-language-reference.md` says “If the implementation and this document disagree, this document is the starting point for resolving the conflict.”
- `docs/repository-layout.md` says the project does not use extra free-form normative status documents and that the five foundational docs are the source of truth.

Why this matters:

- This creates two competing spec authorities.
- An implementer can legitimately justify incompatible behavior from either document.
- A compiler project cannot survive spec ambiguity at the authority level.

Required fix:

- Choose exactly one:
  - Make `docs/fuse-language-reference.md` explicitly non-normative and derivative of `docs/language-guide.md`.
  - Or formally promote it into the normative set and define precedence and synchronization rules.
- Add a rule that no file outside the normative set may define language semantics independently.

### G003 — Repo identity drift: `fuse5` checkout, `fuse4` docs
Severity: `Critical`

Affected files:

- `docs/rules.md`
- `docs/repository-layout.md`
- `docs/language-guide.md`
- `docs/implementation-plan.md`
- `docs/learning-log.md`

Gap:

- The workspace directory is `fuse5`, and git history says “Initial commit of Fuse attempt 5.”
- The foundational docs all declare themselves normative for `fuse4`.
- `docs/repository-layout.md` even defines the future tree as `fuse4/`.

Why this matters:

- This makes it unclear whether the current plan is a carry-over, a fresh reboot, or a document transplant.
- It weakens trust in every “current state” statement.

Required fix:

- Rename the document status lines and repository tree references to match the actual attempt, or explain why the next production attempt is still intentionally named `fuse4`.
- Add a short “attempt lineage” section explaining `fuse3 -> fuse4 -> fuse5` naming and artifact inheritance.

### G004 — The docs blur target architecture description with current planning contract
Severity: `Critical`

Affected files:

- `README.md`
- `docs/repository-layout.md`
- `docs/implementation-plan.md`

Gap:

- Some passages describe the target compiler architecture as settled fact, while other passages describe the future repository or future waves.
- The documents do not consistently distinguish:
  - target-state architecture
  - current planning commitments
  - future implementation artifacts

Why this matters:

- The reader cannot always tell whether a statement is:
  - current project policy
  - future intended repository shape
  - or already-validated implementation fact
- That ambiguity weakens planning discipline even before coding starts.

Required fix:

- Introduce explicit terminology across the docs:
  - `Target architecture`
  - `Current planning contract`
  - `Execution-phase artifact`
- Use those labels consistently so the reader always knows what kind of claim is being made.

### G005 — The project has no canonical “current wave / current phase / current branch of work” state
Severity: `Critical`

Affected files:

- `docs/rules.md`
- `docs/implementation-plan.md`
- `README.md`

Gap:

- The rules require contributors to locate the current wave and phase.
- The repo provides no authoritative pointer to what that current wave is.
- There is no `CURRENT_WAVE.md`, project state file, milestone file, or tracker entry.

Why this matters:

- Multiple contributors can each work against different parts of the plan and all believe they are compliant.
- The plan is detailed, but not operationally anchored.

Required fix:

- Add a single tracked state artifact, for example `PROJECT_STATE.md` or `.omx/current-wave.json`, containing:
  - active wave
  - active phase
  - owner
  - open blockers
  - last reviewed date

### G006 — The implementation plan still permits false completion in several waves
Severity: `Critical`

Affected files:

- `docs/implementation-plan.md`
- `docs/learning-log.md`

Gap:

- The learning log correctly identifies false completion as the main historical failure mode.
- But several waves still defer behavioral proof to later waves:
  - Wave 05 says it has a behavioral proof, then defers real e2e to Wave 09.
  - Wave 07 says match and `?` are behavioral, but the actual proof program is deferred until Wave 09.
- This means a wave can be declared done before its user-visible behavior is runnable.

Why this matters:

- This is exactly the failure mode described in `L013` and `L014`.

Required fix:

- Remove deferred-proof wording from any wave that claims behavior is complete.
- If a behavior cannot be run yet, the wave must say the behavior is not yet complete.
- Introduce explicit intermediate statuses such as:
  - “frontend-complete”
  - “lowering-complete but not runnable”
  - “runnable end-to-end”

### G007 — Wave 05 still contains a monomorphization phase that conflicts with the dedicated generics wave
Severity: `Critical`

Affected files:

- `docs/implementation-plan.md`
- `docs/learning-log.md`

Gap:

- `L015` says generics require a dedicated wave because the earlier attempt to place them inside Wave 05 was structurally wrong.
- But Wave 05 still includes:
  - collecting instantiations
  - validating specialization completeness
  - specializing generic functions
  - integrating monomorphization into the driver pipeline
- Wave 17 then repeats the same work at end-to-end scope.

Why this matters:

- This creates duplicated ownership.
- It weakens the intended “generics are cross-cutting” lesson.
- It encourages implementers to do partial generics work too early, then re-do it later.

Required fix:

- Decide one of two approaches:
  - Keep Wave 05 to checker-only generic semantics and move all specialization/pipeline work to Wave 17.
  - Or move Wave 17 earlier and refactor the whole plan around actual generic integration.

### G008 — There is no explicit wave or phase for the AST-to-HIR bridge, despite it being a documented historical failure source
Severity: `Critical`

Affected files:

- `docs/implementation-plan.md`
- `docs/learning-log.md`

Gap:

- `L013` explicitly names the AST-to-HIR bridge defaulting expression types to `Unknown` as a major hidden failure.
- The implementation plan mentions HIR, checking, and lowering, but does not carve out the AST-to-HIR bridge as its own explicit subsystem with its own proofs and invariants.

Why this matters:

- The bridge is a real integration boundary and has already failed in a way that unit tests missed.
- Leaving it implicit invites the same bug class again.

Required fix:

- Add a dedicated phase or wave for:
  - AST-to-HIR construction from resolved AST
  - type propagation into HIR
  - invariant tests that prove no checked HIR expression loses its resolved type during the bridge

## 2. Cross-Project Execution Gaps

### G009 — Verification commands are heavily Unix-specific despite a required Windows CI matrix
Severity: `High`

Affected files:

- `docs/implementation-plan.md`

Gap:

- Many `Verify:` commands assume Unix shell behavior:
  - `/tmp/...`
  - `PATH="" ...`
  - `grep`
  - `head -20`
  - `diff - <(...)`
  - `cat`
  - chained shell constructs
  - `gcc -x c - -o /tmp/ig`
- The plan also requires Linux, macOS, and Windows CI.

Why this matters:

- A verification plan that is not portable is not a real verification plan for a multi-platform compiler.

Required fix:

- Rewrite all `Verify:` commands into one of:
  - Go test commands
  - project scripts under `tools/`
  - explicit per-platform wrappers
- Ban shell-specific process substitution and ad hoc pipelines in normative verification steps.

### G010 — Several `Verify:` commands are observational or weak, not adversarial proofs
Severity: `High`

Affected files:

- `docs/implementation-plan.md`

Gap:

- Examples of weak verification:
  - “observe green CI”
  - `ls ...`
  - `grep ...`
  - `head -20`
  - `grep -c "Unknown"` as a proxy for semantic correctness
  - checking that a string appears in generated C without also proving runtime behavior

Why this matters:

- These checks can pass while the underlying behavior is still wrong.
- This repeats the self-verifying-plan pattern in a softer form.

Required fix:

- For every task, require one of:
  - unit test proving the invariant
  - e2e proof program
  - deterministic tool under `tools/` that returns non-zero on failure
- Treat plain text inspection commands as supplemental evidence only, never the primary `Verify:` step.

### G011 — No toolchain pinning or host-environment contract is specified
Severity: `High`

Affected files:

- `README.md`
- `docs/repository-layout.md`
- `docs/implementation-plan.md`

Gap:

- The docs require Go, C toolchains, Make, CI on three platforms, and later a native backend.
- There is no canonical toolchain version policy for:
  - Go version
  - C compiler versions
  - supported host OS versions
  - shell assumptions
  - minimum runtime libc/environment assumptions

Why this matters:

- A compiler bootstrap path is extremely sensitive to host toolchain drift.

Required fix:

- Add a “Supported Build Environments” section or dedicated doc specifying:
  - pinned toolchain versions or minimums
  - supported hosts
  - required environment variables
  - reproducibility expectations per host

### G012 — No explicit non-goals or out-of-scope list for v1
Severity: `High`

Affected files:

- `README.md`
- `docs/language-guide.md`
- `docs/implementation-plan.md`

Gap:

- The docs define many desired capabilities, but they do not clearly state what v1 intentionally excludes.
- This is especially dangerous because the language reference introduces far more features than the guide and plan commit to.

Why this matters:

- Teams expand scope silently when the spec surface is aspirational.

Required fix:

- Add a `Non-goals / Not in v1` section in the language guide and README.
- Explicitly classify extra features as:
  - planned later
  - intentionally excluded
  - reference-only ideas

### G013 — No risk register or top-level dependency graph for implementation order
Severity: `High`

Affected files:

- `docs/implementation-plan.md`

Gap:

- The plan is detailed by wave, but it lacks an explicit “highest-risk integration points” register.
- It also lacks a graph of cross-wave dependencies beyond the simple linear wave ordering.

Why this matters:

- The hardest failures in compilers come from hidden cross-cutting dependencies, not from missing task bullets.

Required fix:

- Add a short risk register naming at least:
  - AST-to-HIR bridge correctness
  - generic specialization integration
  - enum layout + pattern lowering
  - drop metadata flow
  - concurrency compiler/runtime integration
  - stage2 parity drift

## 3. File-Specific Gaps

## 3.1 `README.md`

### G014 — README does not explain the project phase model
Severity: `High`

Gap:

- The README does not clearly explain which documents are design-time inputs versus execution-time workflow artifacts.
- It introduces architecture and normative files, but not the phase model that tells a reader how to interpret them.

Required fix:

- Add a short “How to Read This Repo” section explaining:
  - which docs are authoritative now
  - which artifacts are future operational outputs
  - how planning transitions into implementation

### G015 — README onboarding sequence is not phase-aware
Severity: `High`

Gap:

- The onboarding flow lists execution-phase artifacts and planning-phase documents in one flat sequence.
- It does not distinguish “read this now during planning” from “use this once execution begins.”

Required fix:

- Split onboarding into:
  - planning-phase reading order
  - implementation-phase workflow order

### G016 — README has no implementation status summary
Severity: `Medium`

Gap:

- There is no concise snapshot of:
  - current wave
  - current repo maturity
  - what is implemented today
  - what is only specified

Required fix:

- Add a short status matrix near the top.

## 3.2 `docs/rules.md`

### G017 — Rules lack explicit phase-aware activation
Severity: `High`

Gap:

- The rules are written as if every workflow control is always active.
- They do not say which rules activate only once implementation begins or once specific operational artifacts exist.

Required fix:

- Add a short “Rule activation by phase” section that states which rules apply:
  - during planning
  - during Wave 00 bootstrap
  - during implementation waves

### G018 — No rule defines how contradictory documents must be reconciled
Severity: `High`

Gap:

- There is guide precedence, but no process for handling contradiction with non-guide docs that currently claim authority.

Required fix:

- Add a document conflict resolution rule:
  - identify conflict
  - designate owner
  - block implementation of the affected feature until resolved

### G019 — Rules do not define ownership/coherence rules for traits and impls
Severity: `High`

Gap:

- Trait resolution is discussed, but there is no language-level rule for:
  - orphan rules
  - impl overlap
  - coherence

Why this matters:

- These are not optional details in a trait-based systems language.

Required fix:

- Add trait coherence/orphan rules to the language guide and reference them from the rules.

### G020 — No explicit rule for machine-readable project state
Severity: `Medium`

Gap:

- The rules require disciplined progress but rely entirely on prose docs and human memory.

Required fix:

- Add a rule requiring a tracked current-wave/project-state artifact.

## 3.3 `docs/repository-layout.md`

### G021 — The layout doc defines the target tree, but not the ownership and activation rules for getting there
Severity: `High`

Gap:

- It describes the destination tree clearly, but not:
  - when each directory becomes mandatory
  - which wave owns introducing it
  - which directories are target-state only versus immediately normative

Required fix:

- Add an activation/ownership table mapping each top-level path to:
  - introducing wave
  - owning subsystem
  - whether it is required immediately or only once implementation starts

### G022 — The docs claim only five foundational docs are normative, but an extra language reference file already exists
Severity: `High`

Gap:

- This is a live contradiction inside the current tree.

Required fix:

- Either remove `docs/fuse-language-reference.md`, mark it non-normative, or formally include it in the normative set.

### G023 — Package boundaries are described, but interface contracts between major packages are not
Severity: `High`

Gap:

- Example:
  - what exact data flows from resolve to HIR?
  - from check to liveness?
  - from liveness to lower?
  - from monomorph to HIR/lower?

Required fix:

- Add a small package-interface appendix or ADR set for cross-package contracts.

### G024 — No naming conventions for generated artifacts, goldens, fixtures, and proof programs
Severity: `Medium`

Gap:

- The tree names directories, but not how files inside them should be named and organized.

Required fix:

- Add naming conventions for:
  - test fixture files
  - golden files
  - proof program names
  - generated C outputs
  - runtime ABI test files

## 3.4 `docs/language-guide.md`

### G025 — The guide does not fully define v1 scope relative to the much larger language reference
Severity: `Critical`

Gap:

- The guide says features not described here do not exist.
- The reference describes many additional features as ordinary language surface.

Required fix:

- Add a scope boundary section and resolve the reference conflict immediately.

### G026 — Feature status is too coarse to control implementation drift
Severity: `High`

Gap:

- Status is only attached to top-level sections, not the many concrete subfeatures that differ in maturity and scheduling.

Examples:

- `match` semantics
- closures
- optional chaining
- generic type layouts
- channel typing
- FFI surface detail

Required fix:

- Track status at subsection or feature-item level, not just section level.

### G027 — Lexical spec omits BOM handling that the plan requires
Severity: `High`

Gap:

- Wave 01 requires BOM rejection.
- The guide’s lexical section does not currently state the BOM rule.

Required fix:

- Add a lexical contract for BOM handling.

### G028 — Escape and Unicode semantics are delegated to “compiler implementation”
Severity: `High`

Gap:

- String escapes and Unicode escape behavior are not fully specified.
- Identifier Unicode classes are not tied to a concrete Unicode standard/version.

Required fix:

- Define exact Unicode and escape handling rules.

### G029 — Numeric semantics are incomplete
Severity: `High`

Gap:

- The guide does not clearly define:
  - overflow behavior
  - divide-by-zero behavior
  - signed overflow semantics
  - cast semantics for `as`
  - integer literal defaulting in all ambiguous contexts

Required fix:

- Add a numeric semantics section covering arithmetic traps, wrap behavior, checked ops, and casts.

### G030 — Optional chaining semantics are under-specified
Severity: `High`

Gap:

- The guide says optional chaining “propagates failure or absence according to the operand type and language rule of the surrounding expression.”
- That is too vague for implementation.

Required fix:

- Define exact typing and lowering rules for:
  - `Option`
  - `Result`
  - chained accesses
  - method calls
  - interaction with `?`

### G031 — Closure status and semantics are incomplete
Severity: `High`

Gap:

- Expressions mention closures “if enabled by the language version.”
- There is no versioning scheme or enablement mechanism.
- The guide does not define closure types, calling convention, or trait/callable integration.

Required fix:

- Either:
  - fully specify closures as part of v1, or
  - mark them explicitly absent/stubbed until a later wave

### G032 — Pattern matching semantics are incomplete beyond enum dispatch
Severity: `High`

Gap:

- The guide covers dispatch behavior, but not:
  - exhaustiveness checking
  - unreachable-arm detection
  - guard semantics
  - nested pattern typing
  - irrefutable vs refutable contexts

Required fix:

- Expand the pattern section into full static and dynamic semantics.

### G033 — Trait system semantics are incomplete
Severity: `High`

Gap:

- Missing or incomplete:
  - coherence/orphan rules
  - overlap resolution
  - default method conflict rules
  - associated types
  - trait object / dynamic dispatch status
  - object safety rules, if dynamic dispatch is intended

Required fix:

- Define the trait system boundary explicitly.

### G034 — Module system semantics are incomplete
Severity: `High`

Gap:

- The guide does not fully define:
  - root module rules
  - package/module boundaries
  - re-export syntax and semantics
  - visibility model beyond `pub`
  - same-file item ordering rules

Required fix:

- Add complete module/visibility semantics or mark advanced visibility as absent.

### G035 — Concurrency model has a major `spawn` contract ambiguity
Severity: `Critical`

Gap:

- The guide says `spawn` lowers to `fuse_rt_thread_spawn(fn_ptr, env_ptr)` and requires proof of a visible spawned effect.
- The language reference later says `spawn` returns `ThreadHandle[T]` and supports `join()`.
- The guide does not say whether `spawn` is fire-and-forget or handle-returning.

Required fix:

- Decide one concurrency surface and make all docs agree.

### G036 — Concurrency memory model and thread-safety traits are missing
Severity: `High`

Gap:

- The guide does not define:
  - Send/Sync-style traits
  - memory visibility semantics
  - happens-before guarantees
  - lock-ranking enforcement details

Required fix:

- Add a concurrency semantics section, not just syntax/API surface.

### G037 — Error handling semantics for `Option` propagation are incomplete
Severity: `High`

Gap:

- `?` is specified for `Result` clearly.
- `Option` propagation behavior is referenced but not fully spelled out with the same rigor.

Required fix:

- Add the exact `Option[T]` `?` propagation rule with proof examples.

### G038 — The unsafe/FFI chapter is too thin for a systems language
Severity: `High`

Gap:

- Missing or incomplete:
  - calling convention model
  - layout / repr control
  - nullability conventions
  - variadics status
  - pointer casts
  - function pointer ABI
  - extern static semantics

Required fix:

- Either formally exclude these from v1 or specify them completely.

### G039 — Backend contracts are important but incomplete
Severity: `High`

Gap:

- The six backend contracts are strong, but they omit:
  - closure representation
  - trait object representation if supported
  - channel/runtime ABI shapes
  - alignment/padding rules
  - calling conventions
  - enum layout details for struct-like variants

Required fix:

- Add all backend-critical runtime representations that are needed for codegen.

## 3.5 `docs/implementation-plan.md`

### G040 — The plan mixes historical assumed entry states with normative living-plan text
Severity: `High`

Gap:

- Some `State on entry` and `Currently:` lines read like historical snapshots of an imagined repository state rather than stable planning contracts.
- The plan does not clearly distinguish:
  - fixed intended entry conditions for a wave
  - from the actual evolving state of the project as planning changes

Required fix:

- Add wording that makes these fields explicit:
  - `Intended entry state`
  - `Observed project state when wave begins`
- This keeps the plan usable as a living document instead of a frozen narrative.

### G041 — Wave 05 and Wave 07 claim behaviors before runnable proof exists
Severity: `Critical`

Gap:

- Behavioral completion is claimed before the backend/build path exists.

Required fix:

- Rephrase those waves so they only claim pre-runnable completeness, or move their proof obligations earlier with a runnable harness.

### G042 — Wave ordering around generics, self-hosting, and backend retirement is still risky
Severity: `High`

Gap:

- The plan now puts generics before retirement of Go/C, which is good.
- But self-hosting and native backend transition still happen before generics are end-to-end complete, despite the learning log showing that stage2 can self-host while missing important language surface.

Required fix:

- Decide whether self-hosting is allowed to ignore parts of the language surface.
- If yes, say so explicitly and define a “representativeness gap” checklist.
- If no, generics and other missing user-visible features must move earlier.

### G043 — No explicit plan for exhaustiveness checking
Severity: `High`

Gap:

- Pattern matching lowering is planned.
- Exhaustiveness checking is not clearly owned.

Required fix:

- Add exhaustiveness and unreachable-pattern tasks in checking.

### G044 — No explicit plan for visibility/access control semantics
Severity: `Medium`

Gap:

- `pub` exists, but there is no clear wave/phase for full visibility enforcement.

Required fix:

- Add it to resolution/checking.

### G045 — Stage 2 parity is described vaguely
Severity: `High`

Gap:

- “mirrors stage1 architecture closely enough” is not measurable.

Required fix:

- Define a parity checklist:
  - package/module correspondence
  - feature parity
  - known intentional deltas

### G046 — Many Verify steps depend on manual CI observation instead of deterministic commands
Severity: `High`

Gap:

- Examples:
  - “observe green CI”
  - “CI green on all four proof programs”

Required fix:

- Replace manual observation with CI-enforced scripts or explicit workflow checks.

### G047 — There is no explicit documentation synchronization phase
Severity: `Medium`

Gap:

- The project is documentation-driven, but the plan does not contain a recurring task class for reconciling the guide, rules, reference, and layout after each semantic change.

Required fix:

- Add a documentation consistency task to every wave closure.

## 3.6 `docs/learning-log.md`

### G048 — The learning log identifies major failures, but not all of them feed back into explicit plan structure
Severity: `High`

Gap:

- `L013` identifies the AST-to-HIR bridge issue, but the plan does not give it first-class ownership.
- The log is stronger than the plan in some places.

Required fix:

- Add a “traceability back to plan” index mapping each lesson to the exact plan changes that address it.

### G049 — There are no lessons yet about documentation authority drift and verification portability
Severity: `Medium`

Gap:

- Current repo problems include:
  - conflicting spec authority
  - non-portable verify commands
- The log does not yet record those as first-class lessons.

Required fix:

- Add new learning-log entries once these gaps are fixed.

### G050 — The learning log has no planning-phase decision/closure format
Severity: `Medium`

Gap:

- The learning log defines bug entries and wave closures.
- It does not define a planning-phase entry type for:
  - document contradictions resolved
  - scope decisions made before coding
  - design questions closed without code changes
- That leaves an important part of project memory undocumented until implementation starts.

Required fix:

- Add a planning-decision entry format, for example `Pxxx`, for pre-implementation design closures.

## 3.7 `docs/fuse-language-reference.md`

### G051 — The file should not currently be treated as authoritative
Severity: `Critical`

Gap:

- It defines language features independently.
- It lacks the implementation status discipline used by the guide.

Required fix:

- Mark it `non-normative` until synchronized.

### G052 — The file introduces many features not committed in the guide or plan
Severity: `Critical`

Observed examples:

- associated types
- function pointers
- variadic extern functions
- thread handles and joining
- compile-time functions (`const fn`)
- trait objects and dynamic dispatch (`dyn Trait`)
- unions
- conditional compilation
- interior mutability (`Cell`, `RefCell`)
- custom allocators
- visibility granularity (`pub(mod)`)
- callable traits (`Fn`, `FnMut`, `FnOnce`)
- opaque return types (`impl Trait`)

Why this matters:

- These are not small editorial extras. They are major type-system, ABI, and runtime features.

Required fix:

- For each such feature, choose one:
  - add to the guide and plan
  - move to an ideas/backlog doc
  - explicitly mark unsupported in v1

### G053 — The reference contradicts the guide on concurrency surface
Severity: `Critical`

Gap:

- The guide’s `spawn` contract does not define returned handles.
- The reference defines `ThreadHandle[T]`, `join()`, and handle-based parallel mapping.

Required fix:

- Reconcile the concurrency model before any implementation begins.

### G054 — The reference uses “importance” language but no implementation-status language
Severity: `High`

Gap:

- It communicates desirability, not schedule or support level.

Required fix:

- If retained, every section must state:
  - normative status
  - planned wave
  - or “not in v1”

### G055 — The reference expands the keyword surface without corresponding grammar/governance alignment
Severity: `High`

Gap:

- The keyword appendix and examples imply a much broader implemented language surface than the guide and plan support.

Required fix:

- Generate the keyword appendix from the canonical guide, not by hand.

## 4. Missing Design Decisions

These are not merely “nice to have.” They are unresolved implementation decisions that will force inconsistent behavior if left open.

### G056 — Trait coherence and orphan rules
Severity: `Critical`

Required fix:

- Define whether external trait/external type impls are allowed.
- Define overlap resolution and duplicate impl rejection.

### G057 — `spawn` surface and handle semantics
Severity: `Critical`

Required fix:

- Decide whether `spawn` returns:
  - unit / opaque fire-and-forget
  - thread handle
  - `Result[ThreadHandle, E]`

### G058 — Cast semantics for `as`
Severity: `Critical`

Required fix:

- Define numeric casts, pointer casts, trait-object casts, and truncation rules.

### G059 — Exhaustiveness and pattern typing rules
Severity: `Critical`

Required fix:

- Define static checking for patterns before implementing match lowering.

### G060 — FFI ABI surface boundary
Severity: `Critical`

Required fix:

- Define representation, calling conventions, layout control, nullability, and variadic support policy.

### G061 — Concurrency memory model and Send/Sync-like safety model
Severity: `Critical`

Required fix:

- Define the rules before implementing threaded runtime integration.

### G062 — Closure calling convention and representation
Severity: `High`

Required fix:

- Define closure value shape, env layout, lifetime/capture rules, and callable interface.

### G063 — Generic specialization policy across modules and recursion
Severity: `High`

Required fix:

- Define:
  - deduplication boundaries
  - mangling inputs
  - recursive generic emission policy
  - code bloat control expectations

### G064 — Visibility model
Severity: `High`

Required fix:

- Decide whether only `pub/private` exist in v1 or whether finer-grained visibility is supported.

## 5. Immediate Remediation Order

The following should be fixed before implementation work begins:

1. Resolve document authority and make one language spec canonical.
2. Reconcile `fuse4` versus `fuse5` naming.
3. Define the project phase model and document when execution-phase artifacts such as `STUBS.md`, `tests/e2e/README.md`, and wave-closure mechanics become active.
4. Add a canonical current project state artifact naming the active wave and phase.
5. Rewrite non-portable and weak `Verify:` commands into deterministic project-owned scripts/tests.
6. Remove or refactor Wave 05 monomorphization tasks so they do not conflict with Wave 17.
7. Add an explicit AST-to-HIR bridge phase with proof obligations.
8. Reconcile the concurrency model, especially `spawn` and channel semantics.
9. Reconcile the language reference: sync it, de-scope it, or move unsupported features out of the main reference path.
10. Add missing core semantic decisions: casts, trait coherence, exhaustiveness, FFI ABI, memory model.

## 6. Bottom Line

The project has strong instincts:

- it values explicit contracts
- it recognizes prior failure modes
- it correctly treats proof programs and stub tracking as non-optional

But it still has structural gaps that can derail execution:

- the docs do not cleanly separate planning-phase rules from execution-phase rules
- authority is split across documents
- some waves still allow false completion
- the language reference quietly expands scope far beyond the planned implementation
- several critical semantics remain unspecified

If these gaps are fixed first, the probability of implementation success increases materially. If they are not fixed first, the project is at real risk of repeating the same pattern the learning log already warns about: a coherent-looking plan that can still produce a partially real compiler.
