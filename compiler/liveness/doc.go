// Package liveness owns the Fuse ownership / borrow / liveness /
// drop-intent analysis (Rule 3.8, reference §14 ownership, §54
// borrow contracts, §15.5 closure escape, §16 Drop trait).
//
// The pass runs after monomorphization (W08) and before lowering
// (W15+). Its responsibilities are:
//
//   - Enforce the structural borrow rules (reference §54):
//       §54.1  no borrows in struct fields
//       §54.6  returning a borrow requires it to point into a
//              borrowed parameter
//       §54.7  `mutref` is exclusive — coexistence with any other
//              borrow or overlapping `mutref` is rejected
//   - Track moves and reject use-after-move on every control-flow
//     path.
//   - Classify every closure as escaping or non-escaping per
//     §15.5, and reject non-escaping closures at any escape site
//     (storage, return, boxing, `spawn`, `Chan[T]`, `Shared[T]`).
//   - Compute liveness once per function (Rule 3.8), emit
//     last-use / destroy-after metadata, insert drop intent at
//     last-use for every local whose type implements Drop.
//   - Surface drop metadata to codegen so `TypeName_drop(&_lN)`
//     emission is deterministic.
//
// Invariant after Analyze: every rule violation is reported or
// the HIR carries verified ownership metadata; downstream waves
// (W15 MIR consolidation, W17 codegen hardening) consult the
// attached metadata rather than re-analyzing.
package liveness
