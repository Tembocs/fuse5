# Wave 18: CLI and Diagnostics

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: expose the compiler through a coherent command-line interface with
stable diagnostic rendering and developer workflow tooling.

Entry criterion: W17 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- `build`, `run`, `check`, `test`, `fmt`, `doc`, `repl`, `version`, `help`
  dispatch
- diagnostics render in text and JSON
- JSON output parseable and stable
- every compiler diagnostic complies with Rule 6.17 (primary span,
  explanation, suggestion where applicable)
- `fuse fmt` produces byte-stable output
- CLI exit codes documented and consistent
- incremental-compile driver: `fuse check` and `fuse build` reuse cached
  pass outputs when upstream inputs have not changed, using the pass
  fingerprint contract established in W04-P05; editing an unrelated
  function must not invalidate all of its neighbors

Proof of completion:

```
go test ./cmd/fuse/... -v
go test ./compiler/diagnostics/... -v
go test ./compiler/diagnostics/... -run TestDiagnosticQualityRule -v
go test ./compiler/fmt/... -run TestFormatStable -v
go test ./compiler/doc/... -run TestDocCheck -v
go test ./compiler/driver/... -run TestIncrementalRebuild -v
go test ./tests/e2e/... -run TestCliBasicWorkflow -v
go test ./tests/e2e/... -run TestIncrementalEditCycle -v
```

## Phase 00: Stub Audit [W18-P00-STUB-AUDIT]

- Task 01: CLI audit [W18-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W18 -phase P00`

## Phase 01: Subcommand Surface [W18-P01-SUBCOMMANDS]

- Task 01: Subcommand parser [W18-P01-T01-PARSER]
  Verify: `go test ./cmd/fuse/... -run TestSubcommandParser -v`
- Task 02: Wire all commands [W18-P01-T02-WIRE]
  Verify: `go test ./tests/e2e/... -run TestCliBasicWorkflow -v`

## Phase 02: Diagnostics [W18-P02-DIAGNOSTICS]

- Task 01: Text rendering [W18-P02-T01-TEXT]
  Verify: `go test ./compiler/diagnostics/... -run TestTextRendering -v`
- Task 02: JSON rendering [W18-P02-T02-JSON]
  Verify: `go test ./compiler/diagnostics/... -run TestJsonRendering -v`
- Task 03: Rule 6.17 compliance audit [W18-P02-T03-QUALITY-AUDIT]
  DoD: every diagnostic produced by every compiler path so far has a
  primary span, a one-line explanation, and a suggestion where one is
  possible. A checker enumerates diagnostics and asserts structural
  compliance. Diagnostics that cannot offer a suggestion must declare so
  explicitly, not silently.
  Verify: `go test ./compiler/diagnostics/... -run TestDiagnosticQualityRule -v`

## Phase 03: Workflow Tools [W18-P03-WORKFLOW]

- Task 01: `fmt` and `doc` [W18-P03-T01-FMT-DOC]
  Verify: `go test ./compiler/fmt/... -run TestFormatStable -v`
- Task 02: `repl` and `testrunner` [W18-P03-T02-REPL]
  Verify: `go test ./compiler/repl/... -run TestReplRoundTrip -v`

## Phase 04: Incremental Compile Driver [W18-P04-INCREMENTAL]

The pass fingerprint contract in W04-P05 makes caching possible; this
phase makes it real. The driver persists pass outputs keyed by fingerprint
and reuses them when upstream inputs have not changed. This is what turns
`fuse check` into a usable tight-loop tool and feeds W19's language server.

- Task 01: On-disk cache [W18-P04-T01-CACHE]
  DoD: a content-addressed cache under `.fuse-cache/` stores pass outputs
  keyed by fingerprint. Cache layout is documented and versioned; corrupt
  or version-mismatched entries are invalidated on read, not trusted
  silently.
  Verify: `go test ./compiler/driver/... -run TestPassCache -v`
- Task 02: Incremental rebuild [W18-P04-T02-REBUILD]
  DoD: `fuse build` after an unrelated edit re-runs only passes whose
  input fingerprints changed. A test that edits one function in a
  ten-function program must confirm that nine functions' typed HIR is
  served from cache.
  Verify: `go test ./compiler/driver/... -run TestIncrementalRebuild -v`
- Task 03: Edit-cycle proof [W18-P04-T03-EDIT-CYCLE]
  DoD: an e2e test simulates an edit-compile-edit-compile cycle on a
  ~1000 line program and asserts the second compile finishes within a
  declared wall-clock target relative to the first.
  Verify: `go test ./tests/e2e/... -run TestIncrementalEditCycle -v`

## Wave Closure Phase [W18-PCL-WAVE-CLOSURE]

- Task 01: Retire CLI stubs [W18-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W18`
- Task 02: WC018 entry [W18-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC018" docs/learning-log.md`

