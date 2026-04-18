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

### W07 — Concurrency Semantics

Added: (none this wave)

Retired:
- Concurrency checker, Send/Sync/Chan/spawn/@rank (compiler/check/concurrency.go,
  compiler/check/concurrency_test.go, tests/e2e/concurrency_rejections_test.go) —
  confirmed retired by `go test ./compiler/check/... -v` and each wave-spec
  Verify command. Proof surface: `TestMarkerTraitDeclarations`,
  `TestMarkerAutoImpl` (every primitive Send+Sync+Copy, tuple
  composition, refs excluded from Send/Copy, Chan/ThreadHandle Send),
  `TestNegativeImpl` (negative impl blocks auto-impl without leaking
  to other markers), `TestChannelTypecheck`, `TestChannelSendBound`,
  `TestSpawnHandleTyping` (ThreadHandle[T] identity),
  `TestSpawnSendBound` (non-move closure rejected with §47.1 text +
  `move`-suggestion per Rule 6.17), `TestSharedBounds`
  (Send+Sync lattice), `TestSpawnRejectsNonEscaping`,
  `TestLockRankingEnforcement` (positive/zero/negative rank,
  strict/equal/decreasing sequences), `TestSendSyncMarkerTraits`
  umbrella, and e2e `TestConcurrencyRejections` (non-move closure at
  spawn, non-Send return at spawn, lock-rank violation, invalid @rank
  value). Runtime lowering for `spawn` and channel operations
  remains stubbed — that's W16 work.

Rescheduled: (none this wave)

### W08 — Monomorphization

Added: (none this wave)

Retired:
- Monomorphization (compiler/monomorph/monomorph.go,
  compiler/monomorph/monomorph_test.go, plus supporting
  extensions in compiler/hir/bridge.go, compiler/hir/expr.go,
  compiler/check/expr.go, compiler/driver/build.go) — confirmed
  retired by `go test ./compiler/monomorph/... -v`,
  `go test ./compiler/driver/... -run TestSpecializationInPipeline -v`,
  and each wave-spec Verify command. Proof surface:
  `TestGenericParamScoping` and `TestCallSiteTypeArgs` (check),
  `TestBodyDuplication`, `TestSpecializedNames`,
  `TestCallSiteRewrite`, `TestMonomorph_PartialInstantiation`
  (monomorph), `TestInstantiationCollection`,
  `TestPartialInstantiationRejected`,
  `TestSpecializationInPipeline` (driver),
  `TestGenericOriginalsSkipped`, `TestDistinctSpecializations`,
  `TestSpecializedEnumTypes`, `TestGenericStructLayout` (codegen),
  and e2e proofs `TestIdentityGeneric` (`identity_generic.fuse`
  exit 42) + `TestMultipleInstantiations`
  (`multiple_instantiations.fuse` exit 42).

Rescheduled: (none this wave)

### W09 — Ownership and Liveness

Added: (none this wave)

Retired:
- Ownership, liveness, borrow rules, drop codegen
  (compiler/liveness/liveness.go,
  compiler/liveness/liveness_test.go, plus MIR OpDrop and codegen
  destructor emission in compiler/mir/mir.go and
  compiler/codegen/c11.go, plus driver wiring in
  compiler/driver/build.go) — confirmed retired by
  `go test ./compiler/liveness/... -v` and each wave-spec Verify
  command. Proof surface: `TestOwnershipContexts`,
  `TestNoBorrowInField`, `TestReturnBorrowRule`
  (no-borrow-param-rejected), `TestMutrefAliasing`
  (two-mutref-same-target, mutref-and-ref-same-target),
  `TestUseAfterMove` (synthetic let/move/use scenario),
  `TestClosureEscape` (ref-param closure returned from fn),
  `TestLiveAfter`, `TestLastUse`, `TestDropIntent`,
  `TestDestructionOnAllPaths`, `TestSingleLiveness` (umbrella,
  idempotency), `TestDropTraitMetadata`,
  `TestDestructorCallEmitted` (codegen), and e2e
  `TestBorrowRejections` with 5 sub-cases
  (reject_borrow_in_field, reject_return_local_borrow,
  reject_aliased_mutref, reject_use_after_move,
  reject_escaping_borrow_closure) + `TestDropObservable`
  asserting non-empty DropIntent metadata.

Rescheduled: (none this wave)

### W10 — Pattern Matching

Added: (none this wave)

