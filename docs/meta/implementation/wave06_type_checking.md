# Wave 06: Type Checking

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: build a checker that fully types user code and stdlib bodies without
leaving unknown metadata for later passes. This wave covers every
type-system feature whose semantics live in the checker: primitive types,
structs (including newtype `struct T(U);` form), enums (including unions
as distinct declaration form), traits (with coherence), generics (with
parameter scoping), associated types, casts, visibility, function pointer
types, `impl Trait` parameter-position, variadic extern signatures,
`@repr`/`@align` annotations on types.

Pillar alignment:

- Memory safety: the checker must reject invalid ownership-relevant and
  layout-relevant type constructions before they reach lowering or codegen.
- Concurrency safety: the type system foundations required by later
  concurrency rules must be explicit and precise rather than inferred by later
  passes.
- Developer experience: type errors, trait failures, and stdlib-checking
  failures must surface at the checker boundary with concrete diagnostics rather
  than leaking into later compiler stages.

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

## Phase 00: Stub Audit [W06-P00-STUB-AUDIT]

- Task 01: Check audit [W06-P00-T01-CHECK-AUDIT]
  Currently: `compiler/check/` is entering service as a new wave owner and must
  not inherit overdue stubs silently.
  Expected deliveries: audited checker-related stub inventory and committed wave
  entry state.
  Verify: `go run tools/checkstubs/main.go -wave W06 -phase P00`

## Phase 01: Function Signatures and Two-Pass Scheduling [W06-P01-FN-SIGNATURES]

- Task 01: Index all function signatures [W06-P01-T01-INDEX]
  Currently: the checker has no stable signature graph yet.
  Expected deliveries: pre-body function signature registration for every item
  kind the checker owns.
  Verify: `go test ./compiler/check/... -run TestFunctionTypeRegistration -v`
- Task 02: Two-pass checker [W06-P01-T02-TWO-PASS]
  Currently: body checking and signature registration are not yet separated.
  Expected deliveries: pass ordering that checks signatures before bodies and
  leaves no body-dependent signature gaps.
  Verify: `go test ./compiler/check/... -run TestTwoPassChecker -v`

## Phase 02: Nominal Types and Cast/Widen Rules [W06-P02-NOMINAL-CASTS]

- Task 01: Nominal equality [W06-P02-T01-NOMINAL-EQ]
  Currently: nominal type identity is not yet enforced as a checker contract.
  Expected deliveries: nominal equality rules for declared types.
  Verify: `go test ./compiler/check/... -run TestNominalEquality -v`
- Task 02: Primitive method registration [W06-P02-T02-PRIM-METHODS]
  Currently: primitive methods can still be treated as ad hoc builtin behavior.
  Expected deliveries: primitive method lookup wired through the checker.
  Verify: `go test ./compiler/check/... -run TestPrimitiveMethods -v`
- Task 03: Numeric widening [W06-P02-T03-WIDENING]
  Currently: widening rules are specified but not yet enforced.
  Expected deliveries: explicit numeric widening checks.
  Verify: `go test ./compiler/check/... -run TestNumericWidening -v`
- Task 04: Cast semantics [W06-P02-T04-CASTS]
  Currently: cast legality remains unresolved at checker entry.
  Expected deliveries: checker-side `as` cast validation.
  Verify: `go test ./compiler/check/... -run TestCastSemantics -v`

## Phase 03: Trait Lookup and Bound Chains [W06-P03-TRAIT-LOOKUP]

- Task 01: Concrete trait method lookup [W06-P03-T01-CONCRETE]
  Currently: concrete trait method lookup is not yet owned by a distinct checker
  completion step.
  Expected deliveries: method lookup through concrete impls.
  Verify: `go test ./compiler/check/... -run TestConcreteTraitMethodLookup -v`
- Task 02: Bound-chain lookup [W06-P03-T02-BOUND-CHAIN]
  Currently: supertrait and bound-chain resolution is still implicit.
  Expected deliveries: trait-bound lookup through the declared bound chain.
  Verify: `go test ./compiler/check/... -run TestBoundChainLookup -v`
- Task 04: Trait-typed parameters [W06-P03-T04-TRAIT-PARAMS]
  Currently: trait-typed parameter checking can drift into later generic work if
  it is not retired here.
  Expected deliveries: parameter checking for trait-typed positions.
  Verify: `go test ./compiler/check/... -run TestTraitParameters -v`

## Phase 04: Coherence and Associated Type Projection [W06-P04-COHERENCE-ASSOC]

- Task 01: Coherence and orphan rules [W06-P04-T01-COHERENCE]
  Currently: impl coherence is still only specified, not enforced.
  Expected deliveries: coherence and orphan-rule enforcement.
  Verify: `go test ./compiler/check/... -run TestCoherenceOrphan -v`
- Task 02: Associated type projection [W06-P04-T02-ASSOC-PROJECT]
  Currently: associated type projection remains unresolved in the checker.
  Expected deliveries: associated type projection over resolved traits.
  Verify: `go test ./compiler/check/... -run TestAssocTypeProjection -v`
- Task 03: Associated type constraints [W06-P04-T03-ASSOC-CONSTRAINTS]
  Currently: associated type constraints can still be deferred implicitly.
  Expected deliveries: checker enforcement for associated type constraints.
  Verify: `go test ./compiler/check/... -run TestAssocTypeConstraints -v`

## Phase 05: Contextual Inference and Literal Typing [W06-P05-INFERENCE]

