# Fuse Implementation Plan

> Status: normative for Fuse.
>
> This document is the build plan from an empty repository to a self-hosting
> Fuse compiler and the later retirement of bootstrap-only implementation
> languages.

## Overview

Fuse is implemented in stages.

- Stage 1 compiler: Go
- Runtime during bootstrap: C
- Stage 2 compiler: Fuse

The bootstrap stack is fixed. Go and C are allowed during bootstrap because
the project must reach a self-hosted Fuse compiler as quickly and safely as
possible. After Stage 2 compiles itself reliably and a native backend is
stable, Go and C are retired from the compiler implementation path.

The C11 backend is therefore bootstrap infrastructure, not the terminal
backend. Design decisions in HIR, MIR, type identity, ownership analysis, and
pass structure must not depend on C11 in a way that would block the later
native backend.

## Working principles

1. Correctness precedes velocity.
2. Structural fixes beat symptom fixes.
3. No workarounds are allowed in compiler, runtime, or stdlib.
4. Stdlib is the compiler's semantic stress test.
5. Every wave has explicit entry and exit criteria.
6. Every task must be small enough to review and verify directly.
7. Every wave that introduces user-visible behavior owns an end-to-end proof
   program, starting from Wave 05 when the minimal end-to-end spine exists.
8. Features that historically were silently stubbed — monomorphization,
   pattern matching, error propagation, closures, concurrency — each have a
   dedicated wave and their own proof program. No such feature is a phase
   inside somebody else's wave.
9. Every feature in the language reference is scheduled to a concrete wave
   (Rule 2.5). No `Wave TBD`, no "post-v1".
10. Before self-hosting begins, a dedicated Stub Clearance Gate wave
    (W24) confirms the Active stubs table is empty. No stub reaches the
    bootstrap.
11. Developer experience is a first-class promise. Language Server (W19),
    incremental compile (W04+W18), source-level debugging (W17+W26),
    package management (W23), and user-facing documentation (W30) each
    have dedicated schedule entries. The project does not ship with DX
    gaps it cannot defend.
12. Speed is measured, not assumed. The performance corpus (W17-P13) is
    populated during bootstrap and gated by a dedicated Performance Gate
    wave (W27) before Go and C are retired. Thresholds are enforced in
    CI against named reference implementations.
13. Every user-facing compiler diagnostic carries a primary span, a
    one-line explanation, and a suggestion where one is possible
    (Rule 6.17). Diagnostic quality is a correctness concern, not polish.

## Naming conventions

The plan uses globally unique identifiers.

- Wave headings:
  `Wave 06: Type Checking`
- Phase headings:
  `Phase 03: Trait Resolution and Bound Dispatch [W06-P03-TRAIT-RESOLUTION]`
- Task headings:
  `Task 01: Register All Function Types Before Body Checking [W06-P01-T01-FN-TYPE-REGISTRATION]`

All wave, phase, and task numbers are zero-padded.

## Task format

Every task in this plan must be written with:

- a short goal
- a `Currently:` line naming what exists at the start of the task (file:line
  if the code exists, or "not yet started" if the package is empty)
- an exact definition of done
- a `Verify:` line giving the specific command that proves the DoD is met;
  this command must fail if the task has not been completed and must be run
  before the task is marked done
- required regression coverage
- clear scope boundaries

```
Task 01: ... [Wxx-Pyy-Tzz-...]
Currently: ...
DoD: verifiable completion rule.
Verify: go test ./compiler/pkg/... -run TestName -v
```

A Verify command is not satisfied by "it looks correct" or "unit tests pass".
It is satisfied by running the named command and observing the declared
passing output. The agent or contributor executing the task must record the
actual output of the Verify command before marking the task done.

`Verify:` commands must be runnable on Linux, macOS, and Windows. Unix-only
shell forms (process substitution, `/tmp/...` hardcoded paths, bash-specific
features) are forbidden in normative verification steps. Use Go tests,
project-owned scripts under `tools/`, or explicit per-platform wrappers
instead.

## Wave format

Every wave contains:

- **Goal**: one paragraph summary
- **Entry criterion**: what must be true before this wave begins
- **State on entry**: what the codebase actually looks like when the wave
  begins (which packages are empty, which are stubs, which are partial)
- **Exit criteria**: behavioral and structural requirements that must all be
  true
- **Proof of completion**: the specific commands that, when all passing,
  prove the wave is done; these are run in CI and locally before sign-off
- **Phase 00: Stub Audit**: first phase of every wave; committed to STUBS.md
  before other phases begin. This phase must verify no prior wave's stubs
  are overdue (Rule 6.15).
- One or more implementation phases (P01, P02, ...)
- **Wave Closure Phase (PCL)**: last phase of every wave; produces the WCxxx
  learning-log entry, updates STUBS.md with the wave's stub history block
  (Rule 6.16), and confirms the Active stubs table is current.

## Waves at a glance

