// Package hir owns the semantically rich Fuse intermediate representation
// (Rule 3.1 — disjoint IR type families; Rule 3.3 — MIR lowering must
// account for all expression forms).
//
// HIR is the typed semantic layer that sits between the AST
// (compiler/ast) and MIR (compiler/mir, landing in W15). Where the AST
// is syntax-only and preserves source spellings, HIR carries:
//
//   - A typetable.TypeId on every expression, pattern, and type-bearing
//     node. An unresolved type is KindInfer (explicit), not a silent
//     Unknown (L013, L021).
//   - A stable NodeID derived from module path, item path, and local
//     index that survives unrelated source edits (W04-P05-T02).
//   - Structured Pattern nodes (LiteralPat, BindPat, ConstructorPat,
//     WildcardPat, OrPat, RangePat, AtBindPat). Patterns are never
//     stored as text (L007).
//
// The Bridge (AST → HIR) is implemented in this package. It consumes
// the output of compiler/resolve and produces a Program whose every
// expression either has a concrete TypeId or carries an explicit
// KindInfer marker for W06 to resolve. There is no third state.
//
// Builders (NewFn, NewBlock, NewCall, ...) enforce metadata at
// construction: a node cannot be allocated without its required
// fields. Tests that construct HIR via builders therefore also prove
// that the required invariants are upheld.
//
// The pass manifest (Manifest) is the incremental-compilation
// foundation. Every pass declares its inputs, its outputs, and a
// deterministic fingerprint function over inputs. W18's incremental
// driver and W19's language server consume this manifest; W04 lays
// the shape so those later waves do not have to retrofit.
package hir
