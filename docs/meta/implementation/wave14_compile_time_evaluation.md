# Wave 14: Compile-Time Evaluation (`const fn`)

> Part of the [Fuse implementation plan](../implementation-plan.md).


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

## Phase 00: Stub Audit [W14-P00-STUB-AUDIT]

- Task 01: Const evaluator audit [W14-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W14 -phase P00`

## Phase 01: Const Evaluator [W14-P01-EVALUATOR]

- Task 01: Evaluator over checked HIR [W14-P01-T01-EVAL]
  DoD: evaluator supports arithmetic, comparison, bitwise ops, if/loop,
  `match`, struct/tuple construction and destructuring, array indexing
  with constant indices, and recursive calls to other `const fn`.
  Verify: `go test ./compiler/consteval/... -run TestEvaluatorCore -v`
- Task 02: Determinism [W14-P01-T02-DETERMINISM]
  DoD: three runs produce byte-identical results.
  Verify: `go test ./compiler/consteval/... -run TestEvaluatorDeterminism -count=3 -v`

## Phase 02: `const fn` Restrictions [W14-P02-RESTRICTIONS]

- Task 01: Reject non-const operations [W14-P02-T01-RESTRICT]
  DoD: FFI, allocation, thread ops, non-const calls, and interior mutability
  in a `const fn` body produce a diagnostic.
  Verify: `go test ./compiler/consteval/... -run TestConstFnRestrictions -v`

## Phase 03: Const Contexts [W14-P03-CONTEXTS]

- Task 01: `const` initializers [W14-P03-T01-CONST-INIT]
  Verify: `go test ./compiler/consteval/... -run TestConstInit -v`
- Task 02: `static` initializers [W14-P03-T02-STATIC-INIT]
  Verify: `go test ./compiler/consteval/... -run TestStaticInit -v`
- Task 03: Array length expressions [W14-P03-T03-ARRAY-LEN]
  Verify: `go test ./compiler/consteval/... -run TestArrayLenConst -v`
- Task 04: Enum discriminant values [W14-P03-T04-DISCRIMINANT]
  Verify: `go test ./compiler/consteval/... -run TestDiscriminantConst -v`

## Phase 04: Memory Intrinsics in Const Context [W14-P04-MEM-INTRINSICS]

- Task 01: `size_of[T]()` and `align_of[T]()` [W14-P04-T01-SIZEOF]
  DoD: callable in `const` contexts for every concrete type.
  Verify: `go test ./compiler/consteval/... -run TestSizeOfAlignOf -v`

## Phase 05: `const fn` Proof Program [W14-P05-PROOF]

- Task 01: `const_fn.fuse` [W14-P05-T01-PROOF]
  DoD: a program with `const fn factorial` and `const FACT_10: U64 =
  factorial(10);` evaluates to 3628800 at compile time; `main` returns
  `(FACT_10 % 256) as I32` (= 0x80 = 128).
  Verify: `go test ./tests/e2e/... -run TestConstFnProof -v`

## Wave Closure Phase [W14-PCL-WAVE-CLOSURE]

- Task 01: Retire `const fn` stubs [W14-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W14`
- Task 02: WC014 entry [W14-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC014" docs/learning-log.md`

