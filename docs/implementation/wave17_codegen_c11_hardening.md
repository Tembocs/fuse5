# Wave 17: Codegen C11 Hardening

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: enforce every representation contract from reference §57; add
codegen for `@repr(C/packed/Uxx|Ixx)`, `@align(N)`, `@inline`, `@cold`,
compiler intrinsics, variadic call ABI, `Ptr.null[T]()`, `size_of[T]()`/
`align_of[T]()` emission at runtime positions, union layout, and overflow-
aware default policy.

Entry criterion: W16 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- two pointer categories separated (reference §57.1)
- unit erasure total (reference §57.2)
- monomorphization completeness enforced (reference §57.3)
- divergence structural (reference §57.4)
- composite types emitted before use (reference §57.5)
- identifier sanitization and module-qualified mangling (reference §57.6)
- closure representation (reference §57.7)
- trait object representation (reference §57.8)
- channel and thread runtime ABI (reference §57.9)
- alignment and padding stable (reference §57.10)
- `@repr(C)`, `@repr(packed)`, `@repr(Uxx|Ixx)`, `@align(N)` produce
  conforming C layouts
- `@inline`, `@inline(always)`, `@inline(never)`, `@cold` emit the
  corresponding backend annotations
- intrinsics (`unreachable`, `likely`, `unlikely`, `fence`, `prefetch`,
  `assume`) emit correctly
- variadic extern calls follow platform C variadic ABI
- `Ptr.null[T]()` emits `((T*)0)`
- `size_of[T]()` / `align_of[T]()` in runtime position emit literal
- union type emission (largest-field sizing, strictest alignment)
- overflow-aware default: debug panics; release is deterministic per
  target profile
- determinism gates: same source → byte-identical C
- every regression from L001–L015 has a test

Proof of completion:

```
go test ./compiler/codegen/... -v
go test ./compiler/codegen/... -run TestPointerCategories -v
go test ./compiler/codegen/... -run TestTotalUnitErasure -v
go test ./compiler/codegen/... -run TestStructuralDivergence -v
go test ./compiler/codegen/... -run TestIdentifierSanitization -v
go test ./compiler/codegen/... -run TestModuleMangling -v
go test ./compiler/codegen/... -run TestAggregateZeroInit -v
go test ./compiler/codegen/... -run TestReprEmission -v
go test ./compiler/codegen/... -run TestAlignEmission -v
go test ./compiler/codegen/... -run TestInlineEmission -v
go test ./compiler/codegen/... -run TestIntrinsicsEmission -v
go test ./compiler/codegen/... -run TestVariadicCall -v
go test ./compiler/codegen/... -run TestPtrNullEmission -v
go test ./compiler/codegen/... -run TestSizeOfEmission -v
go test ./compiler/codegen/... -run TestUnionLayout -v
go test ./compiler/codegen/... -run TestOverflowPolicy -v
make repro
```

## Phase 00: Stub Audit [W17-P00-STUB-AUDIT]

