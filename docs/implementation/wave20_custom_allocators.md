# Wave 20: Custom Allocators

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: declare the `Allocator` trait, parameterize core collections over
an allocator, and prove a user-defined `BumpAllocator` can back a
collection.

Entry criterion: W19 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- `Allocator` trait declared in stdlib core with `alloc`, `dealloc`,
  `realloc` methods (reference §52.1)
- core collections (`Vec`, `HashMap`, `Box`, etc.) generic over allocator
  parameter; default resolves to `SystemAllocator`
- global allocator declared once per binary; default wraps runtime alloc
- all allocation paths route through the provided allocator; no silent
  fallback to global
- user-defined `BumpAllocator` backs a collection end-to-end
- proof program: `BumpAllocator` allocates twice, collection holds both,
  `reset()` reclaims all memory

Proof of completion:

```
fuse build stdlib/core/alloc/...
go test ./tests/... -run TestAllocatorTrait -v
go test ./tests/... -run TestCollectionsInAllocator -v
go test ./tests/e2e/... -run TestBumpAllocatorProof -v
```

## Phase 00: Stub Audit [W20-P00-STUB-AUDIT]

- Task 01: Allocator audit [W20-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W20 -phase P00`

## Phase 01: Allocator Trait and Global Allocator [W20-P01-TRAIT]

- Task 01: Declare `Allocator` trait [W20-P01-T01-TRAIT]
  Verify: `fuse build stdlib/core/alloc/... && go test ./tests/... -run TestAllocatorTrait -v`
- Task 02: Global allocator with runtime wrapper [W20-P01-T02-GLOBAL]
  DoD: `SystemAllocator` wraps `fuse_rt_alloc_*`; override mechanism
  documented.
  Verify: `go test ./tests/... -run TestGlobalAllocator -v`

## Phase 02: Parameterize Collections [W20-P02-PARAMETERIZE]

- Task 01: `Vec`/`HashMap`/`Box` take allocator param [W20-P02-T01-VEC]
  DoD: every allocation site routes through the provided allocator.
  Verify: `go test ./tests/... -run TestCollectionsInAllocator -v`

## Phase 03: Allocator Proof Program [W20-P03-PROOF]

- Task 01: `bump_allocator.fuse` [W20-P03-T01-PROOF]
  DoD: a program defines a `BumpAllocator` backed by a stack buffer,
  allocates a `Vec[I32, BumpAllocator]`, pushes two values, sums them,
  returns the sum as the exit code. `arena.reset()` must be observable
  (alloc after reset returns original offset).
  Verify: `go test ./tests/e2e/... -run TestBumpAllocatorProof -v`

## Wave Closure Phase [W20-PCL-WAVE-CLOSURE]

- Task 01: Retire allocator stubs [W20-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W20`
- Task 02: WC020 entry [W20-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC020" docs/learning-log.md`

