# Wave 21: Custom Allocators

> Part of the [Fuse implementation plan](../implementation-plan.md).

> This wave owns the allocation contracts described in reference §52.1 and the
> allocatorization of the W20 core surface.


Goal: declare the explicit allocator model, route every heap-owning W20 core
container through that model, and prove that user-defined allocators can back
ordinary Fuse collections end to end. W21 does not invent a second container
surface; it makes the existing core surface allocator-aware without weakening
ownership or diagnostics.

Pillar alignment:

- Memory safety: every allocation, reallocation, and deallocation path must be
  explicit, layout-correct, and tied to a concrete allocator contract. There is
  no silent fallback to a hidden global allocator.
- Concurrency safety: allocator contracts must not bypass `Send`/`Sync` or
  create hidden thread-affinity rules; the wave must document where allocators
  may and may not cross threads.
- Developer experience: the default allocator path, global override point, and
  user-defined allocator hooks must be easy to discover and diagnose.

Entry criterion: W20 done. Phase 00 confirms no overdue stubs.

State on entry: the W20 core surface exists as named public modules, but
heap-owning paths still need to be centralized behind an explicit allocator
contract. The reference already promises allocator semantics, so W21 must turn
that promise into a coherent user-facing API and proof corpus.

Exit criteria:

- `Allocator` trait and its supporting allocation contract ship on stable core
  entry points
- default `SystemAllocator` wraps the runtime allocator surface and is the only
  implicit default
- a single global allocator declaration point exists per binary, with clear
  diagnostics for duplicate declarations or invalid override shapes
- every heap-owning W20 core container routes through an explicit allocator
  parameter or documented default
- there is no silent fallback from an explicit allocator parameter to the
  process-global allocator
- a user-defined `BumpAllocator` or arena-style allocator can back ordinary core
  containers end to end
- the allocator story is documented as part of the public core surface rather
  than as a runtime implementation detail

Proof of completion:

```
fuse build stdlib/core/alloc/...
go test ./tests/stdlib/... -run TestAllocatorTrait -v
go test ./tests/stdlib/... -run TestSystemAllocator -v
go test ./tests/stdlib/... -run TestAllocatorCollections -v
go test ./tests/stdlib/... -run TestAllocatorCoverage -v
go test ./tests/e2e/... -run TestBumpAllocatorProof -v
```

## Phase 00: Stub Audit [W21-P00-STUB-AUDIT]

- Task 01: Allocator audit [W21-P00-T01-AUDIT]
  Currently: allocator work crosses compiler, runtime, and stdlib surfaces, so
  hidden stubs here create structural risk immediately.
  Expected deliveries: audited allocator-related stub list and committed wave
  entry state.
  DoD: every stub that could affect allocator correctness is either retired in
  W21 or recorded as a later-wave dependency with explicit scope boundaries.
  Verify: `go run tools/checkstubs/main.go -wave W21 -phase P00`

## Phase 01: Allocator Contracts and Supporting Types [W21-P01-CONTRACTS]

- Task 01: Declare `Allocator` and its method contract
  [W21-P01-T01-TRAIT]
  Currently: allocation is a required semantic capability, but the public trait
  surface is not yet the explicit owner of that behavior.
  Expected deliveries: `Allocator` with `alloc`, `dealloc`, and `realloc`, plus
  the associated argument and error contract required by the reference.
  DoD: user-defined allocators can be type-checked and used through a stable
  public trait rather than through compiler-private shims.
  Verify: `go test ./tests/stdlib/... -run TestAllocatorTrait -v`
- Task 02: Define allocation diagnostics and supporting types
  [W21-P01-T02-SUPPORT-TYPES]
  Currently: allocation failure and layout mismatch are easy to leave as vague
  runtime behavior unless they are owned explicitly in the plan.
  Expected deliveries: the supporting allocation types, diagnostics, and error
  behavior required to make allocator failures understandable.
  DoD: the allocator contract names how invalid layouts and allocation failures
  are reported instead of leaving them as unspecified runtime accidents.
  Verify: `go test ./tests/stdlib/... -run TestAllocatorTrait -v`

## Phase 02: System and Global Allocator Wiring [W21-P02-SYSTEM-GLOBAL]

- Task 01: Deliver `SystemAllocator` [W21-P02-T01-SYSTEM]
  Currently: runtime allocation exists, but the default allocator path is not
  yet isolated as a user-visible core contract.
  Expected deliveries: `SystemAllocator` as the default allocator wrapper over
  the runtime allocation ABI.
  DoD: the default allocator is explicit, documented, and not a hidden fallback
  path that bypasses the allocator contract.
  Verify: `go test ./tests/stdlib/... -run TestSystemAllocator -v`
