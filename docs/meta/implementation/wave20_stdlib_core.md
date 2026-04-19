# Wave 20: Stdlib Core

> Part of the [Fuse implementation plan](../implementation-plan.md).

> Reference guides for this wave live under `../language-reference/core/`.


Goal: implement the OS-free `core.*` standard library that Stage 1 depends on.
W20 owns the public core module tree itself: scalar foundations, traits,
`Option`, `Result`, strings, formatting, collections, interior mutability,
pointer/size helpers, and overflow-aware arithmetic methods. W21 may extend
these modules with allocator-awareness, but W20 is where the named public core
surface must become real and reviewable.

Pillar alignment:

- Memory safety: `Option`, `Result`, `String`, collections, `Cell`,
  `RefCell`, pointer helpers, and overflow methods must preserve ownership,
  deterministic destruction, and explicit failure semantics rather than hiding
  raw runtime behavior.
- Concurrency safety: re-exported marker and callable traits must remain
  consistent with W07/W16 concurrency rules; `Cell` and `RefCell` do not gain
  silent cross-thread behavior in this wave.
- Developer experience: the public module names and minimum surfaces must match
  the language reference so users can discover the standard library without
  reading compiler internals.

Entry criterion: W19 done. Phase 00 confirms no overdue stubs.

State on entry: the compiler pipeline, diagnostics, and documentation tooling
exist, but the implementation plan still treats much of core stdlib in families
rather than as named public modules. The language-reference tree now names the
core module set explicitly, so this wave must turn those names into owned
deliverables.

Exit criteria:

- stable public modules ship for `core.bool`, `core.int`, `core.int8`,
  `core.int32`, `core.uint8`, `core.uint32`, `core.uint64`, `core.float`,
  `core.float32`, `core.math`, and the primitive method surface required by the
  language reference
- stable public modules ship for `core.traits`, `core.comparable`,
  `core.equatable`, `core.hashable`, `core.hash`, `core.printable`, and
  `core.debuggable`
- stable public modules ship for `core.option`, `core.result`, `core.string`,
  and `core.fmt`
- stable public modules ship for `core.list`, `core.map`, and `core.set`,
  including the iterator and traversal surface those collections require
- `Cell[T]` and `RefCell[T]` ship with explicit runtime borrow tracking where
  required by the reference
- pointer and layout helpers ship: `Ptr.null[T]()`, `is_null()`, `size_of`,
  `align_of`
- overflow-aware methods (`wrapping_*`, `checked_*`, `saturating_*`) ship on
  the numeric surface promised by the reference
- intrinsic `Send`/`Sync`/`Copy`/`Fn`/`FnMut`/`FnOnce` are re-exported through
  stable core entry points rather than ad hoc compiler-only hooks
- bridge files and docs coverage exist for the full W20 core surface
- at least one end-to-end proof exercises the core surface as user code rather
  than as compiler-internal scaffolding

Proof of completion:

```
fuse build stdlib/core/...
go test ./tests/stdlib/... -run TestCoreBoolIntModules -v
go test ./tests/stdlib/... -run TestCoreFloatMathModules -v
go test ./tests/stdlib/... -run TestCorePrimitiveMethods -v
go test ./tests/stdlib/... -run TestCoreComparisonTraits -v
go test ./tests/stdlib/... -run TestCoreHashTraits -v
go test ./tests/stdlib/... -run TestCoreDisplayTraits -v
go test ./tests/stdlib/... -run TestCoreIntrinsicReExports -v
go test ./tests/stdlib/... -run TestCoreOptionModule -v
go test ./tests/stdlib/... -run TestCoreResultModule -v
go test ./tests/stdlib/... -run TestCoreStringModule -v
go test ./tests/stdlib/... -run TestCoreFmtModule -v
go test ./tests/stdlib/... -run TestCoreListModule -v
go test ./tests/stdlib/... -run TestCoreListIteration -v
go test ./tests/stdlib/... -run TestCoreMapModule -v
go test ./tests/stdlib/... -run TestCoreSetModule -v
go test ./tests/stdlib/... -run TestCoreCellModule -v
go test ./tests/stdlib/... -run TestCoreRefCellModule -v
go test ./tests/stdlib/... -run TestCorePtrHelpers -v
go test ./tests/stdlib/... -run TestCoreLayoutHelpers -v
go test ./tests/stdlib/... -run TestCoreOverflowMethods -v
go test ./tests/e2e/... -run TestCoreStdlibProof -v
fuse doc --check stdlib/core/
```

## Phase 00: Stub Audit [W20-P00-STUB-AUDIT]

