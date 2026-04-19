# Wave 29: Targets and Native Expansion

> Part of the [Fuse implementation plan](../implementation-plan.md).

Goal: resume broader target and library work on top of the native
self-hosted compiler. Ecosystem documentation moves to its own wave
(W30).

Pillar alignment:

- Memory safety: new targets must preserve the same ownership, layout, calling,
  and runtime contracts already proven on the earlier native path.
- Concurrency safety: target expansion must not weaken thread, channel, timer,
  or synchronization behavior through target-specific shortcuts.
- Developer experience: each admitted target and each optional `stdlib/ext`
  module must be named explicitly, documented, and testable; no catch-all
  expansion bucket is allowed.

Entry criterion: W28 done.

Exit criteria:

- W29 names the exact non-baseline targets it admits rather than deferring work
  to an unnamed "additional targets" bucket
- the admitted W29 target roster is `linux/arm64`, `macos/arm64`, and
  `windows/arm64`
- each admitted target has a target description, ABI/runtime contract, backend
  support, and CI exercise path
- target expansion proceeds without reintroducing bootstrap debt
- `stdlib/ext/` exists as a governed namespace and hosts only explicitly named
  optional modules admitted by this wave; there is no generic ext-library sweep
- target matrix documented; each supported target is exercised by CI

Proof of completion:

```
go run tools/checktargets/main.go -roster-frozen
go run tools/checktargets/main.go -target linux-arm64
go run tools/checktargets/main.go -target macos-arm64
go run tools/checktargets/main.go -target windows-arm64
go run tools/checkci/main.go -target-matrix
go run tools/checktargets/main.go -ext-roster-frozen
fuse build stdlib/ext/...
```

## Phase 00: Stub Audit [W29-P00-STUB-AUDIT]

- Task 01: Target audit [W29-P00-T01-AUDIT]
  Currently: W29 can easily become a generic expansion bucket unless both target
  and ext-library drift are blocked at entry.
  Expected deliveries: audited W29 target/ext stub inventory and committed entry
  state.
  DoD: no hidden target or ext work enters the wave outside the explicit W29
  roster.
  Verify: `go run tools/checkstubs/main.go -wave W29 -phase P00`

## Phase 01: W29 Target Roster and Contracts [W29-P01-ROSTER]

- Task 01: Freeze the W29 admitted target roster [W29-P01-T01-ROSTER]
  Currently: the prior wave text referred to "additional targets" without
  naming them.
  Expected deliveries: explicit W29 target list: `linux/arm64`, `macos/arm64`,
  `windows/arm64`.
  DoD: no target work is in scope for W29 unless the `(os, arch, abi)` tuple is
  named in the roster.
  Verify: `go run tools/checktargets/main.go -roster-frozen`
- Task 02: Freeze per-target ABI and runtime contracts
  [W29-P01-T02-CONTRACTS]
  Currently: target bring-up can drift if ABI/runtime differences are only
  discovered while implementing codegen.
  Expected deliveries: target-contract records for each admitted target.
  DoD: each target's calling convention, object format, linker/runtime
  requirements, and debugger expectations are documented before implementation
  begins.
  Verify: `go run tools/checktargets/main.go -contracts`

## Phase 02: `linux/arm64` Bring-Up [W29-P02-LINUX-ARM64]

- Task 01: Backend and runtime support for `linux/arm64`
  [W29-P02-T01-LINUX-ARM64-BRINGUP]
  Currently: W29 has not yet named a target-specific bring-up slice.
  Expected deliveries: backend, runtime, and linker support for `linux/arm64`.
  DoD: the admitted Linux ARM64 target reaches the same correctness bar as the
  existing native path.
  Verify: `go run tools/checktargets/main.go -target linux-arm64`
- Task 02: CI exercise for `linux/arm64` [W29-P02-T02-LINUX-ARM64-CI]
  Currently: a target is not complete if it only builds locally.
  Expected deliveries: CI path for `linux/arm64`, whether native or through the
  project's sanctioned cross/executor setup.
  DoD: `linux/arm64` appears in the target matrix with an exercised proof path.
  Verify: `go run tools/checkci/main.go -target linux-arm64`