- Task 02: Deliver the single global allocator declaration rule
  [W21-P02-T02-GLOBAL]
  Currently: global allocator policy can remain underspecified if it is treated
  as a bootstrap implementation detail.
  Expected deliveries: one declaration point per binary, clear override rules,
  and diagnostics for duplicates or invalid declarations.
  DoD: the program-global allocator policy is explicit enough that later waves
  do not have to infer it from runtime code.
  Verify: `go test ./tests/stdlib/... -run TestSystemAllocator -v`

## Phase 03: Allocator-Parameterized Core Owners [W21-P03-CORE-OWNERS]

- Task 01: Route `List` and string-growth paths through explicit allocators
  [W21-P03-T01-LIST-STRING]
  Currently: sequence growth is one of the easiest places for a hidden default
  allocator to survive.
  Expected deliveries: allocator-aware `List` growth and allocator-aware heap
  storage behind `String` where W20 core ownership requires it.
  DoD: every dynamic growth path in these containers routes through the provided
  allocator or the documented default.
  Verify: `go test ./tests/stdlib/... -run TestAllocatorCollections -v`
- Task 02: Route `Map` and `Set` storage through explicit allocators
  [W21-P03-T02-MAP-SET]
  Currently: hash-table storage often drifts away from the declared allocator
  contract because it is implemented in lower-level helpers.
  Expected deliveries: allocator-aware `Map` and `Set` backing storage.
  DoD: hashing, resize, and bucket allocation paths all use the allocator named
  by the owning collection instance.
  Verify: `go test ./tests/stdlib/... -run TestAllocatorCollections -v`
- Task 03: Close allocator coverage for every remaining heap-owning core helper
  [W21-P03-T03-COVERAGE]
  Currently: any heap-owning helper left out of the allocator sweep becomes a
  future hidden escape hatch back to the global allocator.
  Expected deliveries: coverage proof that every heap-owning core helper type
  present by W20 exit follows the same allocator contract.
  DoD: no remaining core heap path silently bypasses the allocator model.
  Verify: `go test ./tests/stdlib/... -run TestAllocatorCoverage -v`

## Phase 04: User Allocator Proofs and Docs [W21-P04-PROOF-DOCS]

- Task 01: Add the `BumpAllocator` proof program [W21-P04-T01-PROOF]
  Currently: a build-only allocator wave can look complete even if user-defined
  allocators do not work in ordinary Fuse programs.
  Expected deliveries: an end-to-end proof program for a bump or arena allocator
  backing a real collection.
  DoD: a program defines a `BumpAllocator` backed by a fixed buffer, allocates a
  core collection with it, performs observable work, and proves `reset()` or
  equivalent arena reuse behavior.
  Verify: `go test ./tests/e2e/... -run TestBumpAllocatorProof -v`
- Task 02: Close allocator docs and regression coverage
  [W21-P04-T02-DOCS-REGRESSIONS]
  Currently: allocator APIs are easy to make technically correct but hard to use
  if the docs and diagnostics are not closed at the same time.
  Expected deliveries: allocator docs coverage and regression matrix covering
  success, failure, and duplicate-global-allocator cases.
  DoD: the allocator feature can be learned from the public docs tree and fails
  loudly when misused.
  Verify: `fuse doc --check stdlib/core/alloc/`

## Wave Closure Phase [W21-PCL-WAVE-CLOSURE]

- Task 01: Retire allocator stubs [W21-PCL-T01-RETIRE]
  Currently: any lingering allocator stub means W20 core containers still have
  hidden implementation debt.
  Expected deliveries: retired allocator stubs and updated stub history.
  DoD: `STUBS.md` contains no Active row for an allocator feature W21 claims to
  ship.
  Verify: `go run tools/checkstubs/main.go -wave W21`
- Task 02: WC021 entry [W21-PCL-T02-CLOSURE-LOG]
  Currently: allocator decisions affect every later wave that touches memory,
  so W21 closure context must be explicit.
  Expected deliveries: `WC021` learning-log entry naming allocator proofs,
  retired stubs, and constraints W22 must preserve.
  DoD: `docs/learning-log.md` contains a complete `WC021` entry aligned with
  this wave's proof commands.
  Verify: `grep "WC021" docs/learning-log.md`

