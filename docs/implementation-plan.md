# Fuse Implementation Plan

> Status: normative for the next production attempt of Fuse (`fuse4`).
>
> This document is the build plan from an empty repository to a self-hosting
> Fuse compiler and the later retirement of bootstrap-only implementation
> languages.

## Overview

Fuse is implemented in stages.

- Stage 1 compiler: Go
- Runtime during bootstrap: C
- Stage 2 compiler: Fuse

The bootstrap stack is fixed. Go and C are allowed during bootstrap because the
project must reach a self-hosted Fuse compiler as quickly and safely as possible.
After Stage 2 compiles itself reliably and a native backend is stable, Go and C
are retired from the compiler implementation path.

The C11 backend is therefore bootstrap infrastructure, not the terminal backend.
Design decisions in HIR, MIR, type identity, ownership analysis, and pass
structure must not depend on C11 in a way that would block the later native
backend.

## Working principles

1. Correctness precedes velocity.
2. Structural fixes beat symptom fixes.
3. No workarounds are allowed in compiler, runtime, or stdlib.
4. Stdlib is the compiler's semantic stress test.
5. Every wave has explicit entry and exit criteria.
6. Every task must be small enough to review and verify directly.

## Naming conventions

The plan uses globally unique identifiers.

- Wave headings:
  `Wave 04: Type Checking and Semantic Validation`
- Phase headings:
  `Phase 03: Trait Resolution and Bound Dispatch [W04-P03-TRAIT-RESOLUTION]`
- Task headings:
  `Task 01: Register All Function Types Before Body Checking [W04-P03-T01-FN-TYPE-REGISTRATION]`

All wave, phase, and task numbers are zero-padded.

## Task format

Every task in this plan must be written with:

- a short goal
- a `Currently:` line naming what exists at the start of the task (file:line
  if the code exists, or "not yet started" if the package is empty)
- an exact definition of done
- a `Verify:` line giving the specific command that proves the DoD is met; this
  command must fail if the task has not been completed and must be run before
  the task is marked done
- required regression coverage
- clear scope boundaries

In this document, task bullets use the compact form:

```
Task 01: ... [Wxx-Pyy-Tzz-...]
Currently: ...
DoD: verifiable completion rule.
Verify: go test ./compiler/pkg/... -run TestName -v
```

A Verify command is not satisfied by "it looks correct" or "unit tests pass".
It is satisfied by running the named command and observing the declared passing
output. The agent or contributor executing the task must record the actual output
of the Verify command before marking the task done.

## Wave format

Every wave contains:

- **Goal**: one paragraph summary
- **Entry criterion**: what must be true before this wave begins
- **State on entry**: what the codebase actually looks like when the wave begins
  (which packages are empty, which are stubs, which are partial)
- **Exit criteria**: behavioral and structural requirements that must all be true
- **Proof of completion**: the specific commands that, when all passing, prove
  the wave is done; these are run in CI and locally before sign-off
- **Phase 00: Stub Audit**: first phase of every wave; committed to STUBS.md
  before other phases begin
- One or more implementation phases (P01, P02, ...)
- **Wave Closure Phase (PCL)**: last phase of every wave; produces the WCxxx
  learning-log entry and confirms STUBS.md is current

## Waves at a glance

| Wave | Theme | Entry criterion | Exit criterion |
|---|---|---|---|
| 00 | Project foundations | — | build, test, CI, and docs scaffold exist |
| 01 | Lexer | Wave 00 done | every token kind and lexical ambiguity covered |
| 02 | Parser and AST | Wave 01 done | all language constructs parse deterministically |
| 03 | Module graph and resolution | Wave 02 done | module graph, imports, and symbols are resolved |
| 04 | TypeTable, HIR, pass manifest | Wave 03 done | typed HIR shape and pass graph are enforced |
| 05 | Type checker | Wave 04 done | stdlib and user bodies type-check with no unknowns |
| 06 | Ownership and liveness | Wave 05 done | single liveness computation and destruction rules hold |
| 07 | HIR to MIR lowering | Wave 06 done | MIR is structurally correct and property-tested |
| 08 | Runtime library | Wave 00 done | bootstrap runtime surface is implemented |
| 09 | C11 backend and contracts | Waves 07 and 08 done | generated C is structurally correct and deterministic |
| 10 | Native build driver and linking | Wave 09 done | end-to-end `fuse build` links working binaries |
| 11 | CLI, diagnostics, workflows | Wave 10 done | user-facing compiler workflow is coherent |
| 12 | Core stdlib | Wave 11 done | core tier ships and stress-tests frontend and backend |
| 13 | Hosted stdlib | Wave 12 done | OS-facing surface and concurrency tier ship |
| 14 | Stage 2 port | Wave 13 done | stage1 compiles stage2 successfully |
| 15 | Self-hosting gate | Wave 14 done | stage2 compiles itself reproducibly |
| 16 | Native backend transition | Wave 15 done | bootstrap C11 dependency is removed |
| 17 | Generics end-to-end | Wave 16 done | generic functions and types compile, run, and produce correct output |
| 18 | Retirement of Go and C | Wave 17 done | Fuse owns the compiler implementation path |
| 19 | Targets and ecosystem | Wave 18 done | cross-target and library growth resume on native base |

## Wave 00: Project Foundations

Goal: establish the repository, module, build, test, tooling, and documentation
foundations required for disciplined compiler work.

Entry criterion: none.

State on entry: empty directory. No files exist. No Go module. No CI. No docs.

Exit criteria:

- `make all` succeeds from a clean checkout
- `go test ./...` succeeds on the initial package set
- CI runs on every push and PR
- the five foundational docs exist and are readable
- `STUBS.md` exists and lists all package stubs created in Phase 02

Proof of completion:

```
make all
go test ./...
# CI must be green on the first push
cat STUBS.md   # must exist and be non-empty
```

### Phase 00: Stub Audit [W00-P00-STUB-AUDIT]

- Task 01: Initialize STUBS.md [W00-P00-T01-INIT-STUBS]
  Currently: file does not exist.
  DoD: STUBS.md exists at the repository root with the required table format
  and a note that all packages are pre-stub (nothing is implemented yet).
  Verify: `test -f STUBS.md && grep "Active stubs" STUBS.md`

### Phase 01: Repository Initialization [W00-P01-REPOSITORY-INITIALIZATION]

- Task 01: Create repository skeleton [W00-P01-T01-REPO-SKELETON]
  Currently: not yet started.
  DoD: top-level source directories and required seed files exist as defined
  in repository-layout.md.
  Verify: `ls cmd compiler runtime stdlib stage2 tests examples tools docs`
  (all directories must exist)
- Task 02: Add foundational docs [W00-P01-T02-FOUNDATIONAL-DOCS]
  Currently: not yet started.
  DoD: language guide, implementation plan, repository layout, rules, and
  learning log exist in `docs/`.
  Verify: `ls docs/language-guide.md docs/implementation-plan.md docs/repository-layout.md docs/rules.md docs/learning-log.md`
- Task 03: Establish archival and generated-file policy
  [W00-P01-T03-ARTIFACT-POLICY]
  Currently: not yet started.
  DoD: generated artifacts are excluded via .gitignore; source-of-truth files
  are explicit and tracked.
  Verify: `git check-ignore build/ dist/ stage2/build/ && git ls-files docs/`

### Phase 02: Go Module and Build Scaffold [W00-P02-GO-MODULE-AND-BUILD-SCAFFOLD]

- Task 01: Initialize Go module [W00-P02-T01-GO-MOD]
  Currently: not yet started.
  DoD: `go.mod` exists and `go build ./...` works.
  Verify: `go build ./...`
- Task 02: Create package stubs [W00-P02-T02-PACKAGE-STUBS]
  Currently: not yet started.
  DoD: all planned Stage 1 packages compile as empty packages. Each stub
  package is listed in STUBS.md.
  Verify: `go build ./compiler/... && grep -c "compiler/" STUBS.md`
- Task 03: Create Stage 1 CLI stub [W00-P02-T03-CLI-STUB]
  Currently: not yet started.
  DoD: `go run ./cmd/fuse` prints a controlled not-yet-implemented message
  and exits with a non-zero code.
  Verify: `go run ./cmd/fuse 2>&1 | grep -i "not yet implemented"`

### Phase 03: Build and CI Baseline [W00-P03-BUILD-AND-CI-BASELINE]

- Task 01: Author Makefile targets [W00-P03-T01-MAKEFILE]
  Currently: not yet started.
  DoD: `all`, `stage1`, `runtime`, `test`, `clean`, `fmt`, and `docs` work.
  Verify: `make all && make test && make clean && make all`
- Task 02: Add CI matrix [W00-P03-T02-CI-MATRIX]
  Currently: not yet started.
  DoD: Linux, macOS, and Windows builds run in CI on every push.
  Verify: observe green CI on a test push; check `.ci/` contains workflow files
- Task 03: Add golden harness [W00-P03-T03-GOLDEN-HARNESS]
  Currently: not yet started.
  DoD: at least one checked-in golden test passes and is byte-stable.
  Verify: `go test ./... -run TestGolden -v`

### Wave Closure Phase [W00-PCL-WAVE-CLOSURE]

- Task 01: Write WC000 learning-log entry [W00-PCL-T01-CLOSURE-LOG]
  DoD: WC000 entry exists in learning-log.md naming: all directories created,
  STUBS.md initialized, CI green, and what Wave 01 must know.
  Verify: `grep "WC000" docs/learning-log.md`

## Wave 01: Lexing and Tokenization

Goal: build a deterministic lexer that covers the full token set and all known
lexical ambiguities.

Entry criterion: Wave 00 done.

State on entry: `compiler/lex/` is an empty stub package that compiles but
contains no logic. `go test ./compiler/lex/...` passes trivially (no tests).

Exit criteria:

- every token kind in the language guide is tested
- BOM is rejected
- nested block comments work
- raw strings obey the full-pattern rule
- `?.` is emitted as one token
- at least one e2e fixture exists in `tests/e2e/` exercising lexical output
- `tests/e2e/README.md` is updated

Proof of completion:

```
go test ./compiler/lex/... -v
go test ./compiler/lex/... -run TestGolden -v
# golden output must be byte-stable across two runs
go test ./compiler/lex/... -run TestRawString -v
go test ./compiler/lex/... -run TestOptionalChain -v
```

### Phase 00: Stub Audit [W01-P00-STUB-AUDIT]

- Task 01: Audit lex package stub [W01-P00-T01-LEX-STUB-AUDIT]
  Currently: `compiler/lex/` exists but is empty.
  DoD: STUBS.md updated to note that the lexer is unimplemented; the CLI stub
  diagnostic for lex-dependent features is confirmed.
  Verify: `grep "lexer" STUBS.md`

### Phase 01: Token Model [W01-P01-TOKEN-MODEL]

- Task 01: Define token kinds [W01-P01-T01-TOKEN-KINDS]
  Currently: no token type defined in compiler/lex/.
  DoD: token enumeration covers punctuation, operators, literals, and keywords
  as listed in language-guide.md §2.3.
  Verify: `go test ./compiler/lex/... -run TestTokenKindCoverage -v`
- Task 02: Define span model [W01-P01-T02-SPAN-MODEL]
  Currently: no span type.
  DoD: each token carries stable source spans (file, byte offset, length).
  Verify: `go test ./compiler/lex/... -run TestSpanStability -v`

### Phase 02: Scanner Core [W01-P02-SCANNER-CORE]

- Task 01: Implement identifier and keyword scanning
  [W01-P02-T01-IDENTIFIERS-AND-KEYWORDS]
  Currently: not yet started.
  DoD: reserved and active keywords tokenize correctly; identifiers are
  distinguished from keywords.
  Verify: `go test ./compiler/lex/... -run TestKeywords -v`
- Task 02: Implement literal scanning [W01-P02-T02-LITERALS]
  Currently: not yet started.
  DoD: integer (all bases), float, string, and raw string forms tokenize
  correctly.
  Verify: `go test ./compiler/lex/... -run TestLiterals -v`
- Task 03: Implement comments and trivia [W01-P02-T03-COMMENTS-AND-TRIVIA]
  Currently: not yet started.
  DoD: line comments, nested block comments, and whitespace are handled;
  nested block comments do not confuse the scanner.
  Verify: `go test ./compiler/lex/... -run TestNestedBlockComment -v`

### Phase 03: Lexical Edge Cases [W01-P03-LEXICAL-EDGE-CASES]