- Task 01: Codegen audit [W17-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W17 -phase P00`

## Phase 01: Type Emission [W17-P01-TYPES]

- Task 01: Types before use [W17-P01-T01-DEFS-FIRST]
  Verify: `go test ./compiler/codegen/... -run TestTypeDefsFirst -v`
- Task 02: Identifier sanitization [W17-P01-T02-SANITIZE]
  Verify: `go test ./compiler/codegen/... -run TestIdentifierSanitization -v`
- Task 03: Module-qualified mangling [W17-P01-T03-MANGLE]
  Verify: `go test ./compiler/codegen/... -run TestModuleMangling -v`

## Phase 02: Pointer Categories [W17-P02-POINTERS]

- Task 01: Two pointer categories [W17-P02-T01-TWO]
  Verify: `go test ./compiler/codegen/... -run TestPointerCategories -v`
- Task 02: Call-site adaptation [W17-P02-T02-CALL-SITE]
  Verify: `go test ./compiler/codegen/... -run TestCallSiteAdaptation -v`
- Task 03: `Ptr.null[T]()` emission [W17-P02-T03-NULL]
  Verify: `go test ./compiler/codegen/... -run TestPtrNullEmission -v`

## Phase 03: Unit and Aggregate [W17-P03-UNIT-AGG]

- Task 01: Total unit erasure [W17-P03-T01-UNIT]
  Verify: `go test ./compiler/codegen/... -run TestTotalUnitErasure -v`
- Task 02: Typed aggregate fallback [W17-P03-T02-AGG]
  Verify: `go test ./compiler/codegen/... -run TestAggregateZeroInit -v`
- Task 03: Union layout [W17-P03-T03-UNION]
  Verify: `go test ./compiler/codegen/... -run TestUnionLayout -v`

## Phase 04: Divergence [W17-P04-DIV]

- Task 01: Structural divergence [W17-P04-T01-DIV]
  Verify: `go test ./compiler/codegen/... -run TestStructuralDivergence -v`

## Phase 05: Layout Control Emission [W17-P05-LAYOUT]

- Task 01: `@repr(C)` / `@repr(packed)` / `@repr(Uxx|Ixx)` emission
  [W17-P05-T01-REPR]
  Verify: `go test ./compiler/codegen/... -run TestReprEmission -v`
- Task 02: `@align(N)` emission [W17-P05-T02-ALIGN]
  Verify: `go test ./compiler/codegen/... -run TestAlignEmission -v`

## Phase 06: Inline and Cold Annotations [W17-P06-INLINE]

- Task 01: `@inline` / `@inline(always)` / `@inline(never)` / `@cold`
  [W17-P06-T01-INLINE]
  DoD: emit corresponding C compiler annotations (`inline`, `__attribute__
  ((always_inline))`, `__attribute__((noinline))`, `__attribute__((cold))`
  or platform equivalents).
  Verify: `go test ./compiler/codegen/... -run TestInlineEmission -v`

## Phase 07: Compiler Intrinsics [W17-P07-INTRINSICS]

- Task 01: `unreachable`, `likely`, `unlikely` [W17-P07-T01-BUILTINS]
  Verify: `go test ./compiler/codegen/... -run TestIntrinsicsEmission -v`
- Task 02: `fence`, `prefetch`, `assume` [W17-P07-T02-MEM-INTRINSICS]
  Verify: `go test ./compiler/codegen/... -run TestMemIntrinsicsEmission -v`

## Phase 08: Variadic Call ABI [W17-P08-VARIADIC]

- Task 01: Variadic call site ABI [W17-P08-T01-VARIADIC]
  DoD: variadic extern calls follow the host platform's C variadic ABI;
  float → double promotion and short-int → int promotion applied at call
  site.
  Verify: `go test ./compiler/codegen/... -run TestVariadicCall -v`

## Phase 09: Memory Intrinsics Emission [W17-P09-MEM-EMISSION]

- Task 01: `size_of[T]()` / `align_of[T]()` emission [W17-P09-T01-SIZEOF]
  DoD: in runtime positions these lower to literal `USize` values.
  Verify: `go test ./compiler/codegen/... -run TestSizeOfEmission -v`
- Task 02: `size_of_val(ref v)` [W17-P09-T02-SIZEOF-VAL]
  Verify: `go test ./compiler/codegen/... -run TestSizeOfValEmission -v`

## Phase 10: Overflow Default Policy [W17-P10-OVERFLOW]

- Task 01: Debug-mode overflow panic [W17-P10-T01-DEBUG-PANIC]
  Verify: `go test ./compiler/codegen/... -run TestOverflowDebugPanic -v`
- Task 02: Release-mode deterministic policy [W17-P10-T02-RELEASE-POLICY]
  DoD: default release behavior is documented (e.g. wrapping) and
  deterministic per target profile; CI goldens pin the choice.
  Verify: `go test ./compiler/codegen/... -run TestOverflowPolicy -v`

## Phase 11: Regressions and Determinism [W17-P11-REG-DET]

- Task 01: L001–L015 regression coverage [W17-P11-T01-REGRESSIONS]
  Verify: `go test ./compiler/codegen/... -run TestHistoricalRegressions -v`
- Task 02: Reproducibility [W17-P11-T02-REPRO]
  Verify: `make repro`

## Wave Closure Phase [W17-PCL-WAVE-CLOSURE]

- Task 01: Retire codegen stubs [W17-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W17`
- Task 02: WC017 entry [W17-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC017" docs/learning-log.md`

