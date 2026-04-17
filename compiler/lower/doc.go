// Package lower owns HIR-to-MIR lowering (Rule 3.1).
//
// At W05 the lowerer handles only the minimal spine — an
// integer-returning `main` whose body is a straight-line return of
// an integer expression. Anything beyond that emits a diagnostic
// rather than silently approximating (Rule 6.9).
//
// Supported W05 expressions:
//
//   - Integer literals (any bit width; narrowing to i64 at MIR).
//   - Binary arithmetic: `+`, `-`, `*`, `/`, `%` on integer operands.
//   - Bare `return EXPR;` statements in a one-block fn body.
//
// Every other HIR form (calls, field access, matches, etc.) produces
// a "W05 spine does not yet lower X" diagnostic. Later waves extend
// the lowerer additively; the W05 surface never shrinks.
package lower