- Task 01: Enforce raw-string full-pattern recognition
  [W01-P03-T01-RAW-STRING-GUARD]
  Currently: not yet started.
  DoD: `r#abc` does not enter raw-string mode and tokenizes as three tokens.
  Verify: `go test ./compiler/lex/... -run TestRawStringGuard -v`
- Task 02: Enforce `?.` longest-match tokenization
  [W01-P03-T02-OPTIONAL-CHAIN-TOKEN]
  Currently: not yet started.
  DoD: parser receives a single `?.` token; `?` and `.` are not emitted
  separately for the `?.` form.
  Verify: `go test ./compiler/lex/... -run TestOptionalChainToken -v`
- Task 03: Add lexer golden and property tests [W01-P03-T03-LEXER-TESTS]
  Currently: not yet started.
  DoD: deterministic lexing corpus exists; golden output reprints stably;
  property tests confirm no input causes panic.
  Verify: `go test ./compiler/lex/... -v` (all pass, no panics on fuzz corpus)

### Wave Closure Phase [W01-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W01-PCL-T01-UPDATE-STUBS]
  DoD: lexer entry removed from STUBS.md; any lexer-adjacent stubs downstream
  are noted.
  Verify: `grep -c "lexer" STUBS.md` (should be 0 if fully retired)
- Task 02: Write WC001 learning-log entry [W01-PCL-T02-CLOSURE-LOG]
  DoD: WC001 exists naming test counts, edge cases found, and what Wave 02
  must know about the token model.
  Verify: `grep "WC001" docs/learning-log.md`

## Wave 02: Parser and AST Construction

Goal: build an AST-only parser that accepts the full surface grammar without
semantic shortcuts.

Entry criterion: Wave 01 done.

State on entry: `compiler/parse/` and `compiler/ast/` are empty stubs. The
lexer is complete. No AST node types exist.

Exit criteria:

- parser handles every grammar construct in the guide
- parser does not panic on malformed input
- AST remains syntax-only
- at least one e2e fixture in `tests/e2e/` exercises full parse round-trips

Proof of completion:

```
go test ./compiler/parse/... -v
go test ./compiler/ast/... -v
go test ./compiler/parse/... -run TestGolden -v
# second golden run must be byte-identical to first
go test ./compiler/parse/... -run TestNopanicOnMalformed -v
```

### Phase 00: Stub Audit [W02-P00-STUB-AUDIT]

- Task 01: Audit parse and ast stubs [W02-P00-T01-PARSE-STUB-AUDIT]
  Currently: both packages are empty.
  DoD: STUBS.md updated noting that parse and ast are unimplemented; all
  features that depend on parsing emit diagnostics if reached.
  Verify: `grep "parse\|ast" STUBS.md`

### Phase 01: AST Surface [W02-P01-AST-SURFACE]

- Task 01: Define AST node set [W02-P01-T01-AST-NODE-SET]
  Currently: no AST types defined.
  DoD: AST node kinds are exhaustive and syntax-only; no semantic fields present.
  Verify: `go build ./compiler/ast/... && go vet ./compiler/ast/...`
- Task 02: Define AST builders or constructors [W02-P01-T02-AST-CONSTRUCTION]
  Currently: not yet started.
  DoD: AST creation is consistent; every node carries correct span information.
  Verify: `go test ./compiler/ast/... -run TestSpanCorrectness -v`

### Phase 02: Core Parsing [W02-P02-CORE-PARSING]

- Task 01: Parse items and declarations [W02-P02-T01-ITEMS-AND-DECLS]
  Currently: not yet started.
  DoD: functions, structs, enums, traits, impls, consts, and imports parse
  to correct AST nodes.
  Verify: `go test ./compiler/parse/... -run TestItemParsing -v`
- Task 02: Parse expressions and statements [W02-P02-T02-EXPRS-AND-STMTS]
  Currently: not yet started.
  DoD: precedence and associativity are correct; expression golden tests pass.
  Verify: `go test ./compiler/parse/... -run TestExprPrecedence -v`
- Task 03: Parse type expressions [W02-P02-T03-TYPE-EXPRS]
  Currently: not yet started.
  DoD: tuples, arrays, slices, pointers, and generics parse correctly.
  Verify: `go test ./compiler/parse/... -run TestTypeExprs -v`

### Phase 03: Ambiguity Control [W02-P03-AMBIGUITY-CONTROL]

- Task 01: Implement struct-literal disambiguation
  [W02-P03-T01-STRUCT-LITERAL-DISAMBIGUATION]
  Currently: not yet started.
  DoD: `IDENT {` is parsed as struct literal only when syntactically valid;
  golden tests cover the ambiguous cases.
  Verify: `go test ./compiler/parse/... -run TestStructLiteralDisambig -v`
- Task 02: Handle optional chaining parse forms
  [W02-P03-T02-OPTIONAL-CHAIN-PARSE]
  Currently: not yet started.
  DoD: `expr?.field` parses correctly as optional chaining, not as two tokens.
  Verify: `go test ./compiler/parse/... -run TestOptionalChainParse -v`
- Task 03: Add parser regression corpus [W02-P03-T03-PARSER-REGRESSIONS]
  Currently: not yet started.
  DoD: ambiguity regressions are covered by golden tests; no malformed input
  causes a panic.
  Verify: `go test ./compiler/parse/... -v && go test ./compiler/parse/... -run TestGolden -count=2 | diff - <(go test ./compiler/parse/... -run TestGolden -count=2)`

### Wave Closure Phase [W02-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W02-PCL-T01-UPDATE-STUBS]
  DoD: parse and ast entries retired from STUBS.md.
  Verify: `grep -c "parse\|ast" STUBS.md` (should be 0 or only semantic stubs)
- Task 02: Write WC002 learning-log entry [W02-PCL-T02-CLOSURE-LOG]
  DoD: WC002 exists naming ambiguity cases found, golden test count, and what
  Wave 03 must know about the AST shape.
  Verify: `grep "WC002" docs/learning-log.md`

## Wave 03: Name Resolution and Module Graph

Goal: resolve symbols, imports, and the module graph without semantic leakage
into later IR layers.

Entry criterion: Wave 02 done.

State on entry: `compiler/resolve/` is an empty stub. The lexer and parser are
complete. No symbol table or scope mechanism exists.

Exit criteria:

- package discovery is deterministic
- import cycles are diagnosed
- qualified enum variant access resolves
- import resolution supports module-first fallback to module-plus-item

Proof of completion:

```
go test ./compiler/resolve/... -v
go test ./compiler/resolve/... -run TestImportCycleDetection -v
go test ./compiler/resolve/... -run TestQualifiedEnumVariant -v
go test ./compiler/resolve/... -run TestModuleFirstFallback -v
```

### Phase 00: Stub Audit [W03-P00-STUB-AUDIT]

- Task 01: Audit resolve stub [W03-P00-T01-RESOLVE-STUB-AUDIT]
  Currently: compiler/resolve/ is an empty package.
  DoD: STUBS.md updated noting that resolution is unimplemented.
  Verify: `grep "resolve" STUBS.md`

### Phase 01: Package Discovery [W03-P01-PACKAGE-DISCOVERY]

- Task 01: Discover module tree [W03-P01-T01-DISCOVER-MODULES]
  Currently: not yet started.
  DoD: source files map deterministically to module paths; the same source
  tree always produces the same module graph in the same order.
  Verify: `go test ./compiler/resolve/... -run TestModuleDiscovery -v`
- Task 02: Build initial module graph [W03-P01-T02-MODULE-GRAPH]
  Currently: not yet started.
  DoD: modules and imports are collected without semantic resolution.
  Verify: `go test ./compiler/resolve/... -run TestModuleGraph -v`

### Phase 02: Symbol Infrastructure [W03-P02-SYMBOL-INFRASTRUCTURE]

- Task 01: Create symbol table and scopes [W03-P02-T01-SYMBOL-TABLE]
  Currently: not yet started.
  DoD: nested scope and module-scope lookup work.
  Verify: `go test ./compiler/resolve/... -run TestScopeLookup -v`
- Task 02: Index top-level symbols [W03-P02-T02-TOP-LEVEL-INDEX]
  Currently: not yet started.
  DoD: items, variants, and exported names are present in the index.
  Verify: `go test ./compiler/resolve/... -run TestTopLevelIndex -v`

### Phase 03: Import and Path Resolution [W03-P03-IMPORT-AND-PATH-RESOLUTION]

- Task 01: Resolve imports with module-first fallback
  [W03-P03-T01-MODULE-FIRST-IMPORT-RESOLUTION]
  Currently: not yet started.
  DoD: `import util.math.Pair` resolves as item import when needed; the
  golden test demonstrates both the module and item cases.
  Verify: `go test ./compiler/resolve/... -run TestModuleFirstFallback -v`
- Task 02: Support qualified enum variant access
  [W03-P03-T02-QUALIFIED-ENUM-VARIANTS]
  Currently: not yet started.
  DoD: `EnumName.Variant` resolves in expressions and patterns.
  Verify: `go test ./compiler/resolve/... -run TestQualifiedEnumVariant -v`
- Task 03: Detect import cycles [W03-P03-T03-IMPORT-CYCLE-DETECTION]
  Currently: not yet started.
  DoD: cyclic imports produce a diagnostic naming the cycle path; the compiler
  does not hang or panic.
  Verify: `go test ./compiler/resolve/... -run TestImportCycleDetection -v`

### Wave Closure Phase [W03-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W03-PCL-T01-UPDATE-STUBS]
  DoD: resolve entries retired from STUBS.md.
  Verify: `grep -c "resolve" STUBS.md`
- Task 02: Write WC003 learning-log entry [W03-PCL-T02-CLOSURE-LOG]
  DoD: WC003 exists naming any resolution edge cases found.
  Verify: `grep "WC003" docs/learning-log.md`

## Wave 04: TypeTable, HIR, and Pass Manifest

Goal: establish the typed semantic IR surface and the pass graph that later
stages rely on.

Entry criterion: Wave 03 done.

State on entry: `compiler/hir/`, `compiler/typetable/` are empty stubs.
Name resolution is complete.

Exit criteria:

- HIR exists and is distinct from AST and MIR
- HIR builders enforce metadata shape
- pass manifest validates declared metadata dependencies
- invariant walkers are in place

Proof of completion:

```
go test ./compiler/hir/... -v
go test ./compiler/typetable/... -v
go test ./compiler/hir/... -run TestInvariantWalkers -v
go test ./compiler/hir/... -run TestBuilderEnforcement -v
```

### Phase 00: Stub Audit [W04-P00-STUB-AUDIT]

- Task 01: Audit HIR and TypeTable stubs [W04-P00-T01-HIR-STUB-AUDIT]
  Currently: both packages are empty.
  DoD: STUBS.md updated noting HIR and TypeTable are unimplemented.
  Verify: `grep "hir\|typetable" STUBS.md`

### Phase 01: Global TypeTable [W04-P01-GLOBAL-TYPETABLE]

- Task 01: Define TypeId and TypeTable [W04-P01-T01-TYPEID-AND-TABLE]
  Currently: no TypeId or TypeTable defined.
  DoD: all later IR layers refer to types by interned IDs; integer comparison
  is sufficient for type equality.
  Verify: `go test ./compiler/typetable/... -run TestTypeInternEquality -v`
- Task 02: Encode nominal identity [W04-P01-T02-NOMINAL-IDENTITY-ENCODING]
  Currently: not yet started.
  DoD: defining symbol and concrete args are part of type identity; two types
  with the same name from different modules are distinct TypeIds.
  Verify: `go test ./compiler/typetable/... -run TestNominalIdentity -v`

### Phase 02: HIR Builders and Metadata [W04-P02-HIR-BUILDERS-AND-METADATA]

- Task 01: Define HIR node set [W04-P02-T01-HIR-NODE-SET]
  Currently: not yet started.
  DoD: HIR node kinds are exhaustive and semantically oriented; none contain
  raw syntax strings in semantic positions.
  Verify: `go build ./compiler/hir/... && go vet ./compiler/hir/...`
- Task 02: Add per-node metadata [W04-P02-T02-HIR-METADATA]
  Currently: not yet started.
  DoD: type, ownership, liveness hooks, divergence, and context fields exist
  on HIR nodes.
  Verify: `go test ./compiler/hir/... -run TestMetadataFields -v`
- Task 03: Enforce builder-only construction [W04-P02-T03-BUILDER-ENFORCEMENT]
  Currently: not yet started.
  DoD: HIR cannot be built ad hoc without metadata defaults; builders reject
  construction with missing required fields.
  Verify: `go test ./compiler/hir/... -run TestBuilderEnforcement -v`

