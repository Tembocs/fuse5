// Package monomorph owns generic-function specialization — the
// "body duplication per concrete type arg" pass that turns generic
// HIR into concrete HIR before lowering (reference §13; L015).
//
// Inputs:
//   - `*hir.Program` after W06 type-checking. Generic call sites
//     carry their explicit turbofish `TypeArgs` on `PathExpr`.
//
// Outputs:
//   - A new `*hir.Program` whose `Modules[*].Items` contains one
//     concrete specialization per (generic fn, type-arg tuple)
//     pair observed at a call site. Generic originals are
//     excluded so lowering and codegen see only concrete fns.
//   - Call sites whose callee was a generic fn are rewritten to
//     reference the specialized symbol and name, with
//     `PathExpr.TypeArgs` cleared.
//
// Specialization identity is deterministic and stable across runs
// (Rule 7.1): the mangled name is `<fn_name>__<TypeName1>_<TypeName2>...`
// where each TypeName is the canonical stringification of the
// concrete TypeId. Two builds with the same program produce the
// same specialization set in the same order.
//
// W08 scope: specialize fn declarations. Generic nominal types
// (Option, Result, generic structs) specialize when referenced at
// a call-site's turbofish; full trait-bound-driven dispatch is W13.
package monomorph