- Task 01: Core stdlib audit [W20-P00-T01-AUDIT]
  Currently: W20 owns a large user-visible surface and therefore carries high
  drift risk if active stubs are not enumerated first.
  Expected deliveries: audited W20 stub list, overdue-stub check, and committed
  `STUBS.md` updates before implementation work begins.
  DoD: every W20-relevant stub is either scheduled to retire here or explicitly
  documented as a later-wave dependency that does not block core stdlib work.
  Verify: `go run tools/checkstubs/main.go -wave W20 -phase P00`

## Phase 01: Core Namespace Ownership [W20-P01-CORE-NAMESPACE]

- Task 01: Name and place every public core module [W20-P01-T01-MODULE-MAP]
  Currently: the language-reference tree names the core modules explicitly, but
  earlier planning still grouped them under broader labels such as "traits" or
  "collections".
  Expected deliveries: stable `stdlib/core/` package layout matching the public
  module names in this wave.
  DoD: every core module named in the W20 exit criteria exists as an owned
  public surface with no anonymous "misc core" or "remaining helpers" bucket.
  Verify: `fuse build stdlib/core/...`
- Task 02: Align module ownership with the reference guides
  [W20-P01-T02-GUIDE-ALIGN]
  Currently: the meta language-reference guides and the implementation plan can
  drift unless the wave names the same modules the guides name.
  Expected deliveries: guide-to-wave alignment for the entire W20 module set.
  DoD: every module named in this wave is also named in the corresponding core
  guide set and in the implementation-tree indexes.
  Verify: `fuse doc --check stdlib/core/`

## Phase 02: Boolean and Numeric Modules [W20-P02-NUMBERS]

- Task 01: Deliver `core.bool` and the signed integer modules
  [W20-P02-T01-BOOL-INT]
  Currently: primitive typing exists in the checker, but the public module
  surfaces for `core.bool`, `core.int`, `core.int8`, and `core.int32` are not
  yet owned by distinct completion criteria.
  Expected deliveries: stable `core.bool`, `core.int`, `core.int8`, and
  `core.int32` modules with the methods and constants promised by the
  reference.
  DoD: each named module can compile and be imported independently, and no
  signed-integer surface remains hidden behind a generic "numbers" umbrella.
  Verify: `go test ./tests/stdlib/... -run TestCoreBoolIntModules -v`
- Task 02: Deliver the unsigned integer, float, and math modules
  [W20-P02-T02-UINT-FLOAT-MATH]
  Currently: unsigned and floating-point semantics are planned, but the public
  module boundaries for `core.uint8`, `core.uint32`, `core.uint64`,
  `core.float`, `core.float32`, and `core.math` are still only implied.
  Expected deliveries: stable modules for those six surfaces, with explicit
  constants, methods, and numeric helpers.
  DoD: the unsigned and floating-point module family is complete without relying
  on signed-number modules to prove its correctness.
  Verify: `go test ./tests/stdlib/... -run TestCoreFloatMathModules -v`
- Task 03: Deliver primitive method registration as a public contract
  [W20-P02-T03-PRIMITIVE-METHODS]
  Currently: primitive method lookup is a type-checker concern first and a
  documented stdlib contract second.
  Expected deliveries: primitive method surfaces exposed through the named
  numeric modules rather than through compiler-private registration only.
  DoD: primitive methods fail loudly if a promised method is missing from a
  specific module.
  Verify: `go test ./tests/stdlib/... -run TestCorePrimitiveMethods -v`

## Phase 03: Trait and Protocol Modules [W20-P03-TRAITS]

- Task 01: Deliver `core.traits`, `core.comparable`, and `core.equatable`
  [W20-P03-T01-COMPARISON-TRAITS]
  Currently: comparison traits are specified in the reference but grouped too
  broadly in the current wave structure.
  Expected deliveries: explicit module ownership for comparison and equality
  protocols.
  DoD: each module can be reviewed and verified independently, with no trait
  family hidden behind a single umbrella task.
  Verify: `go test ./tests/stdlib/... -run TestCoreComparisonTraits -v`
- Task 02: Deliver `core.hashable` and `core.hash`
  [W20-P03-T02-HASH-TRAITS]
  Currently: hashing behavior is easy to leave half-defined when it is bundled
  with unrelated trait work.
  Expected deliveries: stable hashing traits and helper surface through their
  own modules.
  DoD: hashing contracts are complete enough for later `Map` and `Set` work to
  rely on them without hidden TODOs.
  Verify: `go test ./tests/stdlib/... -run TestCoreHashTraits -v`
- Task 03: Deliver `core.printable` and `core.debuggable`
  [W20-P03-T03-DISPLAY-TRAITS]
  Currently: output-related traits are named in the reference, but their module
  ownership can drift into formatting work if not retired separately.
  Expected deliveries: explicit protocol modules for user-facing print and
  debug rendering.
  DoD: display/debug traits are complete before `core.fmt` builds on them.
  Verify: `go test ./tests/stdlib/... -run TestCoreDisplayTraits -v`