## Phase 03: `macos/arm64` and `windows/arm64` Bring-Up [W29-P03-DESKTOP-ARM64]

- Task 01: Backend and runtime support for `macos/arm64`
  [W29-P03-T01-MACOS-ARM64]
  Currently: macOS ARM64 is not yet explicitly owned by a target-specific task.
  Expected deliveries: backend, runtime, and toolchain support for
  `macos/arm64`.
  DoD: macOS ARM64 is complete enough to be named a supported target rather than
  an aspirational entry in the matrix.
  Verify: `go run tools/checktargets/main.go -target macos-arm64`
- Task 02: Backend and runtime support for `windows/arm64`
  [W29-P03-T02-WINDOWS-ARM64]
  Currently: Windows ARM64 is not yet explicitly owned by a target-specific
  task.
  Expected deliveries: backend, runtime, and toolchain support for
  `windows/arm64`.
  DoD: Windows ARM64 is complete enough to be named a supported target rather
  than an aspirational entry in the matrix.
  Verify: `go run tools/checktargets/main.go -target windows-arm64`

## Phase 04: Target Matrix and Release Plumbing [W29-P04-MATRIX]

- Task 01: Publish the expanded target matrix [W29-P04-T01-MATRIX]
  Currently: target support can be implemented piecemeal without a single
  canonical matrix.
  Expected deliveries: documented target matrix naming support level and proof
  path for every admitted target.
  DoD: no target is considered supported without a matrix entry and proof path.
  Verify: `go run tools/checkci/main.go -target-matrix`
- Task 02: Bind release and artifact naming to the target matrix
  [W29-P04-T02-ARTIFACTS]
  Currently: expanded targets can still drift into ad hoc artifact naming or
  packaging if the release surface is not frozen.
  Expected deliveries: artifact naming, packaging, and release rules for the
  admitted target set.
  DoD: each admitted target has a predictable release identity and packaging
  rule.
  Verify: `go run tools/checktargets/main.go -artifacts`

## Phase 05: `stdlib/ext` Admission Contract [W29-P05-EXT-CONTRACT]

- Task 01: Freeze the `stdlib/ext` admission rules [W29-P05-T01-ADMISSION]
  Currently: the prior W29 text allowed a vague "extended libraries" bucket.
  Expected deliveries: explicit admission criteria for `stdlib/ext` modules,
  including documentation, proof, and ownership requirements.
  DoD: no optional library may enter `stdlib/ext` without a named module, proof
  surface, and placement rationale.
  Verify: `go run tools/checktargets/main.go -ext-contract`
- Task 02: Freeze the W29 ext roster [W29-P05-T02-ROSTER]
  Currently: even with admission rules, W29 can still drift unless the optional
  module roster is explicit.
  Expected deliveries: a manifest of the optional modules admitted by W29. The
  manifest may be empty, but it may not be implicit.
  DoD: unnamed optional libraries are out of scope for W29 by construction.
  Verify: `go run tools/checktargets/main.go -ext-roster-frozen`

## Phase 06: Explicit `stdlib/ext` Modules [W29-P06-EXT-MODULES]

- Task 01: Build every admitted `stdlib/ext` module [W29-P06-T01-BUILD]
  Currently: optional libraries can still masquerade as complete if the wave has
  no module-level closure step.
  Expected deliveries: buildable `stdlib/ext` tree containing only the modules
  admitted by the W29 ext roster.
  DoD: the ext tree contains no anonymous backlog bucket and no unnamed pilot
  library.
  Verify: `fuse build stdlib/ext/...`
- Task 02: Prove every admitted `stdlib/ext` module has its own test/proof index
  [W29-P06-T02-PROOF-INDEX]
  Currently: optional libraries are especially prone to weak proof ownership if
  they are grouped under one wave-level build step.
  Expected deliveries: per-module proof/index entries for every admitted ext
  module.
  DoD: each ext module admitted by W29 is individually named in the proof plan,
  even if the roster is intentionally small.
  Verify: `go run tools/checktargets/main.go -ext-proof-index`

## Wave Closure Phase [W29-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W29-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W29`
- Task 02: WC029 entry [W29-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC029" docs/learning-log.md`
