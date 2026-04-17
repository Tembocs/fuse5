// Package check owns semantic analysis and type checking for Fuse
// (reference §2–§13, §24–§56).
//
// The checker consumes the HIR produced by the AST→HIR bridge and
// produces a fully-typed program: every `KindInfer` TypeId that the
// bridge left behind is resolved here, and type errors are
// diagnosed. Unresolved `KindInfer` after Check is a compiler bug,
// not a user error (L013 — no Unknown survives checking).
//
// Shape:
//
//  1. Two-pass design (W06-P01-T02). Pass 1 registers every function
//     signature, struct/enum/trait/union/typealias, and impl block
//     so bodies can reference any top-level item regardless of
//     declaration order. Pass 2 walks bodies and types expressions.
//  2. Contextual inference (W06-P04). An expression's expected type
//     flows from the context (let-type, return-type, fn-argument-
//     slot). Numeric and string literals pick their concrete type
//     from that hint; in its absence integers default to I32, floats
//     to F64.
//  3. Trait resolution (W06-P03). Each `impl Trait for Type` block
//     registers a (Trait, Type) entry in the coherence table;
//     conflicts between two impls for the same pair produce a
//     diagnostic. The orphan rule requires an impl to live in the
//     module that defines either the trait or the implementing type.
//  4. Associated type projection (W06-P05). `Self::Item` inside a
//     trait body is typed as the trait's associated-type parameter;
//     `<T as Trait>::Item` resolves through the (Trait, Type) pair.
//  5. Cast and widening rules (W06-P02). Explicit `as` casts are
//     permitted between primitive numeric types; implicit widening
//     follows the numeric-lattice rules from reference §5.8.
//
// The W06 scope stops where later waves take over: pattern-match
// exhaustiveness (W10), `?` error propagation (W11), closure
// capture analysis (W12), full trait-object vtable layout (W13),
// and const evaluation (W14) each have dedicated waves. Their stubs
// stay Active in STUBS.md until their waves close.
package check
