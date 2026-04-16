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
- `fuse fmt` produces byte-stable output
- CLI exit codes documented and consistent

Proof of completion:

```
go test ./cmd/fuse/... -v
go test ./compiler/diagnostics/... -v
go test ./compiler/fmt/... -run TestFormatStable -v
go test ./compiler/doc/... -run TestDocCheck -v
go test ./tests/e2e/... -run TestCliBasicWorkflow -v
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

## Phase 03: Workflow Tools [W18-P03-WORKFLOW]

- Task 01: `fmt` and `doc` [W18-P03-T01-FMT-DOC]
  Verify: `go test ./compiler/fmt/... -run TestFormatStable -v`
- Task 02: `repl` and `testrunner` [W18-P03-T02-REPL]
  Verify: `go test ./compiler/repl/... -run TestReplRoundTrip -v`

## Wave Closure Phase [W18-PCL-WAVE-CLOSURE]

- Task 01: Retire CLI stubs [W18-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W18`
- Task 02: WC018 entry [W18-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC018" docs/learning-log.md`

