// Package consteval is the Wave 14 compile-time evaluator.
//
// # Scope
//
// The evaluator runs over *checked* HIR — after the checker (W06) has
// replaced every KindInfer TypeId with a concrete type and the bridge
// has produced a valid program. It produces constant values for:
//
//   - `const NAME: T = expr;` initializers (reference §46)
//   - `static NAME: T = expr;` initializers (reference §22)
//   - array length expressions (`[T; N]` where N is a const expression;
//     reference §3.2)
//   - enum discriminant expressions (`Variant = <expr>`)
//   - memory intrinsics in `const` context: `size_of[T]()` and
//     `align_of[T]()` (reference §34.1)
//
// The evaluator is the bootstrap implementation of reference §46.1:
// callers pass a *hir.Program, get back a Result carrying every
// evaluated constant keyed by its defining symbol ID, plus a
// diagnostics slice when evaluation fails.
//
// Determinism contract (Rule 7.1)
//
// The evaluator is deterministic: the same (Program, TypeTable) input
// produces byte-identical output across runs and platforms. Iteration
// over maps is always funneled through a sorted key list; no random
// number generators, time sources, or allocator-dependent iteration
// affect results. `TestEvaluatorDeterminism` asserts this under
// `-count=3`.
//
// Restrictions (reference §46.1)
//
// CheckRestrictions walks a `const fn` body and rejects operations the
// language reference forbids:
//
//   - FFI calls (extern fn)
//   - allocation / deallocation (unsafe blocks stand in at W14 since
//     stdlib is not yet available)
//   - thread operations (`spawn`)
//   - calls to non-`const` functions
//   - interior mutability (recognized by nominal names Cell / RefCell /
//     Atomic* at W14; the real stdlib marker lands with W20)
//
// Each diagnostic carries a primary span, a one-line explanation, and
// a suggestion where one is available (Rule 6.17).
//
// Retires the STUBS.md row "Compile-time evaluation (`const fn`,
// `size_of`, `align_of`)" at Wave 14 closure.
package consteval
