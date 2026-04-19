# Wave 27: Performance Gate

> Part of the [Fuse implementation plan](../implementation-plan.md).

Goal: prove speed. The project claims a compiled systems language with
competitive performance; this wave is where that claim is tested against
concrete thresholds, on both the bootstrap C11 path and the native path,
before Go and C are retired.

The W17-P13 perf baseline seeded `tests/perf/` with the first workloads
and thresholds. This wave grows the corpus, compares Fuse output against
named reference implementations (C, Rust), and turns the perf suite into
a release-blocking CI gate.

Entry criterion: W26 done. Phase 00 confirms no overdue stubs. Native
backend must be complete so comparisons are fair.

State on entry: `tests/perf/` contains the W17-P13 baseline corpus. There
is no release-gate comparison against external reference implementations.
Targets are recorded but not systematically reviewed.

Exit criteria:

- `tests/perf/` corpus covers: compiler self-host (lex + parse + check),
  monomorphization-heavy workload, arithmetic microbenchmarks, memory
  allocator/collection workloads, channel and spawn workloads, and a
  realistic end-user program (at least 5 kloc)
- each benchmark has a reference implementation in C (idiomatic,
  `-O2`-compiled) or Rust (`--release`) with identical semantics
- Fuse native-backend wall-clock is within a stated ratio of the
  reference for each benchmark; the ratio is recorded and enforced
- code-size thresholds recorded for each benchmark's release binary;
  regressions trigger CI failure
- compile-time thresholds recorded for (cold build, incremental build,
  `fuse check`) on a reference workload; regressions trigger CI failure
- memory-footprint ceilings recorded for the compiler itself on the
  reference workload
- perf results published to a tracked dashboard; drift is visible to
  every contributor, not buried in CI logs
- every perf threshold is justified: either matched against a named
  reference implementation, or anchored to a prior run with a commit
  message explaining the change

Proof of completion:

```
go test ./tests/perf/... -v
go test ./tests/perf/... -run TestRuntimePerfVsReference -v
go test ./tests/perf/... -run TestCompileTimeThresholds -v
go test ./tests/perf/... -run TestCodeSizeThresholds -v
go test ./tests/perf/... -run TestCompilerMemoryFootprint -v
go run tools/perfreport/main.go -verify-dashboard
```

## Phase 00: Stub Audit [W27-P00-STUB-AUDIT]

- Task 01: Perf audit [W27-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W27 -phase P00`

## Phase 01: Corpus Expansion [W27-P01-CORPUS]

- Task 01: Broaden the benchmark set [W27-P01-T01-BROADEN]
  DoD: the corpus covers the six categories named in the exit criteria.
  Each benchmark has a deterministic input generator (committed seeds,
  no wall-clock or randomness in output-affecting paths).
  Verify: `go test ./tests/perf/... -run TestCorpusCoverage -v`
- Task 02: Reference implementations committed [W27-P01-T02-REFS]
  DoD: each Fuse benchmark has a matched C or Rust implementation with
  identical observable semantics, committed under `tests/perf/refs/`.
  The build scripts for references are reproducible.
  Verify: `go test ./tests/perf/... -run TestReferenceImplementationsBuild -v`

## Phase 02: Runtime Performance Gate [W27-P02-RUNTIME]

- Task 01: Runtime ratio thresholds [W27-P02-T01-RATIOS]
  DoD: for each benchmark, `tests/perf/README.md` records the maximum
  allowed ratio `fuse_wall_clock / reference_wall_clock`. Exceeding the
  ratio fails CI. Thresholds are justified in commit messages; silent
  bumps are forbidden.
  Verify: `go test ./tests/perf/... -run TestRuntimePerfVsReference -v`
- Task 02: Code-size thresholds [W27-P02-T02-CODE-SIZE]
  DoD: release-binary size ceiling recorded per benchmark; stripped and
  unstripped variants both tracked.
  Verify: `go test ./tests/perf/... -run TestCodeSizeThresholds -v`

## Phase 03: Compile-Time Gate [W27-P03-COMPILE-TIME]

- Task 01: Cold-build threshold [W27-P03-T01-COLD-BUILD]
  DoD: `fuse build` on the reference workload completes under a
  declared wall-clock ceiling on the tier-1 CI host. Ceiling is in
  `tests/perf/README.md` with justification.
  Verify: `go test ./tests/perf/... -run TestColdBuildCeiling -v`
- Task 02: Incremental-build threshold [W27-P03-T02-INCR-BUILD]
  DoD: incremental `fuse build` after a one-function edit completes
  under a declared threshold ratio of the cold build. Tests the
  W04-P05/W18-P04 incremental machinery end-to-end against an actual
  performance budget.
  Verify: `go test ./tests/perf/... -run TestIncrementalBuildRatio -v`
- Task 03: `fuse check` latency [W27-P03-T03-CHECK-LATENCY]
  DoD: `fuse check` on the reference workload is fast enough to drive
  interactive tooling (W19 LSP). Ceiling recorded.
  Verify: `go test ./tests/perf/... -run TestCheckLatency -v`

## Phase 04: Compiler Memory Footprint [W27-P04-MEM]

- Task 01: Peak-RSS ceiling [W27-P04-T01-RSS]
  DoD: peak resident-set size of the compiler process on the reference
  workload has a declared ceiling.
  Verify: `go test ./tests/perf/... -run TestCompilerMemoryFootprint -v`

## Phase 05: Perf Dashboard [W27-P05-DASHBOARD]

- Task 01: Perf history log [W27-P05-T01-HISTORY]
  DoD: every CI run appends a structured perf record under a tracked
  path. Records include commit SHA, host identity, wall-clock, peak RSS,
  code size. History is queryable without re-running benchmarks.
  Verify: `go run tools/perfreport/main.go -verify-history`
- Task 02: Published dashboard [W27-P05-T02-DASHBOARD]
  DoD: a committed tool renders perf trends from the history log into a
  human-readable report. The report is produced on every release commit.
  Verify: `go run tools/perfreport/main.go -verify-dashboard`

## Wave Closure Phase [W27-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W27-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W27`
- Task 02: WC027 entry [W27-PCL-T02-CLOSURE-LOG]
  DoD: WC027 records the final perf ratios against each reference
  implementation, the compile-time budgets that passed, and any
  benchmarks that required threshold adjustments (with justification).
  Verify: `grep "WC027" docs/learning-log.md`
