# STUBS

> Status: normative root-level infrastructure for Fuse.
>
> This file is the live registry of every compiler stub and the append-only
> log of stub lifecycle events. It is governed by
> [docs/rules.md](docs/rules.md) §6.9–§6.16 and
> [docs/repository-layout.md](docs/repository-layout.md) §2.
>
> Every stub listed in the Active stubs table below must emit the declared
> diagnostic when the corresponding language feature is used. Silent stubs
> are forbidden (Rule 6.9). A stub without a concrete retiring wave is a
> defect (Rule 2.5).
>
> The Stub history section is append-only (Rule 10.1, Rule 6.16). Existing
> entries are never edited; new wave-closure blocks are appended at wave
> boundaries.

## Active stubs

At Wave 00 the entire compiler is a forward-declared stub. Each row below
names a language-feature group, the package that will own it, the
placeholder behavior (all present packages are empty doc-only stubs), the
diagnostic the compiler must emit when the feature is used, and the wave
that retires the stub. File:Line is of the form `package/ (not yet
created)` until real code exists; the retiring wave fills in the line
number when it lands the feature.

| Stub | File:Line | Current behavior | Diagnostic emitted | Retiring wave |
|---|---|---|---|---|
| Concurrency checker (Send/Sync/Chan/spawn/@rank) | compiler/check/ (W06 type checker only; no concurrency enforcement) | spawn/Chan/@rank use is untyped at W06 | "concurrency checker not yet implemented" | W07 |
| Monomorphization | compiler/monomorph/ (empty) | no generic specialization | "monomorphization not yet implemented" | W08 |
| Ownership, liveness, borrow rules, drop codegen | compiler/liveness/ (empty) | no ownership enforcement | "ownership/liveness not yet implemented" | W09 |
| Pattern matching dispatch and exhaustiveness | compiler/check/ (W06 type checker only; no match dispatch/exhaustiveness) | match arms type-check but exhaustiveness is not enforced | "pattern matching not yet implemented" | W10 |
| Error propagation (`?` operator) | compiler/lower/ (W05 spine only; no `?` lowering) | `?` operator emits a lowerer diagnostic | "error propagation not yet implemented" | W11 |
| Closures, capture, `move` prefix, Fn/FnMut/FnOnce | compiler/lower/ (W05 spine only; no closure lifting) | closure expressions emit a lowerer diagnostic | "closures not yet implemented" | W12 |
| Trait objects (`dyn Trait`, vtables, object safety) | compiler/codegen/ (W05 C11 subset; no dynamic dispatch) | `dyn Trait` use emits a codegen diagnostic | "trait objects not yet implemented" | W13 |
| Compile-time evaluation (`const fn`, `size_of`, `align_of`) | compiler/ (not yet created consteval/) | no const evaluation | "const fn not yet implemented" | W14 |
| MIR consolidation (casts, fn pointers, slice range, struct update, overflow arithmetic) | compiler/lower/ + compiler/mir/ (W05 minimal subset only) | non-spine forms emit lowerer diagnostics | "MIR lowering not yet implemented" | W15 |
| Runtime ABI (threads, channels, panic, IO) | runtime/src/ (W05 fuse_rt_abort only; threads/channels/IO call abort with "not yet implemented") | stub runtime entries abort at runtime | "runtime not yet implemented" | W16 |
| Codegen C11 hardening (`@repr`, `@align`, `@inline`, intrinsics, variadic, debug info, perf baseline) | compiler/codegen/ (W05 emitter only; no hardening) | hardening decorators are ignored by the W05 emitter | "C11 codegen not yet implemented" | W17 |
| CLI, diagnostics, `fuse fmt/doc/repl`, incremental driver, Rule 6.17 audit | compiler/driver/ (W05 `build` subcommand only; no `run`/`check`/`test`/`fmt`/`doc`/`repl`, no incremental driver) | unimplemented subcommands exit non-zero with a usage message | "subcommand not yet implemented" | W18 |
| Language server (LSP 3.17) | compiler/ (not yet created lsp/) | no LSP server | "fuse lsp not yet implemented" | W19 |
| Stdlib core (traits, primitives, strings, collections, Cell/RefCell, Ptr.null, overflow methods) | stdlib/core/ (empty) | no stdlib | "stdlib core not yet implemented" | W20 |
| Custom allocators (Allocator trait, parameterized collections) | stdlib/core/alloc/ (not yet created) | no allocator trait | "custom allocators not yet implemented" | W21 |
| Stdlib hosted (IO, fs, os, time, thread, sync, channels, network) | stdlib/full/ (empty) | no hosted stdlib | "stdlib hosted not yet implemented" | W22 |
| Package management (manifest, lockfile, resolver, fetcher, registry protocol) | compiler/ (not yet created pkg/) | no package manager | "package manager not yet implemented" | W23 |
| Stub clearance gate | n/a — gating wave | clearance happens at wave entry | n/a — policy wave | W24 |
| Stage 2 self-hosting | stage2/src/ (empty) | no stage2 compiler | "stage 2 compiler not yet ported" | W25 |
| Native backend with DWARF | compiler/ (not yet created codegen/native/) | no native backend | "native backend not yet implemented" | W26 |
| Performance gate (runtime ratios, compile-time budgets, code-size, memory footprint) | tests/perf/ (empty) | no perf gate | "perf gate not yet implemented" | W27 |
| Retirement of Go and C from active path | compiler/ & runtime/ | bootstrap stack active | n/a — retirement wave | W28 |
| Target matrix and `stdlib/ext/` | stdlib/ext/ (empty) | no ext stdlib | "stdlib ext not yet implemented" | W29 |
| Ecosystem documentation (tutorial, book, migration guides, site) | docs/ (tutorial/book/migration/ not yet created) | no user-facing docs | n/a — documentation wave | W30 |

