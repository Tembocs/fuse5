# Fuse Implementation Plan

> Status: normative for Fuse.
>
> This document is the build plan from an empty repository to a self-hosting
> Fuse compiler and the later retirement of bootstrap-only implementation
> languages.

## Overview

Fuse is implemented in stages.

- Stage 1 compiler: Go
- Runtime during bootstrap: C
- Stage 2 compiler: Fuse

The bootstrap stack is fixed. Go and C are allowed during bootstrap because
the project must reach a self-hosted Fuse compiler as quickly and safely as
possible. After Stage 2 compiles itself reliably and a native backend is
stable, Go and C are retired from the compiler implementation path.

The C11 backend is therefore bootstrap infrastructure, not the terminal
backend. Design decisions in HIR, MIR, type identity, ownership analysis, and
pass structure must not depend on C11 in a way that would block the later
native backend.

## Working principles

1. Correctness precedes velocity.
2. Structural fixes beat symptom fixes.
3. No workarounds are allowed in compiler, runtime, or stdlib.
4. Stdlib is the compiler's semantic stress test.
5. Every wave has explicit entry and exit criteria.
6. Every task must be small enough to review and verify directly.
7. Every wave that introduces user-visible behavior owns an end-to-end proof
   program, starting from Wave 05 when the minimal end-to-end spine exists.
8. Features that historically were silently stubbed — monomorphization,
   pattern matching, error propagation, closures, concurrency — each have a
   dedicated wave and their own proof program. No such feature is a phase
   inside somebody else's wave.
9. Every feature in the language reference is scheduled to a concrete wave
   (Rule 2.5). No `Wave TBD`, no "post-v1".
10. Before self-hosting begins, a dedicated Stub Clearance Gate wave
    (W22) confirms the Active stubs table is empty. No stub reaches the
    bootstrap.

## Naming conventions

The plan uses globally unique identifiers.

- Wave headings:
  `Wave 06: Type Checking`
- Phase headings:
  `Phase 03: Trait Resolution and Bound Dispatch [W06-P03-TRAIT-RESOLUTION]`
- Task headings:
  `Task 01: Register All Function Types Before Body Checking [W06-P01-T01-FN-TYPE-REGISTRATION]`

All wave, phase, and task numbers are zero-padded.

## Task format

Every task in this plan must be written with:

- a short goal
- a `Currently:` line naming what exists at the start of the task (file:line
  if the code exists, or "not yet started" if the package is empty)
- an exact definition of done
- a `Verify:` line giving the specific command that proves the DoD is met;
  this command must fail if the task has not been completed and must be run
  before the task is marked done
- required regression coverage
- clear scope boundaries

```
Task 01: ... [Wxx-Pyy-Tzz-...]
Currently: ...
DoD: verifiable completion rule.
Verify: go test ./compiler/pkg/... -run TestName -v
```

A Verify command is not satisfied by "it looks correct" or "unit tests pass".
It is satisfied by running the named command and observing the declared
passing output. The agent or contributor executing the task must record the
actual output of the Verify command before marking the task done.

`Verify:` commands must be runnable on Linux, macOS, and Windows. Unix-only
shell forms (process substitution, `/tmp/...` hardcoded paths, bash-specific
features) are forbidden in normative verification steps. Use Go tests,
project-owned scripts under `tools/`, or explicit per-platform wrappers
instead.

## Wave format

Every wave contains:

- **Goal**: one paragraph summary
- **Entry criterion**: what must be true before this wave begins
- **State on entry**: what the codebase actually looks like when the wave
  begins (which packages are empty, which are stubs, which are partial)
- **Exit criteria**: behavioral and structural requirements that must all be
  true
- **Proof of completion**: the specific commands that, when all passing,
  prove the wave is done; these are run in CI and locally before sign-off
- **Phase 00: Stub Audit**: first phase of every wave; committed to STUBS.md
  before other phases begin. This phase must verify no prior wave's stubs
  are overdue (Rule 6.15).
- One or more implementation phases (P01, P02, ...)
- **Wave Closure Phase (PCL)**: last phase of every wave; produces the WCxxx
  learning-log entry, updates STUBS.md with the wave's stub history block
  (Rule 6.16), and confirms the Active stubs table is current.

## Waves at a glance

| Wave | Theme | Entry criterion | Exit criterion |
|---|---|---|---|
| 00 | Governance and Phase Model | — | build, test, CI, docs scaffold, STUBS.md |
| 01 | Lexer | W00 done | every token kind and lexical ambiguity covered |
| 02 | Parser and AST | W01 done | all language constructs parse deterministically |
| 03 | Resolution | W02 done | module graph, imports, symbols resolved; `@cfg`; visibility |
| 04 | HIR and TypeTable | W03 done | typed HIR shape and pass graph enforced |
| 05 | Minimal End-to-End Spine | W04 done | `fn main() -> I32 { return N; }` runs and exits with N |
| 06 | Type Checking | W05 done | stdlib and user bodies type-check with no unknowns |
| 07 | Concurrency Semantics | W06 done | Send/Sync, Chan[T], spawn, ThreadHandle enforced by checker |
| 08 | Monomorphization | W07 done | generics compile end-to-end with proof programs |
| 09 | Ownership and Liveness | W08 done | single liveness computation + drop codegen with proof |
| 10 | Pattern Matching | W09 done | match dispatch, exhaustiveness, proof program |
| 11 | Error Propagation | W10 done | `?` branch lowering with proof program |
| 12 | Closures and Callable Traits | W11 done | capture, lift, env struct, Fn/FnMut/FnOnce with proof |
| 13 | Trait Objects (`dyn Trait`) | W12 done | fat pointer, vtable layout, object-safe rules, proof |
| 14 | Compile-Time Evaluation (`const fn`) | W13 done | const evaluator over checked HIR with proof |
| 15 | Lowering and MIR Consolidation | W14 done | MIR invariants; casts, fn-pointers, slice range, struct update all lowered |
| 16 | Runtime ABI | W15 done | full runtime replaces stub; IO + threading work |
| 17 | Codegen C11 Hardening | W16 done | all backend contracts enforced; `@repr`, `@align`, intrinsics, variadic, size_of, Ptr.null emission |
| 18 | CLI and Diagnostics | W17 done | `fuse build/run/check/test/fmt/doc/repl` coherent |
| 19 | Stdlib Core | W18 done | core traits, primitives, strings, collections, Cell/RefCell, Ptr.null, overflow methods |
| 20 | Custom Allocators | W19 done | Allocator trait; collections accept allocator; proof program with bump allocator |
| 21 | Stdlib Hosted | W20 done | IO, fs, os, time, thread, sync, channels ship |
| 22 | Stub Clearance Gate | W21 done | Active stubs table is empty; no stub reaches Stage 2 |
| 23 | Stage 2 and Self-Hosting | W22 done | stage1 compiles stage2; stage2 compiles itself reproducibly |
| 24 | Native Backend Transition | W23 done | stage2 compiles without C11 backend dependency |
| 25 | Retirement of Go and C | W24 done | Fuse owns the compiler implementation path |
| 26 | Targets and Ecosystem | W25 done | cross-target and library growth on native base |

Every feature documented in `docs/fuse-language-reference.md` is scheduled
to one or more of the waves above. No feature is deferred to a later
version.

## Wave 00: Governance and Phase Model

Goal: establish the repository, module, build, test, tooling, documentation,
and governance foundations required for disciplined compiler work. Seed
`STUBS.md` with the initial stub inventory covering every unimplemented
language feature.

Entry criterion: none.

State on entry: only the five foundational docs and the `STUBS.md` seed
exist. No Go module. No CI. No compiler packages.

Exit criteria:

- `make all` succeeds from a clean checkout
- `go test ./...` succeeds on the initial package set
- CI runs on every push and PR for Linux, macOS, and Windows
- the five foundational docs exist and are readable
- `STUBS.md` exists with a seeded Active stubs table covering every
  user-visible language feature not yet implemented, and an empty Stub
  history section
- `.claude/current-wave.json` names the active wave for coordination

Proof of completion:

```
make all
go test ./...
go run tools/checkstubs/main.go
```

### Phase 00: Stub Audit [W00-P00-STUB-AUDIT]

- Task 01: Initialize STUBS.md with seed stubs [W00-P00-T01-INIT-STUBS]
  Currently: file does not exist.
  DoD: STUBS.md exists with the Active stubs table seeded with every
  language feature scheduled for a wave greater than W00, each with its
  scheduled retiring wave. Stub history section is empty.
  Verify: `go run tools/checkstubs/main.go -audit-seed`

### Phase 01: Repository Initialization [W00-P01-REPO-INIT]

- Task 01: Create repository skeleton [W00-P01-T01-REPO-SKELETON]
  Verify: `go run tools/checklayout/main.go`
- Task 02: Add foundational docs [W00-P01-T02-FOUNDATIONAL-DOCS]
  Verify: `go run tools/checkdocs/main.go -foundational`
- Task 03: Artifact policy [W00-P01-T03-ARTIFACT-POLICY]
  Verify: `go run tools/checkartifacts/main.go`

### Phase 02: Go Module and Build Scaffold [W00-P02-GO-MODULE]

- Task 01: Initialize Go module [W00-P02-T01-GO-MOD]
  Verify: `go build ./...`
- Task 02: Create package stubs [W00-P02-T02-PACKAGE-STUBS]
  Verify: `go build ./compiler/...`
- Task 03: Create Stage 1 CLI stub [W00-P02-T03-CLI-STUB]
  Verify: `go test ./cmd/fuse/... -run TestCliStub -v`

### Phase 03: Build and CI Baseline [W00-P03-BUILD-CI]

- Task 01: Author Makefile targets [W00-P03-T01-MAKEFILE]
  Verify: `make all && make test && make clean && make all`
- Task 02: Add CI matrix [W00-P03-T02-CI-MATRIX]
  Verify: `go run tools/checkci/main.go`
- Task 03: Add golden harness [W00-P03-T03-GOLDEN-HARNESS]
  Verify: `go test ./tools/goldens/... -run TestGoldenStability -count=2 -v`

### Phase 04: Governance Artifacts [W00-P04-GOVERNANCE]

- Task 01: Create `.claude/current-wave.json`
  [W00-P04-T01-CURRENT-WAVE-FILE]
  Verify: `go run tools/checkgov/main.go -current-wave`
- Task 02: Document phase model [W00-P04-T02-PHASE-MODEL-DOC]
  Verify: `test -f docs/phase-model.md && go run tools/checkdocs/main.go`
- Task 03: Install `tools/checkstubs` [W00-P04-T03-CHECKSTUBS-TOOL]
  Verify: `go test ./tools/checkstubs/... -v`

