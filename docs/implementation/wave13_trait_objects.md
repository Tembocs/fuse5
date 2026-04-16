# Wave 13: Trait Objects (`dyn Trait`)

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: implement dynamic dispatch via `dyn Trait`. Fat-pointer representation,
vtable layout, object safety rules, multi-trait object type, and proof
program.

Entry criterion: W12 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- object-safety checker rejects non-object-safe traits at `dyn Trait` use
  sites (reference §48.1)
- `dyn Trait` lowers to a fat pointer `{ data: Ptr[()], vtable:
  Ptr[Vtable_Trait] }` (reference §57.8)
- vtables emitted as static read-only tables with deterministic layout
  `[size, align, drop_fn, method_1, ...]` (reference §57.8)
- `dyn A + B` produces combined vtable (alphabetical trait order)
- ownership forms `ref dyn Trait`, `mutref dyn Trait`, `owned dyn Trait`
  lower correctly
- proof program: heterogeneous collection of `owned dyn Trait` values
  dispatches correctly

Proof of completion:

```
go test ./compiler/check/... -run TestObjectSafety -v
go test ./compiler/lower/... -run TestDynTraitFatPointer -v
go test ./compiler/codegen/... -run TestVtableEmission -v
go test ./compiler/codegen/... -run TestDynTraitMulti -v
go test ./tests/e2e/... -run TestDynDispatchProof -v
```

## Phase 00: Stub Audit [W13-P00-STUB-AUDIT]

- Task 01: `dyn Trait` audit [W13-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W13 -phase P00`

## Phase 01: Object Safety [W13-P01-OBJECT-SAFETY]

- Task 01: Object-safety checker [W13-P01-T01-OBJECT-SAFETY]
  DoD: traits with generic methods, Self in non-receiver positions,
  associated constants, or non-ref/mutref/owned receivers are rejected at
  `dyn Trait` use sites.
  Verify: `go test ./compiler/check/... -run TestObjectSafety -v`

## Phase 02: Fat Pointer Lowering [W13-P02-FAT-POINTER]

- Task 01: Lower `dyn Trait` to fat pointer [W13-P02-T01-FAT]
  Verify: `go test ./compiler/lower/... -run TestDynTraitFatPointer -v`
- Task 02: Lower `ref dyn Trait`, `mutref dyn Trait`, `owned dyn Trait`
  [W13-P02-T02-OWNERSHIP-FORMS]
  Verify: `go test ./compiler/lower/... -run TestDynOwnershipForms -v`

## Phase 03: Vtable Emission [W13-P03-VTABLE]

- Task 01: Deterministic vtable layout [W13-P03-T01-VTABLE-LAYOUT]
  DoD: per (trait, concrete impl), emit a static vtable with fields
  `[size, align, drop_fn, method_1, ...]` in trait declaration order.
  Verify: `go test ./compiler/codegen/... -run TestVtableEmission -v`
- Task 02: Multi-trait combined vtable [W13-P03-T02-MULTI-TRAIT]
  DoD: `dyn A + B` produces a combined vtable; trait ordering is
  alphabetical for determinism.
  Verify: `go test ./compiler/codegen/... -run TestDynTraitMulti -v`

## Phase 04: Method Dispatch Lowering [W13-P04-DISPATCH]

- Task 01: Call through fat pointer [W13-P04-T01-DISPATCH]
  DoD: a method call on a `dyn Trait` receiver loads the method pointer
  from the vtable and calls it with the data pointer as receiver.
  Verify: `go test ./compiler/codegen/... -run TestDynMethodDispatch -v`

## Phase 05: Dynamic Dispatch Proof [W13-P05-PROOF]

- Task 01: `dyn_dispatch.fuse` [W13-P05-T01-PROOF]
  DoD: a program builds a heterogeneous `List[owned dyn Draw]` containing
  two different implementations, iterates it, calls a method that sums
  into the exit code. The exit code proves both implementations ran.
  Verify: `go test ./tests/e2e/... -run TestDynDispatchProof -v`

## Wave Closure Phase [W13-PCL-WAVE-CLOSURE]

- Task 01: Retire `dyn Trait` stubs [W13-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W13`
- Task 02: WC013 entry [W13-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC013" docs/learning-log.md`