- Task 01: Expected-type inference [W06-P05-T01-EXPECTED]
  Currently: contextual inference is not yet closed as a checker responsibility.
  Expected deliveries: expected-type inference at expression sites.
  Verify: `go test ./compiler/check/... -run TestContextualInference -v`
- Task 02: Zero-arg generic calls [W06-P05-T02-ZERO-ARG]
  Currently: zero-argument generic call resolution can still leak into later
  generic-specialization work.
  Expected deliveries: checker support for zero-arg generic call inference.
  Verify: `go test ./compiler/check/... -run TestZeroArgTypeArgs -v`
- Task 03: Literal typing [W06-P05-T03-LIT-TYPING]
  Currently: literal typing rules are not yet sealed at the checker boundary.
  Expected deliveries: literal typing for the surface forms owned by W06.
  Verify: `go test ./compiler/check/... -run TestLiteralTyping -v`

## Phase 06: Function Pointer Types [W06-P06-FN-POINTERS]

- Task 01: Function pointer types [W06-P06-T01-FN-PTR-TYPE]
  Currently: function pointer types are specified but not yet isolated as their
  own completion boundary.
  Expected deliveries: first-class `fn(A, B) -> R` type checking.
  DoD: `fn(A, B) -> R` is a first-class type. Nominal identity is by
  signature.
  Verify: `go test ./compiler/check/... -run TestFnPointerType -v`

## Phase 07: `impl Trait` Parameter and Return Positions [W06-P07-IMPL-TRAIT]

- Task 01: `impl Trait` parameter-position [W06-P07-T01-IMPL-TRAIT-PARAM]
  Currently: parameter-position `impl Trait` is at risk of drifting into generic
  lowering work if it is not retired in the checker.
  Expected deliveries: checker desugaring and validation for parameter-position
  `impl Trait`.
  DoD: `fn f(x: impl Trait)` desugars to `fn f[T: Trait](x: T)` at check
  time.
  Verify: `go test ./compiler/check/... -run TestImplTraitParam -v`
- Task 02: `impl Trait` return-position [W06-P07-T02-IMPL-TRAIT-RETURN]
  Currently: opaque return checking is specified, but not yet separated from
  function pointer work.
  Expected deliveries: checker-side single-concrete-type enforcement for return
  `impl Trait`.
  DoD: checker records a single concrete return type per `impl Trait`
  return signature; multi-type paths are a diagnostic.
  Verify: `go test ./compiler/check/... -run TestImplTraitReturn -v`

## Phase 08: Unions, Newtypes, and Layout Annotations [W06-P08-LAYOUT-FORMS]

- Task 01: Union declaration check [W06-P08-T01-UNION-CHECK]
  Currently: unions remain an unchecked declaration form on entry.
  Expected deliveries: checker validation for union declarations.
  DoD: `union U { a: T1, b: T2 }` checks field types; union fields may not
  implement `Drop`.
  Verify: `go test ./compiler/check/... -run TestUnionCheck -v`
- Task 02: Newtype pattern [W06-P08-T02-NEWTYPE]
  Currently: newtype checking can still drift into later lowering work if not
  retired as a type-system contract here.
  Expected deliveries: checker validation for single-field tuple-struct
  newtypes.
  DoD: `struct U(T);` is distinct from `T`; methods can be attached.
  Verify: `go test ./compiler/check/... -run TestNewtypePattern -v`
- Task 03: Repr and align annotation validation [W06-P08-T03-REPR-ALIGN]
  Currently: layout annotations remain only specified at entry.
  Expected deliveries: checker validation for repr/align annotations.
  DoD: `@repr(C)`, `@repr(packed)`, `@repr(Uxx|Ixx)`, `@align(N)` validated
  against target type; conflicting annotations produce diagnostics.
  Verify: `go test ./compiler/check/... -run TestReprAnnotationCheck -v`

## Phase 09: Variadic Extern Signatures [W06-P09-VARIADIC]

- Task 01: Variadic extern signatures [W06-P09-T01-VARIADIC-SIG]
  Currently: variadic extern signatures remain unchecked at entry.
  Expected deliveries: checker validation for variadic extern declarations and
  rejection of variadic Fuse functions.
  DoD: `extern fn printf(fmt: Ptr[U8], ...) -> I32;` checks; variadic Fuse
  functions are a diagnostic.
  Verify: `go test ./compiler/check/... -run TestVariadicExternCheck -v`

## Phase 10: Stdlib Body Checking and Checker Regression Matrix [W06-P10-STDLIB]

- Task 01: No stdlib body skips [W06-P10-T01-NO-SKIPS]
  Currently: stdlib body checking can still be skipped while later passes depend
  on checked bodies.
  Expected deliveries: stdlib bodies checked in the same pass as user modules.
  Verify: `go test ./compiler/check/... -run TestStdlibBodyChecking -v`
- Task 02: Checker regression corpus [W06-P10-T02-REGRESSIONS]
  Currently: checker regressions still lack an explicit closure task.
  Expected deliveries: regression matrix covering the user-visible checker
  surface retired by W06.
  Verify: `go test ./compiler/check/... -v`

## Phase 11: Checker E2E Proof [W06-P11-PROOF]

- Task 01: `checker_basic.fuse` [W06-P11-T01-PROOF]
  Currently: unit-level checker tests are not enough to prove user-visible
  success.
  Expected deliveries: end-to-end checker proof program.
  Verify: `go test ./tests/e2e/... -run TestCheckerBasicProof -v`

## Wave Closure Phase [W06-PCL-WAVE-CLOSURE]

- Task 01: Retire check stubs [W06-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W06`
- Task 02: WC006 entry [W06-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC006" docs/learning-log.md`

