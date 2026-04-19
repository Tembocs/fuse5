# Wave 25: Stage 2 and Self-Hosting

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: bring up the self-hosted Fuse compiler and prove it compiles itself
reproducibly.

Pillar alignment:

- Memory safety: the stage2 compiler must preserve the same ownership,
  destruction, and layout contracts as the stage1 compiler rather than relying
  on bootstrap-language escape hatches.
- Concurrency safety: any concurrency-sensitive compiler/runtime surfaces
  carried into stage2 must preserve the W07/W16 contracts; self-hosting is not
  permission to regress them.
- Developer experience: the stage2 compiler must be a usable replacement for
  the stage1 compiler, not merely a minimally self-compiling artifact.

Entry criterion: W24 done. Phase 00 re-verifies that no overdue stubs
block self-hosting entry.

State on entry: `stage2/src/` is empty. All compiler infrastructure is
complete in stage1. STUBS.md may still contain W25+ tracker rows, but
self-hosting is not allowed to begin with overdue stubs or silent stub
behavior.

Exit criteria:

- the stage2 source tree owns a complete compiler package roster rather than a
  partial demo compiler
- stage1 compiles stage2 end-to-end into a usable `fusec2`
- the stage1-built `fusec2` can run the bootstrap-oriented test surface
- stage2 recompiles itself
- reproducibility checks pass

Proof of completion:

```
fuse check stage2/src/compiler/lex/...
fuse check stage2/src/compiler/check/...
fuse check stage2/src/compiler/codegen/...
fuse build stage2/src/... -o stage2_out/fusec2
stage2_out/fusec2 check stage2/src/compiler/lex/...
stage2_out/fusec2 build stage2/src/... -o stage2_out/fusec2_gen2
make repro
go test ./tests/bootstrap/... -v
```

## Phase 00: Stub Audit [W25-P00-STUB-AUDIT]

- Task 01: Self-hosting audit [W25-P00-T01-ZERO-STUBS]
  Currently: the stage2 bring-up can only start if the bootstrap compiler is
  free of overdue residue from earlier waves.
  Expected deliveries: proof that no overdue stub blocks W25 entry and
  that remaining Active rows still name deliberate W25+ work.
  DoD: no Active row retires before W25. If an overdue row exists, the
  wave cannot begin and the missing retirement or reschedule is fixed
  before self-hosting work starts.
  Verify: `go run tools/checkstubs/main.go -wave W25 -phase P00`

## Phase 01: Stage2 Tree, Package Roster, and Build Skeleton [W25-P01-SCAFFOLD]

- Task 01: Freeze the stage2 package roster [W25-P01-T01-ROSTER]
  Currently: `stage2/src/` is empty, so self-hosting can easily drift into a
  partial subset compiler unless package ownership is frozen first.
  Expected deliveries: a package map for the stage2 compiler tree covering the
  compiler pipeline, driver, and command surface required for self-hosting.
  DoD: stage2 has an explicit source-tree roster rather than a vague "port the
  compiler" directive.
  Verify: `fuse check stage2/src/...`
- Task 02: Port the stage2 build skeleton [W25-P01-T02-BUILD-SKELETON]
  Currently: no stage2 build entrypoint or shared infrastructure exists.
  Expected deliveries: stage2 directory layout, build wiring, and foundational
  support code sufficient for package-by-package bring-up.
  DoD: stage2 packages can be checked and built incrementally instead of only as
  one final end-to-end port step.
  Verify: `fuse check stage2/src/compiler/lex/...`

## Phase 02: Frontend Pipeline Port [W25-P02-FRONTEND]

- Task 01: Port the syntax and parse front end [W25-P02-T01-SYNTAX]
  Currently: lexer, parser, and AST sources do not exist in stage2.
  Expected deliveries: stage2 `lex`, `parse`, and supporting syntax packages.
  DoD: the stage2 compiler can check its syntax front end under stage1 without
  depending on later pipeline packages.
  Verify: `fuse check stage2/src/compiler/lex/...`
- Task 02: Port resolution and checked-front-end packages
  [W25-P02-T02-SEMANTIC-FRONTEND]
  Currently: resolution, HIR building, checking, and const-eval are absent in
  stage2.
  Expected deliveries: stage2 `resolve`, `hir`, `check`, and `consteval`
  package set.
  DoD: the semantic front end compiles under stage1 as a coherent stage2 slice,
  independent of backend bring-up.
  Verify: `fuse check stage2/src/compiler/check/...`

## Phase 03: Mid-End and Backend Port [W25-P03-BACKEND]

