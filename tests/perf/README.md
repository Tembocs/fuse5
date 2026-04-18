# Fuse Performance Corpus

W17 Phase 13 establishes the performance baseline. Each benchmark in
this directory has an input, a driver in `perf_test.go`, and a
wall-clock ceiling (or ratio to a named reference) recorded in the
`thresholds.json` table below. Exceeding the ceiling on the tier-1
CI host fails CI.

W27 Performance Gate is the full gate — it extends these thresholds
with compile-time budgets, code-size ceilings, and memory-footprint
checks. W17's role is the first honest measurement so the project
has numbers to defend.

## Benchmarks

| Benchmark                       | Driver                     | Ceiling (tier-1 CI)              | Rationale                                            |
|---------------------------------|----------------------------|----------------------------------|------------------------------------------------------|
| lex+parse a 50kloc-equivalent   | `BenchmarkLexParseCorpus`  | 500 ms wall, 64 MB peak RSS      | compiler self-host proxy                             |
| monomorph-heavy program         | `BenchmarkMonomorphHeavy`  | 100 ms wall                      | worst-case generic instantiation (W08 stress path)   |
| tight arithmetic loop           | `BenchmarkTightArith`      | 100 ns/op                        | integer arithmetic throughput                        |
| channel + spawn round trip      | `BenchmarkChanSpawnRT`     | 50 ms per 1000 messages          | runtime-ABI critical path                            |

Thresholds are deliberately generous at W17. W27 tightens them after
a longer observation window across the CI matrix. A test that flags
a regression should surface a log line including commit SHA, host,
wall-clock, and peak memory — enough metadata to spot drift without
restoring every bar chart.

## CI wiring

`tools/checkci -perf-baseline` asserts this directory has every
benchmark named in the table above and every declared threshold is
a positive number. CI runs `go test ./tests/perf/... -run
TestPerfBaseline -v` and uploads the resulting log as a build
artifact. Perf-only failures do not block PRs at W17; they surface
as a soft-warn annotation. W27 flips the gate from soft to hard.
