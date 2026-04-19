# Wave 18: CLI and Diagnostics

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: expose the compiler through a coherent command-line interface with
stable diagnostic rendering and developer workflow tooling.

Pillar alignment:

- Memory safety: diagnostics and command workflows must expose earlier compiler
  safety failures faithfully rather than obscuring them behind tool output.
- Concurrency safety: no direct concurrency feature ships here, but the command
  surface must preserve later waves' proof and regression paths.
- Developer experience: this wave is explicitly a DX wave, so every subcommand,
  diagnostic rendering path, and incremental workflow must be reviewable as its
  own delivery contract.

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
go test ./cmd/fuse/... -run TestSubcommandParser -v
go test ./cmd/fuse/... -run TestBuildRunCheckDispatch -v
go test ./cmd/fuse/... -run TestWorkflowCommandDispatch -v
go test ./compiler/diagnostics/... -run TestDiagnosticQualityRule -v
go test ./compiler/diagnostics/... -run TestTextRendering -v
go test ./compiler/diagnostics/... -run TestJsonRendering -v
go test ./compiler/fmt/... -run TestFormatStable -v
go test ./compiler/doc/... -run TestDocCheck -v
go test ./compiler/repl/... -run TestReplRoundTrip -v
go test ./compiler/testrunner/... -run TestRunnerInvocation -v
go test ./compiler/driver/... -run TestPassCache -v
go test ./compiler/driver/... -run TestIncrementalRebuild -v
go test ./compiler/driver/... -run TestIncrementalCheckReuse -v
go test ./tests/e2e/... -run TestCliBasicWorkflow -v
go test ./tests/e2e/... -run TestIncrementalEditCycle -v
```

## Phase 00: Stub Audit [W18-P00-STUB-AUDIT]

- Task 01: CLI audit [W18-P00-T01-AUDIT]
  Currently: command-surface drift is easy to miss because many commands share a
  common CLI entrypoint.
  Expected deliveries: audited CLI/diagnostic/incremental stub inventory.
  Verify: `go run tools/checkstubs/main.go -wave W18 -phase P00`

## Phase 01: Parser and Core Command Dispatch [W18-P01-CORE-DISPATCH]

- Task 01: Subcommand parser [W18-P01-T01-PARSER]
  Currently: command parsing is still implicit in the bootstrap CLI stub.
  Expected deliveries: stable argument parser for the W18 command surface.
  Verify: `go test ./cmd/fuse/... -run TestSubcommandParser -v`
- Task 02: Wire `build`, `run`, `check`, and `test`
  [W18-P01-T02-CORE-COMMANDS]
  Currently: the command parser and the user-visible compiler workflow are not
  yet separated into reviewable deliveries.
  Expected deliveries: the core build/run/check/test command set.
  DoD: the core command surface can be reviewed without conflating it with fmt,
  docs, repl, or help/version behavior.
  Verify: `go test ./cmd/fuse/... -run TestBuildRunCheckDispatch -v`

## Phase 02: Workflow Commands and Exit-Code Contract [W18-P02-WORKFLOW-CMDS]

- Task 01: Wire `fmt`, `doc`, and `repl` commands
  [W18-P02-T01-WORKFLOW-COMMANDS]
  Currently: workflow commands are still easy to hide behind one generic
  "wire all commands" step.
  Expected deliveries: `fmt`, `doc`, and `repl` dispatch on the CLI surface.
  DoD: workflow-tool commands are complete enough to test independently from the
  core build/check path.
  Verify: `go test ./cmd/fuse/... -run TestWorkflowCommandDispatch -v`
- Task 02: Wire `version`, `help`, and exit-code rules
  [W18-P02-T02-EXIT-CODES]
  Currently: exit-code behavior and informational commands can drift unless they
  are owned explicitly.
  Expected deliveries: stable informational commands and documented exit codes.
  DoD: command failures and success paths produce deliberate exit-code behavior,
  not ad hoc process termination.
  Verify: `go test ./tests/e2e/... -run TestCliBasicWorkflow -v`

## Phase 03: Text and JSON Diagnostics [W18-P03-DIAGNOSTICS]

- Task 01: Text rendering [W18-P03-T01-TEXT]
  Currently: text diagnostics are still a generic output concern rather than a
  committed CLI contract.
  Expected deliveries: stable text rendering for compiler diagnostics.
  Verify: `go test ./compiler/diagnostics/... -run TestTextRendering -v`
- Task 02: JSON rendering [W18-P03-T02-JSON]
  Currently: JSON diagnostics need an independent compatibility contract.
  Expected deliveries: stable JSON diagnostic output for machine consumers.
  Verify: `go test ./compiler/diagnostics/... -run TestJsonRendering -v`
- Task 03: Rule 6.17 compliance audit [W18-P03-T03-QUALITY-AUDIT]
  Currently: diagnostic structure can regress silently unless there is an
  explicit structural audit.
  Expected deliveries: diagnostic-quality checker over all compiler diagnostics.
  DoD: every diagnostic produced by every compiler path so far has a
  primary span, a one-line explanation, and a suggestion where one is
  possible. A checker enumerates diagnostics and asserts structural
  compliance. Diagnostics that cannot offer a suggestion must declare so
  explicitly, not silently.
  Verify: `go test ./compiler/diagnostics/... -run TestDiagnosticQualityRule -v`

## Phase 04: `fmt` and `doc` Tool Surfaces [W18-P04-FMT-DOC]

- Task 01: `fuse fmt` [W18-P04-T01-FMT]
  Currently: formatting is one workflow tool among several, but still needs its
  own completion boundary.
  Expected deliveries: byte-stable formatter behavior.
  Verify: `go test ./compiler/fmt/... -run TestFormatStable -v`
- Task 02: `fuse doc` [W18-P04-T02-DOC]
  Currently: documentation checks are easy to hide behind generic workflow-tool
  progress.
  Expected deliveries: doc-generation and doc-check command surface.
  Verify: `go test ./compiler/doc/... -run TestDocCheck -v`

## Phase 05: `repl` and `testrunner` [W18-P05-REPL-TESTRUNNER]

- Task 01: `repl` surface [W18-P05-T01-REPL]
  Currently: REPL behavior remains a distinct workflow tool, not a residual CLI
  helper.
  Expected deliveries: runnable REPL command surface.
  Verify: `go test ./compiler/repl/... -run TestReplRoundTrip -v`
- Task 02: `testrunner` surface [W18-P05-T02-TESTRUNNER]
  Currently: test-running workflow can drift into generic CLI behavior if not
  retired separately.
  Expected deliveries: test-runner integration for the user-visible command
  surface.
  Verify: `go test ./compiler/testrunner/... -run TestRunnerInvocation -v`

## Phase 06: On-Disk Cache Contract [W18-P06-CACHE]

The pass fingerprint contract in W04-P05 makes caching possible; this
phase makes it real. The driver persists pass outputs keyed by fingerprint
and reuses them when upstream inputs have not changed. This is what turns
`fuse check` into a usable tight-loop tool and feeds W19's language server.

- Task 01: On-disk cache [W18-P06-T01-CACHE]
  Currently: cache persistence and invalidation behavior are not yet explicit.
  Expected deliveries: content-addressed on-disk pass cache.
  DoD: a content-addressed cache under `.fuse-cache/` stores pass outputs
  keyed by fingerprint. Cache layout is documented and versioned; corrupt
  or version-mismatched entries are invalidated on read, not trusted
  silently.
  Verify: `go test ./compiler/driver/... -run TestPassCache -v`

## Phase 07: Incremental `build` and `check` Reuse [W18-P07-INCREMENTAL-REUSE]

- Task 01: Incremental rebuild for `fuse build`
  [W18-P07-T01-BUILD-REUSE]
  Currently: cache persistence alone does not prove that build reuse is working
  at the command level.
  Expected deliveries: selective rebuild behavior for `fuse build`.
  DoD: `fuse build` after an unrelated edit re-runs only passes whose
  input fingerprints changed. A test that edits one function in a
  ten-function program must confirm that nine functions' typed HIR is
  served from cache.
  Verify: `go test ./compiler/driver/... -run TestIncrementalRebuild -v`
- Task 02: Incremental reuse for `fuse check`
  [W18-P07-T02-CHECK-REUSE]
  Currently: the plan names `fuse check` as part of the incremental promise, but
  the old phase structure did not give it an independent retirement target.
  Expected deliveries: selective reuse behavior for `fuse check`.
  DoD: `fuse check` on an unrelated edit reuses cached pass outputs rather than
  recomputing the whole pipeline.
  Verify: `go test ./compiler/driver/... -run TestIncrementalCheckReuse -v`

## Phase 08: Edit-Cycle Proof and Latency Budget [W18-P08-EDIT-CYCLE]

- Task 01: Edit-cycle proof [W18-P08-T01-EDIT-CYCLE]
  Currently: command-level cache reuse still needs a user-visible proof.
  Expected deliveries: end-to-end edit/compile/edit/compile latency proof.
  DoD: an e2e test simulates an edit-compile-edit-compile cycle on a
  ~1000 line program and asserts the second compile finishes within a
  declared wall-clock target relative to the first.
  Verify: `go test ./tests/e2e/... -run TestIncrementalEditCycle -v`

## Wave Closure Phase [W18-PCL-WAVE-CLOSURE]

- Task 01: Retire CLI stubs [W18-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W18`
- Task 02: WC018 entry [W18-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC018" docs/learning-log.md`