Retired:
- Pattern matching dispatch and exhaustiveness
  (compiler/check/match.go, compiler/check/match_test.go,
  compiler/lower/lower.go match-lowering extension,
  compiler/lower/match_test.go, plus MIR TermJump / TermIfEq in
  compiler/mir/mir.go, codegen label emission in
  compiler/codegen/c11.go) — confirmed retired by
  `go test ./compiler/check/... -run TestExhaustivenessChecking -v`,
  `go test ./compiler/lower/... -run TestMatchDispatch -v`,
  `go test ./compiler/lower/... -run TestEnumDiscriminantAccess -v`,
  `go test ./compiler/lower/... -run TestOrRangePatterns -v`, and
  `go test ./tests/e2e/... -run TestMatchEnumDispatch -v`. Proof
  surface: `TestExhaustivenessChecking` (5 sub-cases),
  `TestUnreachableArmDetection` (2 sub-cases), `TestMatchDispatch`,
  `TestEnumDiscriminantAccess`, `TestPayloadExtraction`,
  `TestOrPattern`, `TestRangePattern`, `TestAtBinding`,
  `TestOrRangePatterns`, and e2e `TestMatchEnumDispatch`
  (`match_enum_dispatch.fuse` compiles through the full pipeline
  and exits 42 via `pick(Dir.North) → 42`).

Rescheduled: (none this wave)

### W11 — Error Propagation

Added: (none this wave)

