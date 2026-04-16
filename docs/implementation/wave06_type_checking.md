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
  Verify: `go run tools/checkstubs/main.go -wave W06 -phase P00`

## Phase 01: Function Type Registration [W06-P01-FN-TYPES]

- Task 01: Index all function signatures [W06-P01-T01-INDEX]
  Verify: `go test ./compiler/check/... -run TestFunctionTypeRegistration -v`
- Task 02: Two-pass checker [W06-P01-T02-TWO-PASS]
  Verify: `go test ./compiler/check/... -run TestTwoPassChecker -v`

## Phase 02: Nominal Identity, Primitives, Casts [W06-P02-NOMINAL]

- Task 01: Nominal equality [W06-P02-T01-NOMINAL-EQ]
  Verify: `go test ./compiler/check/... -run TestNominalEquality -v`
- Task 02: Primitive method registration [W06-P02-T02-PRIM-METHODS]
  Verify: `go test ./compiler/check/... -run TestPrimitiveMethods -v`
- Task 03: Numeric widening [W06-P02-T03-WIDENING]
  Verify: `go test ./compiler/check/... -run TestNumericWidening -v`
- Task 04: Cast semantics [W06-P02-T04-CASTS]
  Verify: `go test ./compiler/check/... -run TestCastSemantics -v`

## Phase 03: Trait Resolution [W06-P03-TRAITS]

- Task 01: Concrete trait method lookup [W06-P03-T01-CONCRETE]
  Verify: `go test ./compiler/check/... -run TestConcreteTraitMethodLookup -v`
- Task 02: Bound-chain lookup [W06-P03-T02-BOUND-CHAIN]
  Verify: `go test ./compiler/check/... -run TestBoundChainLookup -v`
- Task 03: Coherence and orphan rules [W06-P03-T03-COHERENCE]
  Verify: `go test ./compiler/check/... -run TestCoherenceOrphan -v`
- Task 04: Trait-typed parameters [W06-P03-T04-TRAIT-PARAMS]
  Verify: `go test ./compiler/check/... -run TestTraitParameters -v`

## Phase 04: Contextual Inference and Literals [W06-P04-INFERENCE]

- Task 01: Expected-type inference [W06-P04-T01-EXPECTED]
  Verify: `go test ./compiler/check/... -run TestContextualInference -v`
- Task 02: Zero-arg generic calls [W06-P04-T02-ZERO-ARG]
  Verify: `go test ./compiler/check/... -run TestZeroArgTypeArgs -v`
- Task 03: Literal typing [W06-P04-T03-LIT-TYPING]
  Verify: `go test ./compiler/check/... -run TestLiteralTyping -v`

## Phase 05: Associated Types [W06-P05-ASSOC-TYPES]

- Task 01: Associated type projection [W06-P05-T01-PROJECT]
  Verify: `go test ./compiler/check/... -run TestAssocTypeProjection -v`
- Task 02: Associated type constraints [W06-P05-T02-CONSTRAINTS]
  Verify: `go test ./compiler/check/... -run TestAssocTypeConstraints -v`

## Phase 06: Function Pointer and `impl Trait` Types [W06-P06-FN-IMPL]

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

## Phase 07: Unions, Newtype, Repr Annotations [W06-P07-UNIONS-REPR]

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

## Phase 08: Stdlib-Shape Body Checking [W06-P08-STDLIB-SHAPE]

- Task 01: No stdlib body skips [W06-P08-T01-NO-SKIPS]
  Verify: `go test ./compiler/check/... -run TestStdlibBodyChecking -v`
- Task 02: Checker regression corpus [W06-P08-T02-REGRESSIONS]
  Verify: `go test ./compiler/check/... -v`

## Phase 09: Checker E2E Proof [W06-P09-PROOF]

- Task 01: `checker_basic.fuse` [W06-P09-T01-PROOF]
  Verify: `go test ./tests/e2e/... -run TestCheckerBasicProof -v`

## Wave Closure Phase [W06-PCL-WAVE-CLOSURE]

- Task 01: Retire check stubs [W06-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W06`
- Task 02: WC006 entry [W06-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC006" docs/learning-log.md`