## Stub history

The Stub history is append-only. Each wave closure appends a block naming
the stubs added this wave, the stubs retired this wave (with the proof
program or test that confirmed retirement), and any stubs rescheduled
(with reason). Entries are never edited in place (Rule 6.16).

### W00 — Governance and Phase Model

Added:
- Lexer and token model (compiler/lex/ empty) — emits "lexer not yet
  implemented" — retires W01
- Parser and AST (compiler/parse/ empty) — emits "parser not yet
  implemented" — retires W02
- Module resolver (compiler/resolve/ empty) — emits "resolver not yet
  implemented" — retires W03
- HIR and TypeTable (compiler/hir/ empty) — emits "HIR/TypeTable not yet
  implemented" — retires W04
- Minimal end-to-end spine (compiler/driver/ empty) — emits "Stage 1
  driver not yet implemented" — retires W05
- Type checker (compiler/check/ empty) — emits "type checker not yet
  implemented" — retires W06
- Concurrency checker (compiler/check/ empty) — emits "concurrency
  checker not yet implemented" — retires W07
- Monomorphization (compiler/monomorph/ empty) — emits "monomorphization
  not yet implemented" — retires W08
- Ownership, liveness, borrow rules, drop codegen (compiler/liveness/
  empty) — emits "ownership/liveness not yet implemented" — retires W09
- Pattern matching (compiler/check/ empty) — emits "pattern matching not
  yet implemented" — retires W10
- Error propagation (compiler/lower/ empty) — emits "error propagation
  not yet implemented" — retires W11
- Closures (compiler/lower/ empty) — emits "closures not yet implemented"
  — retires W12
- Trait objects (compiler/codegen/ empty) — emits "trait objects not yet
  implemented" — retires W13
- Compile-time evaluation (consteval package not yet created) — emits
  "const fn not yet implemented" — retires W14
- MIR consolidation (compiler/lower/ empty) — emits "MIR lowering not yet
  implemented" — retires W15
- Runtime ABI (runtime/src/ empty) — emits "runtime not yet implemented"
  — retires W16
- Codegen C11 hardening (compiler/codegen/ empty) — emits "C11 codegen
  not yet implemented" — retires W17
- CLI and diagnostics (compiler/driver/ empty) — emits "subcommand not
  yet implemented" — retires W18
- Language server (lsp package not yet created) — emits "fuse lsp not yet
  implemented" — retires W19
- Stdlib core (stdlib/core/ empty) — emits "stdlib core not yet
  implemented" — retires W20
- Custom allocators (stdlib/core/alloc/ not yet created) — emits "custom
  allocators not yet implemented" — retires W21
- Stdlib hosted (stdlib/full/ empty) — emits "stdlib hosted not yet
  implemented" — retires W22
- Package management (pkg package not yet created) — emits "package
  manager not yet implemented" — retires W23
- Stub clearance gate — policy wave, no stub — retires W24
- Stage 2 self-hosting (stage2/src/ empty) — emits "stage 2 compiler not
  yet ported" — retires W25
- Native backend with DWARF (compiler/codegen/native/ not yet created) —
  emits "native backend not yet implemented" — retires W26
- Performance gate (tests/perf/ empty) — emits "perf gate not yet
  implemented" — retires W27
