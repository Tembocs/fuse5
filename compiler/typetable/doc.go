// Package typetable owns interned type identity for the Fuse compiler
// (reference §2.8; Rule 3.7).
//
// The TypeTable is the retirement site for part of the W04 stub. It
// defines TypeId — an opaque integer handle — and the interning
// machinery that turns a structural or nominal type description into a
// TypeId. Two equal types share a TypeId; equality of types is always
// integer comparison (Rule 7.2).
//
// The key contract is §2.8's nominal identity rule: two nominal types
// are the same type if and only if they share the same declared name
// AND the same defining symbol (or module). Two types called `Expr`
// from different modules are distinct; name-only equality is invalid
// in a multi-module compiler.
//
// `Unknown` is deliberately absent from the TypeKind set. The bridge
// (compiler/hir.Bridge) must never synthesize an `Unknown` fallback
// (L013, L021). Types that the bridge genuinely cannot yet resolve —
// for example, a generic use before monomorphization — carry the
// explicit `KindInfer` kind, which the type checker in W06 is required
// to resolve or reject before HIR leaves check.
//
// `KindChannel` and `KindThreadHandle` are defined here at W04 so that
// W07 (concurrency) has a stable type representation to integrate
// with. At W04 those kinds exist in the table but do not yet
// participate in checking.
package typetable
