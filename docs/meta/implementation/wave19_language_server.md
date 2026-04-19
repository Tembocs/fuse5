# Wave 19: Language Server

> Part of the [Fuse implementation plan](../implementation-plan.md).

Goal: ship a conforming Language Server Protocol (LSP) implementation
sufficient for a modern editor experience — diagnostics as you type,
hover showing type information, go-to-definition, completion, and
symbol search. The server reuses the Stage 1 compiler's lexer, parser,
resolver, checker, and diagnostics; it does not fork the pipeline.

Fuse's developer-experience promise is load-bearing on this wave. A
systems language shipping in 2026 without a language server does not
deliver modern DX.

Pillar alignment:

- Memory safety: the LSP must surface ownership, type, and layout failures with
  the same fidelity as the compiler, not a weaker editor-only approximation.
- Concurrency safety: diagnostics and navigation for concurrency-relevant code
  must reuse compiler truth rather than a parallel partial model.
- Developer experience: each editor feature in this wave must have its own
  retirement boundary so the server does not claim "modern DX" while key
  surfaces remain partial.

Entry criterion: W18 done. Phase 00 confirms no overdue stubs. The
incremental-compile driver (W18-P04) is required because a language
server that re-runs the full pipeline on every keystroke is not usable.

State on entry: `compiler/lsp/` is an empty stub package. The CLI has
no `fuse lsp` subcommand.

Exit criteria:

- `fuse lsp` launches an LSP server speaking LSP 3.17 over stdio
- diagnostics stream to the client using W18's JSON rendering
- hover returns a resolved type and a doc comment when present
- goto-definition works for locals, items, types, and trait methods
- completion offers identifiers in scope, with type hints
- document symbols and workspace symbols work
- semantic tokens provided for syntax highlighting
- edits drive the W18 incremental driver; the server does not block the
  client on full recompiles
- a headless LSP client in CI performs a scripted edit-and-query session
  and observes correct responses

Proof of completion:

```
go test ./compiler/lsp/... -v
go test ./compiler/lsp/... -run TestLspInitialize -v
go test ./compiler/lsp/... -run TestLspDiagnosticsStream -v
go test ./compiler/lsp/... -run TestLspHover -v
go test ./compiler/lsp/... -run TestLspGotoDefinition -v
go test ./compiler/lsp/... -run TestLspCompletion -v
go test ./compiler/lsp/... -run TestLspDocumentSymbols -v
go test ./compiler/lsp/... -run TestLspWorkspaceSymbols -v
go test ./compiler/lsp/... -run TestLspSemanticTokens -v
go test ./tests/e2e/... -run TestLspScriptedSession -v
```

## Phase 00: Stub Audit [W19-P00-STUB-AUDIT]

- Task 01: LSP audit [W19-P00-T01-AUDIT]
  Currently: the LSP wave pulls together transport, diagnostics, navigation,
  and completion, so feature drift must be blocked at entry.
  Expected deliveries: audited LSP stub inventory and committed entry state.
  Verify: `go run tools/checkstubs/main.go -wave W19 -phase P00`

## Phase 01: LSP Protocol Skeleton [W19-P01-PROTOCOL]

- Task 01: stdio transport and JSON-RPC framing [W19-P01-T01-TRANSPORT]
  Currently: there is no protocol skeleton in `compiler/lsp/` yet.
  Expected deliveries: stdio transport and JSON-RPC framing for LSP 3.17.
  DoD: the server reads LSP messages framed per the specification,
  dispatches notifications and requests, and writes responses in the
  correct order. Malformed messages produce a protocol error, not a
  crash.
  Verify: `go test ./compiler/lsp/... -run TestLspTransport -v`
- Task 02: Initialize handshake and capability negotiation
  [W19-P01-T02-INITIALIZE]
  Currently: capability negotiation is not yet explicit.
  Expected deliveries: initialize handshake and precise capability surface.
  DoD: server advertises the features implemented by this wave and only
  those. Capabilities not yet implemented must not be advertised.
  Verify: `go test ./compiler/lsp/... -run TestLspInitialize -v`
- Task 03: Document sync [W19-P01-T03-DOC-SYNC]
  Currently: the in-memory document model does not exist yet.
  Expected deliveries: `didOpen`/`didChange`/`didClose`/`didSave` handling over
  the incremental driver.
  DoD: `textDocument/didOpen`, `didChange`, `didClose`, and `didSave`
  maintain an in-memory document model. Changes feed the incremental
  driver rather than restarting the pipeline.
  Verify: `go test ./compiler/lsp/... -run TestLspDocSync -v`

## Phase 02: Diagnostics and Edit Loop [W19-P02-DIAGNOSTICS]