- Task 04: Re-export intrinsic marker and callable traits
  [W20-P03-T04-INTRINSIC-REEXPORTS]
  Currently: `Send`, `Sync`, `Copy`, `Fn`, `FnMut`, and `FnOnce` are still at
  risk of being treated as checker-only concepts.
  Expected deliveries: stable re-export surface for those intrinsic traits.
  DoD: user code imports the intrinsic traits from `core.*`, not from
  implementation detail paths.
  Verify: `go test ./tests/stdlib/... -run TestCoreIntrinsicReExports -v`

## Phase 04: `core.option` and `core.result` [W20-P04-OPTION-RESULT]

- Task 01: Deliver `core.option` [W20-P04-T01-OPTION]
  Currently: `Option` is fundamental to the language shape but still lacks an
  isolated module-level retirement target.
  Expected deliveries: `core.option` constructors, methods, and pattern-facing
  behavior promised by the reference.
  DoD: `Option` can be verified independently from `Result` and is not hidden in
  a combined sum-type bucket.
  Verify: `go test ./tests/stdlib/... -run TestCoreOptionModule -v`
- Task 02: Deliver `core.result` [W20-P04-T02-RESULT]
  Currently: `Result` is easy to treat as "the error type" rather than as a
  module with its own public API contract.
  Expected deliveries: `core.result` constructors, combinators, and value/error
  accessors promised by the reference.
  DoD: `Result` can be reviewed and tested without piggybacking on `Option`
  proof coverage.
  Verify: `go test ./tests/stdlib/... -run TestCoreResultModule -v`

## Phase 05: `core.string` and `core.fmt` [W20-P05-STRING-FMT]

- Task 01: Deliver `core.string` [W20-P05-T01-STRING]
  Currently: string semantics are specified, but the owned module surface is
  still too easy to blur with formatting or collection work.
  Expected deliveries: `core.string` with owned-string operations, views, and
  conversions promised by the reference.
  DoD: the string module can be reviewed as a complete surface independent of
  `core.fmt`.
  Verify: `go test ./tests/stdlib/... -run TestCoreStringModule -v`
- Task 02: Deliver `core.fmt` [W20-P05-T02-FMT]
  Currently: formatting behavior depends on string and protocol work, but it is
  still a separate public module that needs its own completion test.
  Expected deliveries: formatting APIs, format-string rules, and integration
  with `core.printable` and `core.debuggable`.
  DoD: the formatting module is complete enough that failures point to
  `core.fmt`, not generically to "string/formatting" as a bundle.
  Verify: `go test ./tests/stdlib/... -run TestCoreFmtModule -v`

## Phase 06: `core.list` and Sequential Iteration [W20-P06-LIST]

- Task 01: Deliver `core.list` [W20-P06-T01-LIST]
  Currently: sequential collection work is still at risk of being subsumed into
  a generic collection family.
  Expected deliveries: `core.list` with creation, growth, indexing, and
  ownership-preserving mutation.
  DoD: `List[T]` is complete and reviewable without depending on `Map` or `Set`
  progress.
  Verify: `go test ./tests/stdlib/... -run TestCoreListModule -v`
- Task 02: Deliver list iteration and traversal contracts
  [W20-P06-T02-LIST-ITERATION]
  Currently: iteration can drift into later collection work unless W20 binds it
  directly to the list surface.
  Expected deliveries: iterator and traversal behavior promised for
  sequence-like containers at W20 exit.
  DoD: list traversal failures are caught independently of keyed-collection
  regressions.
  Verify: `go test ./tests/stdlib/... -run TestCoreListIteration -v`

## Phase 07: `core.map` and `core.set` [W20-P07-MAP-SET]

- Task 01: Deliver `core.map` [W20-P07-T01-MAP]
  Currently: keyed collections are still too easy to treat as one collective
  success/failure bucket.
  Expected deliveries: `core.map` with hashing/equality integration, lookup,
  insertion, removal, and iteration contracts.
  DoD: `Map[K, V]` has its own completion boundary independent of `Set`.
  Verify: `go test ./tests/stdlib/... -run TestCoreMapModule -v`
- Task 02: Deliver `core.set` [W20-P07-T02-SET]
  Currently: `Set[T]` risks becoming an afterthought attached to `Map` if it is
  not retired separately.
  Expected deliveries: `core.set` with membership, insertion/removal, and
  iteration contracts.
  DoD: `Set[T]` can be signed off independently from `Map[K, V]`.
  Verify: `go test ./tests/stdlib/... -run TestCoreSetModule -v`

## Phase 08: `Cell[T]` and `RefCell[T]` [W20-P08-INTERIOR-MUT]