- Task 01: Port analysis and lowering packages [W25-P03-T01-MIDEND]
  Currently: monomorphization, liveness, MIR, and lowering have not been
  recreated in stage2.
  Expected deliveries: stage2 `monomorph`, `liveness`, `mir`, and `lower`
  packages.
  DoD: the mid-end is complete enough that code generation can depend on it
  without unresolved stage1-only fallbacks.
  Verify: `fuse check stage2/src/compiler/lower/...`
- Task 02: Port backend-facing packages [W25-P03-T02-CODEGEN]
  Currently: code generation, host-compiler coordination, and runtime-interface
  glue are absent in stage2.
  Expected deliveries: stage2 `codegen`, `cc`, and backend-support packages.
  DoD: backend work is no longer hidden inside a single end-to-end build step.
  Verify: `fuse check stage2/src/compiler/codegen/...`

## Phase 04: Driver and Command Surface Port [W25-P04-DRIVER-CLI]

- Task 01: Port the compiler driver [W25-P04-T01-DRIVER]
  Currently: stage2 has no build/check orchestration surface yet.
  Expected deliveries: stage2 `compiler/driver` package and bootstrap build
  entrypoint behavior.
  DoD: stage2 can orchestrate the compiler pipeline rather than only compiling
  isolated packages.
  Verify: `fuse check stage2/src/compiler/driver/...`
- Task 02: Port the stage2 CLI surface [W25-P04-T02-CLI]
  Currently: the self-hosted compiler still lacks its user-visible command
  surface.
  Expected deliveries: stage2 `cmd/fuse` with at least the bootstrap command set
  required for self-compilation.
  DoD: the stage2 command surface is usable enough to drive the bootstrap flow.
  Verify: `fuse check stage2/src/cmd/fuse/...`

## Phase 05: Stage1 Builds Stage2 [W25-P05-STAGE1-BUILD]

- Task 01: Produce the first usable `fusec2`
  [W25-P05-T01-STAGE1-COMPILES-STAGE2]
  Currently: all prior work is package-level unless stage1 can build the full
  stage2 compiler end to end.
  Expected deliveries: stage1-built `stage2_out/fusec2`.
  DoD: stage1 compiles the stage2 tree into a working compiler binary.
  Verify: `fuse build stage2/src/... -o stage2_out/fusec2`
- Task 02: Run the bootstrap smoke surface on the stage1-built compiler
  [W25-P05-T02-STAGE1-BUILT-SMOKE]
  Currently: a built compiler binary is not enough if it cannot exercise the
  expected bootstrap workflows.
  Expected deliveries: smoke coverage proving the stage1-built `fusec2` can run
  compiler actions.
  DoD: the first generated compiler is usable, not just linkable.
  Verify: `go test ./tests/bootstrap/... -run TestStage1BuiltStage2Smoke -v`

## Phase 06: Stage2 Self-Compilation [W25-P06-SELF-COMPILE]

- Task 01: Stage2 compiles its own source tree [W25-P06-T01-STAGE2-COMPILES-SELF]
  Currently: stage2 has only been proven buildable by stage1 so far.
  Expected deliveries: stage2-generated `stage2_out/fusec2_gen2`.
  DoD: the stage1-built compiler can compile the same stage2 source tree into a
  second-generation compiler.
  Verify: `stage2_out/fusec2 build stage2/src/... -o stage2_out/fusec2_gen2`
- Task 02: Run the bootstrap smoke surface on the second-generation compiler
  [W25-P06-T02-GEN2-SMOKE]
  Currently: a second-generation binary could still be linkable but unusable.
  Expected deliveries: smoke coverage proving the generated stage2 compiler can
  drive the same bootstrap-oriented workflows.
  DoD: second-generation smoke coverage passes on the generated compiler.
  Verify: `go test ./tests/bootstrap/... -run TestStage2GeneratedSmoke -v`

## Phase 07: Reproducibility and Bootstrap CI [W25-P07-REPRO]

- Task 01: Bootstrap reproducibility [W25-P07-T01-REPRO]
  Currently: self-compilation without reproducibility still leaves the bootstrap
  process untrustworthy.
  Expected deliveries: reproducibility proof for stage2 artifacts.
  DoD: the project-owned reproducibility check passes for the self-hosted
  compiler build.
  Verify: `make repro`
- Task 02: Gate merges on bootstrap health [W25-P07-T02-GATE]
  Currently: local self-hosting success is insufficient unless CI keeps the
  bootstrap path healthy.
  Expected deliveries: CI gate binding bootstrap health to merges.
  DoD: bootstrap CI fails fast when self-hosting, smoke, or repro regress.
  Verify: `go run tools/checkci/main.go -bootstrap-gate`

## Wave Closure Phase [W25-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W25-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W25`
- Task 02: WC025 entry [W25-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC025" docs/learning-log.md`

