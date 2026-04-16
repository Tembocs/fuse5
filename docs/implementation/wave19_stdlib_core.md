# Wave 19: Stdlib Core

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: implement the OS-free core standard library. Includes core traits,
primitive method surface, strings, collections, iterators, `Cell`/`RefCell`
interior mutability, `Ptr.null` surface, overflow-aware arithmetic methods,
runtime bridge files.

Entry criterion: W18 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- core traits shipped: equality, ordering, hashing, formatting, default
- primitive method surface matches reference §24
- `String` and formatting primitives work
- `List`, `Map`, `Set`, iterators work
- `Cell[T]` and `RefCell[T]` with runtime borrow tracking (reference §51.1)
- `Ptr.null[T]()`, `is_null()` surface (reference §35.1)
- overflow-aware methods (`wrapping_*`, `checked_*`, `saturating_*`) ship
- intrinsic `Send`/`Sync`/`Copy`/`Fn`/`FnMut`/`FnOnce` re-exported
- core bridge files exist per repository-layout.md
- docs coverage check passes

Proof of completion:

```
fuse build stdlib/core/...
go test ./tests/... -run TestCoreStdlib -v
fuse doc --check stdlib/core/
```

## Phase 00: Stub Audit [W19-P00-STUB-AUDIT]

- Task 01: Core stdlib audit [W19-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W19 -phase P00`

## Phase 01: Core Traits and Primitives [W19-P01-TRAITS]

- Task 01: Core traits [W19-P01-T01-TRAITS]
  Verify: `fuse build stdlib/core/traits/...`
- Task 02: Primitive methods [W19-P01-T02-PRIM]
  Verify: `fuse build stdlib/core/primitives/...`
- Task 03: Re-export intrinsic marker and callable traits
  [W19-P01-T03-REEXPORT]
  Verify: `go test ./tests/... -run TestCoreReExports -v`

## Phase 02: Strings, Collections, Iteration [W19-P02-COLLECTIONS]

- Task 01: `String` and formatting [W19-P02-T01-STRING]
  Verify: `fuse build stdlib/core/string/...`
- Task 02: `List`, `Map`, `Set`, iterators [W19-P02-T02-COLLECTIONS]
  Verify: `fuse build stdlib/core/collections/...`

## Phase 03: Interior Mutability [W19-P03-INTERIOR-MUT]

- Task 01: `Cell[T]` [W19-P03-T01-CELL]
  DoD: mutation through shared reference for `Copy` types; not `Send`,
  not `Sync`.
  Verify: `fuse build stdlib/core/cell/... && go test ./tests/... -run TestCell -v`
- Task 02: `RefCell[T]` with runtime borrow tracking [W19-P03-T02-REFCELL]
  DoD: runtime borrow count; `borrow()` increments shared count;
  `borrow_mut()` requires no outstanding borrows; violations panic.
  Verify: `go test ./tests/... -run TestRefCell -v`

## Phase 04: Pointer and Memory Surface [W19-P04-PTR-MEM]

- Task 01: `Ptr.null[T]()` API [W19-P04-T01-NULL]
  Verify: `go test ./tests/... -run TestPtrNull -v`
- Task 02: `size_of[T]()` / `align_of[T]()` wrappers [W19-P04-T02-SIZEOF]
  Verify: `go test ./tests/... -run TestSizeOfWrappers -v`

## Phase 05: Overflow-Aware Arithmetic Methods [W19-P05-OVERFLOW]

- Task 01: `wrapping_*`, `checked_*`, `saturating_*` methods
  [W19-P05-T01-OVERFLOW-METHODS]
  Verify: `go test ./tests/... -run TestOverflowMethods -v`

## Phase 06: Runtime Bridge [W19-P06-BRIDGE]

- Task 01: Core bridge files [W19-P06-T01-FILES]
  Verify: `fuse build stdlib/core/rt_bridge/...`
- Task 02: Docs coverage [W19-P06-T02-DOC]
  Verify: `fuse doc --check stdlib/core/`

## Wave Closure Phase [W19-PCL-WAVE-CLOSURE]

- Task 01: Retire core stdlib stubs [W19-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W19`
- Task 02: WC019 entry [W19-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC019" docs/learning-log.md`