### Phase 03: Pass Graph and Invariants [W04-P03-PASS-GRAPH-AND-INVARIANTS]

- Task 01: Define pass manifest [W04-P03-T01-PASS-MANIFEST]
  Currently: not yet started.
  DoD: passes declare reads and writes explicitly; the manifest rejects passes
  that consume metadata they haven't declared.
  Verify: `go test ./compiler/hir/... -run TestPassManifest -v`
- Task 02: Implement invariant walkers [W04-P03-T02-INVARIANT-WALKERS]
  Currently: not yet started.
  DoD: post-pass structural checks run in debug and CI modes; a failing
  invariant produces a diagnostic naming the violating pass.
  Verify: `go test ./compiler/hir/... -run TestInvariantWalkers -v`
- Task 03: Prohibit nondeterministic IR collections
  [W04-P03-T03-DETERMINISTIC-IR]
  Currently: not yet started.
  DoD: HIR and MIR do not rely on builtin map iteration order; golden tests
  produce byte-identical output across runs.
  Verify: `go test ./compiler/hir/... -run TestDeterministicOrder -count=3`
  (all three runs must produce identical output)

### Wave Closure Phase [W04-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W04-PCL-T01-UPDATE-STUBS]
  DoD: HIR and TypeTable entries retired from STUBS.md.
  Verify: `grep -c "hir\|typetable" STUBS.md`
- Task 02: Write WC004 learning-log entry [W04-PCL-T02-CLOSURE-LOG]
  DoD: WC004 exists.
  Verify: `grep "WC004" docs/learning-log.md`

## Wave 05: Type Checking and Semantic Validation

Goal: build a checker that fully types user code and stdlib bodies without
leaving unknown metadata for later passes.

Entry criterion: Wave 04 done.

State on entry: `compiler/check/` is an empty stub. HIR, TypeTable, and
resolution are complete. No type-checking logic exists.

Exit criteria:

- no checked HIR node retains `Unknown` type metadata
- all function declaration nodes are typed before body checking
- stdlib bodies are checked in the same pass as user modules
- trait-bound lookup works through supertraits
- contextual generic inference works in constructor-style calls
- behavioral proof: a program that calls a method on a concrete type and uses
  the result compiles, runs, and returns the correct value

Proof of completion:

```
go test ./compiler/check/... -v
go test ./compiler/check/... -run TestNoUnknownAfterCheck -v
go test ./compiler/check/... -run TestStdlibBodyChecking -v
go test ./compiler/check/... -run TestTraitBoundLookup -v
# run the hello_exit.fuse proof program end-to-end once Wave 09 is available
```

### Phase 00: Stub Audit [W05-P00-STUB-AUDIT]

- Task 01: Audit check stub [W05-P00-T01-CHECK-STUB-AUDIT]
  Currently: compiler/check/ is an empty package.
  DoD: STUBS.md updated noting that type checking is unimplemented.
  Verify: `grep "check\|type.check" STUBS.md`

### Phase 01: Function Type Registration [W05-P01-FN-TYPE-REGISTRATION]

- Task 01: Index all function signatures [W05-P01-T01-INDEX-FN-SIGNATURES]
  Currently: not yet started.
  DoD: top-level functions, impl methods, and externs all receive function
  types in Pass 1 before any body is checked.
  Verify: `go test ./compiler/check/... -run TestFunctionTypeRegistration -v`
- Task 02: Separate signature registration from body checking
  [W05-P01-T02-TWO-PASS-CHECKER]
  Currently: not yet started.
  DoD: checker runs a signature pass before body analysis; no impl method
  retains Unknown function type after Pass 1.
  Verify: `go test ./compiler/check/... -run TestTwoPassChecker -v`

### Phase 02: Nominal Identity and Equality [W05-P02-NOMINAL-IDENTITY-AND-EQUALITY]

- Task 01: Implement nominal type equality [W05-P02-T01-NOMINAL-EQUALITY]
  Currently: not yet started.
  DoD: same-name types from different modules are distinct TypeIds.
  Verify: `go test ./compiler/check/... -run TestNominalEquality -v`
- Task 02: Register primitive method surface [W05-P02-T02-PRIMITIVE-METHODS]
  Currently: not yet started.
  DoD: primitive method calls resolve during body checking without user-declared
  impls.
  Verify: `go test ./compiler/check/... -run TestPrimitiveMethods -v`
- Task 03: Implement numeric widening rules [W05-P02-T03-NUMERIC-WIDENING]
  Currently: not yet started.
  DoD: legal mixed-width arithmetic and comparisons type-check; illegal ones
  produce diagnostics.
  Verify: `go test ./compiler/check/... -run TestNumericWidening -v`

### Phase 03: Trait Resolution and Bound Dispatch [W05-P03-TRAIT-RESOLUTION]

- Task 01: Implement trait method lookup on concrete types
  [W05-P03-T01-CONCRETE-TRAIT-METHOD-LOOKUP]
  Currently: not yet started.
  DoD: trait-implemented methods resolve on concrete receivers.
  Verify: `go test ./compiler/check/... -run TestConcreteTraitMethodLookup -v`
- Task 02: Implement bound-chain lookup on type parameters
  [W05-P03-T02-BOUND-CHAIN-LOOKUP]
  Currently: not yet started.
  DoD: bounds and supertraits are searched recursively; a bound of `Hashable`
  resolves `Equatable` methods when `Hashable: Equatable`.
  Verify: `go test ./compiler/check/... -run TestBoundChainLookup -v`
- Task 03: Support trait-typed parameters as interfaces
  [W05-P03-T03-TRAIT-PARAMETERS-AS-INTERFACES]
  Currently: not yet started.
  DoD: concrete implementers are accepted at trait-typed call sites.
  Verify: `go test ./compiler/check/... -run TestTraitParameters -v`

### Phase 04: Contextual Inference and Literals [W05-P04-CONTEXTUAL-INFERENCE-AND-LITERALS]

- Task 01: Infer generics from expected type [W05-P04-T01-EXPECTED-TYPE-INFERENCE]
  Currently: not yet started.
  DoD: constructor-style calls infer type args from context; `List.new()` in a
  `List[Expr]` context infers `Expr`.
  Verify: `go test ./compiler/check/... -run TestContextualInference -v`
- Task 02: Handle explicit type args on zero-arg generic calls
  [W05-P04-T02-ZERO-ARG-TYPE-ARGS]
  Currently: not yet started.
  DoD: no-value-argument generic helpers still specialize correctly.
  Verify: `go test ./compiler/check/... -run TestZeroArgTypeArgs -v`
- Task 03: Normalize literal typing [W05-P04-T03-LITERAL-TYPING]
  Currently: not yet started.
  DoD: integer and float literals pick contextually valid types when required.
  Verify: `go test ./compiler/check/... -run TestLiteralTyping -v`

### Phase 05: Stdlib Body Checking [W05-P05-STDLIB-BODY-CHECKING]

- Task 01: Remove stdlib body skips [W05-P05-T01-REMOVE-STDLIB-SKIPS]
  Currently: not yet started (check stub has no skip logic yet, but the
  architecture must not introduce one).
  DoD: stdlib and user modules are checked uniformly in the same pass.
  Verify: `go test ./compiler/check/... -run TestStdlibBodyChecking -v`
- Task 02: Fix exposed semantic gaps [W05-P05-T02-STDLIB-STRESS-FIXES]
  Currently: not yet started.
  DoD: stdlib methods, traits, and patterns type-check without workarounds.
  Verify: `go test ./compiler/check/... -run TestStdlibStress -v`
- Task 03: Add checker regression corpus [W05-P05-T03-CHECKER-REGRESSIONS]
  Currently: not yet started.
  DoD: each fixed checker class has a dedicated regression test.
  Verify: `go test ./compiler/check/... -v` (all regressions pass)

### Phase 06: Monomorphization [W05-P06-MONOMORPHIZATION]

Note: this phase establishes the monomorph package interface but does NOT
implement full pipeline integration. Full generics require Wave 17.

- Task 01: Collect concrete instantiations at call sites
  [W05-P06-T01-COLLECT-INSTANTIATIONS]
  Currently: compiler/monomorph/ exists as a placeholder.
  DoD: all generic function and type usages produce concrete type argument sets
  in the monomorph context during checking.
  Verify: `go test ./compiler/monomorph/... -run TestCollect -v`
- Task 02: Validate specialization completeness
  [W05-P06-T02-VALIDATE-SPECIALIZATIONS]
  Currently: not yet started.
  DoD: partially-resolved type arguments are rejected before lowering.
  Verify: `go test ./compiler/monomorph/... -run TestPartialSpecializationRejected -v`
- Task 03: Specialize generic functions into concrete HIR/MIR
  [W05-P06-T03-SPECIALIZE-FUNCTIONS]
  Currently: not yet started.
  DoD: each concrete instantiation produces a distinct lowered function
  (stub integration into driver deferred to Wave 17).
  Verify: `go test ./compiler/monomorph/... -run TestSpecialize -v`
- Task 04: Integrate monomorph into the driver pipeline
  [W05-P06-T04-INTEGRATE-PIPELINE]
  Currently: not yet started.
  DoD: `Build()` runs monomorphization between checking and lowering.
  Note: full e2e correctness of generics is Wave 17's responsibility. This task
  establishes the call site only.
  Verify: `go test ./compiler/driver/... -run TestMonomorphInPipeline -v`

### Wave Closure Phase [W05-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W05-PCL-T01-UPDATE-STUBS]
  DoD: check entries retired; any features checked but not yet lowered are
  added to STUBS.md with appropriate diagnostics.
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC005 learning-log entry [W05-PCL-T02-CLOSURE-LOG]
  DoD: WC005 exists.
  Verify: `grep "WC005" docs/learning-log.md`

## Wave 06: Ownership, Liveness, and Destruction

Goal: compute ownership and liveness once and expose it as stable metadata for
all later passes.

Entry criterion: Wave 05 done.

State on entry: `compiler/liveness/` is an empty stub. Type checking is
complete.

Exit criteria:

- ownership metadata is complete on HIR
- liveness is computed exactly once per function
- destruction behavior is inserted based on last use and ownership

Proof of completion:

```
go test ./compiler/liveness/... -v
go test ./compiler/liveness/... -run TestSingleLiveness -v
go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v
```

### Phase 00: Stub Audit [W06-P00-STUB-AUDIT]

- Task 01: Audit liveness stub [W06-P00-T01-LIVENESS-STUB-AUDIT]
  Currently: compiler/liveness/ is empty.
  DoD: STUBS.md updated.
  Verify: `grep "liveness" STUBS.md`

### Phase 01: Ownership Semantics [W06-P01-OWNERSHIP-SEMANTICS]

- Task 01: Model ownership contexts [W06-P01-T01-OWNERSHIP-CONTEXTS]
  Currently: not yet started.
  DoD: value, ref, mutref, owned, and move contexts are tracked explicitly.
  Verify: `go test ./compiler/liveness/... -run TestOwnershipContexts -v`
- Task 02: Enforce implicit and explicit borrow rules
  [W06-P01-T02-BORROW-RULES]
  Currently: not yet started.
  DoD: mutable-receiver implicit borrow and invalid escapes are both enforced.
  Verify: `go test ./compiler/liveness/... -run TestBorrowRules -v`

### Phase 02: Single Liveness Computation [W06-P02-SINGLE-LIVENESS-COMPUTATION]

- Task 01: Compute per-node live-after data [W06-P02-T01-LIVE-AFTER]
  Currently: not yet started.
  DoD: HIR nodes carry live-after metadata; computation runs once per function.
  Verify: `go test ./compiler/liveness/... -run TestLiveAfter -v`
- Task 02: Expose last-use and destroy-after metadata
  [W06-P02-T02-LAST-USE-AND-DESTROY-AFTER]
  Currently: not yet started.
  DoD: later passes do not need to recompute liveness.
  Verify: `go test ./compiler/liveness/... -run TestLastUse -v`

### Phase 03: Deterministic Destruction [W06-P03-DETERMINISTIC-DESTRUCTION]

- Task 01: Insert drop intent semantically [W06-P03-T01-DROP-INTENT]
  Currently: not yet started.
  DoD: owned locals have deterministic destruction behavior on all control flow
  paths including branches, loops, and early returns.
  Verify: `go test ./compiler/liveness/... -run TestDropIntent -v`