- Task 01: Streaming diagnostics [W19-P02-T01-STREAM]
  Currently: the server does not yet stream compiler diagnostics after edits.
  Expected deliveries: streaming diagnostics over
  `textDocument/publishDiagnostics`.
  DoD: after each accepted change, the server publishes diagnostics for
  the affected file over `textDocument/publishDiagnostics`. Latency from
  change to diagnostic for a small file is bounded by the incremental
  driver's per-pass budget.
  Verify: `go test ./compiler/lsp/... -run TestLspDiagnosticsStream -v`
- Task 02: Diagnostic quality passthrough [W19-P02-T02-QUALITY]
  Currently: suggestions and structural diagnostic data can still be lost at the
  LSP boundary.
  Expected deliveries: Rule 6.17-preserving diagnostics and quick-fix exposure.
  DoD: every diagnostic rendered over LSP preserves the Rule 6.17
  structure (primary span, explanation, suggestion where present). LSP's
  `CodeAction` surface exposes suggestions as quick-fixes when they are
  mechanical.
  Verify: `go test ./compiler/lsp/... -run TestLspQuickFixes -v`

## Phase 03: Hover and Go-to-Definition [W19-P03-HOVER-GOTO]

- Task 01: Hover type information [W19-P03-T01-HOVER]
  Currently: hover results are not yet wired to compiler truth.
  Expected deliveries: hover support for identifiers and call sites.
  DoD: hovering over an identifier returns its resolved type and any
  doc comment. Hovering over a call returns the callee's signature.
  Verify: `go test ./compiler/lsp/... -run TestLspHover -v`
- Task 02: Go-to-definition [W19-P03-T02-GOTO-DEF]
  Currently: definition navigation is not yet a separately reviewable feature.
  Expected deliveries: goto-definition across locals, items, and imports.
  DoD: goto works for local bindings, items (`fn`, `struct`, `enum`,
  `trait`, `const`, `static`, `type`), trait methods (resolves to the
  concrete impl when statically known, otherwise to the trait method),
  and imported names.
  Verify: `go test ./compiler/lsp/... -run TestLspGotoDefinition -v`

## Phase 04: Document and Workspace Symbols [W19-P04-SYMBOLS]

- Task 01: Document symbols [W19-P04-T01-DOCUMENT-SYMBOLS]
  Currently: symbol extraction is not yet separated from hover/goto behavior.
  Expected deliveries: hierarchical `textDocument/documentSymbol` support.
  DoD: document-symbol output for a file is complete and independently testable.
  Verify: `go test ./compiler/lsp/... -run TestLspDocumentSymbols -v`
- Task 02: Workspace symbols [W19-P04-T02-WORKSPACE-SYMBOLS]
  Currently: workspace search can drift into document-symbol work unless it is
  retired separately.
  Expected deliveries: `workspace/symbol` support over the indexed workspace.
  DoD: `textDocument/documentSymbol` and `workspace/symbol` return a
  hierarchical symbol list for the file and a filtered list for the
  workspace.
  Verify: `go test ./compiler/lsp/... -run TestLspWorkspaceSymbols -v`

## Phase 05: Scope-Aware Completion [W19-P05-COMPLETION]

- Task 01: Scope-aware completion [W19-P05-T01-COMPLETE]
  Currently: completion behavior is still grouped with highlighting in the old
  phase structure.
  Expected deliveries: scope-aware identifier and method completion.
  DoD: `textDocument/completion` returns identifiers in scope at the
  cursor, with type hints in the detail field. Method completion on
  a receiver filters by the receiver's type and trait impls.
  Verify: `go test ./compiler/lsp/... -run TestLspCompletion -v`

## Phase 06: Semantic Tokens [W19-P06-SEMANTIC-TOKENS]

- Task 01: Semantic tokens [W19-P06-T01-SEM-TOKENS]
  Currently: semantic-token output is a distinct editor feature, but still easy
  to hide inside a broader completion/highlighting phase.
  Expected deliveries: semantic-token stream for syntax highlighting.
  DoD: `textDocument/semanticTokens/full` returns a token stream
  classifying identifiers by kind (type, function, variable, parameter,
  keyword, etc.). Editors driving highlighting from this stream produce
  a readable result.
  Verify: `go test ./compiler/lsp/... -run TestLspSemanticTokens -v`

## Phase 07: End-to-End LSP Proof [W19-P07-PROOF]

- Task 01: Headless scripted session [W19-P07-T01-SCRIPTED]
  Currently: unit tests alone are not enough to prove the integrated editor
  experience.
  Expected deliveries: scripted headless LSP session in CI.
  DoD: a test harness launches `fuse lsp`, sends a canned sequence of
  initialize → open → edit → hover → goto → complete, and asserts the
  responses match expected goldens. The session runs in CI on every
  push.
  Verify: `go test ./tests/e2e/... -run TestLspScriptedSession -v`

## Wave Closure Phase [W19-PCL-WAVE-CLOSURE]

- Task 01: Retire LSP stubs [W19-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W19`
- Task 02: WC019 entry [W19-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC019" docs/learning-log.md`
