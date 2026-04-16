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
- debug info emitted through the C backend: `-g` passthrough plus a
  Fuse→C line mapping good enough for `gdb`/`lldb` to break on a Fuse
  source line and print a local by Fuse-level name
- performance baseline established: `tests/perf/` seeded with benchmarks
  for the compiler's own hot paths and for user programs; thresholds
  recorded in `tests/perf/README.md` and gated in CI

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
go test ./compiler/codegen/... -run TestDebugInfoEmission -v
go test ./tests/e2e/... -run TestDebugBreakpointInGdb -v
go test ./tests/perf/... -run TestPerfBaseline -v
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

## Phase 12: Debug Info Through C Backend [W17-P12-DEBUG-INFO]

Fuse's developer-experience promise requires source-level debugging from
the first shipping compiler. Bootstrap debug info rides on the C backend:
the emitted C preserves original Fuse line numbers via `#line` directives,
and the C compiler is invoked with `-g` so the native debugger sees Fuse
sources directly.

- Task 01: `#line` directive emission [W17-P12-T01-LINE-DIRECTIVES]
  DoD: every MIR-derived C statement is preceded by a `#line N "path.fuse"`
  directive when the originating HIR node has a span. Synthesized code
  with no originating span is explicitly marked.
  Verify: `go test ./compiler/codegen/... -run TestLineDirectives -v`
- Task 02: `-g` passthrough and build integration [W17-P12-T02-GFLAG]
  DoD: `fuse build --debug` passes `-g` to the C compiler; `fuse build`
  without `--debug` does not. Release builds stay free of debug bloat.
  Verify: `go test ./compiler/cc/... -run TestDebugFlagPassthrough -v`
- Task 03: Fuse-named local visibility [W17-P12-T03-LOCAL-NAMES]
  DoD: local variable names in emitted C preserve their Fuse-level
  identifiers (through sanitization), so `gdb` / `lldb` can read them
  by the name the user wrote.
  Verify: `go test ./compiler/codegen/... -run TestDebugInfoEmission -v`
- Task 04: End-to-end gdb breakpoint proof [W17-P12-T04-GDB-PROOF]
  DoD: an e2e test compiles `tests/e2e/debug_breakpoint.fuse` with
  `--debug`, runs it under a scripted `gdb` session, breaks on a Fuse
  source line, reads a local by Fuse name, and confirms the values.
  Test is skipped if no debugger is available on the host but runs in CI
  on Linux at minimum.
  Verify: `go test ./tests/e2e/... -run TestDebugBreakpointInGdb -v`

## Phase 13: Performance Baseline [W17-P13-PERF-BASELINE]

The project claims speed; the project must measure speed. This phase
establishes `tests/perf/` and commits thresholds that the rest of the
project agrees to hold. W27 later becomes the full gate; this phase is the
first honest measurement.

- Task 01: Create `tests/perf/` corpus [W17-P13-T01-CORPUS]
  DoD: the corpus contains at minimum: (a) a compiler self-host proxy
  (lexing and parsing a 50kloc corpus), (b) a monomorphization-heavy
  workload, (c) a tight arithmetic loop, (d) a channel-and-spawn workload.
  Each benchmark has an input, a driver, and a threshold.
  Verify: `go test ./tests/perf/... -run TestPerfCorpusPresent -v`
- Task 02: Record baseline thresholds [W17-P13-T02-THRESHOLDS]
  DoD: `tests/perf/README.md` lists every benchmark with its wall-clock
  ceiling (or ratio to a named C reference) for the tier-1 CI host. A
  regression past the ceiling fails CI.
  Verify: `go test ./tests/perf/... -run TestPerfBaseline -v`
- Task 03: CI wiring [W17-P13-T03-CI]
  DoD: the perf suite runs on the tier-1 CI host. Results are logged
  with sufficient metadata (commit, host, wall-clock, peak memory) to
  detect drift over time.
  Verify: `go run tools/checkci/main.go -perf-baseline`

## Wave Closure Phase [W17-PCL-WAVE-CLOSURE]

- Task 01: Retire codegen stubs [W17-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W17`
- Task 02: WC017 entry [W17-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC017" docs/learning-log.md`

