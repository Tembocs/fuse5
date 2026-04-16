# Wave 00: Governance and Phase Model

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: establish the repository, module, build, test, tooling, documentation,
and governance foundations required for disciplined compiler work. Seed
`STUBS.md` with the initial stub inventory covering every unimplemented
language feature.

Entry criterion: none.

State on entry: only the five foundational docs and the `STUBS.md` seed
exist. No Go module. No CI. No compiler packages.

Exit criteria:

- `make all` succeeds from a clean checkout
- `go test ./...` succeeds on the initial package set
- CI runs on every push and PR for Linux, macOS, and Windows
- the five foundational docs exist and are readable
- `STUBS.md` exists with a seeded Active stubs table covering every
  user-visible language feature not yet implemented, and an empty Stub
  history section
- `.claude/current-wave.json` names the active wave for coordination

Proof of completion:

```
make all
go test ./...
go run tools/checkstubs/main.go
```

## Phase 00: Stub Audit [W00-P00-STUB-AUDIT]

- Task 01: Initialize STUBS.md with seed stubs [W00-P00-T01-INIT-STUBS]
  Currently: file does not exist.
  DoD: STUBS.md exists with the Active stubs table seeded with every
  language feature scheduled for a wave greater than W00, each with its
  scheduled retiring wave. Stub history section is empty.
  Verify: `go run tools/checkstubs/main.go -audit-seed`

## Phase 01: Repository Initialization [W00-P01-REPO-INIT]

- Task 01: Create repository skeleton [W00-P01-T01-REPO-SKELETON]
  Verify: `go run tools/checklayout/main.go`
- Task 02: Add foundational docs [W00-P01-T02-FOUNDATIONAL-DOCS]
  Verify: `go run tools/checkdocs/main.go -foundational`
- Task 03: Artifact policy [W00-P01-T03-ARTIFACT-POLICY]
  Verify: `go run tools/checkartifacts/main.go`

## Phase 02: Go Module and Build Scaffold [W00-P02-GO-MODULE]

- Task 01: Initialize Go module [W00-P02-T01-GO-MOD]
  Verify: `go build ./...`
- Task 02: Create package stubs [W00-P02-T02-PACKAGE-STUBS]
  Verify: `go build ./compiler/...`
- Task 03: Create Stage 1 CLI stub [W00-P02-T03-CLI-STUB]
  Verify: `go test ./cmd/fuse/... -run TestCliStub -v`

## Phase 03: Build and CI Baseline [W00-P03-BUILD-CI]

- Task 01: Author Makefile targets [W00-P03-T01-MAKEFILE]
  Verify: `make all && make test && make clean && make all`
- Task 02: Add CI matrix [W00-P03-T02-CI-MATRIX]
  Verify: `go run tools/checkci/main.go`
- Task 03: Add golden harness [W00-P03-T03-GOLDEN-HARNESS]
  Verify: `go test ./tools/goldens/... -run TestGoldenStability -count=2 -v`

## Phase 04: Governance Artifacts [W00-P04-GOVERNANCE]

- Task 01: Create `.claude/current-wave.json`
  [W00-P04-T01-CURRENT-WAVE-FILE]
  Verify: `go run tools/checkgov/main.go -current-wave`
- Task 02: Document phase model [W00-P04-T02-PHASE-MODEL-DOC]
  Verify: `test -f docs/phase-model.md && go run tools/checkdocs/main.go`
- Task 03: Install `tools/checkstubs` [W00-P04-T03-CHECKSTUBS-TOOL]
  Verify: `go test ./tools/checkstubs/... -v`

## Wave Closure Phase [W00-PCL-WAVE-CLOSURE]

- Task 01: Update STUBS.md stub history [W00-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -history-current-wave W00`
- Task 02: Write WC000 learning-log entry [W00-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC000" docs/learning-log.md`

