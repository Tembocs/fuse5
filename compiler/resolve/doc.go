// Package resolve owns module discovery, symbols, imports, `@cfg`
// evaluation, and visibility enforcement (reference §11.6, §18, §50, §53).
//
// The resolver is the retirement site for the W03 stub. It runs after
// parsing and before HIR construction (W04). Its inputs are a set of
// parsed `*ast.File`s with their module paths and a BuildConfig; its
// outputs are the resolved module graph, symbol table, and a binding map
// from syntactic path sites to resolved symbols. The AST is not mutated
// (Rule 3.2 — AST stays syntax-only).
//
// Resolution is deterministic: iteration over modules and scopes uses
// pre-sorted keys, and diagnostics are stable across runs (Rule 7.1,
// Rule 7.4).
//
// The resolver's contracts:
//
//   - Module discovery mirrors directory structure: `<root>/foo/bar.fuse`
//     belongs to module `foo.bar` (reference §18.1).
//   - Imports use module-first resolution: `import a.b.c` first tries
//     module `a.b.c`; if that module does not exist, retries with `c` as
//     an item inside module `a.b` (reference §18.7).
//   - Cyclic imports produce a diagnostic naming the cycle path
//     (reference §18.7).
//   - `@cfg` predicates are evaluated here, before any other resolution
//     work; items whose predicate is false are excluded from every
//     downstream name resolution step (reference §50.1).
//   - Qualified enum variant paths `Enum.Variant` resolve to the hoisted
//     variant symbol (reference §11.6, §18.6).
//   - Visibility is enforced at every use site across all four levels —
//     private, `pub(mod)`, `pub(pkg)`, `pub` (reference §53.1). A
//     narrower visibility never shadows a wider one.
package resolve