- Task 02: Test loops, breaks, and early returns
  [W06-P03-T02-CONTROL-FLOW-DESTRUCTION]
  Currently: not yet started.
  DoD: destruction remains correct across complex control flow.
  Verify: `go test ./compiler/liveness/... -run TestDestructionOnAllPaths -v`

### Wave Closure Phase [W06-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W06-PCL-T01-UPDATE-STUBS]
  DoD: liveness entries retired; drop codegen stub (actual C emission deferred
  to Wave 09) added to STUBS.md with diagnostic.
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC006 learning-log entry [W06-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC006" docs/learning-log.md`

## Wave 07: HIR to MIR Lowering

Goal: lower semantically complete HIR into explicit MIR without losing type,
ownership, or control-flow invariants.

Entry criterion: Wave 06 done.

State on entry: `compiler/lower/` and `compiler/mir/` are empty stubs.
Liveness and type checking are complete.

Exit criteria:

- MIR blocks terminate structurally
- no move-after-move invariant violations exist
- method calls and field accesses are disambiguated correctly
- diverging control flow does not create phantom locals
- match lowering dispatches on discriminants (behavioral, not structural)
- `?` operator lowers to a branch (behavioral, not structural)
- behavioral proof: a match expression with three arms returns the correct arm's
  value when executed

Proof of completion:

```
go test ./compiler/lower/... -v
go test ./compiler/mir/... -v
go test ./compiler/lower/... -run TestMatchDispatch -v
go test ./compiler/lower/... -run TestQuestionBranch -v
# proof program match_enum_dispatch.fuse must pass in tests/e2e/ once Wave 09 is available
```

### Phase 00: Stub Audit [W07-P00-STUB-AUDIT]

- Task 01: Audit lower and mir stubs [W07-P00-T01-LOWER-STUB-AUDIT]
  Currently: both packages are empty.
  DoD: STUBS.md updated noting lower and mir are unimplemented.
  Verify: `grep "lower\|mir" STUBS.md`

### Phase 01: MIR Core [W07-P01-MIR-CORE]

- Task 01: Define MIR instruction set [W07-P01-T01-MIR-INSTRS]
  Currently: not yet started.
  DoD: borrow, move, drop, call, field, and constant operations are explicit
  MIR instruction types.
  Verify: `go build ./compiler/mir/... && go vet ./compiler/mir/...`
- Task 02: Define MIR builders [W07-P01-T02-MIR-BUILDERS]
  Currently: not yet started.
  DoD: block and local construction is centralized through builders.
  Verify: `go test ./compiler/mir/... -run TestMIRBuilders -v`

### Phase 02: Control Flow Lowering [W07-P02-CONTROL-FLOW-LOWERING]

- Task 01: Lower branching and loops [W07-P02-T01-BRANCHES-AND-LOOPS]
  Currently: not yet started.
  DoD: join blocks exist only when control flow truly reaches them.
  Verify: `go test ./compiler/lower/... -run TestBranchLowering -v`
- Task 02: Seal blocks on terminators [W07-P02-T02-SEAL-BLOCKS]
  Currently: not yet started.
  DoD: `return`, `break`, and `continue` do not reopen fallthrough blocks.
  Verify: `go test ./compiler/lower/... -run TestSealedBlocks -v`
- Task 03: Model divergence structurally [W07-P02-T03-DIVERGENCE-STRUCTURE]
  Currently: not yet started.
  DoD: no fake post-divergence temporaries appear in MIR; diverging calls have
  no MIR successors.
  Verify: `go test ./compiler/lower/... -run TestStructuralDivergence -v`

### Phase 03: Calls, Methods, and Fields [W07-P03-CALLS-METHODS-AND-FIELDS]

- Task 01: Lower borrow expressions as borrow instructions
  [W07-P03-T01-BORROW-INSTRS]
  Currently: not yet started.
  DoD: `ref` and `mutref` always lower to `InstrBorrow`; no generic unary
  lowering path handles them.
  Verify: `go test ./compiler/lower/... -run TestBorrowInstr -v`
- Task 02: Disambiguate method calls from field reads
  [W07-P03-T02-METHOD-VS-FIELD]
  Currently: not yet started.
  DoD: method calls do not lower to field-address instructions; the lowerer
  distinguishes call position from non-call position.
  Verify: `go test ./compiler/lower/... -run TestMethodVsField -v`
- Task 03: Lower enum constructors and bare variant values
  [W07-P03-T03-ENUM-CONSTRUCTORS]
  Currently: not yet started.
  DoD: enum variant values and constructor calls lower distinctly and correctly.
  Verify: `go test ./compiler/lower/... -run TestEnumConstructors -v`

### Phase 04: Pattern Match Lowering [W07-P04-PATTERN-MATCH-LOWERING]

- Task 01: Add structured pattern nodes to HIR
  [W07-P04-T01-HIR-PATTERN-NODES]
  Currently: HIR stores patterns as `PatternDesc string` (text only).
  DoD: MatchArm carries structured Pattern (LiteralPat, BindPat,
  ConstructorPat, WildcardPat), not a text description.
  Verify: `go test ./compiler/hir/... -run TestPatternNodes -v`
- Task 02: Lower match to cascading branches
  [W07-P04-T02-MATCH-TO-BRANCHES]
  Currently: lowerer emits `TermGoto(firstArmBlock)` unconditionally.
  DoD: each arm produces a condition check and branch; wildcard arms produce
  unconditional jumps; arms are tested in source order. A match with N arms
  produces N-1 conditional branches in MIR.
  Verify: `go test ./compiler/lower/... -run TestMatchToBranches -v`
- Task 03: Lower enum discriminant access
  [W07-P04-T03-ENUM-DISCRIMINANT-ACCESS]
  Currently: not yet started.
  DoD: match on enum types reads the tag field and branches by tag value; the
  generated MIR contains a field-read instruction for `_tag`.
  Verify: `go test ./compiler/lower/... -run TestEnumDiscriminantAccess -v`

### Phase 05: Error Propagation Lowering [W07-P05-ERROR-PROPAGATION-LOWERING]

- Task 01: Type-check ? operator on Result and Option
  [W07-P05-T01-QUESTION-TYPECHECK]
  Currently: `checkQuestion()` returns `Unknown` type.
  DoD: `?` on `Result[T, E]` returns `T`; `?` on `Option[T]` returns `T`;
  type errors are diagnosed.
  Verify: `go test ./compiler/check/... -run TestQuestionTypecheck -v`
- Task 02: Lower ? to branch-and-early-return
  [W07-P05-T02-QUESTION-LOWERING]
  Currently: `lowerExpr(QuestionExpr)` returns `lowerExpr(n.Expr)` (pass-through).
  DoD: `?` emits a discriminant check on `_tag`, extracts `_f0` on success,
  and returns the Err/None wrapper on failure. The resulting MIR contains a
  conditional branch.
  Verify: `go test ./compiler/lower/... -run TestQuestionBranch -v`

### Phase 06: Closure Lowering [W07-P06-CLOSURE-LOWERING]

- Task 01: Implement capture analysis
  [W07-P06-T01-CAPTURE-ANALYSIS]
  Currently: lowerer returns `constUnit()` for closure expressions.
  DoD: closure bodies are scanned for references to outer variables; captured
  variables are collected before lowering begins.
  Verify: `go test ./compiler/lower/... -run TestCaptureAnalysis -v`
- Task 02: Generate environment struct and lift closure body
  [W07-P06-T02-CLOSURE-LIFTING]
  Currently: no environment struct or lifted function is generated.
  DoD: each closure produces a lifted function taking an env parameter and
  a struct type holding captured variables.
  Verify: `go test ./compiler/lower/... -run TestClosureLifting -v`
- Task 03: Emit closure construction at expression site
  [W07-P06-T03-CLOSURE-CONSTRUCTION]
  Currently: not yet started.
  DoD: closure expressions emit struct init for the environment and pair it
  with the lifted function pointer.
  Verify: `go test ./compiler/lower/... -run TestClosureConstruction -v`

### Wave Closure Phase [W07-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W07-PCL-T01-UPDATE-STUBS]
  DoD: lower and mir entries retired; any features lowered but not yet codegenned
  remain in STUBS.md with diagnostics.
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC007 learning-log entry [W07-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC007" docs/learning-log.md`

## Wave 08: Runtime Library

Goal: implement the bootstrap runtime surface in C with a stable ABI and a small
trusted code footprint.

Entry criterion: Wave 00 done.

State on entry: `runtime/` directory exists with the header structure but no
implementations. The include/ directory may have placeholder headers.

Exit criteria:

- all required runtime entry points exist
- runtime tests pass on supported host platforms
- runtime ABI matches the language guide and backend contracts

Proof of completion:

```
make runtime
# runtime tests (platform-specific):
cd runtime && make test
```

### Phase 00: Stub Audit [W08-P00-STUB-AUDIT]

- Task 01: Audit runtime stubs [W08-P00-T01-RUNTIME-STUB-AUDIT]
  Currently: runtime/src/ is empty or has placeholder files.
  DoD: STUBS.md updated noting which runtime entry points are unimplemented.
  Verify: `grep "runtime" STUBS.md`

### Phase 01: Runtime Surface [W08-P01-RUNTIME-SURFACE]

- Task 01: Define runtime header [W08-P01-T01-RUNTIME-HEADER]
  Currently: not yet started.
  DoD: all bootstrap runtime entry points are declared in runtime/include/.
  Verify: `gcc -fsyntax-only runtime/include/fuse_rt.h`
- Task 02: Implement memory and panic primitives
  [W08-P01-T02-MEMORY-AND-PANIC]
  Currently: not yet started.
  DoD: allocation, deallocation, panic, and abort surface compile and pass
  runtime tests.
  Verify: `cd runtime && make test && ./tests/test_memory && ./tests/test_panic`

### Phase 02: IO, Process, and Time [W08-P02-IO-PROCESS-AND-TIME]

- Task 01: Implement basic IO surface [W08-P02-T01-BASIC-IO]
  Currently: not yet started.
  DoD: stdout, stderr, file, and minimal path operations work.
  Verify: `cd runtime && ./tests/test_io`
- Task 02: Implement process and time surface [W08-P02-T02-PROCESS-AND-TIME]
  Currently: not yet started.
  DoD: arguments, environment, clock, and process control work.
  Verify: `cd runtime && ./tests/test_process`

### Phase 03: Threads and Synchronization [W08-P03-THREADS-AND-SYNC]

- Task 01: Implement thread and TLS surface [W08-P03-T01-THREAD-AND-TLS]
  Currently: not yet started.
  DoD: spawn and thread-local primitives work.
  Verify: `cd runtime && ./tests/test_thread`
- Task 02: Implement synchronization surface [W08-P03-T02-SYNC]
  Currently: not yet started.
  DoD: mutex, condition, and related runtime helpers work.
  Verify: `cd runtime && ./tests/test_sync`

### Wave Closure Phase [W08-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W08-PCL-T01-UPDATE-STUBS]
  DoD: runtime entries retired; spawn/channel compiler-side integration remains
  stubbed with diagnostics until Wave 13.
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC008 learning-log entry [W08-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC008" docs/learning-log.md`

## Wave 09: C11 Backend and Representation Contracts

Goal: emit correct, deterministic C11 from concrete MIR while enforcing the
backend contracts documented in the language guide.

Entry criterion: Waves 07 and 08 done.

State on entry: `compiler/codegen/` is an empty stub. MIR lowering is complete.
Runtime ABI is defined.

Exit criteria:

- composite types are emitted before use
- unresolved types never reach codegen
- pointer categories are handled correctly
- unit erasure is total
- divergence and aggregate fallbacks are emitted correctly
- behavioral proof: a hello-world Fuse program compiles, links, and runs
  producing exit code 0
- `tests/e2e/hello_exit.fuse` passes in CI
- `tests/e2e/README.md` is updated

Proof of completion:

```
go test ./compiler/codegen/... -v
# end-to-end:
fuse build tests/e2e/hello_exit.fuse -o /tmp/hello_exit
/tmp/hello_exit
echo "exit: $?"   # must be 0
go test ./tests/e2e/... -run TestHelloExit -v
```

### Phase 00: Stub Audit [W09-P00-STUB-AUDIT]

- Task 01: Audit codegen stub [W09-P00-T01-CODEGEN-STUB-AUDIT]
  Currently: compiler/codegen/ is empty.
  DoD: STUBS.md updated.
  Verify: `grep "codegen" STUBS.md`

### Phase 01: Type Emission and Naming [W09-P01-TYPE-EMISSION-AND-NAMING]

- Task 01: Emit composite type definitions before function bodies
  [W09-P01-T01-TYPE-DEFS-FIRST]
  Currently: not yet started.
  DoD: generated C has no use-before-definition composite types; gcc compiles
  the output without error.
  Verify: `go test ./compiler/codegen/... -run TestTypeDefsFirst -v`
- Task 02: Sanitize identifiers and avoid collisions
  [W09-P01-T02-IDENTIFIER-SANITIZATION]
  Currently: not yet started.
  DoD: names are legal C identifiers and stable across builds; C keywords are
  escaped.
  Verify: `go test ./compiler/codegen/... -run TestIdentifierSanitization -v`
- Task 03: Encode module-qualified identity in mangling
  [W09-P01-T03-MODULE-QUALIFIED-MANGLING]
  Currently: not yet started.
  DoD: same-name items from different modules do not collide after mangling.
  Verify: `go test ./compiler/codegen/... -run TestModuleMangling -v`

### Phase 02: Pointer Categories and Borrow Semantics [W09-P02-POINTER-CATEGORIES-AND-BORROWS]

- Task 01: Distinguish borrow pointers from `Ptr[T]` values
  [W09-P02-T01-TWO-POINTER-CATEGORIES]
  Currently: not yet started.
  DoD: codegen tracks borrow pointers and `Ptr[T]` values separately; implicit
  deref applies only to borrow pointers.
  Verify: `go test ./compiler/codegen/... -run TestPointerCategories -v`
- Task 02: Adapt call sites to ref and mutref signatures
  [W09-P02-T02-CALL-SITE-ADAPTATION]
  Currently: not yet started.
  DoD: value and borrow arguments are passed correctly at every call site.
  Verify: `go test ./compiler/codegen/... -run TestCallSiteAdaptation -v`

### Phase 03: Unit Erasure and Aggregate Emission [W09-P03-UNIT-ERASURE-AND-AGGREGATE-EMISSION]

- Task 01: Erase unit consistently [W09-P03-T01-TOTAL-UNIT-ERASURE]
  Currently: not yet started.
  DoD: no ghost unit payloads or params remain in generated C at any site.
  Verify: `go test ./compiler/codegen/... -run TestTotalUnitErasure -v`
- Task 02: Emit typed aggregate zero-initializers
  [W09-P03-T02-TYPED-AGGREGATE-FALLBACKS]
  Currently: not yet started.
  DoD: aggregate fallbacks are never scalar `0`; the form is `(FuseType_Foo){0}`.
  Verify: `go test ./compiler/codegen/... -run TestAggregateZeroInit -v`

### Phase 04: Divergence and Equality Lowering [W09-P04-DIVERGENCE-AND-EQUALITY-LOWERING]

- Task 01: Emit divergence structurally [W09-P04-T01-STRUCTURAL-DIVERGENCE]
  Currently: not yet started.
  DoD: no undeclared locals are read after diverging calls.
  Verify: `go test ./compiler/codegen/... -run TestStructuralDivergence -v`
- Task 02: Lower equality and comparison semantically
  [W09-P04-T02-SEMANTIC-EQUALITY]
  Currently: not yet started.
  DoD: non-scalar equality does not compile down to invalid raw C comparisons.
  Verify: `go test ./compiler/codegen/... -run TestSemanticEquality -v`

### Phase 05: Drop Codegen [W09-P05-DROP-CODEGEN]

- Task 01: Flow Drop trait metadata to codegen
  [W09-P05-T01-DROP-TRAIT-METADATA]
  Currently: not yet started.
  DoD: codegen can determine whether a type has a Drop implementation via the
  type table or a side table.
  Verify: `go test ./compiler/codegen/... -run TestDropTraitMetadata -v`
- Task 02: Emit destructor calls for InstrDrop
  [W09-P05-T02-EMIT-DESTRUCTOR-CALLS]
  Currently: codegen emits `/* drop _lN */` comment placeholder.
  DoD: `InstrDrop` on types with Drop impls emits `TypeName_drop(&_lN);`
  in generated C; types without Drop emit nothing.
  Verify: `go test ./compiler/codegen/... -run TestDestructorCallEmitted -v`
  (test must grep generated C for `_drop(` and confirm its presence)
- Task 03: Test drop codegen with owned resources
  [W09-P05-T03-DROP-CODEGEN-TESTS]
  Currently: not yet started.
  DoD: regression tests verify destructor calls appear in generated C for a
  type that implements Drop.
  Verify: `go test ./compiler/codegen/... -run TestDropCodegenOwned -v`

### Phase 06: Backend Regression Closure [W09-P06-BACKEND-REGRESSION-CLOSURE]

- Task 01: Add regression tests from fuse3 bug history
  [W09-P06-T01-BUG-HISTORY-REGRESSIONS]
  Currently: not yet started.
  DoD: codegen bugs documented in learning-log L004–L006 have direct regression
  coverage.
  Verify: `go test ./compiler/codegen/... -run TestL004Regression -run TestL005Regression -run TestL006Regression -v`

### Wave Closure Phase [W09-PCL-WAVE-CLOSURE]

- Task 01: Add hello_exit.fuse to tests/e2e/ [W09-PCL-T01-HELLO-PROOF]
  DoD: `tests/e2e/hello_exit.fuse` exists; `tests/e2e/README.md` updated; CI
  passes on this program.
  Verify: `go test ./tests/e2e/... -run TestHelloExit -v`
- Task 02: Update STUBS.md [W09-PCL-T02-UPDATE-STUBS]
  DoD: codegen entries retired; any remaining stubs documented.
  Verify: `go run tools/checkstubs/main.go`
- Task 03: Write WC009 learning-log entry [W09-PCL-T03-CLOSURE-LOG]
  Verify: `grep "WC009" docs/learning-log.md`

## Wave 10: Native Build Driver and Linking

Goal: turn generated C and the runtime library into working native artifacts.

Entry criterion: Wave 09 done.

State on entry: `compiler/cc/` and `compiler/driver/` are stubs or partial.
C backend is complete.

Exit criteria:

- `fuse build` produces working binaries and libraries
- runtime library discovery is deterministic
- build errors are rendered clearly
- behavioral proof: `fuse build tests/e2e/hello_exit.fuse` produces a binary
  that exits with code 0

Proof of completion:

```
fuse build tests/e2e/hello_exit.fuse -o /tmp/hello_exit && /tmp/hello_exit
echo $?   # must be 0
go test ./compiler/driver/... -v
```

### Phase 00: Stub Audit [W10-P00-STUB-AUDIT]

- Task 01: Audit driver and cc stubs [W10-P00-T01-DRIVER-STUB-AUDIT]
  Currently: both packages are stubs.
  DoD: STUBS.md updated.
  Verify: `grep "driver\|cc" STUBS.md`

### Phase 01: Compiler Invocation [W10-P01-COMPILER-INVOCATION]

- Task 01: Detect host C compiler [W10-P01-T01-COMPILER-DETECTION]
  Currently: not yet started.
  DoD: supported host toolchains (gcc, clang) are discovered reliably on Linux,
  macOS, and Windows.
  Verify: `go test ./compiler/cc/... -run TestCompilerDetection -v`
- Task 02: Emit compile and link arguments [W10-P01-T02-COMPILE-LINK-ARGS]
  Currently: not yet started.
  DoD: target, optimization, debug, and output flags work correctly.
  Verify: `go test ./compiler/cc/... -run TestCompileArgs -v`

### Phase 02: Runtime Discovery and Linking [W10-P02-RUNTIME-DISCOVERY-AND-LINKING]

- Task 01: Discover or build runtime library [W10-P02-T01-RUNTIME-DISCOVERY]
  Currently: not yet started.
  DoD: package builds find the runtime deterministically; same source tree
  always resolves the same runtime path.
  Verify: `go test ./compiler/driver/... -run TestRuntimeDiscovery -v`
- Task 02: Link artifacts [W10-P02-T02-LINK-ARTIFACTS]
  Currently: not yet started.
  DoD: executables link correctly; the resulting binary runs.
  Verify: `fuse build tests/e2e/hello_exit.fuse -o /tmp/he && /tmp/he && echo ok`

### Phase 03: End-to-End Validation [W10-P03-END-TO-END-VALIDATION]

- Task 01: Build and run examples [W10-P03-T01-BUILD-AND-RUN-EXAMPLES]
  Currently: not yet started.
  DoD: hello_exit and representative examples pass.
  Verify: `go test ./tests/e2e/... -v`
- Task 02: Add end-to-end failure diagnostics
  [W10-P03-T02-E2E-DIAGNOSTICS]
  Currently: not yet started.
  DoD: build errors point to generated C only as implementation failures, not
  user errors.
  Verify: `go test ./compiler/driver/... -run TestBuildErrorDiagnostics -v`

### Wave Closure Phase [W10-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md and tests/e2e/README.md [W10-PCL-T01-UPDATE-DOCS]
  Verify: `go run tools/checkstubs/main.go && grep "hello_exit" tests/e2e/README.md`
- Task 02: Write WC010 learning-log entry [W10-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC010" docs/learning-log.md`

## Wave 11: CLI, Diagnostics, and Developer Workflows

Goal: expose the compiler through a coherent command-line interface and stable
developer workflow.

Entry criterion: Wave 10 done.

State on entry: `cmd/fuse/` exists as a minimal stub. Build pipeline is complete.

Exit criteria:

- build, run, check, test, fmt, doc, repl, version, and help flows exist
- diagnostics are readable in text and JSON
- tooling behavior is deterministic

Proof of completion:

```
fuse build --help
fuse version
fuse check tests/e2e/hello_exit.fuse
go test ./compiler/diagnostics/... -v
```

### Phase 00: Stub Audit [W11-P00-STUB-AUDIT]

- Task 01: Audit CLI stubs [W11-P00-T01-CLI-STUB-AUDIT]
  Currently: cmd/fuse prints not-yet-implemented for all subcommands.
  DoD: STUBS.md updated listing each subcommand that is not yet wired.
  Verify: `grep "subcommand\|cli" STUBS.md`

### Phase 01: CLI Surface [W11-P01-CLI-SURFACE]

- Task 01: Implement subcommand parser [W11-P01-T01-SUBCOMMAND-PARSER]
  Currently: cmd/fuse has no real subcommand parsing.
  DoD: common flags and subcommand-specific flags work.
  Verify: `fuse --help && fuse build --help && fuse check --help`
- Task 02: Wire top-level commands [W11-P01-T02-COMMAND-WIRING]
  Currently: not yet started.
  DoD: build, run, check, test, fmt, doc, repl, version, and help dispatch
  to their respective implementations.
  Verify: `fuse version` (prints version); `fuse build tests/e2e/hello_exit.fuse -o /tmp/he && /tmp/he`

### Phase 02: Diagnostics and Formatting [W11-P02-DIAGNOSTICS-AND-FORMATTING]

- Task 01: Stabilize diagnostic rendering [W11-P02-T01-DIAGNOSTIC-RENDERING]
  Currently: not yet started.
  DoD: human-readable diagnostics include spans and context; golden tests
  are byte-stable.
  Verify: `go test ./compiler/diagnostics/... -run TestDiagnosticRendering -v`
- Task 02: Add JSON output mode [W11-P02-T02-JSON-DIAGNOSTICS]
  Currently: not yet started.
  DoD: machine-readable diagnostics are emitted consistently when `-json` flag
  is passed.
  Verify: `fuse check nonexistent.fuse -json 2>&1 | python3 -m json.tool`

### Phase 03: Workflow Tools [W11-P03-WORKFLOW-TOOLS]

- Task 01: Implement doc and format workflows [W11-P03-T01-DOC-AND-FMT]
  Currently: not yet started.
  DoD: public API docs and formatting workflows are usable.
  Verify: `fuse fmt examples/ && fuse doc stdlib/core/`
- Task 02: Implement REPL and test runner workflows
  [W11-P03-T02-REPL-AND-TESTRUNNER]
  Currently: not yet started.
  DoD: developer-facing tools run against the same compiler core.
  Verify: `echo 'fn main() -> I32 { return 0; }' | fuse repl`

### Wave Closure Phase [W11-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W11-PCL-T01-UPDATE-STUBS]
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC011 learning-log entry [W11-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC011" docs/learning-log.md`

## Wave 12: Core Standard Library

Goal: implement the OS-free core standard library and use it as a semantic and
backend stress suite.

Entry criterion: Wave 11 done.

State on entry: `stdlib/core/` exists but is largely empty. All compiler
infrastructure is in place.

Exit criteria:

- core traits, primitives, strings, collections, iterators, and formatting ship
- all core public APIs are documented
- core library passes tests and compiler stress cases

Proof of completion:

```
fuse build stdlib/core/...
go test ./tests/... -run TestCoreStdlib -v
```

### Phase 00: Stub Audit [W12-P00-STUB-AUDIT]

- Task 01: Audit core stdlib stubs [W12-P00-T01-CORE-STUB-AUDIT]
  Currently: stdlib/core/ is mostly empty.
  DoD: STUBS.md updated noting which core modules are missing.
  Verify: `grep "stdlib.core" STUBS.md`

### Phase 01: Core Traits and Primitive Surface [W12-P01-CORE-TRAITS-AND-PRIMITIVES]

- Task 01: Ship core traits [W12-P01-T01-CORE-TRAITS]
  Currently: not yet started.
  DoD: equality, hashing, comparison, formatting, and default traits exist.
  Verify: `fuse build stdlib/core/traits/...`
- Task 02: Implement primitive methods and aliases
  [W12-P01-T02-PRIMITIVE-METHOD-SURFACE]
  Currently: not yet started.
  DoD: integer, float, bool, and char methods match language contracts.
  Verify: `fuse build stdlib/core/primitives/...`

### Phase 02: Strings, Collections, and Iteration [W12-P02-STRINGS-COLLECTIONS-AND-ITERATION]

- Task 01: Implement String and formatting primitives
  [W12-P02-T01-STRING-AND-FMT]
  Currently: not yet started.
  DoD: `String` and formatter builders are usable.
  Verify: `fuse build stdlib/core/string/...`
- Task 02: Implement List, Map, Set, and iterators
  [W12-P02-T02-COLLECTIONS-AND-ITERATORS]
  Currently: not yet started.
  DoD: collections stress generics, traits, and ownership correctly.
  Verify: `fuse build stdlib/core/collections/...`

### Phase 03: Runtime Bridge Layer [W12-P03-RUNTIME-BRIDGE-LAYER]

- Task 01: Implement core bridge files [W12-P03-T01-CORE-BRIDGE-FILES]
  Currently: bridge files listed in repository-layout.md do not exist.
  DoD: only approved bridge files contain unsafe runtime hooks; all are listed
  in STUBS.md during transition and removed when complete.
  Verify: `fuse build stdlib/core/rt_bridge/...`
- Task 02: Add doc coverage checks [W12-P03-T02-DOC-COVERAGE-CHECKS]
  Currently: not yet started.
  DoD: all public stdlib APIs have docs; CI runs `fuse doc --check`.
  Verify: `fuse doc --check stdlib/core/`

### Wave Closure Phase [W12-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W12-PCL-T01-UPDATE-STUBS]
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC012 learning-log entry [W12-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC012" docs/learning-log.md`

## Wave 13: Hosted Standard Library

Goal: implement the hosted stdlib tier on top of core while preserving the core
versus hosted boundary.

Entry criterion: Wave 12 done.

State on entry: `stdlib/full/` is empty. Core stdlib is complete.

Exit criteria:

- full tier IO, fs, os, time, thread, sync, and chan modules exist
- concurrency surface passes threaded tests
- hosted modules do not leak back into core

Proof of completion:

```
fuse build stdlib/full/...
go test ./tests/... -run TestHostedStdlib -v
go test ./tests/... -run TestConcurrency -v
```

### Phase 00: Stub Audit [W13-P00-STUB-AUDIT]

- Task 01: Audit hosted stdlib stubs [W13-P00-T01-HOSTED-STUB-AUDIT]
  Currently: stdlib/full/ is empty; spawn and channel compiler-side integration
  is in STUBS.md from Wave 08.
  DoD: STUBS.md confirmed current.
  Verify: `grep "spawn\|channel\|stdlib.full" STUBS.md`

### Phase 01: IO and OS Surface [W13-P01-IO-AND-OS-SURFACE]

- Task 01: Implement IO modules [W13-P01-T01-IO-MODULES]
  Currently: not yet started.
  DoD: stdin, stdout, stderr, files, and dirs work.
  Verify: `fuse build stdlib/full/io/...`
- Task 02: Implement OS modules [W13-P01-T02-OS-MODULES]
  Currently: not yet started.
  DoD: env, process, args, and time modules work.
  Verify: `fuse build stdlib/full/os/...`

### Phase 02: Threads, Sync, and Channels [W13-P02-THREADS-SYNC-AND-CHANNELS]

- Task 01: Implement thread and handle modules
  [W13-P02-T01-THREAD-AND-HANDLE]
  Currently: not yet started.
  DoD: spawn and thread handles work.
  Verify: `fuse build stdlib/full/thread/...`
- Task 02: Implement sync modules [W13-P02-T02-SYNC-MODULES]
  Currently: not yet started.
  DoD: mutex, rwlock, cond, once, and shared APIs work.
  Verify: `fuse build stdlib/full/sync/...`
- Task 03: Implement channels [W13-P02-T03-CHANNELS]
  Currently: not yet started.
  DoD: channel operations reflect the concurrency model from the guide.
  Verify: `fuse build stdlib/full/chan/...`

### Phase 03: Compiler-Side Concurrency Integration [W13-P03-COMPILER-CONCURRENCY-INTEGRATION]

- Task 01: Add channel type kind to the type table
  [W13-P03-T01-CHANNEL-TYPE-KIND]
  Currently: `KindChannel` does not exist in the type table; channel types
  produce Unknown in the checker.
  DoD: `KindChannel` exists in the type table with an element type parameter;
  `Chan[I32]` interns as a distinct TypeId.
  Verify: `go test ./compiler/typetable/... -run TestChannelTypeKind -v`
- Task 02: Lower spawn to runtime thread creation
  [W13-P03-T02-SPAWN-TO-RUNTIME]
  Currently: `SpawnExpr` lowers to `EmitCall(dest, arg, nil, Unit, false)`.
  DoD: `spawn expr` emits `fuse_rt_thread_spawn(fn, arg)` in codegen; no
  synchronous call is emitted.
  Verify: `go test ./compiler/codegen/... -run TestSpawnEmission -v`
  (test must grep generated C for `fuse_rt_thread_spawn`)
- Task 03: Lower channel operations to runtime calls
  [W13-P03-T03-CHANNEL-OPS-TO-RUNTIME]
  Currently: channel operations lower to unknown calls.
  DoD: send, recv, and close emit corresponding `fuse_rt_chan_*` calls.
  Verify: `go test ./compiler/codegen/... -run TestChannelOpsEmission -v`
- Task 04: Type-check channel expressions with element types
  [W13-P03-T04-CHANNEL-TYPECHECK]
  Currently: not yet started.
  DoD: `Chan[I32]` is a valid type; send/recv are type-checked against the
  element type; a send of `Bool` to `Chan[I32]` is a type error.
  Verify: `go test ./compiler/check/... -run TestChannelTypecheck -v`

### Wave Closure Phase [W13-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W13-PCL-T01-UPDATE-STUBS]
  DoD: spawn and channel stubs retired.
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC013 learning-log entry [W13-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC013" docs/learning-log.md`

## Wave 14: Stage 2 Compiler Port

Goal: bring up the self-hosted Fuse compiler source tree until it builds cleanly
with Stage 1.

Entry criterion: Wave 13 done.

State on entry: `stage2/src/` is empty. All stdlib tiers are complete.

Exit criteria:

- stage2 source mirrors stage1 architecture closely enough to bootstrap
- stage1 compiles stage2 end to end
- stage2 build failures are reduced to real stage2 defects, not stage1 gaps

Proof of completion:

```
fuse build stage2/src/...
fuse check stage2/src/...
```

### Phase 00: Stub Audit [W14-P00-STUB-AUDIT]

- Task 01: Audit stage2 stubs [W14-P00-T01-STAGE2-STUB-AUDIT]
  Currently: stage2/src/ is empty.
  DoD: STUBS.md documents that stage2 is being ported.
  Verify: `grep "stage2" STUBS.md`

### Phase 01: Stage 2 Frontend Parity [W14-P01-STAGE2-FRONTEND-PARITY]

- Task 01: Port frontend modules [W14-P01-T01-PORT-FRONTEND]
  Currently: not yet started.
  DoD: lex, parse, resolve, HIR, and checker sources exist in Fuse.
  Verify: `fuse check stage2/src/compiler/lex/...`
- Task 02: Port core driver and helpers [W14-P01-T02-PORT-DRIVER-HELPERS]
  Currently: not yet started.
  DoD: stage2 has the minimal executable architecture to compile itself.
  Verify: `fuse build stage2/src/... -o /tmp/stage2_compiler`

### Phase 02: Stage 2 Backend Bring-Up [W14-P02-STAGE2-BACKEND-BRING-UP]

- Task 01: Close frontend checker gaps [W14-P02-T01-CLOSE-FRONTEND-GAPS]
  Currently: not yet started.
  DoD: `fuse check stage2` succeeds with no Unknown types.
  Verify: `fuse check stage2/src/... 2>&1 | grep -c "Unknown"` (must be 0)
- Task 02: Close backend contract gaps [W14-P02-T02-CLOSE-BACKEND-GAPS]
  Currently: not yet started.
  DoD: stage2 generated C compiles significantly beyond example programs.
  Verify: `fuse build stage2/src/... 2>&1 | head -20`

### Phase 03: Stage 2 Build Closure [W14-P03-STAGE2-BUILD-CLOSURE]

- Task 01: Eliminate stage2 C generation breakpoints
  [W14-P03-T01-ELIMINATE-STAGE2-C-BREAKPOINTS]
  Currently: not yet started.
  DoD: no remaining failures are explained by missing stage1 contracts.
  Verify: `fuse build stage2/src/...` (zero errors)
- Task 02: Build stage2 end to end [W14-P03-T02-BUILD-STAGE2-END-TO-END]
  Currently: not yet started.
  DoD: stage1 produces a working stage2 compiler artifact.
  Verify: `fuse build stage2/src/... -o /tmp/fusec2 && /tmp/fusec2 --version`

### Wave Closure Phase [W14-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W14-PCL-T01-UPDATE-STUBS]
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC014 learning-log entry [W14-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC014" docs/learning-log.md`

## Wave 15: Self-Hosting Gate

Goal: prove that Fuse can compile itself reproducibly.

Entry criterion: Wave 14 done.

State on entry: stage2 compiles with stage1 but has not compiled itself yet.

Exit criteria:

- stage1 compiles stage2 successfully
- stage2 recompiles itself successfully
- output equivalence or reproducibility checks pass according to project policy

Proof of completion:

```
fuse build stage2/src/... -o /tmp/fusec2
/tmp/fusec2 build stage2/src/... -o /tmp/fusec2_gen2
make repro   # reproducibility check
```

### Phase 00: Stub Audit [W15-P00-STUB-AUDIT]

- Task 01: Audit self-hosting gaps [W15-P00-T01-SELF-HOSTING-AUDIT]
  Currently: stage2 has not compiled itself.
  DoD: STUBS.md documents any compiler features stage2 uses that are still
  stubbed.
  Verify: `go run tools/checkstubs/main.go`

### Phase 01: First Self-Compilation [W15-P01-FIRST-SELF-COMPILATION]

- Task 01: Compile stage2 with stage1 [W15-P01-T01-STAGE1-COMPILES-STAGE2]
  Currently: not yet done at this wave's scope.
  DoD: a working stage2 artifact is produced.
  Verify: `fuse build stage2/src/... -o /tmp/fusec2 && /tmp/fusec2 --version`
- Task 02: Compile stage2 with stage2 [W15-P01-T02-STAGE2-COMPILES-ITSELF]
  Currently: not yet started.
  DoD: self-compilation completes successfully.
  Verify: `/tmp/fusec2 build stage2/src/... -o /tmp/fusec2_gen2 && /tmp/fusec2_gen2 --version`

### Phase 02: Reproducibility and Equivalence [W15-P02-REPRODUCIBILITY-AND-EQUIVALENCE]

- Task 01: Implement bootstrap reproducibility check
  [W15-P02-T01-BOOTSTRAP-REPRO-CHECK]
  Currently: not yet started.
  DoD: multi-generation outputs compare according to policy.
  Verify: `make repro` (exits 0)
- Task 02: Gate merges on bootstrap health [W15-P02-T02-GATE-ON-BOOTSTRAP]
  Currently: not yet started.
  DoD: self-hosting regressions are release-blocking; CI fails on regression.
  Verify: CI pipeline includes bootstrap health check

### Wave Closure Phase [W15-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W15-PCL-T01-UPDATE-STUBS]
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC015 learning-log entry [W15-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC015" docs/learning-log.md`

## Wave 16: Native Backend Transition

Goal: remove the bootstrap C11 backend from the compiler implementation path by
introducing the native backend on top of the same semantic contracts.

Entry criterion: Wave 15 done.

State on entry: stage2 compiles itself via C11. Native backend does not exist.

Exit criteria:

- native backend exists and passes correctness gates
- stage2 no longer depends on C11 codegen for its own build path

Proof of completion:

```
fuse build --backend=native tests/e2e/hello_exit.fuse -o /tmp/he_native && /tmp/he_native
echo $?   # must be 0
```

### Phase 00: Stub Audit [W16-P00-STUB-AUDIT]

- Task 01: Audit native backend stubs [W16-P00-T01-NATIVE-BACKEND-AUDIT]
  Currently: native backend does not exist.
  DoD: STUBS.md documents that native backend is planned for this wave.
  Verify: `grep "native.backend" STUBS.md`

### Phase 01: Native Backend Foundation [W16-P01-NATIVE-BACKEND-FOUNDATION]

- Task 01: Define native backend interface [W16-P01-T01-NATIVE-BACKEND-INTERFACE]
  Currently: not yet started.
  DoD: native backend consumes MIR without C11-specific assumptions.
  Verify: `go build ./compiler/codegen/native/...`
- Task 02: Reuse backend contracts [W16-P01-T02-REUSE-BACKEND-CONTRACTS]
  Currently: not yet started.
  DoD: pointer, unit, monomorphization, and divergence contracts remain intact
  in the native backend.
  Verify: `go test ./compiler/codegen/native/... -run TestBackendContracts -v`

### Phase 02: Native Backend Closure [W16-P02-NATIVE-BACKEND-CLOSURE]

- Task 01: Compile stage2 through native path [W16-P02-T01-COMPILE-STAGE2-NATIVELY]
  Currently: not yet started.
  DoD: stage2 builds without C11 backend dependency.
  Verify: `fuse build --backend=native stage2/src/... -o /tmp/fusec2_native && /tmp/fusec2_native --version`
- Task 02: Remove bootstrap-only C11 requirement
  [W16-P02-T02-REMOVE-C11-REQUIREMENT]
  Currently: not yet started.
  DoD: C11 backend becomes optional; compiler builds without it in the active path.
  Verify: `fuse build --backend=native tests/e2e/hello_exit.fuse -o /tmp/he && /tmp/he`

### Wave Closure Phase [W16-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W16-PCL-T01-UPDATE-STUBS]
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC016 learning-log entry [W16-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC016" docs/learning-log.md`

## Wave 17: Generics End-to-End

Goal: make generic functions and generic types compile through the full pipeline
and produce correct running programs. Currently the `monomorph` package exists
but is not integrated, the checker resolves generic type args but does not
propagate them to specialization, and no generic program has ever compiled to a
working binary.

Entry criterion: Wave 16 done.

State on entry:
- `compiler/monomorph/` has `Record`, `Substitute`, and `IsGeneric` but is not
  called from the driver pipeline
- `compiler/check/checker.go:208-224` ignores `fn.GenericParams` during body
  checking
- `compiler/check/checker.go:checkCall` does not match explicit type args
- No call-site scanner collects concrete instantiations
- No body duplication produces concrete function variants
- No call-site rewriting changes generic call targets
- `compiler/driver/` goes directly from checking to HIR building
- Generic enum types have no concrete field layout in codegen
- `compiler/lower/lower.go:389` treats `?` as a pass-through

Exit criteria:

- a generic function called with two different concrete types produces two
  distinct specialized functions in the generated C
- `Option[I32]` and `Result[I32, Bool]` compile and run through pattern matching
- all proof programs in this wave compile, link, run, and return the expected
  exit code

Proof of completion:

```
go test ./tests/e2e/... -run TestIdentityGeneric -v       # exit code 42
go test ./tests/e2e/... -run TestMultipleInstantiations -v # exit code 13
go test ./tests/e2e/... -run TestOptionMatch -v            # exit code 42
go test ./tests/e2e/... -run TestErrorPropagation -v       # exit code 43
# CI must be green on all four proof programs on Linux, macOS, Windows
```

Proof programs (each must pass as a CI-gated e2e test):

```fuse
// P1: basic generic function (tests/e2e/identity_generic.fuse)
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity[I32](42); }
// expected: exit code 42

// P2: two instantiations of the same generic (tests/e2e/multiple_instantiations.fuse)
fn first[T](a: T, b: T) -> T { return a; }
fn main() -> I32 {
    let x = first[I32](10, 20);
    let y = first[I32](3, 7);
    return x + y;
}
// expected: exit code 13

// P3: generic type (Option) with pattern matching (tests/e2e/option_match.fuse)
// requires enum layout, discriminant, and match dispatch

// P4: error propagation with generics (tests/e2e/error_propagation.fuse)
// see language-guide.md §13
```

### Phase 00: Stub Audit [W17-P00-STUB-AUDIT]

- Task 01: Document all generics stubs [W17-P00-T01-GENERICS-STUB-AUDIT]
  Currently: see "State on entry" above.
  DoD: STUBS.md updated with all 10 gaps identified in L015, each with the
  file:line, current behavior, and the phase that retires it.
  Verify: `grep -c "W17" STUBS.md` (must be ≥ 10)

### Phase 01: Generic Parameter Scoping in the Checker [W17-P01-GENERIC-PARAM-SCOPING]

Currently the checker resolves type expressions but does not register generic
type parameters (`T`, `U`) as in-scope types during body checking of generic
functions. This phase ensures `T` is a valid type inside the body.

- Task 01: Register generic params in the function's local type scope
  [W17-P01-T01-REGISTER-GENERIC-PARAMS]
  Currently: `compiler/check/checker.go:208-224` ignores `fn.GenericParams`.
  DoD: inside `fn identity[T](x: T) -> T { return x; }`, the parameter `x`
  resolves to type `T` (a GenericParam TypeId), and the return type resolves to
  `T`.
  Verify: `go test ./compiler/check/... -run TestGenericParamScoping -v`
- Task 02: Resolve explicit type arguments at call sites
  [W17-P01-T02-RESOLVE-CALL-SITE-TYPE-ARGS]
  Currently: `checkCall` does not match explicit type args like
  `identity[I32](42)` to the generic function's type parameters.
  DoD: `identity[I32](42)` resolves `T=I32` and the call's return type is
  `I32`.
  Verify: `go test ./compiler/check/... -run TestCallSiteTypeArgs -v`
- Task 03: Add checker tests for generic param scoping
  [W17-P01-T03-CHECKER-GENERIC-TESTS]
  Currently: no tests for generic param scoping exist.
  DoD: unit tests verify that generic param types flow through function bodies
  and that explicit type args at call sites resolve correctly.
  Verify: `go test ./compiler/check/... -run TestGenericParam -v`

### Phase 02: Instantiation Collection [W17-P02-INSTANTIATION-COLLECTION]

The `compiler/monomorph/` package has `Context.Record()` but it is never called.

- Task 01: Scan checked AST for generic call sites
  [W17-P02-T01-SCAN-GENERIC-CALL-SITES]
  Currently: no code walks the checked AST to find calls with type arguments.
  DoD: after checking, the driver collects all concrete instantiations via
  `monomorph.Context.Record()`.
  Verify: `go test ./compiler/driver/... -run TestInstantiationCollection -v`
- Task 02: Validate instantiation completeness
  [W17-P02-T02-VALIDATE-INSTANTIATION-COMPLETENESS]
  Currently: `monomorph.Record()` rejects partial specializations but is never
  called from the pipeline.
  DoD: a call like `identity[Unknown]()` produces a diagnostic in the integrated
  pipeline.
  Verify: `go test ./compiler/driver/... -run TestPartialInstantiationRejected -v`
- Task 03: Add driver tests for instantiation collection
  [W17-P02-T03-INSTANTIATION-COLLECTION-TESTS]
  Currently: no driver tests for this.
  DoD: driver tests verify that `Build()` with a generic program produces the
  expected instantiation set.
  Verify: `go test ./compiler/driver/... -run TestGenericInstantiations -v`

### Phase 03: Generic Function Body Specialization [W17-P03-BODY-SPECIALIZATION]

- Task 01: Implement AST-level body duplication with type substitution
  [W17-P03-T01-AST-BODY-DUPLICATION]
  Currently: no code duplicates a function body.
  DoD: given `fn identity[T](x: T) -> T { return x; }` and `T=I32`, a new
  concrete `FnDecl` is produced with `T` replaced by `I32` in all parameter
  types, return types, and body type expressions.
  Verify: `go test ./compiler/monomorph/... -run TestBodyDuplication -v`
- Task 02: Generate specialized function names
  [W17-P03-T02-SPECIALIZED-FUNCTION-NAMES]
  Currently: no naming scheme for specialized functions exists.
  DoD: `identity[I32]` produces a function named `identity__I32` (deterministic
  scheme). The name is used consistently in declaration and at call sites.
  Verify: `go test ./compiler/monomorph/... -run TestSpecializedNames -v`
- Task 03: Rewrite call sites to reference specialized names
  [W17-P03-T03-REWRITE-CALL-SITES]
  Currently: call sites reference the generic name.
  DoD: `identity[I32](42)` in the caller's body is rewritten to call
  `identity__I32(42)`.
  Verify: `go test ./compiler/monomorph/... -run TestCallSiteRewrite -v`
- Task 04: Add unit tests for body specialization
  [W17-P03-T04-SPECIALIZATION-TESTS]
  Currently: not yet started.
  DoD: unit tests verify that a generic function produces a correctly typed
  concrete function after substitution, and that call sites reference the
  specialized name.
  Verify: `go test ./compiler/monomorph/... -v`

### Phase 04: Driver Pipeline Integration [W17-P04-DRIVER-PIPELINE-INTEGRATION]

Wire the specialization output into the existing driver pipeline so specialized
functions flow through HIR → liveness → MIR → codegen.

- Task 01: Insert specialization step between checking and HIR building
  [W17-P04-T01-INSERT-SPECIALIZATION-STEP]
  Currently: `compiler/driver/` goes directly from checking to HIR building.
  DoD: the driver runs instantiation collection, then body specialization,
  then feeds the specialized concrete functions into the HIR builder.
  Verify: `go test ./compiler/driver/... -run TestSpecializationInPipeline -v`
- Task 02: Skip generic function originals in codegen
  [W17-P04-T02-SKIP-GENERIC-ORIGINALS]
  Currently: generic functions with unresolved type params would produce invalid
  C if lowered.
  DoD: functions with `GenericParams` are not lowered unless they have been
  specialized. Only concrete specializations reach codegen.
  Verify: `go test ./compiler/codegen/... -run TestGenericOriginalsSkipped -v`
- Task 03: Verify generated C for basic generic proof program
  [W17-P04-T03-VERIFY-GENERATED-C]
  Currently: no generic program has produced C output.
  DoD: `identity[I32](42)` produces C containing a function `Fuse_identity__I32`
  that takes `int32_t` and returns `int32_t`, and `main` calls it. The C
  compiles with gcc.
  Verify: `fuse build tests/e2e/identity_generic.fuse -emit-c | gcc -x c - -o /tmp/ig && /tmp/ig; echo $?`
  (must print 42)

### Phase 05: Proof Program P1 — Basic Generic Function [W17-P05-PROOF-P1]

- Task 01: Add e2e test for basic generic function
  [W17-P05-T01-E2E-BASIC-GENERIC]
  Currently: `tests/e2e/identity_generic.fuse` does not exist.
  DoD: the program compiles, runs, and exits with code 42. The test is
  committed to `tests/e2e/` and `tests/e2e/README.md` is updated.
  Verify: `go test ./tests/e2e/... -run TestIdentityGeneric -v`
- Task 02: Fix any failures surfaced by the proof program
  [W17-P05-T02-FIX-P1-FAILURES]
  Currently: unknown until P1 runs.
  DoD: the e2e test passes. Any bugs found are fixed with regressions and
  learning-log entries.
  Verify: `go test ./tests/e2e/... -run TestIdentityGeneric -v` (exit 0)

### Phase 06: Multiple Instantiations [W17-P06-MULTIPLE-INSTANTIATIONS]

- Task 01: Verify two instantiations produce distinct functions
  [W17-P06-T01-TWO-INSTANTIATIONS]
  Currently: `monomorph.Record()` deduplicates, so `identity[I32]` and
  `identity[Bool]` should produce two entries — but the pipeline is not
  integrated yet.
  DoD: generated C contains both `Fuse_identity__I32` and `Fuse_identity__Bool`.
  Verify: `fuse build tests/e2e/multiple_instantiations.fuse -emit-c | grep -c "identity__"`
  (must be ≥ 2)
- Task 02: Add e2e test for multiple instantiations
  [W17-P06-T02-E2E-MULTIPLE-INSTANTIATIONS]
  Currently: `tests/e2e/multiple_instantiations.fuse` does not exist.
  DoD: the program compiles, runs, and exits with code 13.
  Verify: `go test ./tests/e2e/... -run TestMultipleInstantiations -v`

### Phase 07: Generic Types (Option, Result) [W17-P07-GENERIC-TYPES]

- Task 01: Specialize generic enum types
  [W17-P07-T01-SPECIALIZE-GENERIC-ENUMS]
  Currently: `Option[I32]` is interned as a TypeId with TypeArgs but no
  concrete enum layout is generated.
  DoD: `Option[I32]` produces a concrete C struct with a `_tag` field and
  a payload field of type `int32_t`.
  Verify: `fuse build tests/e2e/option_match.fuse -emit-c | grep "_tag"` (must match)
- Task 02: Emit specialized enum type definitions in codegen
  [W17-P07-T02-EMIT-SPECIALIZED-ENUM-TYPES]
  Currently: codegen emits `typedef struct Name Name;` for enums but does not
  emit field definitions for tag and payload.
  DoD: generated C for `Option[I32]` includes a struct definition with `int _tag;`
  and `int32_t _f0;` fields.
  Verify: `fuse build tests/e2e/option_match.fuse -emit-c | grep "_f0"` (must match)
- Task 03: Specialize generic struct types
  [W17-P07-T03-SPECIALIZE-GENERIC-STRUCTS]
  Currently: not yet started.
  DoD: generic structs with type parameters produce concrete field layouts
  after substitution.
  Verify: `go test ./compiler/codegen/... -run TestGenericStructLayout -v`

### Phase 08: Enum Construction and Destructuring with Generics [W17-P08-ENUM-GENERICS]

- Task 01: Emit specialized enum variant constructors
  [W17-P08-T01-SPECIALIZED-ENUM-CONSTRUCTORS]
  Currently: not yet started.
  DoD: `Option.Some[I32](42)` produces C that initializes the specialized
  `Option_I32` struct with `_tag=0` and `_f0=42`.
  Verify: `go test ./compiler/codegen/... -run TestSpecializedEnumConstructors -v`
- Task 02: Lower match on specialized enums to discriminant dispatch
  [W17-P08-T02-MATCH-ON-SPECIALIZED-ENUMS]
  Currently: not yet started.
  DoD: `match opt { Some(v) { ... } None { ... } }` on `Option[I32]` reads
  `_tag`, branches, and extracts `_f0` as `int32_t`.
  Verify: `go test ./compiler/codegen/... -run TestMatchOnSpecializedEnum -v`
- Task 03: Add e2e test for Option with pattern matching
  [W17-P08-T03-E2E-OPTION-MATCH]
  Currently: `tests/e2e/option_match.fuse` does not exist.
  DoD: a program that creates `Some(42)`, matches on it, and returns the
  inner value compiles, runs, and exits with code 42.
  Verify: `go test ./tests/e2e/... -run TestOptionMatch -v`

### Phase 09: Error Propagation with Generics [W17-P09-ERROR-PROPAGATION-GENERICS]

The `?` operator was wired for the lowerer (L009) but requires a specialized
`Result[T, E]` type to work end-to-end.

- Task 01: Verify `?` on specialized Result type
  [W17-P09-T01-QUESTION-ON-SPECIALIZED-RESULT]
  Currently: `?` lowering emits a branch, but `Result[I32, Bool]` has no
  concrete layout yet.
  DoD: `?` on `Result[I32, Bool]` reads `_tag`, branches, extracts `_f0` as
  `int32_t` on success, and early-returns the Result on failure.
  Verify: `go test ./compiler/codegen/... -run TestQuestionOnSpecializedResult -v`
- Task 02: Add e2e test for error propagation
  [W17-P09-T02-E2E-ERROR-PROPAGATION]
  Currently: `tests/e2e/error_propagation.fuse` does not exist.
  DoD: a program that uses `?` on a Result value compiles, runs, and returns
  the expected exit code for both the Ok and Err paths.
  Verify: `go test ./tests/e2e/... -run TestErrorPropagation -v`

### Phase 10: Regression Closure [W17-P10-REGRESSION-CLOSURE]

- Task 01: Add regression tests for all fixed generic edge cases
  [W17-P10-T01-GENERIC-REGRESSIONS]
  Currently: regressions not yet written.
  DoD: every bug found during this wave has a regression test.
  Verify: `go test ./... -run TestGenericRegression -v`
- Task 02: Update learning log with any new lessons
  [W17-P10-T02-UPDATE-LEARNING-LOG]
  Currently: not yet started.
  DoD: learning-log entries exist for any design gaps or architectural lessons
  discovered during generic implementation.
  Verify: `grep "L016\|L017" docs/learning-log.md` (or whatever the next entries are)
- Task 03: Verify all proof programs pass in CI
  [W17-P10-T03-CI-PROOF-PROGRAMS]
  Currently: not yet started.
  DoD: the e2e test suite including all generic proof programs passes in the
  CI matrix (Linux, macOS, Windows).
  Verify: CI green on all four proof programs across all three platforms

### Wave Closure Phase [W17-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W17-PCL-T01-UPDATE-STUBS]
  DoD: all 10 generic stubs from STUBS.md retired; no new stubs introduced.
  Verify: `grep -c "W17" STUBS.md` (must be 0)
- Task 02: Update tests/e2e/README.md [W17-PCL-T02-UPDATE-E2E-README]
  DoD: all four proof programs listed in README.md with expected outputs.
  Verify: `grep "identity_generic\|multiple_instantiations\|option_match\|error_propagation" tests/e2e/README.md`
- Task 03: Write WC017 learning-log entry [W17-PCL-T03-CLOSURE-LOG]
  Verify: `grep "WC017" docs/learning-log.md`

## Wave 18: Retirement of Go and C from the Compiler Path

Goal: complete the transition from bootstrap implementation languages to a Fuse
compiler implemented and built by Fuse.

Entry criterion: Wave 17 done.

State on entry: Generics work end-to-end. Go is still required to build Stage 1.
C is still required as a bootstrap backend.

Exit criteria:

- Fuse owns the compiler implementation path
- Go is no longer required to build the compiler
- C is no longer required as a backend or runtime implementation dependency in
  the compiler path

Proof of completion:

```
# must succeed without Go toolchain in PATH:
fuse build stage2/src/... -o /tmp/fusec2_final
/tmp/fusec2_final --version
```

### Phase 00: Stub Audit [W18-P00-STUB-AUDIT]

- Task 01: Audit bootstrap dependencies [W18-P00-T01-BOOTSTRAP-AUDIT]
  Currently: Go and C both required.
  DoD: STUBS.md documents all remaining bootstrap dependencies.
  Verify: `go run tools/checkstubs/main.go`

### Phase 01: Retire Go [W18-P01-RETIRE-GO]

- Task 01: Freeze Stage 1 as archival bootstrap tool [W18-P01-T01-FREEZE-STAGE1]
  Currently: Stage 1 is the active compiler.
  DoD: Stage 1 is no longer required for ordinary compiler development.
  Verify: `fuse build tests/e2e/hello_exit.fuse -o /tmp/he_fuse && /tmp/he_fuse`
- Task 02: Remove Go from active compiler build workflow
  [W18-P01-T02-REMOVE-GO-FROM-WORKFLOW]
  Currently: `make stage1` requires Go.
  DoD: supported build path no longer invokes Go.
  Verify: `PATH="" make stage1 2>&1 | grep -v "go"` (Go not invoked)

### Phase 02: Retire C [W18-P02-RETIRE-C]

- Task 01: Replace C runtime dependencies as required
  [W18-P02-T01-REPLACE-C-RUNTIME]
  Currently: runtime/ is C.
  DoD: compiler implementation path no longer requires C runtime code.
  Verify: `fuse build tests/e2e/hello_exit.fuse -o /tmp/he && /tmp/he`
- Task 02: Remove C from compiler bootstrap assumptions
  [W18-P02-T02-REMOVE-C-FROM-BOOTSTRAP-ASSUMPTIONS]
  Currently: C11 required for codegen.
  DoD: compiler implementation is Fuse-only.
  Verify: `fuse build --backend=native tests/e2e/hello_exit.fuse -o /tmp/he && /tmp/he`

### Wave Closure Phase [W18-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W18-PCL-T01-UPDATE-STUBS]
  Verify: `go run tools/checkstubs/main.go` (or fuse equivalent after retirement)
- Task 02: Write WC018 learning-log entry [W18-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC018" docs/learning-log.md`

## Wave 19: Targets and Ecosystem Growth

Goal: resume broader target and library work on top of the self-hosted native
compiler.

Entry criterion: Wave 18 done.

State on entry: Fuse owns the compiler. Bootstrap languages are retired.

Exit criteria:

- target expansion and library growth occur without reintroducing bootstrap debt

Proof of completion:

```
fuse build --target=<new-target> tests/e2e/hello_exit.fuse -o /tmp/he_<target>
```

### Phase 00: Stub Audit [W19-P00-STUB-AUDIT]

- Task 01: Audit target and ecosystem stubs [W19-P00-T01-ECOSYSTEM-AUDIT]
  Currently: only the initial target (host) is supported.
  DoD: STUBS.md documents planned targets.
  Verify: `grep "target" STUBS.md`

### Phase 01: Additional Targets [W19-P01-ADDITIONAL-TARGETS]

- Task 01: Add target descriptions [W19-P01-T01-TARGET-DESCRIPTIONS]
  Currently: not yet started.
  DoD: each supported target has a documented ABI and validation path.
  Verify: `ls docs/targets/`
- Task 02: Add target CI [W19-P01-T02-TARGET-CI]
  Currently: not yet started.
  DoD: target regressions are visible immediately in CI.
  Verify: CI green on new target build

### Phase 02: Extended Libraries [W19-P02-EXTENDED-LIBRARIES]

- Task 01: Implement ext stdlib modules [W19-P02-T01-EXT-STDLIB]
  Currently: stdlib/ext/ is empty.
  DoD: optional libraries build on the stable core and hosted tiers.
  Verify: `fuse build stdlib/ext/...`
- Task 02: Publish ecosystem guidance [W19-P02-T02-ECOSYSTEM-GUIDANCE]
  Currently: not yet started.
  DoD: package authors have clear compatibility and safety rules.
  Verify: `ls docs/ecosystem-guide.md`

### Wave Closure Phase [W19-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md [W19-PCL-T01-UPDATE-STUBS]
  Verify: `go run tools/checkstubs/main.go`
- Task 02: Write WC019 learning-log entry [W19-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC019" docs/learning-log.md`

## Cross-cutting constraints

The following rules apply to every wave.

- Determinism is a release-level requirement.
- No unresolved types may reach codegen.
- No pass may recompute liveness independently.
- Invariant walkers remain enabled in debug and CI contexts.
- Stdlib failures are compiler signals, not library excuses.
- Workarounds are forbidden.
- Each non-trivial bug must produce both a regression and a learning-log entry.
- Every wave that introduces a user-visible feature must include at least one
  end-to-end proof program: a Fuse source file that compiles, links, runs, and
  produces a verified output. The proof program must fail if the feature is
  stubbed (Rule 6.8).
- Exit criteria must include behavioral requirements ("this program produces
  exit code N"), not only structural ones ("HIR nodes carry metadata").
  Structural criteria are necessary but never sufficient alone (Rule 6.10).
- Every task must carry a `Currently:` line naming the existing state at the
  file and line level where possible, and a `Verify:` line giving the command
  that proves the task done. A task whose DoD cannot be expressed as a runnable
  command is not ready to be worked.
- Stubs must emit compiler diagnostics, not silent defaults. A feature that
  parses and type-checks but is not lowered must produce an error, not a
  silently wrong program (Rule 6.9). Every stub must be in STUBS.md.
- Every wave requires a Phase 00 Stub Audit committed before other phases begin
  (Rule 6.12) and a Wave Closure Phase committed before sign-off (Rule 6.14).
- `STUBS.md` must be updated at every wave boundary (Rule 6.13).
- `tests/e2e/README.md` must be updated whenever a proof program is added.