| Wave | Theme | Entry criterion | Exit criterion |
|---|---|---|---|
| [00](implementation/wave00_governance.md) | Governance and Phase Model | — | build, test, CI, docs scaffold, STUBS.md |
| [01](implementation/wave01_lexer.md) | Lexer | W00 done | every token kind and lexical ambiguity covered |
| [02](implementation/wave02_parser_and_ast.md) | Parser and AST | W01 done | all language constructs parse deterministically |
| [03](implementation/wave03_resolution.md) | Resolution | W02 done | module graph, imports, symbols resolved; `@cfg`; visibility |
| [04](implementation/wave04_hir_and_typetable.md) | HIR and TypeTable | W03 done | typed HIR shape, pass graph, incremental-compile foundation |
| [05](implementation/wave05_minimal_end_to_end_spine.md) | Minimal End-to-End Spine | W04 done | `fn main() -> I32 { return N; }` runs and exits with N |
| [06](implementation/wave06_type_checking.md) | Type Checking | W05 done | stdlib and user bodies type-check with no unknowns |
| [07](implementation/wave07_concurrency_semantics.md) | Concurrency Semantics | W06 done | Send/Sync, Chan[T], spawn, ThreadHandle enforced by checker |
| [08](implementation/wave08_monomorphization.md) | Monomorphization | W07 done | generics compile end-to-end with proof programs |
| [09](implementation/wave09_ownership_and_liveness.md) | Ownership and Liveness | W08 done | borrow rules (field / return / aliasing / use-after-move) enforced; drop codegen with proof |
| [10](implementation/wave10_pattern_matching.md) | Pattern Matching | W09 done | match dispatch, exhaustiveness, proof program |
| [11](implementation/wave11_error_propagation.md) | Error Propagation | W10 done | `?` branch lowering with proof program |
| [12](implementation/wave12_closures_and_callable_traits.md) | Closures and Callable Traits | W11 done | capture, lift, env struct, Fn/FnMut/FnOnce with proof |
| [13](implementation/wave13_trait_objects.md) | Trait Objects (`dyn Trait`) | W12 done | fat pointer, vtable layout, object-safe rules, proof |
| [14](implementation/wave14_compile_time_evaluation.md) | Compile-Time Evaluation (`const fn`) | W13 done | const evaluator over checked HIR with proof |
| [15](implementation/wave15_lowering_and_mir_consolidation.md) | Lowering and MIR Consolidation | W14 done | MIR invariants; casts, fn-pointers, slice range, struct update all lowered |
| [16](implementation/wave16_runtime_abi.md) | Runtime ABI | W15 done | full runtime replaces stub; IO + threading work |
| [17](implementation/wave17_codegen_c11_hardening.md) | Codegen C11 Hardening | W16 done | backend contracts enforced; debug info via C; perf baseline seeded |
| [18](implementation/wave18_cli_and_diagnostics.md) | CLI and Diagnostics | W17 done | `fuse build/run/check/test/fmt/doc/repl` + diagnostic-quality audit + incremental driver |
| [19](implementation/wave19_language_server.md) | Language Server | W18 done | LSP 3.17 server: diagnostics, hover, goto-def, completion, symbols |
| [20](implementation/wave20_stdlib_core.md) | Stdlib Core | W19 done | core traits, primitives, strings, collections, Cell/RefCell, Ptr.null, overflow methods |
| [21](implementation/wave21_custom_allocators.md) | Custom Allocators | W20 done | Allocator trait; collections accept allocator; proof program with bump allocator |
| [22](implementation/wave22_stdlib_hosted.md) | Stdlib Hosted | W21 done | IO, fs, os, time, thread, sync, channels, network ship |
| [23](implementation/wave23_package_management.md) | Package Management | W22 done | manifest, lockfile, resolver, fetcher, registry protocol, two-crate proof |
| [24](implementation/wave24_stub_clearance_gate.md) | Stub Clearance Gate | W23 done | Active stubs table is empty; no stub reaches Stage 2 |
| [25](implementation/wave25_stage2_and_self_hosting.md) | Stage 2 and Self-Hosting | W24 done | stage1 compiles stage2; stage2 compiles itself reproducibly |
| [26](implementation/wave26_native_backend_transition.md) | Native Backend Transition | W25 done | stage2 compiles without C11 backend dependency; native DWARF |
| [27](implementation/wave27_performance_gate.md) | Performance Gate | W26 done | runtime ratios, compile-time budgets, code-size, memory footprint gated in CI |
| [28](implementation/wave28_retirement_of_go_and_c.md) | Retirement of Go and C | W27 done | Fuse owns the compiler implementation path |
| [29](implementation/wave29_targets_and_native_expansion.md) | Targets and Native Expansion | W28 done | cross-target matrix and `stdlib/ext/` on native base |
| [30](implementation/wave30_ecosystem_documentation.md) | Ecosystem Documentation | W29 done | tutorial, book, migration guides, ecosystem guide, published docs site |

Every feature documented in `docs/fuse-language-reference.md` is scheduled
to one or more of the waves above. No feature is deferred to a later
version.


## Per-wave plans