### Wave Closure Phase [W00-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md stub history [W00-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -history-current-wave W00`
- Task 02: Write WC000 learning-log entry [W00-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC000" docs/learning-log.md`

## Wave 01: Lexer

Goal: build a deterministic lexer that covers the full token set and every
lexical ambiguity listed in the language reference.

Entry criterion: W00 done. Phase 00 of this wave confirms no overdue stubs.

State on entry: `compiler/lex/` is an empty stub package.

Exit criteria:

- every token kind in reference §1 is tested
- BOM is rejected (reference §1.10)
- nested block comments work
- raw strings obey the full-pattern rule (reference §1.10)
- `?.` is emitted as one token (reference §1.10)
- c-string literal `c"..."` lexes correctly (reference §42.5)
- golden tests are byte-stable across three runs

Proof of completion:

```
go test ./compiler/lex/... -v
go test ./compiler/lex/... -run TestGolden -count=3 -v
```

### Phase 00: Stub Audit [W01-P00-STUB-AUDIT]

- Task 01: Audit lex stub [W01-P00-T01-LEX-STUB-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W01 -phase P00`

### Phase 01: Token Model [W01-P01-TOKEN-MODEL]

- Task 01: Define token kinds [W01-P01-T01-TOKEN-KINDS]
  Verify: `go test ./compiler/lex/... -run TestTokenKindCoverage -v`
- Task 02: Define span model [W01-P01-T02-SPAN-MODEL]
  Verify: `go test ./compiler/lex/... -run TestSpanStability -v`

### Phase 02: Scanner Core [W01-P02-SCANNER]

- Task 01: Identifier and keyword scanning [W01-P02-T01-IDENT-KEYWORD]
  Verify: `go test ./compiler/lex/... -run TestKeywords -v`
- Task 02: Literal scanning [W01-P02-T02-LITERALS]
  DoD: integer (all bases), float, string, raw string, and c-string literal
  forms tokenize correctly.
  Verify: `go test ./compiler/lex/... -run TestLiterals -v`
- Task 03: Comments and trivia [W01-P02-T03-TRIVIA]
  Verify: `go test ./compiler/lex/... -run TestNestedBlockComment -v`

### Phase 03: Lexical Edge Cases [W01-P03-EDGES]

- Task 01: Raw string full-pattern rule [W01-P03-T01-RAW-STRING]
  Verify: `go test ./compiler/lex/... -run TestRawStringGuard -v`
- Task 02: `?.` longest-match [W01-P03-T02-OPTIONAL-CHAIN]
  Verify: `go test ./compiler/lex/... -run TestOptionalChainToken -v`
- Task 03: BOM rejection [W01-P03-T03-BOM]
  Verify: `go test ./compiler/lex/... -run TestBomRejection -v`
- Task 04: Golden and fuzz coverage [W01-P03-T04-TESTS]
  Verify: `go test ./compiler/lex/... -run TestLexerFuzz -v`

### Wave Closure Phase [W01-PCL-WAVE-CLOSURE]

- Task 01: Retire lexer stub [W01-PCL-T01-RETIRE-STUB]
  Verify: `go run tools/checkstubs/main.go -wave W01 -retired lexer`
- Task 02: WC001 entry [W01-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC001" docs/learning-log.md`

## Wave 02: Parser and AST

Goal: build an AST-only parser that accepts the full surface grammar
defined in reference Appendix C without semantic shortcuts.

Entry criterion: W01 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/parse/` and `compiler/ast/` are empty stubs.

Exit criteria:

- parser handles every grammar construct in reference Appendix C, including
  `dyn Trait`, `impl Trait`, `union`, decorators (`@value`, `@rank`,
  `@repr`, `@align`, `@inline`, `@cold`, `@cfg`), `const fn`, or-patterns,
  range patterns, `@`-binding patterns, slice range indexing, and struct
  update syntax
- parser does not panic on malformed input
- AST remains syntax-only (no resolved symbols, no types, no metadata)
- struct literal disambiguation works per reference §10.7
- optional chaining parses per reference §1.10
- golden tests are byte-stable

Proof of completion:

```
go test ./compiler/parse/... -v
go test ./compiler/ast/... -v
go test ./compiler/parse/... -run TestGolden -count=3 -v
go test ./compiler/parse/... -run TestNopanicOnMalformed -v
```

### Phase 00: Stub Audit [W02-P00-STUB-AUDIT]

- Task 01: Audit [W02-P00-T01-PARSE-STUB-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W02 -phase P00`

### Phase 01: AST Surface [W02-P01-AST]

- Task 01: Define AST node set [W02-P01-T01-NODES]
  Verify: `go test ./compiler/ast/... -run TestAstNodeCompleteness -v`
- Task 02: Builders and span correctness [W02-P01-T02-BUILDERS]
  Verify: `go test ./compiler/ast/... -run TestSpanCorrectness -v`

### Phase 02: Core Parsing [W02-P02-CORE]

- Task 01: Items and declarations [W02-P02-T01-ITEMS]
  DoD: functions (including `const fn`), structs, enums, traits, impls,
  consts, statics, type aliases, externs (including variadic externs),
  unions parse correctly.
  Verify: `go test ./compiler/parse/... -run TestItemParsing -v`
- Task 02: Expressions and statements [W02-P02-T02-EXPRS]
  DoD: precedence per reference Appendix B; expression goldens pass; struct
  update syntax (`..base`), slice range indexing parse correctly.
  Verify: `go test ./compiler/parse/... -run TestExprPrecedence -v`
- Task 03: Type expressions [W02-P02-T03-TYPES]
  DoD: tuples, arrays, slices, raw pointers, generics, fn-types, `dyn
  Trait`, `impl Trait`, unions all parse.
  Verify: `go test ./compiler/parse/... -run TestTypeExprs -v`
- Task 04: Patterns [W02-P02-T04-PATTERNS]
  DoD: literal, wildcard, bind, tuple, struct, constructor, or, range,
  `@`-binding patterns all parse.
  Verify: `go test ./compiler/parse/... -run TestPatternParsing -v`
- Task 05: Decorators [W02-P02-T05-DECORATORS]
  DoD: `@value`, `@rank(N)`, `@repr(C)`, `@repr(packed)`, `@repr(Uxx)`,
  `@repr(Ixx)`, `@align(N)`, `@inline`, `@inline(always)`, `@inline(never)`,
  `@cold`, `@cfg(...)` parse with predicate arguments.
  Verify: `go test ./compiler/parse/... -run TestDecoratorParsing -v`

### Phase 03: Ambiguity Control [W02-P03-AMBIGUITY]

- Task 01: Struct-literal disambiguation [W02-P03-T01-STRUCT-LITERAL]
  Verify: `go test ./compiler/parse/... -run TestStructLiteralDisambig -v`
- Task 02: Optional chaining parse [W02-P03-T02-OPTIONAL-CHAIN]
  Verify: `go test ./compiler/parse/... -run TestOptionalChainParse -v`
- Task 03: Malformed input corpus [W02-P03-T03-MALFORMED]
  Verify: `go test ./compiler/parse/... -run TestNopanicOnMalformed -v`

### Wave Closure Phase [W02-PCL-WAVE-CLOSURE]

- Task 01: Retire parse/ast stubs [W02-PCL-T01-RETIRE-STUB]
  Verify: `go run tools/checkstubs/main.go -wave W02 -retired parse,ast`
- Task 02: WC002 entry [W02-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC002" docs/learning-log.md`

## Wave 03: Resolution

Goal: resolve symbols, imports, and the module graph; evaluate `@cfg`
predicates; enforce visibility levels.

Entry criterion: W02 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/resolve/` is an empty stub.

Exit criteria:

- module discovery deterministic
- imports resolve with module-first fallback (reference §18.7)
- import cycles diagnosed without hang
- qualified enum variant access resolves (reference §11.6)
- `@cfg` evaluated at resolve time; items with false predicate removed
  from the HIR-input stream (reference §50.1)
- four visibility levels enforced: private / `pub(mod)` / `pub(pkg)` / `pub`
  (reference §53.1)

Proof of completion:

```
go test ./compiler/resolve/... -v
go test ./compiler/resolve/... -run TestImportCycleDetection -v
go test ./compiler/resolve/... -run TestQualifiedEnumVariant -v
go test ./compiler/resolve/... -run TestModuleFirstFallback -v
go test ./compiler/resolve/... -run TestCfgEvaluation -v
go test ./compiler/resolve/... -run TestVisibilityEnforcement -v
```

### Phase 00: Stub Audit [W03-P00-STUB-AUDIT]

- Task 01: Resolve audit [W03-P00-T01-RESOLVE-STUB-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W03 -phase P00`

### Phase 01: Module Graph [W03-P01-MODULE-GRAPH]

- Task 01: Discover modules [W03-P01-T01-DISCOVER]
  Verify: `go test ./compiler/resolve/... -run TestModuleDiscovery -count=3 -v`
- Task 02: Build module graph [W03-P01-T02-GRAPH]
  Verify: `go test ./compiler/resolve/... -run TestModuleGraph -v`

### Phase 02: Symbols and Scopes [W03-P02-SYMBOLS]

- Task 01: Symbol table and scopes [W03-P02-T01-SYMBOLS]
  Verify: `go test ./compiler/resolve/... -run TestScopeLookup -v`
- Task 02: Top-level indexing [W03-P02-T02-INDEX]
  Verify: `go test ./compiler/resolve/... -run TestTopLevelIndex -v`

### Phase 03: Import and Path Resolution [W03-P03-IMPORTS]

- Task 01: Module-first fallback [W03-P03-T01-MODULE-FIRST]
  Verify: `go test ./compiler/resolve/... -run TestModuleFirstFallback -v`
- Task 02: Qualified enum variants [W03-P03-T02-QUALIFIED-VARIANTS]
  Verify: `go test ./compiler/resolve/... -run TestQualifiedEnumVariant -v`
- Task 03: Import cycles [W03-P03-T03-CYCLES]
  Verify: `go test ./compiler/resolve/... -run TestImportCycleDetection -v`

### Phase 04: Conditional Compilation [W03-P04-CFG]

- Task 01: `@cfg` evaluator [W03-P04-T01-CFG-EVAL]
  DoD: `@cfg(os = "linux")`, `not`, `all`, `any`, `feature = "name"` all
  evaluate; items with false predicate are removed from the HIR input.
  Verify: `go test ./compiler/resolve/... -run TestCfgEvaluation -v`
- Task 02: Duplicate-item resolution [W03-P04-T02-CFG-DUPLICATES]
  Verify: `go test ./compiler/resolve/... -run TestCfgDuplicates -v`

### Phase 05: Visibility Enforcement [W03-P05-VISIBILITY]

- Task 01: Enforce four visibility levels [W03-P05-T01-VIS-LEVELS]
  Verify: `go test ./compiler/resolve/... -run TestVisibilityEnforcement -v`

### Wave Closure Phase [W03-PCL-WAVE-CLOSURE]

- Task 01: Retire resolve stubs [W03-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W03 -retired resolve,cfg,visibility`
- Task 02: WC003 entry [W03-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC003" docs/learning-log.md`

## Wave 04: HIR and TypeTable

Goal: establish the typed semantic IR surface, the pass graph, and the
AST-to-HIR bridge with proven type preservation.

Entry criterion: W03 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/hir/` and `compiler/typetable/` are empty stubs.

Exit criteria:

- HIR exists, is distinct from AST, carries required metadata
- HIR builders enforce required metadata at construction
- TypeTable interns all types; equality is integer comparison
- nominal identity includes defining symbol (reference §2.8)
- pass manifest validates declared metadata dependencies
- invariant walkers run in debug and CI
- AST-to-HIR bridge preserves resolved types — no `Unknown` defaults (L013)
- `KindChannel` and `KindThreadHandle` type kinds defined (used in W07)

Proof of completion:

```
go test ./compiler/hir/... -v
go test ./compiler/typetable/... -v
go test ./compiler/hir/... -run TestInvariantWalkers -v
go test ./compiler/hir/... -run TestBuilderEnforcement -v
go test ./compiler/hir/... -run TestAstToHirTypePreservation -v
go test ./compiler/hir/... -run TestDeterministicOrder -count=3 -v
```

### Phase 00: Stub Audit [W04-P00-STUB-AUDIT]

- Task 01: HIR and TypeTable audit [W04-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W04 -phase P00`

### Phase 01: TypeTable [W04-P01-TYPETABLE]

- Task 01: TypeId and interning [W04-P01-T01-TYPEID]
  Verify: `go test ./compiler/typetable/... -run TestTypeInternEquality -v`
- Task 02: Nominal identity [W04-P01-T02-NOMINAL-IDENTITY]
  Verify: `go test ./compiler/typetable/... -run TestNominalIdentity -v`
- Task 03: Channel type kind stub [W04-P01-T03-CHANNEL-KIND-STUB]
  DoD: `KindChannel` defined; checker integration is in STUBS.md,
  retiring W07.
  Verify: `go test ./compiler/typetable/... -run TestChannelTypeKindExists -v`
- Task 04: Thread handle type kind stub [W04-P01-T04-THREAD-HANDLE-STUB]
  DoD: `KindThreadHandle` defined; checker integration retires W07.
  Verify: `go test ./compiler/typetable/... -run TestThreadHandleKindExists -v`

### Phase 02: HIR Node Set and Metadata [W04-P02-HIR-NODES]

- Task 01: HIR node set [W04-P02-T01-NODES]
  DoD: HIR nodes are semantically oriented; patterns are structured nodes
  (`LiteralPat`, `BindPat`, `ConstructorPat`, `WildcardPat`, `OrPat`,
  `RangePat`, `AtBindPat`), not text (L007).
  Verify: `go test ./compiler/hir/... -run TestHirNodeSet -v`
- Task 02: Metadata fields [W04-P02-T02-METADATA]
  Verify: `go test ./compiler/hir/... -run TestMetadataFields -v`
- Task 03: Builder enforcement [W04-P02-T03-BUILDERS]
  Verify: `go test ./compiler/hir/... -run TestBuilderEnforcement -v`

### Phase 03: AST-to-HIR Bridge [W04-P03-BRIDGE]

- Task 01: Bridge with type propagation [W04-P03-T01-BRIDGE-IMPL]
  DoD: no expression receives `Unknown` as its default type (L013 defense).
  Verify: `go test ./compiler/hir/... -run TestAstToHirTypePreservation -v`
- Task 02: Bridge invariant walker [W04-P03-T02-BRIDGE-INVARIANT]
  Verify: `go test ./compiler/hir/... -run TestBridgeInvariant -v`

### Phase 04: Pass Graph and Determinism [W04-P04-PASS-GRAPH]

- Task 01: Pass manifest [W04-P04-T01-MANIFEST]
  Verify: `go test ./compiler/hir/... -run TestPassManifest -v`
- Task 02: Invariant walkers [W04-P04-T02-INVARIANTS]
  Verify: `go test ./compiler/hir/... -run TestInvariantWalkers -v`
- Task 03: Deterministic IR collections [W04-P04-T03-DETERMINISM]
  Verify: `go test ./compiler/hir/... -run TestDeterministicOrder -count=3 -v`

### Wave Closure Phase [W04-PCL-WAVE-CLOSURE]

- Task 01: Retire HIR/TypeTable/bridge stubs [W04-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W04 -retired hir,typetable,bridge`
- Task 02: WC004 entry [W04-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC004" docs/learning-log.md`

## Wave 05: Minimal End-to-End Spine

Goal: make a trivial Fuse program compile, link, run, and return a chosen
exit code. Closes the L013 deferred-proof failure mode.

Entry criterion: W04 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/lower/`, `compiler/mir/`, `compiler/codegen/`,
`compiler/cc/`, `compiler/driver/`, `runtime/` are empty stubs.

Exit criteria:

- `tests/e2e/hello_exit.fuse` compiles, links, runs, returns 0 on Linux,
  macOS, Windows
- `tests/e2e/exit_with_value.fuse` returns a chosen nonzero code (e.g. 42)
- spine MIR, codegen, runtime, driver packages alive with minimal impls
- every feature beyond int-returning main either emits a diagnostic or is
  in STUBS.md with a retiring wave
- `tests/e2e/README.md` created with the two proof programs recorded

Proof of completion:

```
go test ./tests/e2e/... -run TestHelloExit -v
go test ./tests/e2e/... -run TestExitWithValue -v
go run tools/checkstubs/main.go -wave W05
```

### Phase 00: Stub Audit [W05-P00-STUB-AUDIT]

- Task 01: Spine audit [W05-P00-T01-SPINE-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W05 -phase P00`

### Phase 01: Minimal MIR [W05-P01-MIR]

- Task 01: Minimal MIR instruction set [W05-P01-T01-MIR-MIN]
  DoD: `const-int`, `return`, `binary-add/sub/mul/div/mod`, basic block
  with terminator. Anything else produces a lowerer diagnostic.
  Verify: `go test ./compiler/mir/... -run TestMinimalMir -v`
- Task 02: Minimal lowerer [W05-P01-T02-LOWER-MIN]
  Verify: `go test ./compiler/lower/... -run TestMinimalLowerIntReturn -v`

### Phase 02: Minimal Codegen [W05-P02-CODEGEN]

- Task 01: Minimal C11 emitter [W05-P02-T01-CODEGEN-MIN]
  Verify: `go test ./compiler/codegen/... -run TestMinimalCodegenC -v`
- Task 02: C compiler detection [W05-P02-T02-CC]
  Verify: `go test ./compiler/cc/... -run TestCCDetection -v`

### Phase 03: Stub Runtime [W05-P03-RUNTIME]

- Task 01: Minimal runtime [W05-P03-T01-RT-MIN]
  DoD: `runtime/include/fuse_rt.h` declares the ABI surface with stubs
  for all functions except process-entry and `fuse_rt_abort`.
  Verify: `go test ./runtime/tests/... -run TestStubRuntime -v`

### Phase 04: Minimal Driver [W05-P04-DRIVER]

- Task 01: `fuse build` for int-returning main [W05-P04-T01-DRIVER-MIN]
  Verify: `go test ./compiler/driver/... -run TestMinimalBuildInvocation -v`
- Task 02: CLI subcommand minimum [W05-P04-T02-CLI-MIN]
  Verify: `go test ./cmd/fuse/... -run TestMinimalCli -v`

### Phase 05: Proof Programs [W05-P05-PROOF]

- Task 01: `hello_exit.fuse` [W05-P05-T01-HELLO]
  Verify: `go test ./tests/e2e/... -run TestHelloExit -v`
- Task 02: `exit_with_value.fuse` [W05-P05-T02-EXIT-VALUE]
  Verify: `go test ./tests/e2e/... -run TestExitWithValue -v`

### Wave Closure Phase [W05-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md stub history [W05-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W05`
- Task 02: WC005 entry [W05-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC005" docs/learning-log.md`

## Wave 06: Type Checking

Goal: build a checker that fully types user code and stdlib bodies without
leaving unknown metadata for later passes. This wave covers every
type-system feature whose semantics live in the checker: primitive types,
structs (including newtype `struct T(U);` form), enums (including unions
as distinct declaration form), traits (with coherence), generics (with
parameter scoping), associated types, casts, visibility, function pointer
types, `impl Trait` parameter-position, variadic extern signatures,
`@repr`/`@align` annotations on types.

Entry criterion: W05 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/check/` is an empty stub.

Exit criteria:

- no checked HIR node retains `Unknown` metadata
- two-pass checker (signatures first, then bodies)
- stdlib bodies checked in the same pass as user modules (L002 defense)
- trait-bound lookup through supertraits (reference §12.7)
- coherence and orphan rules enforced (reference §12.7)
- associated types resolve (reference §30.1)
- `as` cast semantics enforced (reference §28.1)
- numeric widening enforced (reference §5.8)
- function pointer types (reference §29.1)
- union declarations checked (reference §49.1)
- newtype pattern (single-field tuple struct form) checked
- opaque return `-> impl Trait` single-concrete-type-per-function rule
  (reference §56.1)
- `@repr(C)` / `@repr(packed)` / `@repr(Uxx|Ixx)` / `@align(N)` validated
  (reference §37.5)
- variadic extern signatures checked
- four visibility levels enforced on every use site

Proof of completion:

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
```

### Phase 00: Stub Audit [W06-P00-STUB-AUDIT]

- Task 01: Check audit [W06-P00-T01-CHECK-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W06 -phase P00`

### Phase 01: Function Type Registration [W06-P01-FN-TYPES]

- Task 01: Index all function signatures [W06-P01-T01-INDEX]
  Verify: `go test ./compiler/check/... -run TestFunctionTypeRegistration -v`
- Task 02: Two-pass checker [W06-P01-T02-TWO-PASS]
  Verify: `go test ./compiler/check/... -run TestTwoPassChecker -v`

### Phase 02: Nominal Identity, Primitives, Casts [W06-P02-NOMINAL]

- Task 01: Nominal equality [W06-P02-T01-NOMINAL-EQ]
  Verify: `go test ./compiler/check/... -run TestNominalEquality -v`
- Task 02: Primitive method registration [W06-P02-T02-PRIM-METHODS]
  Verify: `go test ./compiler/check/... -run TestPrimitiveMethods -v`
- Task 03: Numeric widening [W06-P02-T03-WIDENING]
  Verify: `go test ./compiler/check/... -run TestNumericWidening -v`
- Task 04: Cast semantics [W06-P02-T04-CASTS]
  Verify: `go test ./compiler/check/... -run TestCastSemantics -v`

### Phase 03: Trait Resolution [W06-P03-TRAITS]

- Task 01: Concrete trait method lookup [W06-P03-T01-CONCRETE]
  Verify: `go test ./compiler/check/... -run TestConcreteTraitMethodLookup -v`
- Task 02: Bound-chain lookup [W06-P03-T02-BOUND-CHAIN]
  Verify: `go test ./compiler/check/... -run TestBoundChainLookup -v`
- Task 03: Coherence and orphan rules [W06-P03-T03-COHERENCE]
  Verify: `go test ./compiler/check/... -run TestCoherenceOrphan -v`
- Task 04: Trait-typed parameters [W06-P03-T04-TRAIT-PARAMS]
  Verify: `go test ./compiler/check/... -run TestTraitParameters -v`

### Phase 04: Contextual Inference and Literals [W06-P04-INFERENCE]

- Task 01: Expected-type inference [W06-P04-T01-EXPECTED]
  Verify: `go test ./compiler/check/... -run TestContextualInference -v`
- Task 02: Zero-arg generic calls [W06-P04-T02-ZERO-ARG]
  Verify: `go test ./compiler/check/... -run TestZeroArgTypeArgs -v`
- Task 03: Literal typing [W06-P04-T03-LIT-TYPING]
  Verify: `go test ./compiler/check/... -run TestLiteralTyping -v`

### Phase 05: Associated Types [W06-P05-ASSOC-TYPES]

- Task 01: Associated type projection [W06-P05-T01-PROJECT]
  Verify: `go test ./compiler/check/... -run TestAssocTypeProjection -v`
- Task 02: Associated type constraints [W06-P05-T02-CONSTRAINTS]
  Verify: `go test ./compiler/check/... -run TestAssocTypeConstraints -v`

### Phase 06: Function Pointer and `impl Trait` Types [W06-P06-FN-IMPL]

- Task 01: Function pointer types [W06-P06-T01-FN-PTR-TYPE]
  DoD: `fn(A, B) -> R` is a first-class type. Nominal identity is by
  signature.
  Verify: `go test ./compiler/check/... -run TestFnPointerType -v`
- Task 02: `impl Trait` parameter-position [W06-P06-T02-IMPL-TRAIT-PARAM]
  DoD: `fn f(x: impl Trait)` desugars to `fn f[T: Trait](x: T)` at check
  time.
  Verify: `go test ./compiler/check/... -run TestImplTraitParam -v`
- Task 03: `impl Trait` return-position [W06-P06-T03-IMPL-TRAIT-RETURN]
  DoD: checker records a single concrete return type per `impl Trait`
  return signature; multi-type paths are a diagnostic.
  Verify: `go test ./compiler/check/... -run TestImplTraitReturn -v`

### Phase 07: Unions, Newtype, Repr Annotations [W06-P07-UNIONS-REPR]

- Task 01: Union declaration check [W06-P07-T01-UNION-CHECK]
  DoD: `union U { a: T1, b: T2 }` checks field types; union fields may not
  implement `Drop`.
  Verify: `go test ./compiler/check/... -run TestUnionCheck -v`
- Task 02: Newtype pattern [W06-P07-T02-NEWTYPE]
  DoD: `struct U(T);` is distinct from `T`; methods can be attached.
  Verify: `go test ./compiler/check/... -run TestNewtypePattern -v`
- Task 03: Repr and align annotation validation [W06-P07-T03-REPR-ALIGN]
  DoD: `@repr(C)`, `@repr(packed)`, `@repr(Uxx|Ixx)`, `@align(N)` validated
  against target type; conflicting annotations produce diagnostics.
  Verify: `go test ./compiler/check/... -run TestReprAnnotationCheck -v`
- Task 04: Variadic extern signatures [W06-P07-T04-VARIADIC-SIG]
  DoD: `extern fn printf(fmt: Ptr[U8], ...) -> I32;` checks; variadic Fuse
  functions are a diagnostic.
  Verify: `go test ./compiler/check/... -run TestVariadicExternCheck -v`

### Phase 08: Stdlib-Shape Body Checking [W06-P08-STDLIB-SHAPE]

- Task 01: No stdlib body skips [W06-P08-T01-NO-SKIPS]
  Verify: `go test ./compiler/check/... -run TestStdlibBodyChecking -v`
- Task 02: Checker regression corpus [W06-P08-T02-REGRESSIONS]
  Verify: `go test ./compiler/check/... -v`

### Phase 09: Checker E2E Proof [W06-P09-PROOF]

- Task 01: `checker_basic.fuse` [W06-P09-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestCheckerBasicProof -v`

### Wave Closure Phase [W06-PCL-WAVE-CLOSURE]

- Task 01: Retire check stubs [W06-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W06`
- Task 02: WC006 entry [W06-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC006" docs/learning-log.md`

## Wave 07: Concurrency Semantics

Goal: enforce `Send`/`Sync`/`Copy` marker traits, `Chan[T]` type checking,
`spawn` semantics (handle-returning), `ThreadHandle[T]`, and `@rank`
lock ordering. Checker-side enforcement only; runtime lowering happens in
W16.

Entry criterion: W06 done. Phase 00 confirms no overdue stubs.

State on entry: channel and thread-handle kind stubs exist in TypeTable
(added W04). Checker does not yet enforce Send/Sync bounds.

Exit criteria:

- `Send`, `Sync`, `Copy` declared as intrinsic marker traits; auto-impl
  rules enforced (reference §47.1)
- negative impls (`impl !Send for T {}`) work in the declaring module
- `Chan[T]` type checking rejects element-type mismatches (reference §17.6)
- `spawn fn() -> T { ... }` typed as `ThreadHandle[T]` (reference §39.1)
- spawn requires `Send` captures
- `@rank(N)` lock ordering enforced structurally (reference §17.6)

Proof of completion:

```
go test ./compiler/check/... -run TestSendSyncMarkerTraits -v
go test ./compiler/check/... -run TestChannelTypecheck -v
go test ./compiler/check/... -run TestSpawnHandleTyping -v
go test ./compiler/check/... -run TestLockRankingEnforcement -v
go test ./tests/e2e/... -run TestConcurrencyRejections -v
```

### Phase 00: Stub Audit [W07-P00-STUB-AUDIT]

- Task 01: Concurrency audit [W07-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W07 -phase P00`

### Phase 01: Marker Traits [W07-P01-MARKERS]

- Task 01: Intrinsic `Send`/`Sync`/`Copy` [W07-P01-T01-DECLARE]
  Verify: `go test ./compiler/check/... -run TestMarkerTraitDeclarations -v`
- Task 02: Auto-impl rules [W07-P01-T02-AUTO-IMPL]
  Verify: `go test ./compiler/check/... -run TestMarkerAutoImpl -v`
- Task 03: Negative impls [W07-P01-T03-NEG-IMPL]
  Verify: `go test ./compiler/check/... -run TestNegativeImpl -v`

### Phase 02: Channel Typecheck [W07-P02-CHANNEL]

- Task 01: `Chan[T]` type operations [W07-P02-T01-CHAN]
  Verify: `go test ./compiler/check/... -run TestChannelTypecheck -v`
- Task 02: Send bound on element type [W07-P02-T02-CHAN-SEND-BOUND]
  Verify: `go test ./compiler/check/... -run TestChannelSendBound -v`

### Phase 03: Spawn Typecheck [W07-P03-SPAWN]

- Task 01: Spawn typing to `ThreadHandle[T]` [W07-P03-T01-SPAWN-TYPING]
  Verify: `go test ./compiler/check/... -run TestSpawnHandleTyping -v`
- Task 02: Send bound on captured environment [W07-P03-T02-SPAWN-SEND]
  Verify: `go test ./compiler/check/... -run TestSpawnSendBound -v`
- Task 03: `Shared[T]` bounds [W07-P03-T03-SHARED-BOUND]
  Verify: `go test ./compiler/check/... -run TestSharedBounds -v`

### Phase 04: Lock Ranking [W07-P04-RANKING]

- Task 01: `@rank(N)` structural enforcement [W07-P04-T01-RANK]
  Verify: `go test ./compiler/check/... -run TestLockRankingEnforcement -v`

### Phase 05: Concurrency Proof Program [W07-P05-PROOF]

- Task 01: Rejection proof [W07-P05-T01-REJECTIONS]
  DoD: checker proof file asserts several rejections fire with the
  expected diagnostic (non-Send spawn, wrong-type send, double-lock rank).
  Verify: `go test ./tests/e2e/... -run TestConcurrencyRejections -v`

### Wave Closure Phase [W07-PCL-WAVE-CLOSURE]

- Task 01: Retire concurrency stubs [W07-PCL-T01-RETIRE]
  DoD: channel type kind, ThreadHandle type kind, marker traits retired.
  Runtime-side lowering (spawn → runtime call, channel ops → runtime call)
  remains stubbed with diagnostics, retiring W16.
  Verify: `go run tools/checkstubs/main.go -wave W07`
- Task 02: WC007 entry [W07-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC007" docs/learning-log.md`

## Wave 08: Monomorphization

Goal: make generic functions and generic types compile through the full
pipeline and produce correct running programs (L015).

Entry criterion: W07 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- generic parameters scope correctly in bodies
- explicit type args at call sites resolve
- driver collects concrete instantiations after checking
- partial specializations rejected
- body duplication produces concrete functions with type substitution
- specialized function names are deterministic and distinct
- call sites rewritten to reference specialized names
- only concrete instantiations reach codegen
- unresolved types are a hard error before codegen
- proof programs: identity[I32], multiple instantiations; Option/Result
  specialization

Proof of completion:

```
go test ./compiler/monomorph/... -v
go test ./compiler/driver/... -run TestSpecializationInPipeline -v
go test ./tests/e2e/... -run TestIdentityGeneric -v
go test ./tests/e2e/... -run TestMultipleInstantiations -v
```

### Phase 00: Stub Audit [W08-P00-STUB-AUDIT]

- Task 01: Monomorphization audit [W08-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W08 -phase P00`

### Phase 01: Generic Parameter Scoping [W08-P01-SCOPING]

- Task 01: Register generic params [W08-P01-T01-REGISTER]
  Verify: `go test ./compiler/check/... -run TestGenericParamScoping -v`
- Task 02: Resolve explicit type args [W08-P01-T02-CALL-SITE-TYPE-ARGS]
  Verify: `go test ./compiler/check/... -run TestCallSiteTypeArgs -v`

### Phase 02: Instantiation Collection [W08-P02-COLLECT]

- Task 01: Scan for generic call sites [W08-P02-T01-SCAN]
  Verify: `go test ./compiler/driver/... -run TestInstantiationCollection -v`
- Task 02: Validate completeness [W08-P02-T02-VALIDATE]
  Verify: `go test ./compiler/driver/... -run TestPartialInstantiationRejected -v`

### Phase 03: Body Specialization [W08-P03-SPECIALIZE]

- Task 01: AST-level body duplication [W08-P03-T01-DUPLICATE]
  Verify: `go test ./compiler/monomorph/... -run TestBodyDuplication -v`
- Task 02: Specialized function names [W08-P03-T02-NAMES]
  Verify: `go test ./compiler/monomorph/... -run TestSpecializedNames -v`
- Task 03: Call-site rewriting [W08-P03-T03-REWRITE]
  Verify: `go test ./compiler/monomorph/... -run TestCallSiteRewrite -v`

### Phase 04: Driver Pipeline Integration [W08-P04-PIPELINE]

- Task 01: Insert specialization step [W08-P04-T01-INSERT]
  Verify: `go test ./compiler/driver/... -run TestSpecializationInPipeline -v`
- Task 02: Skip generic originals in codegen [W08-P04-T02-SKIP-ORIGINALS]
  Verify: `go test ./compiler/codegen/... -run TestGenericOriginalsSkipped -v`

### Phase 05: Generic Types (Option, Result) [W08-P05-GENERIC-TYPES]

- Task 01: Specialize generic enums [W08-P05-T01-SPEC-ENUMS]
  Verify: `go test ./compiler/codegen/... -run TestSpecializedEnumTypes -v`
- Task 02: Specialize generic structs [W08-P05-T02-SPEC-STRUCTS]
  Verify: `go test ./compiler/codegen/... -run TestGenericStructLayout -v`

### Phase 06: Generic Proof Programs [W08-P06-PROOF]

- Task 01: `identity_generic.fuse` [W08-P06-T01-IDENTITY]
  Verify: `go test ./tests/e2e/... -run TestIdentityGeneric -v`
- Task 02: `multiple_instantiations.fuse` [W08-P06-T02-MULTIPLE]
  Verify: `go test ./tests/e2e/... -run TestMultipleInstantiations -v`
- Task 03: Distinct specializations in C [W08-P06-T03-DISTINCT]
  Verify: `go test ./compiler/codegen/... -run TestDistinctSpecializations -v`

### Wave Closure Phase [W08-PCL-WAVE-CLOSURE]

- Task 01: Retire generics stubs [W08-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W08`
- Task 02: WC008 entry [W08-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC008" docs/learning-log.md`

## Wave 09: Ownership and Liveness

Goal: compute ownership and liveness once, enforce no-borrow-in-struct-field,
and emit actual destructor calls for types with `Drop` impls.

Entry criterion: W08 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- ownership metadata complete on HIR
- liveness computed exactly once per function
- borrow lowering uses `InstrBorrow` with precise kind
- no borrows in struct fields (reference §54.1)
- drop metadata flows from checker to codegen
- `InstrDrop` emits `TypeName_drop(&_lN);` for types with `Drop`
- proof program: type with `Drop` impl demonstrates observable cleanup

Proof of completion:

```
go test ./compiler/liveness/... -v
go test ./compiler/liveness/... -run TestSingleLiveness -v
go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v
go test ./compiler/codegen/... -run TestDestructorCallEmitted -v
go test ./tests/e2e/... -run TestDropObservable -v
```

### Phase 00: Stub Audit [W09-P00-STUB-AUDIT]

- Task 01: Liveness audit [W09-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W09 -phase P00`

### Phase 01: Ownership Semantics [W09-P01-OWNERSHIP]

- Task 01: Ownership contexts [W09-P01-T01-CONTEXTS]
  Verify: `go test ./compiler/liveness/... -run TestOwnershipContexts -v`
- Task 02: Borrow rules and no-borrow-in-field [W09-P01-T02-BORROW]
  Verify: `go test ./compiler/liveness/... -run TestBorrowRules -v`

### Phase 02: Single Liveness Computation [W09-P02-LIVENESS]

- Task 01: Live-after data [W09-P02-T01-LIVE-AFTER]
  Verify: `go test ./compiler/liveness/... -run TestLiveAfter -v`
- Task 02: Last-use and destroy-after [W09-P02-T02-LAST-USE]
  Verify: `go test ./compiler/liveness/... -run TestLastUse -v`

### Phase 03: Drop Intent and Codegen [W09-P03-DROP]

- Task 01: Insert drop intent [W09-P03-T01-DROP-INTENT]
  Verify: `go test ./compiler/liveness/... -run TestDropIntent -v`
- Task 02: Drop trait metadata [W09-P03-T02-DROP-METADATA]
  Verify: `go test ./compiler/codegen/... -run TestDropTraitMetadata -v`
- Task 03: Emit destructor calls [W09-P03-T03-EMIT]
  Verify: `go test ./compiler/codegen/... -run TestDestructorCallEmitted -v`

### Phase 04: Control-Flow Destruction [W09-P04-CFLOW-DROP]

- Task 01: Loops, breaks, early returns [W09-P04-T01-CFLOW]
  Verify: `go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v`

### Phase 05: Drop Proof Program [W09-P05-PROOF]

- Task 01: `drop_observable.fuse` [W09-P05-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestDropObservable -v`

### Wave Closure Phase [W09-PCL-WAVE-CLOSURE]

- Task 01: Retire liveness and drop stubs [W09-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W09`
- Task 02: WC009 entry [W09-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC009" docs/learning-log.md`

## Wave 10: Pattern Matching

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

### Phase 00: Stub Audit [W10-P00-STUB-AUDIT]

- Task 01: Pattern matching audit [W10-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W10 -phase P00`

### Phase 01: Exhaustiveness [W10-P01-EXHAUSTIVENESS]

- Task 01: Exhaustiveness over enums and bools [W10-P01-T01-ENUMS]
  Verify: `go test ./compiler/check/... -run TestExhaustivenessChecking -v`
- Task 02: Unreachable arm detection [W10-P01-T02-UNREACHABLE]
  Verify: `go test ./compiler/check/... -run TestUnreachableArmDetection -v`

### Phase 02: Match Lowering [W10-P02-LOWER]

- Task 01: Cascading branches [W10-P02-T01-CASCADE]
  Verify: `go test ./compiler/lower/... -run TestMatchDispatch -v`
- Task 02: Discriminant access [W10-P02-T02-DISCRIM]
  Verify: `go test ./compiler/lower/... -run TestEnumDiscriminantAccess -v`
- Task 03: Payload extraction [W10-P02-T03-PAYLOAD]
  Verify: `go test ./compiler/lower/... -run TestPayloadExtraction -v`

### Phase 03: Extended Pattern Forms [W10-P03-EXT-PATTERNS]

- Task 01: Or-pattern lowering [W10-P03-T01-OR]
  Verify: `go test ./compiler/lower/... -run TestOrPattern -v`
- Task 02: Range pattern lowering [W10-P03-T02-RANGE]
  Verify: `go test ./compiler/lower/... -run TestRangePattern -v`
- Task 03: `@`-binding [W10-P03-T03-AT]
  Verify: `go test ./compiler/lower/... -run TestAtBinding -v`

### Phase 04: Match Proof Program [W10-P04-PROOF]

- Task 01: `match_enum_dispatch.fuse` [W10-P04-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestMatchEnumDispatch -v`

### Wave Closure Phase [W10-PCL-WAVE-CLOSURE]

- Task 01: Retire pattern matching stubs [W10-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W10`
- Task 02: WC010 entry [W10-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC010" docs/learning-log.md`

## Wave 11: Error Propagation

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

### Phase 00: Stub Audit [W11-P00-STUB-AUDIT]

- Task 01: `?` audit [W11-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W11 -phase P00`

### Phase 01: Type Checking `?` [W11-P01-CHECK]

- Task 01: `?` on Result [W11-P01-T01-RESULT]
  Verify: `go test ./compiler/check/... -run TestQuestionTypecheck -v`
- Task 02: `?` on Option [W11-P01-T02-OPTION]
  Verify: `go test ./compiler/check/... -run TestQuestionOptionTypecheck -v`

### Phase 02: Lowering `?` [W11-P02-LOWER]

- Task 01: Branch-and-early-return [W11-P02-T01-BRANCH]
  Verify: `go test ./compiler/lower/... -run TestQuestionBranch -v`

### Phase 03: Proof Program [W11-P03-PROOF]

- Task 01: `error_propagation.fuse` [W11-P03-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestErrorPropagation -v`

### Wave Closure Phase [W11-PCL-WAVE-CLOSURE]

- Task 01: Retire `?` stubs [W11-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W11`
- Task 02: WC011 entry [W11-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC011" docs/learning-log.md`

## Wave 12: Closures and Callable Traits

Goal: capture analysis, environment-struct generation, closure lifting,
automatic `Fn`/`FnMut`/`FnOnce` implementation, call desugaring.

Entry criterion: W11 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- capture analysis scans closure bodies for outer-variable references
- environment struct type generated per closure
- closure body lifted to standalone MIR function
- closure expression emits struct init + function pointer pair
- `Fn`/`FnMut`/`FnOnce` declared as intrinsic traits; stdlib core re-exports
- closures auto-implement the tightest matching trait
- function pointers auto-implement all three
- call desugaring: `f(args)` → `f.call(args)` / `call_mut` / `call_once`
  based on tightest bound

Proof of completion:

```
go test ./compiler/lower/... -run TestCaptureAnalysis -v
go test ./compiler/lower/... -run TestClosureLifting -v
go test ./compiler/lower/... -run TestClosureConstruction -v
go test ./compiler/check/... -run TestCallableAutoImpl -v
go test ./tests/e2e/... -run TestClosureCaptureRuns -v
```

### Phase 00: Stub Audit [W12-P00-STUB-AUDIT]

- Task 01: Closure audit [W12-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W12 -phase P00`

### Phase 01: Capture and Lift [W12-P01-CAPTURE-LIFT]

- Task 01: Capture analysis [W12-P01-T01-CAPTURE]
  Verify: `go test ./compiler/lower/... -run TestCaptureAnalysis -v`
- Task 02: Environment struct + lifted body [W12-P01-T02-LIFT]
  Verify: `go test ./compiler/lower/... -run TestClosureLifting -v`
- Task 03: Closure construction [W12-P01-T03-CONSTRUCT]
  Verify: `go test ./compiler/lower/... -run TestClosureConstruction -v`

### Phase 02: Callable Traits [W12-P02-CALLABLE]

- Task 01: Intrinsic `Fn`/`FnMut`/`FnOnce` [W12-P02-T01-DECLARE]
  Verify: `go test ./compiler/check/... -run TestCallableTraitDeclaration -v`
- Task 02: Auto-impl [W12-P02-T02-AUTO]
  Verify: `go test ./compiler/check/... -run TestCallableAutoImpl -v`
- Task 03: Call desugaring [W12-P02-T03-DESUGAR]
  Verify: `go test ./compiler/lower/... -run TestCallDesugar -v`

### Phase 03: Closure Proof Program [W12-P03-PROOF]

- Task 01: `closure_capture.fuse` [W12-P03-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestClosureCaptureRuns -v`

### Wave Closure Phase [W12-PCL-WAVE-CLOSURE]

- Task 01: Retire closure stubs [W12-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W12`
- Task 02: WC012 entry [W12-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC012" docs/learning-log.md`

## Wave 13: Trait Objects (`dyn Trait`)

Goal: implement dynamic dispatch via `dyn Trait`. Fat-pointer representation,
vtable layout, object safety rules, multi-trait object type, and proof
program.

Entry criterion: W12 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- object-safety checker rejects non-object-safe traits at `dyn Trait` use
  sites (reference §48.1)
- `dyn Trait` lowers to a fat pointer `{ data: Ptr[()], vtable:
  Ptr[Vtable_Trait] }` (reference §57.8)
- vtables emitted as static read-only tables with deterministic layout
  `[size, align, drop_fn, method_1, ...]` (reference §57.8)
- `dyn A + B` produces combined vtable (alphabetical trait order)
- ownership forms `ref dyn Trait`, `mutref dyn Trait`, `owned dyn Trait`
  lower correctly
- proof program: heterogeneous collection of `owned dyn Trait` values
  dispatches correctly

Proof of completion:

```
go test ./compiler/check/... -run TestObjectSafety -v
go test ./compiler/lower/... -run TestDynTraitFatPointer -v
go test ./compiler/codegen/... -run TestVtableEmission -v
go test ./compiler/codegen/... -run TestDynTraitMulti -v
go test ./tests/e2e/... -run TestDynDispatchProof -v
```

### Phase 00: Stub Audit [W13-P00-STUB-AUDIT]

- Task 01: `dyn Trait` audit [W13-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W13 -phase P00`

### Phase 01: Object Safety [W13-P01-OBJECT-SAFETY]

- Task 01: Object-safety checker [W13-P01-T01-OBJECT-SAFETY]
  DoD: traits with generic methods, Self in non-receiver positions,
  associated constants, or non-ref/mutref/owned receivers are rejected at
  `dyn Trait` use sites.
  Verify: `go test ./compiler/check/... -run TestObjectSafety -v`

### Phase 02: Fat Pointer Lowering [W13-P02-FAT-POINTER]

- Task 01: Lower `dyn Trait` to fat pointer [W13-P02-T01-FAT]
  Verify: `go test ./compiler/lower/... -run TestDynTraitFatPointer -v`
- Task 02: Lower `ref dyn Trait`, `mutref dyn Trait`, `owned dyn Trait`
  [W13-P02-T02-OWNERSHIP-FORMS]
  Verify: `go test ./compiler/lower/... -run TestDynOwnershipForms -v`

### Phase 03: Vtable Emission [W13-P03-VTABLE]

- Task 01: Deterministic vtable layout [W13-P03-T01-VTABLE-LAYOUT]
  DoD: per (trait, concrete impl), emit a static vtable with fields
  `[size, align, drop_fn, method_1, ...]` in trait declaration order.
  Verify: `go test ./compiler/codegen/... -run TestVtableEmission -v`
- Task 02: Multi-trait combined vtable [W13-P03-T02-MULTI-TRAIT]
  DoD: `dyn A + B` produces a combined vtable; trait ordering is
  alphabetical for determinism.
  Verify: `go test ./compiler/codegen/... -run TestDynTraitMulti -v`

### Phase 04: Method Dispatch Lowering [W13-P04-DISPATCH]

- Task 01: Call through fat pointer [W13-P04-T01-DISPATCH]
  DoD: a method call on a `dyn Trait` receiver loads the method pointer
  from the vtable and calls it with the data pointer as receiver.
  Verify: `go test ./compiler/codegen/... -run TestDynMethodDispatch -v`

### Phase 05: Dynamic Dispatch Proof [W13-P05-PROOF]

- Task 01: `dyn_dispatch.fuse` [W13-P05-T01-PROOF]
  DoD: a program builds a heterogeneous `List[owned dyn Draw]` containing
  two different implementations, iterates it, calls a method that sums
  into the exit code. The exit code proves both implementations ran.
  Verify: `go test ./tests/e2e/... -run TestDynDispatchProof -v`

### Wave Closure Phase [W13-PCL-WAVE-CLOSURE]

- Task 01: Retire `dyn Trait` stubs [W13-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W13`
- Task 02: WC013 entry [W13-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC013" docs/learning-log.md`

## Wave 14: Compile-Time Evaluation (`const fn`)

Goal: implement a deterministic compile-time evaluator that operates on
checked HIR and supports `const fn`, const contexts (`const`, `static`,
array lengths, enum discriminant values), and memory intrinsics in
const position (`size_of[T]()`, `align_of[T]()`).

Entry criterion: W13 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- const evaluator operates over checked HIR and is deterministic
  (same input → same bytes)
- `const fn` body restrictions enforced: no FFI, no allocation, no threads,
  no non-`const` calls, no interior mutability (reference §46.1)
- `const` initializers evaluate at compile time; `static` initializers too
- array length expressions that are `const` evaluate to fixed usize
- enum discriminant expressions evaluate
- `size_of[T]()` and `align_of[T]()` return `USize` computed at compile
  time for every concrete type
- proof program: `const FACT_10: U64 = factorial(10);` evaluates to the
  correct value at compile time; program exits with FACT_10 % 256

Proof of completion:

```
go test ./compiler/consteval/... -v
go test ./compiler/consteval/... -run TestConstFnRestrictions -v
go test ./compiler/consteval/... -run TestSizeOfAlignOf -v
go test ./tests/e2e/... -run TestConstFnProof -v
```

### Phase 00: Stub Audit [W14-P00-STUB-AUDIT]

- Task 01: Const evaluator audit [W14-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W14 -phase P00`

### Phase 01: Const Evaluator [W14-P01-EVALUATOR]

- Task 01: Evaluator over checked HIR [W14-P01-T01-EVAL]
  DoD: evaluator supports arithmetic, comparison, bitwise ops, if/loop,
  `match`, struct/tuple construction and destructuring, array indexing
  with constant indices, and recursive calls to other `const fn`.
  Verify: `go test ./compiler/consteval/... -run TestEvaluatorCore -v`
- Task 02: Determinism [W14-P01-T02-DETERMINISM]
  DoD: three runs produce byte-identical results.
  Verify: `go test ./compiler/consteval/... -run TestEvaluatorDeterminism -count=3 -v`

### Phase 02: `const fn` Restrictions [W14-P02-RESTRICTIONS]

- Task 01: Reject non-const operations [W14-P02-T01-RESTRICT]
  DoD: FFI, allocation, thread ops, non-const calls, and interior mutability
  in a `const fn` body produce a diagnostic.
  Verify: `go test ./compiler/consteval/... -run TestConstFnRestrictions -v`

### Phase 03: Const Contexts [W14-P03-CONTEXTS]

- Task 01: `const` initializers [W14-P03-T01-CONST-INIT]
  Verify: `go test ./compiler/consteval/... -run TestConstInit -v`
- Task 02: `static` initializers [W14-P03-T02-STATIC-INIT]
  Verify: `go test ./compiler/consteval/... -run TestStaticInit -v`
- Task 03: Array length expressions [W14-P03-T03-ARRAY-LEN]
  Verify: `go test ./compiler/consteval/... -run TestArrayLenConst -v`
- Task 04: Enum discriminant values [W14-P03-T04-DISCRIMINANT]
  Verify: `go test ./compiler/consteval/... -run TestDiscriminantConst -v`

### Phase 04: Memory Intrinsics in Const Context [W14-P04-MEM-INTRINSICS]

- Task 01: `size_of[T]()` and `align_of[T]()` [W14-P04-T01-SIZEOF]
  DoD: callable in `const` contexts for every concrete type.
  Verify: `go test ./compiler/consteval/... -run TestSizeOfAlignOf -v`

### Phase 05: `const fn` Proof Program [W14-P05-PROOF]

- Task 01: `const_fn.fuse` [W14-P05-T01-PROOF]
  DoD: a program with `const fn factorial` and `const FACT_10: U64 =
  factorial(10);` evaluates to 3628800 at compile time; `main` returns
  `(FACT_10 % 256) as I32` (= 0x80 = 128).
  Verify: `go test ./tests/e2e/... -run TestConstFnProof -v`

### Wave Closure Phase [W14-PCL-WAVE-CLOSURE]

- Task 01: Retire `const fn` stubs [W14-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W14`
- Task 02: WC014 entry [W14-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC014" docs/learning-log.md`

## Wave 15: Lowering and MIR Consolidation

Goal: consolidate lowering across all features introduced so far. Add
cast lowering, function pointer lowering, slice range indexing, struct
update syntax, and overflow-aware arithmetic lowering. Enforce every
MIR invariant at pass boundaries.

Entry criterion: W14 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- MIR blocks terminate structurally (reference §57.4)
- sealed blocks stay sealed after `return`/`break`/`continue`
- no move-after-move violations
- method calls lower distinctly from field reads
- equality lowers through type-specific semantics for non-scalars
- optional chaining lowers to conditional read
- `as` cast lowers per reference §28.1
- function pointer values lower per reference §29.1
- slice range `arr[a..b]` lowers to slice descriptor (reference §32.1)
- struct update `{ ..base }` lowers with explicit-field precedence
  (reference §45.1)
- overflow-aware method calls (`wrapping_add`, `checked_add`, etc.)
  lower to the right MIR ops (reference §33.1)
- invariant walkers green on every pass boundary
- property tests cover MIR transformations

Proof of completion:

```
go test ./compiler/lower/... -v
go test ./compiler/mir/... -v
go test ./compiler/lower/... -run TestInvariantWalkersPass -v
go test ./compiler/lower/... -run TestCastLowering -v
go test ./compiler/lower/... -run TestFnPointerLowering -v
go test ./compiler/lower/... -run TestSliceRangeLowering -v
go test ./compiler/lower/... -run TestStructUpdateLowering -v
go test ./compiler/lower/... -run TestOverflowArithmeticLowering -v
go test ./tests/property/... -run TestMirTransforms -v
```

### Phase 00: Stub Audit [W15-P00-STUB-AUDIT]

- Task 01: MIR audit [W15-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W15 -phase P00`

### Phase 01: Structural Invariants [W15-P01-INVARIANTS]

- Task 01: Sealed blocks [W15-P01-T01-SEALED]
  Verify: `go test ./compiler/lower/... -run TestSealedBlocks -v`
- Task 02: Structural divergence [W15-P01-T02-DIV]
  Verify: `go test ./compiler/lower/... -run TestStructuralDivergence -v`
- Task 03: No-move-after-move [W15-P01-T03-NO-MOVE-AFTER]
  Verify: `go test ./compiler/lower/... -run TestNoMoveAfterMove -v`

### Phase 02: Operator and Method Lowering [W15-P02-OPS]

- Task 01: Borrow instruction [W15-P02-T01-BORROW]
  Verify: `go test ./compiler/lower/... -run TestBorrowInstr -v`
- Task 02: Method vs field [W15-P02-T02-METHOD-FIELD]
  Verify: `go test ./compiler/lower/... -run TestMethodVsField -v`
- Task 03: Semantic equality [W15-P02-T03-SEMANTIC-EQ]
  Verify: `go test ./compiler/lower/... -run TestSemanticEquality -v`
- Task 04: Optional chaining [W15-P02-T04-OPTIONAL-CHAIN]
  Verify: `go test ./compiler/lower/... -run TestOptionalChainLowering -v`

### Phase 03: Cast, FnPointer, Slice Range, Struct Update [W15-P03-EXPR-FORMS]

- Task 01: Cast lowering [W15-P03-T01-CAST]
  Verify: `go test ./compiler/lower/... -run TestCastLowering -v`
- Task 02: Function pointer lowering [W15-P03-T02-FN-POINTER]
  Verify: `go test ./compiler/lower/... -run TestFnPointerLowering -v`
- Task 03: Slice range indexing [W15-P03-T03-SLICE-RANGE]
  Verify: `go test ./compiler/lower/... -run TestSliceRangeLowering -v`
- Task 04: Struct update syntax [W15-P03-T04-STRUCT-UPDATE]
  Verify: `go test ./compiler/lower/... -run TestStructUpdateLowering -v`

### Phase 04: Overflow-Aware Arithmetic Lowering [W15-P04-OVERFLOW]

- Task 01: `wrapping_*`, `checked_*`, `saturating_*` lower correctly
  [W15-P04-T01-OVERFLOW]
  DoD: each method lowers to distinct MIR op tags that codegen (W17) picks
  up for policy-appropriate emission.
  Verify: `go test ./compiler/lower/... -run TestOverflowArithmeticLowering -v`

### Phase 05: Property Tests [W15-P05-PROPERTY]

- Task 01: MIR transform property tests [W15-P05-T01-PROPERTY]
  Verify: `go test ./tests/property/... -run TestMirTransforms -v`

### Wave Closure Phase [W15-PCL-WAVE-CLOSURE]

- Task 01: Retire lower/mir stubs [W15-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W15`
- Task 02: WC015 entry [W15-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC015" docs/learning-log.md`

## Wave 16: Runtime ABI

Goal: replace the W05 stub runtime with a full implementation. Wire
`spawn`, channel ops, panic, `ThreadHandle.join()`, and all IO/process/
time/thread/sync surface to real runtime calls.

Entry criterion: W15 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- runtime provides memory, panic, IO, process, time, thread, sync, channel
  surface
- runtime tests pass on Linux, macOS, Windows
- `spawn` lowers to `fuse_rt_thread_spawn`
- `handle.join()` lowers to `fuse_rt_thread_join` returning
  `Result[T, ThreadError]`
- channel ops lower to `fuse_rt_chan_send/recv/close`
- panic lowers to `fuse_rt_panic`
- proof programs: spawn produces observable effect; channel round-trip
  works

Proof of completion:

```
make runtime
cd runtime && make test
go test ./compiler/codegen/... -run TestSpawnEmission -v
go test ./compiler/codegen/... -run TestChannelOpsEmission -v
go test ./compiler/codegen/... -run TestJoinEmission -v
go test ./tests/e2e/... -run TestSpawnObservable -v
go test ./tests/e2e/... -run TestChannelRoundTrip -v
```

### Phase 00: Stub Audit [W16-P00-STUB-AUDIT]

- Task 01: Runtime audit [W16-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W16 -phase P00`

### Phase 01: Memory and Panic [W16-P01-MEM-PANIC]

- Task 01: Runtime header [W16-P01-T01-HEADER]
  Verify: `go run tools/checkruntime/main.go -header-syntax`
- Task 02: Memory and panic impl [W16-P01-T02-MEM-PANIC]
  Verify: `cd runtime && make test`

### Phase 02: IO, Process, Time [W16-P02-IO-PROC-TIME]

- Task 01: IO [W16-P02-T01-IO]
  Verify: `cd runtime && ./tests/test_io`
- Task 02: Process and time [W16-P02-T02-PROC-TIME]
  Verify: `cd runtime && ./tests/test_process`

### Phase 03: Threads and Sync [W16-P03-THREAD-SYNC]

- Task 01: Thread spawn and TLS [W16-P03-T01-SPAWN]
  Verify: `cd runtime && ./tests/test_thread`
- Task 02: Sync primitives [W16-P03-T02-SYNC]
  Verify: `cd runtime && ./tests/test_sync`

### Phase 04: Compiler-Runtime Bridging [W16-P04-BRIDGE]

- Task 01: Spawn lowering [W16-P04-T01-SPAWN-LOWER]
  Verify: `go test ./compiler/codegen/... -run TestSpawnEmission -v`
- Task 02: Join lowering [W16-P04-T02-JOIN]
  Verify: `go test ./compiler/codegen/... -run TestJoinEmission -v`
- Task 03: Channel op lowering [W16-P04-T03-CHAN-LOWER]
  Verify: `go test ./compiler/codegen/... -run TestChannelOpsEmission -v`

### Phase 05: Concurrency Proof Programs [W16-P05-PROOF]

- Task 01: `spawn_observable.fuse` [W16-P05-T01-SPAWN-PROOF]
  Verify: `go test ./tests/e2e/... -run TestSpawnObservable -v`
- Task 02: `channel_round_trip.fuse` [W16-P05-T02-CHAN-PROOF]
  Verify: `go test ./tests/e2e/... -run TestChannelRoundTrip -v`

### Wave Closure Phase [W16-PCL-WAVE-CLOSURE]

- Task 01: Retire runtime stubs [W16-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W16`
- Task 02: WC016 entry [W16-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC016" docs/learning-log.md`

## Wave 17: Codegen C11 Hardening

Goal: enforce every representation contract from reference §57; add
codegen for `@repr(C/packed/Uxx|Ixx)`, `@align(N)`, `@inline`, `@cold`,
compiler intrinsics, variadic call ABI, `Ptr.null[T]()`, `size_of[T]()`/
`align_of[T]()` emission at runtime positions, union layout, and overflow-
aware default policy.

Entry criterion: W16 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- two pointer categories separated (reference §57.1)
- unit erasure total (reference §57.2)
- monomorphization completeness enforced (reference §57.3)
- divergence structural (reference §57.4)
- composite types emitted before use (reference §57.5)
- identifier sanitization and module-qualified mangling (reference §57.6)
- closure representation (reference §57.7)
- trait object representation (reference §57.8)
- channel and thread runtime ABI (reference §57.9)
- alignment and padding stable (reference §57.10)
- `@repr(C)`, `@repr(packed)`, `@repr(Uxx|Ixx)`, `@align(N)` produce
  conforming C layouts
- `@inline`, `@inline(always)`, `@inline(never)`, `@cold` emit the
  corresponding backend annotations
- intrinsics (`unreachable`, `likely`, `unlikely`, `fence`, `prefetch`,
  `assume`) emit correctly
- variadic extern calls follow platform C variadic ABI
- `Ptr.null[T]()` emits `((T*)0)`
- `size_of[T]()` / `align_of[T]()` in runtime position emit literal
- union type emission (largest-field sizing, strictest alignment)
- overflow-aware default: debug panics; release is deterministic per
  target profile
- determinism gates: same source → byte-identical C
- every regression from L001–L015 has a test

Proof of completion:

```
go test ./compiler/codegen/... -v
go test ./compiler/codegen/... -run TestPointerCategories -v
go test ./compiler/codegen/... -run TestTotalUnitErasure -v
go test ./compiler/codegen/... -run TestStructuralDivergence -v
go test ./compiler/codegen/... -run TestIdentifierSanitization -v
go test ./compiler/codegen/... -run TestModuleMangling -v
go test ./compiler/codegen/... -run TestAggregateZeroInit -v
go test ./compiler/codegen/... -run TestReprEmission -v
go test ./compiler/codegen/... -run TestAlignEmission -v
go test ./compiler/codegen/... -run TestInlineEmission -v
go test ./compiler/codegen/... -run TestIntrinsicsEmission -v
go test ./compiler/codegen/... -run TestVariadicCall -v
go test ./compiler/codegen/... -run TestPtrNullEmission -v
go test ./compiler/codegen/... -run TestSizeOfEmission -v
go test ./compiler/codegen/... -run TestUnionLayout -v
go test ./compiler/codegen/... -run TestOverflowPolicy -v
make repro
```

### Phase 00: Stub Audit [W17-P00-STUB-AUDIT]

- Task 01: Codegen audit [W17-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W17 -phase P00`

### Phase 01: Type Emission [W17-P01-TYPES]

- Task 01: Types before use [W17-P01-T01-DEFS-FIRST]
  Verify: `go test ./compiler/codegen/... -run TestTypeDefsFirst -v`
- Task 02: Identifier sanitization [W17-P01-T02-SANITIZE]
  Verify: `go test ./compiler/codegen/... -run TestIdentifierSanitization -v`
- Task 03: Module-qualified mangling [W17-P01-T03-MANGLE]
  Verify: `go test ./compiler/codegen/... -run TestModuleMangling -v`

### Phase 02: Pointer Categories [W17-P02-POINTERS]

- Task 01: Two pointer categories [W17-P02-T01-TWO]
  Verify: `go test ./compiler/codegen/... -run TestPointerCategories -v`
- Task 02: Call-site adaptation [W17-P02-T02-CALL-SITE]
  Verify: `go test ./compiler/codegen/... -run TestCallSiteAdaptation -v`
- Task 03: `Ptr.null[T]()` emission [W17-P02-T03-NULL]
  Verify: `go test ./compiler/codegen/... -run TestPtrNullEmission -v`

### Phase 03: Unit and Aggregate [W17-P03-UNIT-AGG]

- Task 01: Total unit erasure [W17-P03-T01-UNIT]
  Verify: `go test ./compiler/codegen/... -run TestTotalUnitErasure -v`
- Task 02: Typed aggregate fallback [W17-P03-T02-AGG]
  Verify: `go test ./compiler/codegen/... -run TestAggregateZeroInit -v`
- Task 03: Union layout [W17-P03-T03-UNION]
  Verify: `go test ./compiler/codegen/... -run TestUnionLayout -v`

### Phase 04: Divergence [W17-P04-DIV]

- Task 01: Structural divergence [W17-P04-T01-DIV]
  Verify: `go test ./compiler/codegen/... -run TestStructuralDivergence -v`

### Phase 05: Layout Control Emission [W17-P05-LAYOUT]

- Task 01: `@repr(C)` / `@repr(packed)` / `@repr(Uxx|Ixx)` emission
  [W17-P05-T01-REPR]
  Verify: `go test ./compiler/codegen/... -run TestReprEmission -v`
- Task 02: `@align(N)` emission [W17-P05-T02-ALIGN]
  Verify: `go test ./compiler/codegen/... -run TestAlignEmission -v`

### Phase 06: Inline and Cold Annotations [W17-P06-INLINE]

- Task 01: `@inline` / `@inline(always)` / `@inline(never)` / `@cold`
  [W17-P06-T01-INLINE]
  DoD: emit corresponding C compiler annotations (`inline`, `__attribute__
  ((always_inline))`, `__attribute__((noinline))`, `__attribute__((cold))`
  or platform equivalents).
  Verify: `go test ./compiler/codegen/... -run TestInlineEmission -v`

### Phase 07: Compiler Intrinsics [W17-P07-INTRINSICS]

- Task 01: `unreachable`, `likely`, `unlikely` [W17-P07-T01-BUILTINS]
  Verify: `go test ./compiler/codegen/... -run TestIntrinsicsEmission -v`
- Task 02: `fence`, `prefetch`, `assume` [W17-P07-T02-MEM-INTRINSICS]
  Verify: `go test ./compiler/codegen/... -run TestMemIntrinsicsEmission -v`

### Phase 08: Variadic Call ABI [W17-P08-VARIADIC]

- Task 01: Variadic call site ABI [W17-P08-T01-VARIADIC]
  DoD: variadic extern calls follow the host platform's C variadic ABI;
  float → double promotion and short-int → int promotion applied at call
  site.
  Verify: `go test ./compiler/codegen/... -run TestVariadicCall -v`

### Phase 09: Memory Intrinsics Emission [W17-P09-MEM-EMISSION]

- Task 01: `size_of[T]()` / `align_of[T]()` emission [W17-P09-T01-SIZEOF]
  DoD: in runtime positions these lower to literal `USize` values.
  Verify: `go test ./compiler/codegen/... -run TestSizeOfEmission -v`
- Task 02: `size_of_val(ref v)` [W17-P09-T02-SIZEOF-VAL]
  Verify: `go test ./compiler/codegen/... -run TestSizeOfValEmission -v`

### Phase 10: Overflow Default Policy [W17-P10-OVERFLOW]

- Task 01: Debug-mode overflow panic [W17-P10-T01-DEBUG-PANIC]
  Verify: `go test ./compiler/codegen/... -run TestOverflowDebugPanic -v`
- Task 02: Release-mode deterministic policy [W17-P10-T02-RELEASE-POLICY]
  DoD: default release behavior is documented (e.g. wrapping) and
  deterministic per target profile; CI goldens pin the choice.
  Verify: `go test ./compiler/codegen/... -run TestOverflowPolicy -v`

### Phase 11: Regressions and Determinism [W17-P11-REG-DET]

- Task 01: L001–L015 regression coverage [W17-P11-T01-REGRESSIONS]
  Verify: `go test ./compiler/codegen/... -run TestHistoricalRegressions -v`
- Task 02: Reproducibility [W17-P11-T02-REPRO]
  Verify: `make repro`

### Wave Closure Phase [W17-PCL-WAVE-CLOSURE]

- Task 01: Retire codegen stubs [W17-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W17`
- Task 02: WC017 entry [W17-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC017" docs/learning-log.md`

## Wave 18: CLI and Diagnostics

Goal: expose the compiler through a coherent command-line interface with
stable diagnostic rendering and developer workflow tooling.

Entry criterion: W17 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- `build`, `run`, `check`, `test`, `fmt`, `doc`, `repl`, `version`, `help`
  dispatch
- diagnostics render in text and JSON
- JSON output parseable and stable
- `fuse fmt` produces byte-stable output
- CLI exit codes documented and consistent

Proof of completion:

```
go test ./cmd/fuse/... -v
go test ./compiler/diagnostics/... -v
go test ./compiler/fmt/... -run TestFormatStable -v
go test ./compiler/doc/... -run TestDocCheck -v
go test ./tests/e2e/... -run TestCliBasicWorkflow -v
```

### Phase 00: Stub Audit [W18-P00-STUB-AUDIT]

- Task 01: CLI audit [W18-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W18 -phase P00`

### Phase 01: Subcommand Surface [W18-P01-SUBCOMMANDS]

- Task 01: Subcommand parser [W18-P01-T01-PARSER]
  Verify: `go test ./cmd/fuse/... -run TestSubcommandParser -v`
- Task 02: Wire all commands [W18-P01-T02-WIRE]
  Verify: `go test ./tests/e2e/... -run TestCliBasicWorkflow -v`

### Phase 02: Diagnostics [W18-P02-DIAGNOSTICS]

- Task 01: Text rendering [W18-P02-T01-TEXT]
  Verify: `go test ./compiler/diagnostics/... -run TestTextRendering -v`
- Task 02: JSON rendering [W18-P02-T02-JSON]
  Verify: `go test ./compiler/diagnostics/... -run TestJsonRendering -v`

### Phase 03: Workflow Tools [W18-P03-WORKFLOW]

- Task 01: `fmt` and `doc` [W18-P03-T01-FMT-DOC]
  Verify: `go test ./compiler/fmt/... -run TestFormatStable -v`
- Task 02: `repl` and `testrunner` [W18-P03-T02-REPL]
  Verify: `go test ./compiler/repl/... -run TestReplRoundTrip -v`

### Wave Closure Phase [W18-PCL-WAVE-CLOSURE]

- Task 01: Retire CLI stubs [W18-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W18`
- Task 02: WC018 entry [W18-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC018" docs/learning-log.md`

## Wave 19: Stdlib Core

Goal: implement the OS-free core standard library. Includes core traits,
primitive method surface, strings, collections, iterators, `Cell`/`RefCell`
interior mutability, `Ptr.null` surface, overflow-aware arithmetic methods,
runtime bridge files.

Entry criterion: W18 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- core traits shipped: equality, ordering, hashing, formatting, default
- primitive method surface matches reference §24
- `String` and formatting primitives work
- `List`, `Map`, `Set`, iterators work
- `Cell[T]` and `RefCell[T]` with runtime borrow tracking (reference §51.1)
- `Ptr.null[T]()`, `is_null()` surface (reference §35.1)
- overflow-aware methods (`wrapping_*`, `checked_*`, `saturating_*`) ship
- intrinsic `Send`/`Sync`/`Copy`/`Fn`/`FnMut`/`FnOnce` re-exported
- core bridge files exist per repository-layout.md
- docs coverage check passes

Proof of completion:

```
fuse build stdlib/core/...
go test ./tests/... -run TestCoreStdlib -v
fuse doc --check stdlib/core/
```

### Phase 00: Stub Audit [W19-P00-STUB-AUDIT]

- Task 01: Core stdlib audit [W19-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W19 -phase P00`

### Phase 01: Core Traits and Primitives [W19-P01-TRAITS]

- Task 01: Core traits [W19-P01-T01-TRAITS]
  Verify: `fuse build stdlib/core/traits/...`
- Task 02: Primitive methods [W19-P01-T02-PRIM]
  Verify: `fuse build stdlib/core/primitives/...`
- Task 03: Re-export intrinsic marker and callable traits
  [W19-P01-T03-REEXPORT]
  Verify: `go test ./tests/... -run TestCoreReExports -v`

### Phase 02: Strings, Collections, Iteration [W19-P02-COLLECTIONS]

- Task 01: `String` and formatting [W19-P02-T01-STRING]
  Verify: `fuse build stdlib/core/string/...`
- Task 02: `List`, `Map`, `Set`, iterators [W19-P02-T02-COLLECTIONS]
  Verify: `fuse build stdlib/core/collections/...`

### Phase 03: Interior Mutability [W19-P03-INTERIOR-MUT]

- Task 01: `Cell[T]` [W19-P03-T01-CELL]
  DoD: mutation through shared reference for `Copy` types; not `Send`,
  not `Sync`.
  Verify: `fuse build stdlib/core/cell/... && go test ./tests/... -run TestCell -v`
- Task 02: `RefCell[T]` with runtime borrow tracking [W19-P03-T02-REFCELL]
  DoD: runtime borrow count; `borrow()` increments shared count;
  `borrow_mut()` requires no outstanding borrows; violations panic.
  Verify: `go test ./tests/... -run TestRefCell -v`

### Phase 04: Pointer and Memory Surface [W19-P04-PTR-MEM]

- Task 01: `Ptr.null[T]()` API [W19-P04-T01-NULL]
  Verify: `go test ./tests/... -run TestPtrNull -v`
- Task 02: `size_of[T]()` / `align_of[T]()` wrappers [W19-P04-T02-SIZEOF]
  Verify: `go test ./tests/... -run TestSizeOfWrappers -v`

### Phase 05: Overflow-Aware Arithmetic Methods [W19-P05-OVERFLOW]

- Task 01: `wrapping_*`, `checked_*`, `saturating_*` methods
  [W19-P05-T01-OVERFLOW-METHODS]
  Verify: `go test ./tests/... -run TestOverflowMethods -v`

### Phase 06: Runtime Bridge [W19-P06-BRIDGE]

- Task 01: Core bridge files [W19-P06-T01-FILES]
  Verify: `fuse build stdlib/core/rt_bridge/...`
- Task 02: Docs coverage [W19-P06-T02-DOC]
  Verify: `fuse doc --check stdlib/core/`

### Wave Closure Phase [W19-PCL-WAVE-CLOSURE]

- Task 01: Retire core stdlib stubs [W19-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W19`
- Task 02: WC019 entry [W19-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC019" docs/learning-log.md`

## Wave 20: Custom Allocators

Goal: declare the `Allocator` trait, parameterize core collections over
an allocator, and prove a user-defined `BumpAllocator` can back a
collection.

Entry criterion: W19 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- `Allocator` trait declared in stdlib core with `alloc`, `dealloc`,
  `realloc` methods (reference §52.1)
- core collections (`Vec`, `HashMap`, `Box`, etc.) generic over allocator
  parameter; default resolves to `SystemAllocator`
- global allocator declared once per binary; default wraps runtime alloc
- all allocation paths route through the provided allocator; no silent
  fallback to global
- user-defined `BumpAllocator` backs a collection end-to-end
- proof program: `BumpAllocator` allocates twice, collection holds both,
  `reset()` reclaims all memory

Proof of completion:

```
fuse build stdlib/core/alloc/...
go test ./tests/... -run TestAllocatorTrait -v
go test ./tests/... -run TestCollectionsInAllocator -v
go test ./tests/e2e/... -run TestBumpAllocatorProof -v
```

### Phase 00: Stub Audit [W20-P00-STUB-AUDIT]

- Task 01: Allocator audit [W20-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W20 -phase P00`

### Phase 01: Allocator Trait and Global Allocator [W20-P01-TRAIT]

- Task 01: Declare `Allocator` trait [W20-P01-T01-TRAIT]
  Verify: `fuse build stdlib/core/alloc/... && go test ./tests/... -run TestAllocatorTrait -v`
- Task 02: Global allocator with runtime wrapper [W20-P01-T02-GLOBAL]
  DoD: `SystemAllocator` wraps `fuse_rt_alloc_*`; override mechanism
  documented.
  Verify: `go test ./tests/... -run TestGlobalAllocator -v`

### Phase 02: Parameterize Collections [W20-P02-PARAMETERIZE]

- Task 01: `Vec`/`HashMap`/`Box` take allocator param [W20-P02-T01-VEC]
  DoD: every allocation site routes through the provided allocator.
  Verify: `go test ./tests/... -run TestCollectionsInAllocator -v`

### Phase 03: Allocator Proof Program [W20-P03-PROOF]

- Task 01: `bump_allocator.fuse` [W20-P03-T01-PROOF]
  DoD: a program defines a `BumpAllocator` backed by a stack buffer,
  allocates a `Vec[I32, BumpAllocator]`, pushes two values, sums them,
  returns the sum as the exit code. `arena.reset()` must be observable
  (alloc after reset returns original offset).
  Verify: `go test ./tests/e2e/... -run TestBumpAllocatorProof -v`

### Wave Closure Phase [W20-PCL-WAVE-CLOSURE]

- Task 01: Retire allocator stubs [W20-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W20`
- Task 02: WC020 entry [W20-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC020" docs/learning-log.md`

## Wave 21: Stdlib Hosted

Goal: implement hosted stdlib on top of core; preserve the core/hosted
boundary.

Entry criterion: W20 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- IO, fs, os, time, thread (with `ThreadHandle`), sync, channel modules
  ship
- concurrency surface passes threaded tests
- hosted modules do not leak back into core

Proof of completion:

```
fuse build stdlib/full/...
go test ./tests/... -run TestHostedStdlib -v
go test ./tests/... -run TestConcurrency -v
```

### Phase 00: Stub Audit [W21-P00-STUB-AUDIT]

- Task 01: Hosted stdlib audit [W21-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W21 -phase P00`

### Phase 01: IO and OS [W21-P01-IO-OS]

- Task 01: IO modules [W21-P01-T01-IO]
  Verify: `fuse build stdlib/full/io/...`
- Task 02: OS modules [W21-P01-T02-OS]
  Verify: `fuse build stdlib/full/os/...`

### Phase 02: Threads, Sync, Channels [W21-P02-CONCURRENCY]

- Task 01: `ThreadHandle` module [W21-P02-T01-THREAD]
  DoD: `ThreadHandle[T]`, `spawn`, `join()`, detach on drop.
  Verify: `fuse build stdlib/full/thread/...`
- Task 02: Sync modules [W21-P02-T02-SYNC]
  DoD: `Mutex`, `RwLock`, `Cond`, `Once`, `Shared`, `@rank` enforcement
  already live in checker.
  Verify: `fuse build stdlib/full/sync/...`
- Task 03: Channels [W21-P02-T03-CHAN]
  Verify: `fuse build stdlib/full/chan/...`

### Wave Closure Phase [W21-PCL-WAVE-CLOSURE]

- Task 01: Retire hosted stubs [W21-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W21`
- Task 02: WC021 entry [W21-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC021" docs/learning-log.md`

## Wave 22: Stub Clearance Gate

Goal: single-purpose wave whose sole exit criterion is an empty Active
stubs table. The compiler does not proceed to Stage 2 self-hosting with
any unimplemented feature remaining.

This wave exists specifically because every past attempt at Fuse has
reached self-hosting with silent stubs still in place, and those stubs
later surfaced as missing features (L013, L015). The clearance wave is
the forcing function that guarantees they cannot slip through.

Entry criterion: W21 done. Phase 00 confirms no overdue stubs.

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

### Phase 00: Stub Audit [W22-P00-STUB-AUDIT]

- Task 01: Enumerate remaining stubs [W22-P00-T01-ENUMERATE]
  DoD: the Phase 00 audit produces a prioritized list of every remaining
  stub with its retirement path. A stub without a clear retirement path
  is escalated to the user.
  Verify: `go run tools/checkstubs/main.go -wave W22 -phase P00 -enumerate`

### Phase 01: Retire Remaining Stubs [W22-P01-RETIRE]

This phase has one task per remaining stub. Stubs are retired in reverse
dependency order: leaf features first, cross-cutting features last. Each
retirement follows the normal pattern: retire the stub in the code, add a
proof program that would fail if the stub were reinstated, update
STUBS.md.

- Task 01..N: one task per stub, each with its own `[W22-P01-Txx-...]`
  identifier.
  DoD: every stub retired with a corresponding proof program committed to
  `tests/e2e/` or a checker regression test. Each retirement records a
  line in the Stub history log.
  Verify: `go run tools/checkstubs/main.go -wave W22 -retired <stub-name>`

### Phase 02: Reference Status Audit [W22-P02-REF-AUDIT]

- Task 01: Verify every reference section is `DONE` [W22-P02-T01-REF]
  DoD: `tools/checkref -all-done` reports every feature section tagged
  `DONE — Wxx` pointing at the wave that retired it.
  Verify: `go run tools/checkref/main.go -all-done`
- Task 02: Verify every feature has a committed proof or regression
  [W22-P02-T02-PROOFS]
  DoD: `tests/e2e/README.md` lists a program or test for every reference
  feature. Orphan references are a CI failure.
  Verify: `go run tools/checkref/main.go -proof-coverage`

### Phase 03: Clean Build Gate [W22-P03-CLEAN-BUILD]

- Task 01: Full test suite passes on all hosts [W22-P03-T01-FULL-TEST]
  Verify: CI green on Linux, macOS, Windows for the full suite
  (unit + e2e + property + bootstrap harness).
- Task 02: Empty Active stubs [W22-P03-T02-EMPTY]
  Verify: `go run tools/checkstubs/main.go -require-empty-active`

### Wave Closure Phase [W22-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W22-PCL-T01-HISTORY]
  DoD: `## W22` block in STUBS.md Stub history lists every stub retired
  this wave, with the proof program that confirmed it.
  Verify: `go run tools/checkstubs/main.go -wave W22`
- Task 02: Write WC022 learning-log entry [W22-PCL-T02-CLOSURE-LOG]
  DoD: WC022 records which stubs remained until this late (and why), what
  the clearance wave surfaced, and the assertion that no stub remains.
  Verify: `grep "WC022" docs/learning-log.md`

## Wave 23: Stage 2 and Self-Hosting

Goal: bring up the self-hosted Fuse compiler and prove it compiles itself
reproducibly.

Entry criterion: W22 done. Phase 00 re-verifies the Active stubs table
is empty.

State on entry: `stage2/src/` is empty. All compiler infrastructure
complete. STUBS.md Active table is empty.

Exit criteria:

- stage1 compiles stage2 end-to-end
- stage2 recompiles itself
- reproducibility checks pass

Proof of completion:

```
fuse build stage2/src/... -o stage2_out/fusec2
stage2_out/fusec2 build stage2/src/... -o stage2_out/fusec2_gen2
make repro
go test ./tests/bootstrap/... -v
```

### Phase 00: Stub Audit [W23-P00-STUB-AUDIT]

- Task 01: Self-hosting audit [W23-P00-T01-ZERO-STUBS]
  DoD: Active stubs table empty. If not empty, the wave cannot begin and
  the missing retirements go back to W22.
  Verify: `go run tools/checkstubs/main.go -wave W23 -phase P00 -require-empty-active`

### Phase 01: Port Stage 2 [W23-P01-PORT]

- Task 01: Port frontend [W23-P01-T01-FRONTEND]
  Verify: `fuse check stage2/src/compiler/lex/...`
- Task 02: Port driver [W23-P01-T02-DRIVER]
  Verify: `fuse build stage2/src/... -o stage2_out/fusec2`

### Phase 02: First Self-Compilation [W23-P02-SELF-COMPILE]

- Task 01: stage1 compiles stage2 [W23-P02-T01-STAGE1-COMPILES-STAGE2]
  Verify: `fuse build stage2/src/... -o stage2_out/fusec2`
- Task 02: stage2 compiles itself [W23-P02-T02-STAGE2-COMPILES-ITSELF]
  Verify: `stage2_out/fusec2 build stage2/src/... -o stage2_out/fusec2_gen2`

### Phase 03: Reproducibility [W23-P03-REPRO]

- Task 01: Bootstrap reproducibility [W23-P03-T01-REPRO]
  Verify: `make repro`
- Task 02: Gate merges on bootstrap health [W23-P03-T02-GATE]
  Verify: `go run tools/checkci/main.go -bootstrap-gate`

### Wave Closure Phase [W23-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W23-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W23`
- Task 02: WC023 entry [W23-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC023" docs/learning-log.md`

## Wave 24: Native Backend Transition

Goal: remove the bootstrap C11 backend from the compiler implementation
path by standing up a native backend on the same semantic contracts.

Entry criterion: W23 done. Phase 00 re-verifies the Active stubs table
is empty.

Exit criteria:

- native backend exists and passes correctness gates
- backend contracts §57.1–§57.10 hold for native as well
- stage2 builds through native path
- every committed e2e proof program passes with `--backend=native`

Proof of completion:

```
fuse build --backend=native stage2/src/... -o stage2_out/fusec2_native
stage2_out/fusec2_native --version
go test ./tests/e2e/... -run TestNativeBackendAllProofs -v
```

### Phase 00: Stub Audit [W24-P00-STUB-AUDIT]

- Task 01: Native backend audit [W24-P00-T01-ZERO]
  Verify: `go run tools/checkstubs/main.go -wave W24 -phase P00 -require-empty-active`

### Phase 01: Native Backend Foundation [W24-P01-FOUNDATION]

- Task 01: Backend interface [W24-P01-T01-INTERFACE]
  Verify: `go build ./compiler/codegen/native/...`
- Task 02: Reuse backend contracts [W24-P01-T02-CONTRACTS]
  Verify: `go test ./compiler/codegen/native/... -run TestBackendContracts -v`

### Phase 02: Native Backend Proof [W24-P02-PROOF]

- Task 01: All e2e proofs through native [W24-P02-T01-ALL-PROOFS]
  Verify: `go test ./tests/e2e/... -run TestNativeBackendAllProofs -v`
- Task 02: stage2 through native [W24-P02-T02-STAGE2-NATIVE]
  Verify: `fuse build --backend=native stage2/src/... && stage2_out/fusec2_native --version`

### Wave Closure Phase [W24-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W24-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W24`
- Task 02: WC024 entry [W24-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC024" docs/learning-log.md`

## Wave 25: Retirement of Go and C

Goal: complete the transition to a Fuse-owned compiler implementation path.

Entry criterion: W24 done.

Exit criteria:

- Fuse owns the compiler implementation path
- Go not required to build the compiler in the active path
- C not required as a backend or runtime dependency in the active path

Proof of completion:

```
go run tools/checkgoc/main.go -retirement
fuse build stage2/src/... -o stage2_out/fusec2_final
stage2_out/fusec2_final --version
```

### Phase 00: Stub Audit [W25-P00-STUB-AUDIT]

- Task 01: Bootstrap dependency audit [W25-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W25 -phase P00`

### Phase 01: Retire Go [W25-P01-RETIRE-GO]

- Task 01: Freeze Stage 1 [W25-P01-T01-FREEZE]
  Verify: `go run tools/checkgoc/main.go -freeze-stage1`
- Task 02: Remove Go from active workflow [W25-P01-T02-REMOVE-GO]
  Verify: `go run tools/checkgoc/main.go -active-path-no-go`

### Phase 02: Retire C [W25-P02-RETIRE-C]

- Task 01: Replace C runtime dependencies [W25-P02-T01-REPLACE-C]
  Verify: `go run tools/checkgoc/main.go -active-path-no-c`
- Task 02: Remove C from bootstrap assumptions [W25-P02-T02-C-FREE]
  Verify: `fuse build --backend=native tests/e2e/hello_exit.fuse -o /tmp/he && /tmp/he`

### Wave Closure Phase [W25-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W25-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W25`
- Task 02: WC025 entry [W25-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC025" docs/learning-log.md`

## Wave 26: Targets and Ecosystem

Goal: resume broader target and library work on top of the native
self-hosted compiler.

Entry criterion: W25 done.

Exit criteria:

- target expansion proceeds without reintroducing bootstrap debt
- `stdlib/ext/` hosts optional libraries

Proof of completion:

```
fuse build --target=<new-target> tests/e2e/hello_exit.fuse -o /tmp/he_<target>
fuse build stdlib/ext/...
```

### Phase 00: Stub Audit [W26-P00-STUB-AUDIT]

- Task 01: Target audit [W26-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W26 -phase P00`

### Phase 01: Additional Targets [W26-P01-TARGETS]

- Task 01: Target descriptions [W26-P01-T01-DESCRIPTIONS]
  Verify: `go run tools/checktargets/main.go`
- Task 02: Target CI [W26-P01-T02-TARGET-CI]
  Verify: `go run tools/checkci/main.go -target-matrix`

### Phase 02: Extended Libraries [W26-P02-EXT]

- Task 01: Implement ext stdlib [W26-P02-T01-EXT-STDLIB]
  Verify: `fuse build stdlib/ext/...`
- Task 02: Ecosystem guidance [W26-P02-T02-GUIDANCE]
  Verify: `test -f docs/ecosystem-guide.md`

### Wave Closure Phase [W26-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W26-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W26`
- Task 02: WC026 entry [W26-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC026" docs/learning-log.md`

## Cross-cutting constraints

The following rules apply to every wave.

- Determinism is a release-level requirement (Rule 7.1).
- No unresolved types may reach codegen (Rule 3.9).
- No pass may recompute liveness independently (Rule 3.8).
- Invariant walkers remain enabled in debug and CI (Rule 3.5).
- Stdlib failures are compiler signals, not library excuses (Rule 5.1).
- Workarounds are forbidden (Rule 4.2).
- Each non-trivial bug produces both a regression and a learning-log entry
  (Rules 4.3, 4.4).
- Every wave that introduces user-visible behavior includes at least one
  end-to-end proof program that fails if the feature is stubbed (Rule 6.8).
- Exit criteria include behavioral requirements, not only structural ones
  (Rule 6.10).
- Every task has `Currently:`, DoD, and `Verify:`. Verify commands must be
  portable across Linux, macOS, and Windows.
- Stubs emit diagnostics, not silent defaults (Rule 6.9).
- Every wave begins with a Phase 00 Stub Audit (Rule 6.12) and ends with a
  Wave Closure Phase (Rule 6.14).
- STUBS.md is updated at every wave boundary (Rule 6.13) with Active stubs
  table and Stub history append (Rule 6.16).
- Overdue stubs block wave entry (Rule 6.15).
- Every feature in the language reference is scheduled to a concrete wave
  (Rule 2.5). No `Wave TBD`, no "post-v1".
- `tests/e2e/README.md` is updated whenever a proof program is added.
- W22 Stub Clearance Gate and W23 Stage 2 and W24 Native Backend require
  the Active stubs table to be empty at entry.