- Task 01: Deliver `Cell[T]` [W20-P08-T01-CELL]
  Currently: interior mutability is specified, but `Cell` and `RefCell` are
  still bundled too closely for independent review.
  Expected deliveries: `Cell[T]` for shared-reference mutation of `Copy` data.
  DoD: `Cell` can be signed off independently, including its non-`Send` and
  non-`Sync` behavior.
  Verify: `go test ./tests/stdlib/... -run TestCoreCellModule -v`
- Task 02: Deliver `RefCell[T]` [W20-P08-T02-REFCELL]
  Currently: runtime borrow tracking is a materially different contract from
  `Cell`, but current planning makes them easy to retire together.
  Expected deliveries: `RefCell[T]` with runtime borrow-count enforcement.
  DoD: `RefCell` violations panic with the documented behavior and do not rely
  on `Cell` coverage to prove correctness.
  Verify: `go test ./tests/stdlib/... -run TestCoreRefCellModule -v`

## Phase 09: Pointer, Layout, and Overflow Helpers [W20-P09-PTR-LAYOUT-OVERFLOW]

- Task 01: Deliver pointer-null helpers [W20-P09-T01-PTR]
  Currently: pointer helper APIs are promised by the reference but still easy to
  hide inside backend-oriented helpers.
  Expected deliveries: `Ptr.null[T]()` and `is_null()` on stable core entry
  points.
  DoD: pointer-null behavior is verifiable independently from layout and
  overflow helpers.
  Verify: `go test ./tests/stdlib/... -run TestCorePtrHelpers -v`
- Task 02: Deliver layout helpers [W20-P09-T02-LAYOUT]
  Currently: `size_of` and `align_of` are often treated as intrinsic plumbing
  rather than as user-facing contracts.
  Expected deliveries: `size_of[T]()` and `align_of[T]()` through the public
  core surface.
  DoD: layout-query behavior is fully specified and independently testable.
  Verify: `go test ./tests/stdlib/... -run TestCoreLayoutHelpers -v`
- Task 03: Deliver overflow-aware arithmetic methods
  [W20-P09-T03-OVERFLOW]
  Currently: overflow policy is part of the language contract, but the method
  surface still needs a narrow retirement boundary.
  Expected deliveries: `wrapping_*`, `checked_*`, and `saturating_*` methods on
  the numeric modules that promise them.
  DoD: overflow helper failures point to their own task rather than to a mixed
  pointer/layout phase.
  Verify: `go test ./tests/stdlib/... -run TestCoreOverflowMethods -v`

## Phase 10: Bridge Surface, Proof, and Docs [W20-P10-BRIDGE-PROOF]

- Task 01: Stabilize the core runtime bridge surface [W20-P10-T01-BRIDGE]
  Currently: bridge files can still pass as thin implementation detail without
  a public-surface completion check.
  Expected deliveries: bridge files and symbol boundaries required by the core
  library, aligned with repository-layout rules.
  DoD: the core stdlib compiles end to end without hidden out-of-tree bridge
  dependencies.
  Verify: `fuse build stdlib/core/...`
- Task 02: Add the core stdlib proof program [W20-P10-T02-PROOF]
  Currently: broad build/test passes are not enough to prove that W20 behaves
  correctly as a user-visible library wave.
  Expected deliveries: an end-to-end proof program that exercises the core
  module tree as ordinary Fuse code.
  DoD: the proof fails if W20 leaves any named core module stubbed or only
  partially wired.
  Verify: `go test ./tests/e2e/... -run TestCoreStdlibProof -v`
- Task 03: Close docs coverage for the core tree [W20-P10-T03-DOCS]
  Currently: the core module tree can still drift from the reference unless the
  docs surface is validated as part of the wave exit.
  Expected deliveries: docs coverage and module-guide parity for the entire W20
  public surface.
  DoD: the generated docs surface matches the named modules and minimum APIs W20
  claims to ship.
  Verify: `fuse doc --check stdlib/core/`

## Wave Closure Phase [W20-PCL-WAVE-CLOSURE]

- Task 01: Retire core stdlib stubs [W20-PCL-T01-RETIRE]
  Currently: any remaining W20 stub would mean the public core surface still
  contains hidden implementation debt.
  Expected deliveries: retired W20 stdlib stubs and updated `STUBS.md` history.
  DoD: no Active stub row remains for a feature this wave claimed to retire.
  Verify: `go run tools/checkstubs/main.go -wave W20`
- Task 02: WC020 entry [W20-PCL-T02-CLOSURE-LOG]
  Currently: the closure log is the only durable place to record what core
  stdlib proofs and residual risks actually shipped.
  Expected deliveries: `WC020` learning-log entry naming proofs, retired stubs,
  and what W21 must know.
  DoD: `docs/learning-log.md` contains a complete `WC020` entry aligned with
  this wave's proof commands.
  Verify: `grep "WC020" docs/learning-log.md`