Full detail for each wave — goal, entry criterion, state on entry, exit criteria, proof of completion, implementation phases, and wave closure phase — lives in [`docs/implementation/`](implementation/). One file per wave.

- [Wave 00: Governance and Phase Model](implementation/wave00_governance.md)
- [Wave 01: Lexer](implementation/wave01_lexer.md)
- [Wave 02: Parser and AST](implementation/wave02_parser_and_ast.md)
- [Wave 03: Resolution](implementation/wave03_resolution.md)
- [Wave 04: HIR and TypeTable](implementation/wave04_hir_and_typetable.md)
- [Wave 05: Minimal End-to-End Spine](implementation/wave05_minimal_end_to_end_spine.md)
- [Wave 06: Type Checking](implementation/wave06_type_checking.md)
- [Wave 07: Concurrency Semantics](implementation/wave07_concurrency_semantics.md)
- [Wave 08: Monomorphization](implementation/wave08_monomorphization.md)
- [Wave 09: Ownership and Liveness](implementation/wave09_ownership_and_liveness.md)
- [Wave 10: Pattern Matching](implementation/wave10_pattern_matching.md)
- [Wave 11: Error Propagation](implementation/wave11_error_propagation.md)
- [Wave 12: Closures and Callable Traits](implementation/wave12_closures_and_callable_traits.md)
- [Wave 13: Trait Objects (`dyn Trait`)](implementation/wave13_trait_objects.md)
- [Wave 14: Compile-Time Evaluation (`const fn`)](implementation/wave14_compile_time_evaluation.md)
- [Wave 15: Lowering and MIR Consolidation](implementation/wave15_lowering_and_mir_consolidation.md)
- [Wave 16: Runtime ABI](implementation/wave16_runtime_abi.md)
- [Wave 17: Codegen C11 Hardening](implementation/wave17_codegen_c11_hardening.md)
- [Wave 18: CLI and Diagnostics](implementation/wave18_cli_and_diagnostics.md)
- [Wave 19: Language Server](implementation/wave19_language_server.md)
- [Wave 20: Stdlib Core](implementation/wave20_stdlib_core.md)
- [Wave 21: Custom Allocators](implementation/wave21_custom_allocators.md)
- [Wave 22: Stdlib Hosted](implementation/wave22_stdlib_hosted.md)
- [Wave 23: Package Management](implementation/wave23_package_management.md)
- [Wave 24: Stub Clearance Gate](implementation/wave24_stub_clearance_gate.md)
- [Wave 25: Stage 2 and Self-Hosting](implementation/wave25_stage2_and_self_hosting.md)
- [Wave 26: Native Backend Transition](implementation/wave26_native_backend_transition.md)
- [Wave 27: Performance Gate](implementation/wave27_performance_gate.md)
- [Wave 28: Retirement of Go and C](implementation/wave28_retirement_of_go_and_c.md)
- [Wave 29: Targets and Native Expansion](implementation/wave29_targets_and_native_expansion.md)
- [Wave 30: Ecosystem Documentation](implementation/wave30_ecosystem_documentation.md)

## Cross-cutting constraints

The following rules apply to every wave.

- Determinism is a release-level requirement (Rule 7.1).
- No unresolved types may reach codegen (Rule 3.9).
- No pass may recompute liveness independently (Rule 3.8).
- Invariant walkers remain enabled in debug and CI (Rule 3.5).
- Stdlib failures are compiler signals, not library excuses (Rule 5.1).
- Workarounds are forbidden (Rule 4.2).
- Each non-trivial bug produces both a regression and a learning-log entry
  (Rules 4.3, 4.4).
- Every wave that introduces user-visible behavior includes at least one
  end-to-end proof program that fails if the feature is stubbed (Rule 6.8).
- Exit criteria include behavioral requirements, not only structural ones
  (Rule 6.10).
- Every task has `Currently:`, DoD, and `Verify:`. Verify commands must be
  portable across Linux, macOS, and Windows.
- Stubs emit diagnostics, not silent defaults (Rule 6.9).
- Every wave begins with a Phase 00 Stub Audit (Rule 6.12) and ends with a
  Wave Closure Phase (Rule 6.14).
- STUBS.md is updated at every wave boundary (Rule 6.13) with Active stubs
  table and Stub history append (Rule 6.16).
- Overdue stubs block wave entry (Rule 6.15).
- Every feature in the language reference is scheduled to a concrete wave
  (Rule 2.5). No `Wave TBD`, no "post-v1".
- `tests/e2e/README.md` is updated whenever a proof program is added.
- `tests/perf/README.md` is updated whenever a benchmark is added;
  threshold changes are justified in commit messages (Rule 6.17 also
  applies to perf regressions masked as routine).
- Every compiler diagnostic carries a primary span, a one-line
  explanation, and a suggestion where applicable (Rule 6.17).
- W24 Stub Clearance Gate, W25 Stage 2, and W26 Native Backend all
  require the Active stubs table to be empty at entry.
- W27 Performance Gate must pass before W28 retires the bootstrap
  language surface.