- Retirement of Go and C — retirement wave, no stub — retires W28
- Target matrix and stdlib/ext (stdlib/ext/ empty) — emits "stdlib ext
  not yet implemented" — retires W29
- Ecosystem documentation (docs/book /tutorial /migration not yet
  created) — documentation wave, no stub — retires W30

Retired: (none this wave — W00 is the seeding wave)

Rescheduled: (none this wave)

### W01 — Lexer

Added: (none this wave)

Retired:
- Lexer and token model (compiler/lex/scanner.go, compiler/lex/token.go,
  compiler/lex/span.go) — confirmed retired by `go test
  ./compiler/lex/... -v` and `go test ./compiler/lex/... -run TestGolden
  -count=3 -v`. Proof surface: TestTokenKindCoverage, TestKeywords,
  TestLiterals, TestNestedBlockComment, TestRawStringGuard,
  TestOptionalChainToken, TestBomRejection, TestSpanStability,
  TestLexerFuzz, TestGolden (four golden fixtures under
  compiler/lex/testdata/).

Rescheduled: (none this wave)

### W02 — Parser and AST

Added: (none this wave)

Retired:
- Parser and AST (compiler/ast/*.go, compiler/parse/*.go) — confirmed
  retired by `go test ./compiler/ast/... -v`, `go test
  ./compiler/parse/... -v`, and `go test ./compiler/parse/... -run
  TestGolden -count=3 -v`. Proof surface: TestAstNodeCompleteness,
  TestSpanCorrectness, TestItemParsing, TestExprPrecedence,
  TestTypeExprs, TestPatternParsing, TestDecoratorParsing,
  TestStructLiteralDisambig, TestOptionalChainParse,
  TestNopanicOnMalformed (40 malformed-input cases), TestGolden (five
  golden fixtures under compiler/parse/testdata/).

Rescheduled: (none this wave)

### W03 — Resolution

Added: (none this wave)

Retired:
- Module resolver (compiler/resolve/*.go) — confirmed retired by `go
  test ./compiler/resolve/... -v`, with each wave-spec Verify command
  producing its declared passing output: `TestModuleDiscovery -count=3`,
  `TestModuleGraph`, `TestScopeLookup`, `TestTopLevelIndex`,
  `TestModuleFirstFallback`, `TestQualifiedEnumVariant`,
  `TestImportCycleDetection`, `TestCfgEvaluation`, `TestCfgDuplicates`,
  `TestVisibilityEnforcement`. Proof surface: determinism-checked
  filesystem discovery, ModuleGraph with sorted Order and Edges,
  symbol-table scope chain, top-level indexing across every item kind
  plus enum variant hoisting, module-first import fallback (reference
  §18.7), qualified enum variant resolution across a FieldExpr+PathExpr
  chain (reference §11.6), Tarjan-based import cycle detection that
  covers self-edges and multi-module cycles, `@cfg` predicate evaluator
  supporting `key = "value"`, `feature = "x"`, `not`, `all`, `any`,
  nested combinators, duplicate-item detection across `@cfg` overlaps
  (reference §50.1), and four-level visibility enforcement
  (private / pub(mod) / pub(pkg) / pub — reference §53.1).

Rescheduled: (none this wave)

### W04 — HIR and TypeTable

Added: (none this wave)

Retired:
- HIR and TypeTable (compiler/typetable/kind.go, compiler/typetable/type.go,
  compiler/typetable/table.go, compiler/hir/doc.go, compiler/hir/node.go,
  compiler/hir/identity.go, compiler/hir/item.go, compiler/hir/expr.go,
  compiler/hir/pat.go, compiler/hir/stmt.go, compiler/hir/program.go,
  compiler/hir/builder.go, compiler/hir/bridge.go, compiler/hir/invariant.go,
  compiler/hir/manifest.go, compiler/hir/incremental.go) — confirmed retired
  by `go test ./compiler/typetable/... -v`, `go test ./compiler/hir/... -v`,
  and each wave-spec Verify command. Proof surface: `TestTypeInternEquality`,
  `TestNominalIdentity`, `TestChannelTypeKindExists`, `TestThreadHandleKindExists`,
  `TestInferIsExplicit`, `TestHirNodeSet`, `TestMetadataFields`,
  `TestBuilderEnforcement` (10 sub-cases), `TestBuilderEnforcement_HappyPath`,
  `TestAstToHirTypePreservation` (6 sub-cases including nominal identity
  propagation across fn signatures and struct-literal constructors),
  `TestBridgeInvariant`, `TestInvariantWalkers` (clean + synthetic violation),
  `TestPassManifest` (5 sub-cases including cycle detection), `TestDeterministicOrder`
  (-count=3 confirms stable topological order), `TestPassFingerprintStable`
  (-count=3 confirms byte-identical fingerprints across runs),
  `TestStableNodeIdentity` (unrelated edit does not shift NodeIDs),
  `TestIncrementalSubstitutable` (invalidating one function's HIR re-runs
  only its dependent passes).

Rescheduled: (none this wave)

### W05 — Minimal End-to-End Spine

Added: (none this wave)

Retired:
- Minimal end-to-end spine (compiler/mir/mir.go, compiler/lower/lower.go,
  compiler/codegen/c11.go, compiler/cc/compiler.go, compiler/driver/build.go,
  cmd/fuse/main.go with the `build` subcommand, runtime/include/fuse_rt.h,
  runtime/src/abort.c) — confirmed retired by `go test ./... -v` and each
  wave-spec Verify command. Proof surface: `TestMinimalMir`,
  `TestMinimalLowerIntReturn` (+ `_Rejects` for every non-spine HIR form
  that must emit a diagnostic), `TestMinimalCodegenC` (literal main,
  arithmetic main, deterministic output, rejects unsupported op),
  `TestCCDetection` / `TestCCDetection_HonorsEnv` /
  `TestCCDetection_ErrorWhenAbsent`, `TestStubRuntime` (fuse_rt.h
  declares every W05/W07/W16/W22 surface; abort.c implements the W05
  abort), `TestMinimalBuildInvocation` (full pipeline: Fuse source →
  C11 → host compile → run → check exit code), `TestMinimalCli` (four
  sub-cases), `TestHelloExit` (exit 0), `TestExitWithValue` (exit 42
  via `6 * 7`). `tests/e2e/README.md` created as the proof-program
  registry per Rule 6.8.

Rescheduled: (none this wave)

### W06 — Type Checking

Added: (none this wave)

Retired:
- Type checker (compiler/check/checker.go, compiler/check/body.go,
  compiler/check/expr.go, compiler/check/trait.go,
  compiler/check/assoc.go, compiler/check/items.go,
  compiler/check/repr.go, compiler/check/invariant.go) — confirmed
  retired by `go test ./compiler/check/... -v` and each wave-spec
  Verify command. Proof surface: `TestFunctionTypeRegistration`,
  `TestTwoPassChecker`, `TestNominalEquality`, `TestPrimitiveMethods`,
  `TestNumericWidening` (widen + narrow-rejection sub-cases),
  `TestCastSemantics` (numeric-ok + bool-rejected),
  `TestConcreteTraitMethodLookup`, `TestTraitBoundLookup`
  (umbrella), `TestBoundChainLookup`, `TestCoherenceOrphan`
  (conflicting impls + orphan rule), `TestTraitParameters`,
  `TestContextualInference` (I64 literal hinted from let),
  `TestZeroArgTypeArgs`, `TestLiteralTyping`,
  `TestAssocTypeProjection`, `TestAssocTypeConstraints`,
  `TestFnPointerType`, `TestImplTraitParam`, `TestImplTraitReturn`,
  `TestUnionCheck` (primitive-ok + non-trivial-rejected),
  `TestNewtypePattern`, `TestReprAnnotationCheck` (8 sub-cases
  including conflicting @repr, int-on-struct rejection,
  non-power-of-two @align), `TestVariadicExternCheck`,
  `TestStdlibBodyChecking`, `TestNoUnknownAfterCheck`. End-to-end
  proof `TestCheckerBasicProof` compiles `checker_basic.fuse`
  (a multi-fn program with typed parameters and a direct call)
  and confirms exit 42.

Additionally extended (non-retiring-this-wave):
- MIR gained `OpParam` and `OpCall` so lowered code can read
  parameters and invoke named functions. The lowerer's spine
  opens up to multi-fn programs with typed int parameters.
- Codegen emits forward declarations for non-main functions so
  mutually-referential definitions compile regardless of source
  order; main narrows its i64 return to int via explicit cast.
- Driver pipeline now runs Check between the HIR bridge and MIR
  lowering. Failing type-check diagnostics surface through the
  driver's error return with the stage name "type checking".
- Resolver: single-segment type paths are now tolerated silently,
  since they may be generic parameters (`T`) the bridge registers.
- HIR bridge: added a generic-scope so `fn id[T](x: T)` types
  each bare `T` as a KindGenericParam TypeId; FieldExpr chains
  that the resolver bound to a module-qualified item now lower
  to a single PathExpr carrying the resolved symbol.

Rescheduled: (none this wave)