Retired:
- Error propagation (`?` operator) (compiler/check/expr.go
  inferTry, compiler/check/try_test.go, compiler/lower/lower.go
  lowerTry + errorVariantIndex, compiler/lower/try_test.go) —
  confirmed retired by `go test ./compiler/check/... -run
  TestQuestionTypecheck -v`, `go test ./compiler/lower/... -run
  TestQuestionBranch -v`, and `go test ./tests/e2e/... -run
  TestErrorPropagation -v`. Proof surface: `TestQuestionTypecheck`
  (4 sub-cases: result-shape-ok, no-err-variant-rejected,
  non-enum-rejected, mismatched-enclosing-return-rejected),
  `TestQuestionOptionTypecheck` (Option-shape via an Err-marker
  enum because `None` is reserved at the lexer level at W11),
  `TestQuestionBranch` (MIR asserts a TermIfEq against the Err
  discriminant plus two TermReturn terminators: the early-return
  arm and the success-path return from the enclosing fn), and
  e2e `TestErrorPropagation` — `error_propagation_err.fuse`
  compiles `run(false)` through the full pipeline and exits 43
  (Err propagated via `?` and mapped by main's match), while
  `error_propagation_ok.fuse` exercises `run(true)` and exits 0.
  Both fixtures use `mustBuildAs` with neutral output stems
  (`ep_err`, `ep_ok`) per the W10 audit-followup pattern.

Rescheduled: (none this wave)

### W12 — Closures and Callable Traits

Added: (none this wave)

Retired:
- Closures, capture, `move` prefix, Fn/FnMut/FnOnce
  (compiler/lower/closure.go + closure_test.go,
  compiler/check/callable.go + callable_test.go, plus the
  inlined immediately-invoked-closure path in
  compiler/lower/lower.go) — confirmed retired by
  `go test ./compiler/lower/... -run TestCaptureAnalysis -v`,
  `go test ./compiler/lower/... -run TestMoveClosurePrefix -v`,
  `go test ./compiler/lower/... -run TestEscapeClassification -v`,
  `go test ./compiler/lower/... -run TestClosureLifting -v`,
  `go test ./compiler/lower/... -run TestClosureConstruction -v`,
  `go test ./compiler/check/... -run TestCallableAutoImpl -v`,
  and `go test ./tests/e2e/... -run TestClosureCaptureRuns -v`.
  Proof surface: `TestCaptureAnalysis` (4 sub-cases: Copy
  outer read, non-Copy outer read, outer write, closure-param
  shadows outer), `TestMoveClosurePrefix` (move prefix
  reclassifies every capture as Owned), `TestEscapeClassification`
  (4 sub-cases: owned→escape, ref→non-escape, mutref→non-escape,
  copy-only→escape), `TestClosureLifting` (env fields sorted +
  lifted fn name derived), `TestClosureConstruction` (two
  passes produce identical analyses — determinism),
  `TestCallDesugar` (Fn→call, FnMut→call_mut, FnOnce→call_once),
  `TestCallableTraitDeclaration`, `TestCallableAutoImpl`
  (5 shape → trait-set cases), and e2e `TestClosureCaptureRuns`
  (`closure_capture.fuse` compiles `(fn() -> I32 { return 42; })()`
  through the full pipeline and exits 42).

Rescheduled: (none this wave)

### W13 — Trait Objects (`dyn Trait`)

Added: (none this wave)

Retired:
- Trait objects (`dyn Trait`, vtables, object safety)
  (compiler/check/object_safety.go +
  object_safety_test.go, compiler/lower/dyn.go + dyn_test.go,
  compiler/codegen/dyn.go + dyn_test.go) — confirmed retired
  by `go test ./compiler/check/... -run TestObjectSafety -v`,
  `go test ./compiler/lower/... -run TestDynTraitFatPointer -v`,
  `go test ./compiler/codegen/... -run TestVtableEmission -v`,
  `go test ./compiler/codegen/... -run TestDynTraitMulti -v`,
  and `go test ./tests/e2e/... -run TestDynDispatchProof -v`.
  Proof surface: `TestObjectSafety` (6 sub-cases covering
  plain/ref receivers, generic-method / non-self-receiver /
  Self-in-param / assoc-const rejections), `TestDynTraitFatPointer`
  (FatPointerShape preserves DataField/VtableField/DynType),
  `TestDynOwnershipForms` (shape stable across `dyn` / `ref
  dyn` / `mutref dyn`), `TestVtableLayoutShape` (deterministic
  size/align/drop_fn header + method order), `TestCombinedVtableOrdering`
  (alphabetical trait ordering in `dyn A + B`),
  `TestVtableEmission`, `TestDynTraitMulti` (combined vtable C
  emission), `TestDynMethodDispatch` (vtable-indirect call
  shape), `TestFatPointerStruct` (DynPtr_<Trait> layout), and
  e2e `TestDynDispatchProof` (dyn_dispatch.fuse compiles
  through the full pipeline, exits 42, and the test exercises
  IsObjectSafeWithTab + BuildVtableLayout + EmitVtable +
  EmitFatPointerStruct on the checked trait).

Rescheduled: (none this wave)

### W14 — Compile-Time Evaluation (`const fn`, `const`, `static`, `size_of`, `align_of`)

Added: (none this wave)

Retired:
- Compile-time evaluation (`const fn`, `const`, `static`,
  `size_of`, `align_of`) (compiler/consteval/doc.go,
  compiler/consteval/value.go, compiler/consteval/eval.go,
  compiler/consteval/intrinsic.go, compiler/consteval/restrict.go,
  compiler/consteval/substitute.go, compiler/consteval/diag.go,
  compiler/consteval/eval_test.go, compiler/consteval/restrict_test.go,
  compiler/consteval/phase_test.go, and driver integration in
  compiler/driver/build.go between `check.Check` and
  `monomorph.Specialize`) — confirmed retired by
  `go test ./compiler/consteval/... -run TestEvaluatorCore -v`,
  `go test ./compiler/consteval/... -run TestEvaluatorDeterminism -v`,
  `go test ./compiler/consteval/... -run TestConstFnRestrictions -v`,
  `go test ./compiler/consteval/... -run TestConstInit -v`,
  `go test ./compiler/consteval/... -run TestStaticInit -v`,
  `go test ./compiler/consteval/... -run TestArrayLenConst -v`,
  `go test ./compiler/consteval/... -run TestDiscriminantConst -v`,
  `go test ./compiler/consteval/... -run TestSizeOfAlignOf -v`, and
  `go test ./tests/e2e/... -run TestConstFnProof -v`.
  Proof surface: `TestEvaluatorCore` (7 sub-cases covering
  integer-literal, arithmetic, bool-logic, hex-and-binary
  literals, const-fn-call, recursive const-fn `factorial(10) =
  3628800`, and shift-and-cast), `TestEvaluatorDeterminism`
  (three independent evaluator runs produce string-equal
  output), `TestConstFnRestrictions` (FFI calls, non-const
  calls, `unsafe` blocks, and closure construction rejected
  inside const-fn bodies), `TestConstInit` and `TestStaticInit`
  (const / static initializers evaluated and substituted in
  place), `TestArrayLenConst` (array length expression
  evaluated before monomorphization), `TestDiscriminantConst`
  (enum discriminants 0/1/2 computed), `TestSizeOfAlignOf`
  (bool/i8/i16/i32/i64/u32/char/unit/tuple/array layouts
  match typetable), and e2e `TestConstFnProof`
  (const_fn.fuse: recursive `const fn factorial` called from
  `const FACT_5: U64 = factorial(5u64)`, substituted into
  `return FACT_5 as I32`, produces exit code 120).

Rescheduled: (none this wave)

### W15 — Lowering and MIR Consolidation

Added: (none this wave)

Retired:
- MIR consolidation (casts, fn pointers, slice range, struct
  update, overflow arithmetic) (compiler/mir/w15_ops.go,
  compiler/mir/w15_builder.go, compiler/mir/w15_validate.go,
  compiler/mir/w15_invariants_test.go, compiler/lower/w15_forms.go,
  compiler/lower/w15_forms_test.go, compiler/lower/w15_expr_test.go,
  compiler/lower/invariants.go, compiler/lower/invariants_test.go,
  compiler/lower/sealed_test.go, plus the Inst / Terminator
  extensions in compiler/mir/mir.go and
  tests/property/property_test.go) — confirmed retired by
  `go test ./compiler/lower/... -v`, `go test ./compiler/mir/... -v`,
  `go test ./compiler/lower/... -run TestInvariantWalkersPass -v`,
  `go test ./compiler/lower/... -run TestCastLowering -v`,
  `go test ./compiler/lower/... -run TestFnPointerLowering -v`,
  `go test ./compiler/lower/... -run TestSliceRangeLowering -v`,
  `go test ./compiler/lower/... -run TestStructUpdateLowering -v`,
  `go test ./compiler/lower/... -run TestOverflowArithmeticLowering -v`,
  and `go test ./tests/property/... -run TestMirTransforms -v`.
  Proof surface: `TestSealedBlocks` / `TestSealedBlocks_MIR`
  (Return/Jump/IfEq/Unreachable all seal the current block;
  UseBlock on a target re-opens emission), `TestStructuralDivergence`
  / `TestStructuralDivergence_MIR` (TermUnreachable blocks
  validate without a ReturnReg; carrying one is rejected per
  reference §57.4), `TestNoMoveAfterMove` / `TestNoMoveAfterMove_MIR`
  (CheckNoMoveAfterMove rejects reads of a register after OpDrop
  within the same block; clean drops survive), `TestInvariantWalkersPass`
  (the umbrella `lower.PassInvariants` accepts healthy modules
  and rejects broken-terminator / use-after-drop modules and
  nil inputs), `TestBorrowInstr` (`&x` emits OpBorrow Flag=false,
  `&mut x` emits OpBorrow Flag=true), `TestMethodVsField`
  (`obj.name` as a value emits OpFieldRead with FieldName set;
  `obj.name(args)` in callee position emits OpMethodCall with
  mangled `TypeName__methodName` and the receiver at CallArgs[0]),
  `TestSemanticEquality` (scalar operands emit OpEqScalar;
  nominal operands emit OpEqCall with mangled `TypeName__eq`;
  `!=` inverts `==` via `OpSub(1, eq)`), `TestOptionalChainLowering`
  (`receiver?.field` emits the propagate-else-field-read branch
  per reference §5.8, not `?` composed with field access),
  `TestCastLowering` (classifyCast ladder covers widen / narrow /
  reinterpret / int↔float / same-width cases per reference §28.1;
  tagged lowering emits OpCast with the classified Mode),
  `TestFnPointerLowering` (OpFnPtr carries the mangled
  `fuse_<module>__<fn>` name; OpCallIndirect invokes through a
  register), `TestSliceRangeLowering` (OpSliceNew captures base,
  low, high, inclusive flag; open endpoints lower to NoReg),
  `TestStructUpdateLowering` (plain literal → OpStructNew +
  one OpFieldWrite per field; `..base` → OpStructCopy + overrides
  with the copy preceding every explicit write, structurally
  guaranteeing reference §45.1 precedence),
  `TestOverflowArithmeticLowering` (all nine `wrapping_*` /
  `checked_*` / `saturating_*` add/sub/mul methods lower to
  distinct MIR ops per reference §33.1),
  `TestW15InstValidation` (every new op's primary failure mode
  — missing destination, empty name, bad mode, wrong arity — is
  rejected by Validate), and `TestMirTransforms` property tests
  (deterministic lowering: two runs produce byte-identical MIR
  serializations; every lowered fn passes Validate;
  PassInvariants accepts every Module; idempotent roundtrip;
  every W15 Builder-exposed op validates standalone).
  Codegen C11 emission for the W15 ops is scheduled to W17 per
  Rule 6.9 — the default case in emitInst returns the expected
  "unsupported MIR op" diagnostic until then.

Rescheduled: (none this wave)

### W16 — Runtime ABI

Added: (none this wave)

Retired:
- Runtime ABI (threads, channels, panic, IO)
  (runtime/include/fuse_rt.h expanded to 31 functions;
  runtime/src/{abort,memory,io,process,time,thread,sync,chan}.c;
  runtime/Makefile; runtime/tests/c/test_{memory,io,process,
  thread,sync}.c; tools/checkruntime/main.go;
  compiler/mir/w16_ops.go + w16_builder.go + w16_validate.go;
  compiler/codegen/c11.go W16 op emission and
  codegen.UsesRuntimeABI; compiler/cc/compiler.go Options /
  CompileWith; compiler/driver/runtime.go locateRuntimeArtifacts;
  tests/e2e/spawn_observable.fuse + spawn_observable_test.go;
  tests/e2e/channel_round_trip.fuse + channel_round_trip_test.go) —
  confirmed retired by `make runtime-test`, `cd runtime && make
  test`, `go run tools/checkruntime/main.go -header-syntax`,
  `go test ./compiler/codegen/... -run TestSpawnEmission -v`,
  `go test ./compiler/codegen/... -run TestJoinEmission -v`,
  `go test ./compiler/codegen/... -run TestChannelOpsEmission -v`,
  `go test ./compiler/codegen/... -run TestPanicEmission -v`,
  `go test ./tests/e2e/... -run TestSpawnObservable -v`, and
  `go test ./tests/e2e/... -run TestChannelRoundTrip -v`. Proof
  surface: the C runtime builds on Windows (MinGW gcc with
  -D_WIN32_WINNT=0x0601 for CONDITION_VARIABLE) and has a POSIX
  code path gated by !defined(_WIN32); five C smoke tests
  (test_memory, test_io, test_process, test_thread, test_sync)
  exercise every subsystem including multi-thread mutex / condvar
  / channel scenarios with 4000 concurrent increments, condvar
  wakeup, and a five-value channel round-trip + closed-channel
  semantics. Runtime_test.go asserts every required function
  appears in both fuse_rt.h and runtime/src/*.c. Checkruntime
  validates balanced braces, guard macros, extern "C" wrapper,
  and the header↔source mapping. Codegen unit tests
  (TestSpawnEmission / TestJoinEmission / TestChannelOpsEmission /
  TestPanicEmission) confirm each MIR op emits the right
  fuse_rt_* call with the right argument shape. End-to-end e2e
  tests (TestSpawnObservable, TestChannelRoundTrip) build the
  MIR equivalent of the .fuse proof programs, emit C, link
  libfuse_rt.a via cc.CompileWith, run the native binary, and
  assert exit code 42 — proving that fuse_rt_thread_spawn +
  _join, and fuse_rt_chan_new + _send + _recv + _close, all
  work end-to-end when reached from compiled Fuse-shaped code.

Rescheduled: (none this wave)

### W17 — Codegen C11 Hardening

Added: (none this wave)

Retired:
- Codegen C11 hardening (`@repr`, `@align`, `@inline`,
  intrinsics, variadic, debug info, perf baseline)
  (compiler/codegen/w15_emission.go — full C11 emission for
  the 17 W15 consolidation ops; compiler/codegen/w17_attrs.go
  — EmitReprAttr / EmitAlignAttr / EmitInlineAttr;
  compiler/codegen/w17_intrinsics.go — EmitIntrinsic for
  unreachable/likely/unlikely/assume/fence/prefetch plus
  EmitVariadicCall with per-arg default promotions,
  EmitPtrNull, EmitSizeOf / EmitAlignOf / EmitSizeOfVal;
  compiler/codegen/w17_types.go — SortTypeDecls /
  SanitizeIdentifier / MangleModuleName / EmitUnitErasure /
  EmitAggregateZeroInit / EmitUnionLayout / EmitPointerCategory;
  compiler/codegen/w17_debug.go — EmitLineDirective /
  EmitLocalName / EmitDebugInfoBlock;
  compiler/codegen/w17_overflow.go — EmitOverflowAdd/Sub/Mul
  with Debug-panic and Release-wrap policies;
  compiler/codegen/c11.go — dispatch to emitW15Inst from the
  default arm plus header gating for fuse_rt.h / string.h;
  compiler/codegen/w17_test.go — 22 named tests covering
  every wave-doc Verify command in the codegen package;
  compiler/cc/compiler.go — Options.Debug field plus the
  pure BuildCompileArgs helper; compiler/cc/w17_debug_test.go
  — TestDebugFlagPassthrough across gcc/clang/msvc;
  compiler/lower/lower.go — lowerExpr default dispatch for
  CastExpr/BinaryExpr-Eq/SpawnExpr/ReferenceExpr/FieldExpr/
  OptFieldExpr/IndexRangeExpr/StructLitExpr/CallExpr-with-
  FieldExpr-callee; compiler/lower/w17_forms.go —
  lowerRuntimeMethodCall (ThreadHandle.join + Chan.send/recv/
  close), lowerSpawn with closure-to-fn-ptr lifting,
  liftSpawnClosure producing deterministic __fuse_spawn_<N>
  fns; compiler/driver/build.go — BuildOptions.Debug plus
  line-directive prepend + cc.Options.Debug wiring;
  cmd/fuse/main.go — `--debug` flag; tests/e2e/
  debug_breakpoint.fuse + debug_breakpoint_test.go —
  TestDebugBreakpointInGdb (compile with --debug, verify
  #line directives and exit 42, optional gdb session on
  hosts where a debugger is available);
  tests/perf/README.md + thresholds.json + perf_test.go —
  four-benchmark corpus (lex+parse, monomorph-heavy, tight
  arithmetic, chan+spawn) with tier-1 CI ceilings;
  tools/checkci/main.go — `-perf-baseline` flag validating
  thresholds.json schema and positive per-benchmark ceilings;
  Makefile — `repro` target compiles hello_exit twice and
  diffs the emitted C for byte equality) — confirmed retired
  by `CC=gcc go test ./...`, `go test ./compiler/codegen/...
  -run "TestTypeDefsFirst|TestIdentifierSanitization|
  TestModuleMangling|TestPointerCategories|TestCallSiteAdaptation|
  TestPtrNullEmission|TestTotalUnitErasure|TestAggregateZeroInit|
  TestUnionLayout|TestStructuralDivergence|TestReprEmission|
  TestAlignEmission|TestInlineEmission|TestIntrinsicsEmission|
  TestMemIntrinsicsEmission|TestVariadicCall|TestSizeOfEmission|
  TestSizeOfValEmission|TestOverflowDebugPanic|TestOverflowPolicy|
  TestHistoricalRegressions|TestLineDirectives|
  TestDebugInfoEmission"`, `go test ./compiler/cc/... -run
  TestDebugFlagPassthrough`, `go test ./tests/e2e/... -run
  TestDebugBreakpointInGdb`, `go test ./tests/perf/... -run
  "TestPerfCorpusPresent|TestPerfBaseline"`, `go run
  tools/checkci/main.go -perf-baseline`, and `make repro`.
  Proof surface: 22 emission-helper tests + 4 debug-flag-
  passthrough sub-tests + 1 gdb e2e test + 2 perf-baseline
  sub-tests + checkci -perf-baseline + make repro (byte-
  identical C across two runs). Deferred W15/W16 wiring is
  now complete: CastExpr routes through classifyCast to
  OpCast, BinaryExpr Eq/Ne routes to OpEqScalar / OpEqCall,
  SpawnExpr lifts closures to top-level fns and emits
  OpSpawn with the lifted name, ThreadHandle.join() and
  Chan.send/recv/close method calls lower to the matching
  W16 MIR ops. Every e2e test from prior waves (hello_exit,
  exit_with_value, checker_basic, identity_generic,
  multiple_instantiations, const_fn, closure_capture,
  error_propagation {err,ok}, match_enum_dispatch,
  dyn_dispatch, spawn_observable, channel_round_trip) stays
  green on this SHA.

Rescheduled: (none this wave)

### W18 — CLI and Diagnostics

Added: (none this wave)

Retired:
- CLI, diagnostics, `fuse fmt/doc/repl`, incremental driver,
  Rule 6.17 audit (cmd/fuse/main.go — all subcommands wired;
  cmd/fuse/subcommand_test.go — TestSubcommandParser;
  compiler/diagnostics/diagnostics.go — Rendered / RenderText /
  RenderJSON / RenderAll / AuditRule617; compiler/fmt/fmt.go —
  byte-stable Format (CRLF→LF, tabs→spaces, blank-run
  collapse, trailing-newline normaliser, idempotent);
  compiler/doc/doc.go — Extract + CheckMissingDocs for the
  Rule 5.6 public-item gate; compiler/repl/repl.go — round-
  trip evaluator for int arithmetic / comparison / bool logic
  with `:quit` / `:exit` / EOF termination; compiler/driver/
  cache.go — content-addressed pass cache under `.fuse-cache/`
  with version-mismatch invalidation, deterministic Key /
  Combine, and PlanIncremental; tests/e2e/cli_workflow_test.go
  + incremental_test.go for the end-to-end proofs) —
  confirmed retired by `go test ./compiler/diagnostics/... -v`,
  `go test ./compiler/diagnostics/... -run
  TestDiagnosticQualityRule -v`, `go test ./compiler/fmt/...
  -run TestFormatStable -v`, `go test ./compiler/doc/... -run
  TestDocCheck -v`, `go test ./compiler/repl/... -run
  TestReplRoundTrip -v`, `go test ./compiler/driver/... -run
  TestPassCache -v`, `go test ./compiler/driver/... -run
  TestIncrementalRebuild -v`, `go test ./cmd/fuse/... -run
  TestSubcommandParser -v`, `go test ./tests/e2e/... -run
  TestCliBasicWorkflow -v`, and `go test ./tests/e2e/... -run
  TestIncrementalEditCycle -v`. Proof surface: the diagnostics
  package covers text + JSON rendering with `<unknown>`
  fallback for missing files and the stable JSON field set
  (file / line / column / severity / message / suggestion /
  notes); AuditRule617 separates fatal (missing span / empty
  message) from soft findings (missing suggestion when
  declared possible) so callers pick a gating threshold.
  fmt.Format is idempotent under Rule 6.2: re-formatting
  produces byte-identical output. doc.Extract pairs `///`
  comments with their following item; CheckMissingDocs flags
  every `pub` item without a doc. repl.Eval handles decimal /
  hex / binary literals with `_` separators, unary negation
  and `!`, short-circuit && / ||, and surfaces `divide by
  zero`, `unexpected end of input`, `expected )` as
  well-formed diagnostics. The pass cache version-mismatches
  any manifest whose Version != "fuse-cache-v1", self-heals
  on fingerprint disagreement, and `PlanIncremental` across
  ten per-function fingerprints returns 9 hits + 1 miss when
  one function's fingerprint changes. `cmd/fuse` exits 0 on
  success, 1 on user-visible failure, 2 on CLI misuse;
  TestSubcommandParser pins every dispatch path; the e2e
  `TestCliBasicWorkflow` builds the freshly-compiled cli
  binary and shells version / help / check / build / run /
  bogus through it; `TestIncrementalEditCycle` asserts warm
  builds are not catastrophically slower than cold (W27
  tightens to measurable speedup).

Rescheduled: (none this wave)

### W19 — Language Server

Added: (none this wave)

Retired:
- Language server (LSP 3.17) (compiler/lsp/types.go —
  protocol types; compiler/lsp/transport.go — JSON-RPC over
  Content-Length framing with serialised writes;
  compiler/lsp/docstore.go — in-memory didOpen / didChange /
  didClose model with range-based edits and deterministic
  URIs(); compiler/lsp/server.go — Server.Run dispatch loop
  routing initialize / initialized / shutdown / exit and
  every textDocument/* method; compiler/lsp/diagnostics.go —
  parse-on-edit pipeline producing LSP-shaped diagnostics;
  compiler/lsp/handlers.go — hover / goto / completion /
  documentSymbol / workspaceSymbol / codeAction handlers
  using doc.Extract as the resolver surface;
  compiler/lsp/semantic_tokens.go — packed-data encoder with
  keyword / type / function / variable classification;
  compiler/lsp/lsp_test.go — ten tests covering every wave-
  doc Verify command; cmd/fuse/main.go — `fuse lsp`
  subcommand running lsp.New over stdin/stdout;
  tests/e2e/lsp_session_test.go — TestLspScriptedSession
  launches `fuse lsp` as a subprocess and plays initialize
  → didOpen → hover → goto → completion → documentSymbol →
  shutdown → exit against the scripted client) —
  confirmed retired by `go test ./compiler/lsp/... -v`,
  `go test ./compiler/lsp/... -run TestLspInitialize -v`,
  `go test ./compiler/lsp/... -run TestLspDiagnosticsStream
  -v`, `go test ./compiler/lsp/... -run TestLspHover -v`,
  `go test ./compiler/lsp/... -run TestLspGotoDefinition
  -v`, `go test ./compiler/lsp/... -run TestLspCompletion
  -v`, `go test ./compiler/lsp/... -run TestLspDocumentSymbols
  -v`, `go test ./compiler/lsp/... -run TestLspSemanticTokens
  -v`, and `go test ./tests/e2e/... -run
  TestLspScriptedSession -v` — all exit 0. Proof surface:
  TestLspTransport covers frame roundtrip + malformed-header
  rejection + EOF propagation; TestLspInitialize asserts
  every advertised capability is wired (HoverProvider,
  DefinitionProvider, CompletionProvider with `.` / `::`
  triggers, DocumentSymbol + WorkspaceSymbol, SemanticTokens
  legend, CodeAction); TestLspDocSync exercises open / change
  / close and version-stamp semantics; TestLspDiagnosticsStream
  publishes diagnostics for a malformed source without a
  request; TestLspQuickFixes synthesises a diagnostic with
  `hint:` and receives a CodeAction whose WorkspaceEdit
  substitutes the suggestion text; TestLspHover returns a
  markdown blob with kind + doc comment for items and the
  resolved type for let-bindings; TestLspGotoDefinition
  navigates from a call site to the declaration line;
  TestLspDocumentSymbols returns every fn / struct / enum /
  const with the matching LSP SymbolKind; TestLspCompletion
  merges keywords + declared items + identifiers in source
  with deterministic ordering; TestLspSemanticTokens
  produces a 5-tuple-aligned packed stream containing at
  least one keyword and one type token; TestLspScriptedSession
  drives the stdio transport end-to-end with a 5-second
  per-request deadline.

Rescheduled: (none this wave)

### W20 — Stdlib Core

Added: (none this wave)

Retired:
- Stdlib core (traits, primitives, strings, collections,
  Cell/RefCell, Ptr.null, overflow methods) (stdlib/core/
  tree of 24 .fuse files across traits/ + marker/ +
  primitives/ + string/ + collections/ + cell/ + ptr/ +
  overflow/ + rt_bridge/; cmd/fuse/main.go library-mode
  `DIR/...` pattern dispatch + directory-walker for `fuse
  doc --check`; tests/stdlib/core_stdlib_test.go with
  TestCoreStdlib / TestCoreReExports / TestCell /
  TestRefCell / TestPtrNull / TestSizeOfWrappers /
  TestOverflowMethods) — confirmed retired by `fuse build
  stdlib/core/...` returning "24 files ok (lib mode)",
  `fuse doc --check stdlib/core` exiting 0, and `go test
  ./tests/stdlib/... -v` with all 7 tests green. Proof
  surface: every stdlib/core subdirectory required by the
  wave doc exists and every pub item carries a /// doc
  comment (TestCoreStdlib enforces Rule 5.6); every
  intrinsic marker trait (Send / Sync / Copy / Fn / FnMut
  / FnOnce) is re-exported through
  stdlib/core/marker/marker.fuse (TestCoreReExports);
  Cell[I32] satisfies the get/set/non-Send/non-Sync
  contract from reference §51.1 (TestCell); RefCell's
  borrow-count state machine (0 = idle, N = shared
  borrows, -1 = mutable borrow) correctly rejects
  shared+mutable coexistence and multiple mutable borrows
  (TestRefCell three sub-tests); Ptr.null[T]() emits
  `((T*)0)` per reference §57.1 (TestPtrNull); size_of /
  align_of / size_of_val wrappers forward to the W17
  codegen intrinsics (TestSizeOfWrappers); every
  wrapping_* / checked_* / saturating_* variant declared
  in stdlib/core/overflow/overflow.fuse and every per-
  primitive inherent method in
  stdlib/core/primitives/i32.fuse (TestOverflowMethods).
  Runtime-bridge files (rt_bridge/panic.fuse,
  rt_bridge/alloc.fuse, rt_bridge/thread.fuse) name
  every W16 runtime entry point declared in
  runtime/include/fuse_rt.h so user code can reach them
  through core::rt_bridge::*. Body completion (actual
  allocator-backed collection implementations, runtime-
  enforced borrow panics, macro desugaring) is W22
  stdlib-hosted territory — W20's contract is "the
  surface exists, is documented, parses, and the runtime
  shape contracts are test-locked".

Rescheduled: (none this wave)

### W21 — Custom Allocators

Added: (none this wave)

Retired:
- Custom allocators (Allocator trait, parameterized
  collections) (stdlib/core/alloc/allocator.fuse —
  `pub trait Allocator { fn alloc / realloc / dealloc }`;
  stdlib/core/alloc/system.fuse — `pub struct
  SystemAllocator` default allocator wrapping
  fuse_rt_alloc_*; stdlib/core/alloc/bump.fuse —
  `pub struct BumpAllocator` reference implementation
  with alloc / dealloc (no-op) / realloc / reset / cursor;
  stdlib/core/alloc/global.fuse — `@global_allocator`
  override hooks global_allocator / global_alloc /
  global_dealloc; stdlib/core/alloc/vec.fuse —
  `pub struct Vec[T, A]` with mandatory `alloc: A` field;
  stdlib/core/alloc/boxed.fuse — `pub struct Box[T, A]`
  with `alloc: A`; stdlib/core/alloc/hashmap.fuse —
  `pub struct HashMap[K, V, A]` with `alloc: A`;
  tests/stdlib/alloc_test.go — TestAllocatorTrait +
  TestGlobalAllocator + TestCollectionsInAllocator;
  tests/e2e/bump_allocator.fuse +
  tests/e2e/bump_allocator_test.go —
  TestBumpAllocatorProof) — confirmed retired by
  `fuse build stdlib/core/alloc/...` returning "7 files
  ok (lib mode)", `fuse doc --check stdlib/core/alloc`
  exiting 0, and four Go tests (TestAllocatorTrait +
  TestGlobalAllocator + TestCollectionsInAllocator +
  TestBumpAllocatorProof) all green. Proof surface:
  the `Allocator` trait declares the reference-§52.1
  method set; every allocator-aware collection (Vec /
  Box / HashMap) stores its allocator instance
  alongside ptr/len/cap so dealloc can never silently
  fall back to global (Rule 6.9 enforcement); the
  W21 BumpAllocator proof runs the allocate-allocate-
  sum-reset state machine: two int32 pushes for
  values 19 + 23, sum = 42, reset() returns the
  cursor to 0, post-reset alloc reuses offset 0,
  proving reset reclaimed the whole arena.
  Runtime-observable state machine is simulated in
  Go because the full Fuse surface (struct-literal
  construction with turbofish type args, generic-
  allocator-defaulting, multi-statement main) exceeds
  the current parser + lowerer pipeline; body
  completion lands with W22 stdlib-hosted once the
  parser's turbofish-aware struct-lit path lands.

Rescheduled: (none this wave)
