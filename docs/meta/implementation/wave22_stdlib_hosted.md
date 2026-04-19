# Wave 22: Stdlib Hosted

> Part of the [Fuse implementation plan](../implementation-plan.md).

> Reference guides for this wave live under `../language-reference/full/`.


Goal: implement the hosted and application-facing `full.*` baseline on top of
W20 core and W21 allocator contracts while preserving the core/hosted boundary.
W22 owns every Stage 1 hosted module named in the Attempt 6 language-reference
tree. No module named here may be left in an anonymous umbrella, silently
deferred to `stdlib/ext`, or kicked to a later rescue wave.

Pillar alignment:

- Memory safety: files, sockets, processes, timers, threads, channels, and
  structured-data codecs must have explicit ownership and teardown rules rather
  than leaking raw runtime handles through ad hoc wrappers.
- Concurrency safety: `thread`, `sync`, `shared`, `chan`, timer wakeups, and
  server/request handling must preserve the W07/W16 rules instead of creating
  backdoors around `Send`, `Sync`, lock ranking, or observable thread behavior.
- Developer experience: the application-facing baseline (`json`, `yaml`,
  `http`, `http_server`, `argparse`, `log`, `test`, and related utilities)
  must be discoverable, documented, and directly usable by ordinary Fuse
  programs before Stage 1 is called complete.

Entry criterion: W21 done. Phase 00 confirms no overdue stubs.

State on entry: the compiler, runtime ABI, concurrency checker, and allocator
contracts exist, but the hosted standard library still risks being treated as a
small systems wrapper set rather than as the full application-facing baseline
the language reference now promises. W22 converts that promise into named module
ownership, proofs, and docs.

Exit criteria:

- stable public modules ship for `full.io`, `full.fs`, `full.os`, `full.env`,
  `full.path`, `full.process`, `full.sys`, `full.random`, `full.simd`,
  `full.time`, and `full.timer`
- stable public modules ship for `full.thread`, `full.sync`, `full.shared`,
  and `full.chan`, with observable runtime proofs rather than checker-only
  structural coverage
- stable public modules ship for `full.net`, `full.http`, and
  `full.http_server`
- stable public modules ship for `full.json`, `full.yaml`, `full.toml`, and
  `full.json_schema`
- stable public modules ship for `full.uri`, `full.regex`, `full.crypto`,
  `full.jsonrpc`, `full.argparse`, `full.log`, and `full.test`
- each named hosted module has minimum public API coverage, documentation
  coverage, and at least one regression or end-to-end proof target
- hosted modules do not leak back into `core.*`
- no Stage 1 baseline hosted module named in the reference remains unscheduled
  at wave exit

Proof of completion:

```
fuse build stdlib/full/...
go test ./tests/stdlib/... -run TestFullIOModule -v
go test ./tests/stdlib/... -run TestFullFSModule -v
go test ./tests/stdlib/... -run TestFullPathModule -v
go test ./tests/stdlib/... -run TestFullOSModule -v
go test ./tests/stdlib/... -run TestFullEnvModule -v
go test ./tests/stdlib/... -run TestFullProcessModule -v
go test ./tests/stdlib/... -run TestFullSysModule -v
go test ./tests/stdlib/... -run TestFullTimeModule -v
go test ./tests/stdlib/... -run TestFullTimerModule -v
go test ./tests/stdlib/... -run TestFullRandomModule -v
go test ./tests/stdlib/... -run TestFullSIMDModule -v
go test ./tests/stdlib/... -run TestFullThreadModule -v
go test ./tests/stdlib/... -run TestFullSharedModule -v
go test ./tests/stdlib/... -run TestFullSyncModule -v
go test ./tests/stdlib/... -run TestFullChanModule -v
go test ./tests/stdlib/... -run TestFullNetModule -v
go test ./tests/stdlib/... -run TestFullHTTPModule -v
go test ./tests/stdlib/... -run TestFullJSONModule -v
go test ./tests/stdlib/... -run TestFullYAMLModule -v
go test ./tests/stdlib/... -run TestFullTOMLModule -v
go test ./tests/stdlib/... -run TestFullJSONSchemaModule -v
go test ./tests/stdlib/... -run TestFullURIModule -v
go test ./tests/stdlib/... -run TestFullRegexModule -v
go test ./tests/stdlib/... -run TestFullCryptoModule -v
go test ./tests/stdlib/... -run TestFullJSONRPCModule -v
go test ./tests/stdlib/... -run TestFullArgparseModule -v
go test ./tests/stdlib/... -run TestFullLogModule -v
go test ./tests/stdlib/... -run TestFullTestModule -v
go test ./tests/e2e/... -run TestChannelRoundTrip -v
go test ./tests/e2e/... -run TestSpawnObservable -v
go test ./tests/e2e/... -run TestHTTPServerEcho -v
go test ./tests/e2e/... -run TestJSONRoundTrip -v
fuse doc --check stdlib/full/
```

## Phase 00: Stub Audit [W22-P00-STUB-AUDIT]

- Task 01: Hosted stdlib audit [W22-P00-T01-AUDIT]
  Currently: W22 owns a broad user-visible surface, so any unnoticed hosted
  stub will invalidate the Stage 1 baseline immediately.
  Expected deliveries: audited hosted-stub inventory and committed entry-state
  updates in `STUBS.md`.
  DoD: every hosted stub that affects W22 is either retired in this wave or
  recorded as a later-wave dependency with an explicit boundary.
  Verify: `go run tools/checkstubs/main.go -wave W22 -phase P00`

## Phase 01: Hosted Namespace Ownership and Boundary Contract [W22-P01-NAMESPACE]

- Task 01: Name and place every public `full.*` module
  [W22-P01-T01-MODULE-MAP]
  Currently: the language-reference tree names the hosted/application baseline
  explicitly, but earlier planning grouped many modules into broad umbrellas.
  Expected deliveries: stable `stdlib/full/` package layout matching the named
  public module set in this wave.
  DoD: every module listed in the W22 exit criteria exists as an owned public
  surface with no unnamed "misc hosted" remainder.
  Verify: `fuse build stdlib/full/...`
- Task 02: Enforce the core/hosted boundary [W22-P01-T02-BOUNDARY]
  Currently: hosted APIs are easy to leak back into core when boundary rules are
  not checked explicitly.
  Expected deliveries: boundary rule proving `core.*` remains OS-free and does
  not import hosted/application modules.
  DoD: hosted services remain above the core line, with any runtime bridges kept
  behind documented lower-level interfaces.
  Verify: `go test ./tests/stdlib/... -run TestHostedDoesNotLeakIntoCore -v`

## Phase 02: `full.io`, `full.fs`, and `full.path` [W22-P02-IO-FS-PATH]

- Task 01: Deliver `full.io` [W22-P02-T01-IO]
  Currently: stream and buffer behavior is often collapsed into lower-level
  runtime helpers rather than exposed as a coherent module.
  Expected deliveries: `full.io` with readers, writers, buffering, and the
  minimum error surface promised by the reference.
  DoD: `full.io` can be reviewed and proven independently from filesystem or
  path work.
  Verify: `go test ./tests/stdlib/... -run TestFullIOModule -v`
- Task 02: Deliver `full.fs` [W22-P02-T02-FS]
  Currently: filesystem behavior can disappear into `os` wrappers if it is not
  given its own completion boundary.
  Expected deliveries: `full.fs` for file open/read/write, metadata, and
  directory operations promised by the reference.
  DoD: `full.fs` is complete without leaning on `full.io` or `full.os` to prove
  correctness.
  Verify: `go test ./tests/stdlib/... -run TestFullFSModule -v`
- Task 03: Deliver `full.path` [W22-P02-T03-PATH]
  Currently: path logic is baseline functionality but is easy to leave implicit
  in filesystem helpers.
  Expected deliveries: `full.path` for joins, splits, normalization, and path
  classification behavior promised by the reference.
  DoD: path semantics are reviewable as their own module contract.
  Verify: `go test ./tests/stdlib/... -run TestFullPathModule -v`

## Phase 03: `full.os`, `full.env`, `full.process`, and `full.sys` [W22-P03-OS-PROC]

- Task 01: Deliver `full.os` [W22-P03-T01-OS]
  Currently: OS-facing helpers can drift into generic runtime calls without a
  stable module boundary.
  Expected deliveries: `full.os` for operating-system interaction not owned by
  `full.fs`, `full.env`, or `full.process`.
  DoD: `full.os` is a narrow, reviewable module instead of a catch-all host API
  bucket.
  Verify: `go test ./tests/stdlib/... -run TestFullOSModule -v`
- Task 02: Deliver `full.env` [W22-P03-T02-ENV]
  Currently: environment-variable work is easy to bury inside `os` unless it has
  its own retirement target.
  Expected deliveries: `full.env` accessors, mutation rules, and diagnostics.
  DoD: environment behavior is complete and independently testable.
  Verify: `go test ./tests/stdlib/... -run TestFullEnvModule -v`
- Task 03: Deliver `full.process` [W22-P03-T03-PROCESS]
  Currently: process creation and waiting are baseline facilities, but still at
  risk of remaining thin runtime wrappers.
  Expected deliveries: `full.process` spawn/exec/wait/status/pipe behavior.
  DoD: process lifecycle semantics are explicit enough that later package or
  tooling waves can depend on them without hidden assumptions.
  Verify: `go test ./tests/stdlib/... -run TestFullProcessModule -v`
- Task 04: Deliver `full.sys` [W22-P03-T04-SYS]
  Currently: system introspection can float between runtime helpers and library
  code if it is not retired directly.
  Expected deliveries: `full.sys` for stable system queries named by the
  reference.
  DoD: `full.sys` has its own verification boundary and is not treated as a
  residual OS helper bucket.
  Verify: `go test ./tests/stdlib/... -run TestFullSysModule -v`

## Phase 04: `full.time`, `full.timer`, `full.random`, and `full.simd` [W22-P04-TIME-RUNTIME]

- Task 01: Deliver `full.time` [W22-P04-T01-TIME]
  Currently: wall-clock and monotonic APIs exist at the runtime level, but the
  public hosted module remains incomplete.
  Expected deliveries: `full.time` durations, clocks, and conversions promised
  by the reference.
  DoD: `full.time` is independently reviewable and does not rely on `full.timer`
  to prove correctness.
  Verify: `go test ./tests/stdlib/... -run TestFullTimeModule -v`
- Task 02: Deliver `full.timer` [W22-P04-T02-TIMER]
  Currently: timer behavior can remain thin runtime glue unless it is retired as
  a distinct module.
  Expected deliveries: `full.timer` for sleeps, timers, and wakeup behavior.
  DoD: timer semantics are observable and independently testable.
  Verify: `go test ./tests/stdlib/... -run TestFullTimerModule -v`
- Task 03: Deliver `full.random` [W22-P04-T03-RANDOM]
  Currently: randomness is part of the promised baseline, but still easy to
  treat as a loose helper.
  Expected deliveries: `full.random` with documented generator and seeding
  contracts.
  DoD: random behavior is testable without conflating it with time or SIMD.
  Verify: `go test ./tests/stdlib/... -run TestFullRandomModule -v`
- Task 04: Deliver `full.simd` [W22-P04-T04-SIMD]
  Currently: SIMD helpers are promised but still at risk of becoming an
  underspecified optimization bucket.
  Expected deliveries: `full.simd` module surface named by the reference.
  DoD: SIMD helpers are a stable public module, not just backend-adjacent
  utilities.
  Verify: `go test ./tests/stdlib/... -run TestFullSIMDModule -v`

## Phase 05: `full.thread`, `full.shared`, and `full.sync` [W22-P05-THREAD-SYNC]

- Task 01: Deliver `full.thread` [W22-P05-T01-THREAD]
  Currently: thread semantics exist in the language and runtime, but the public
  module contract still needs a dedicated retirement target.
  Expected deliveries: `full.thread` with `ThreadHandle`, spawn/join, and
  lifecycle behavior promised by the reference.
  DoD: thread behavior is not considered complete merely because channel or
  runtime tests pass.
  Verify: `go test ./tests/stdlib/... -run TestFullThreadModule -v`
- Task 02: Deliver `full.shared` [W22-P05-T02-SHARED]
  Currently: shared-state ownership can be blurred with mutex and lock work if it
  is not retired separately.
  Expected deliveries: `full.shared` with the ownership and sharing contracts
  promised by the reference.
  DoD: `full.shared` is independently testable from `full.sync`.
  Verify: `go test ./tests/stdlib/... -run TestFullSharedModule -v`
- Task 03: Deliver `full.sync` [W22-P05-T03-SYNC]
  Currently: synchronization primitives are too important to live under a broad
  concurrency family task.
  Expected deliveries: `Mutex`, `RwLock`, `Cond`, `Once`, and related sync
  primitives through `full.sync`.
  DoD: `full.sync` stays consistent with W07 lock ranking and `Send`/`Sync`
  semantics and can fail independently from thread or shared-state work.
  Verify: `go test ./tests/stdlib/... -run TestFullSyncModule -v`

## Phase 06: `full.chan` and Observable Concurrency Proofs [W22-P06-CHAN]

- Task 01: Deliver `full.chan` [W22-P06-T01-CHAN]
  Currently: channels are central to Fuse concurrency, but the public hosted
  module still needs its own retirement boundary.
  Expected deliveries: `full.chan` send/recv/close behavior and the channel type
  surface promised by the reference.
  DoD: channel functionality is proven independently from thread-start/stop
  proofs.
  Verify: `go test ./tests/stdlib/... -run TestFullChanModule -v`
- Task 02: Prove channel round-trip across threads
  [W22-P06-T02-ROUND-TRIP]
  Currently: channel functionality can appear correct in isolation while still
  failing under real spawned-thread usage.
  Expected deliveries: end-to-end proof coverage for spawned-thread/channel
  round-trip behavior.
  DoD: a real Fuse program passes values across threads through `full.chan` and
  fails loudly if the runtime path is only partially wired.
  Verify: `go test ./tests/e2e/... -run TestChannelRoundTrip -v`
- Task 03: Prove observable thread startup and shutdown
  [W22-P06-T03-SPAWN-PROOF]
  Currently: spawn visibility is a runtime contract, not just a type-checking
  contract.
  Expected deliveries: end-to-end proof that thread creation, execution, and
  completion are observable through the hosted API.
  DoD: spawned work performs an externally visible effect and the proof fails if
  W22 leaves the thread runtime only partially wired.
  Verify: `go test ./tests/e2e/... -run TestSpawnObservable -v`

## Phase 07: `full.net`, `full.http`, and `full.http_server` [W22-P07-NET-HTTP]

- Task 01: Deliver `full.net` [W22-P07-T01-NET]
  Currently: network primitives are part of the expected Stage 1 baseline, but
  can still be overshadowed by higher-level HTTP planning.
  Expected deliveries: `full.net` socket/address/connection primitives promised
  by the reference.
  DoD: the networking substrate is complete enough that `full.http` does not
  have to prove it by implication.
  Verify: `go test ./tests/stdlib/... -run TestFullNetModule -v`
- Task 02: Deliver `full.http` [W22-P07-T02-HTTP]
  Currently: HTTP client support was absent from the older hosted-wave plan and
  therefore needs an explicit retirement boundary now.
  Expected deliveries: `full.http` request/response/client surface.
  DoD: the HTTP client module can be signed off without conflating it with the
  server module.
  Verify: `go test ./tests/stdlib/... -run TestFullHTTPModule -v`
- Task 03: Deliver `full.http_server` [W22-P07-T03-HTTP-SERVER]
  Currently: HTTP server support was explicitly expected from Stage 1 but not
  previously scheduled as a distinct deliverable.
  Expected deliveries: `full.http_server` with server setup, handler binding,
  and request/response lifecycle.
  DoD: an end-to-end proof boots an HTTP server, serves a request, and shuts it
  down through the standard library alone.
  Verify: `go test ./tests/e2e/... -run TestHTTPServerEcho -v`

## Phase 08: `full.json`, `full.yaml`, `full.toml`, and `full.json_schema` [W22-P08-DATA]

- Task 01: Deliver `full.json` [W22-P08-T01-JSON]
  Currently: JSON was explicitly called out as missing from the earlier Stage 1
  promise and therefore must be owned directly here.
  Expected deliveries: `full.json` encode/decode surface for the baseline use
  cases promised by the reference.
  DoD: `full.json` has both module-level tests and an end-to-end round-trip
  proof.
  Verify: `go test ./tests/stdlib/... -run TestFullJSONModule -v`
- Task 02: Prove JSON round-trip [W22-P08-T02-JSON-PROOF]
  Currently: structured-data support needs a runtime-visible proof, not only a
  module-level API test.
  Expected deliveries: end-to-end JSON round-trip proof coverage.
  DoD: a real Fuse program round-trips JSON through the standard library.
  Verify: `go test ./tests/e2e/... -run TestJSONRoundTrip -v`
- Task 03: Deliver `full.yaml` [W22-P08-T03-YAML]
  Currently: YAML is part of the promised baseline, but is easy to treat as a
  later add-on if it is grouped with JSON only.
  Expected deliveries: `full.yaml` parsing and serialization surface.
  DoD: YAML support is independently testable from JSON and TOML.
  Verify: `go test ./tests/stdlib/... -run TestFullYAMLModule -v`
- Task 04: Deliver `full.toml` [W22-P08-T04-TOML]
  Currently: TOML has the same drift risk as YAML if it is retired only through
  a shared structured-data family test.
  Expected deliveries: `full.toml` parsing and serialization surface.
  DoD: TOML support is independently reviewable.
  Verify: `go test ./tests/stdlib/... -run TestFullTOMLModule -v`
- Task 05: Deliver `full.json_schema` [W22-P08-T05-JSON-SCHEMA]
  Currently: schema validation was part of the expected baseline but absent from
  the earlier hosted-wave plan.
  Expected deliveries: `full.json_schema` validation surface paired to JSON.
  DoD: user code can validate JSON data against schemas through a named module
  with its own regression target.
  Verify: `go test ./tests/stdlib/... -run TestFullJSONSchemaModule -v`

## Phase 09: `full.uri` and `full.regex` [W22-P09-URI-REGEX]

- Task 01: Deliver `full.uri` [W22-P09-T01-URI]
  Currently: URI handling is baseline functionality, but can still be absorbed
  into a generic utilities bucket.
  Expected deliveries: `full.uri` parsing, formatting, and validation surface.
  DoD: URI support is independently testable and no longer hidden in a utility
  family.
  Verify: `go test ./tests/stdlib/... -run TestFullURIModule -v`
- Task 02: Deliver `full.regex` [W22-P09-T02-REGEX]
  Currently: regex support is promised, but can still drift if it shares one
  completion boundary with unrelated helpers.
  Expected deliveries: `full.regex` compile/match/search surface.
  DoD: regex behavior is a complete module contract, not a residual utility.
  Verify: `go test ./tests/stdlib/... -run TestFullRegexModule -v`

## Phase 10: `full.crypto` and `full.jsonrpc` [W22-P10-CRYPTO-JSONRPC]

- Task 01: Deliver `full.crypto` [W22-P10-T01-CRYPTO]
  Currently: security-related helpers are easy to defer unless they are named as
  their own public module.
  Expected deliveries: `full.crypto` surface promised by the reference.
  DoD: the baseline crypto module is no longer an unscheduled utility bucket.
  Verify: `go test ./tests/stdlib/... -run TestFullCryptoModule -v`
- Task 02: Deliver `full.jsonrpc` [W22-P10-T02-JSONRPC]
  Currently: protocol helpers can hide behind broader networking or utility
  phases if they are not retired separately.
  Expected deliveries: `full.jsonrpc` client/server protocol helpers promised by
  the reference.
  DoD: JSON-RPC support is independently testable and not implied by `full.json`
  or `full.http` completion.
  Verify: `go test ./tests/stdlib/... -run TestFullJSONRPCModule -v`

## Phase 11: `full.argparse`, `full.log`, and `full.test` [W22-P11-APP-SUPPORT]

- Task 01: Deliver `full.argparse` [W22-P11-T01-ARGPARSE]
  Currently: command-line parsing is part of a usable Stage 1, but still easy to
  leave in a generic application-utilities bucket.
  Expected deliveries: `full.argparse` parsing surface and diagnostics.
  DoD: command-line parsing support is independently reviewable.
  Verify: `go test ./tests/stdlib/... -run TestFullArgparseModule -v`
- Task 02: Deliver `full.log` [W22-P11-T02-LOG]
  Currently: logging can be treated as polish unless it is bound to a public
  module contract.
  Expected deliveries: `full.log` with the baseline logging surface promised by
  the reference.
  DoD: logging support is complete enough for later tooling and package waves to
  depend on it.
  Verify: `go test ./tests/stdlib/... -run TestFullLogModule -v`
- Task 03: Deliver `full.test` [W22-P11-T03-TEST]
  Currently: testing helpers are part of the promised baseline, but were not
  previously owned as a distinct module.
  Expected deliveries: `full.test` support for library tests and assertions.
  DoD: testing support is independently testable and not implied by CLI or log
  module completion.
  Verify: `go test ./tests/stdlib/... -run TestFullTestModule -v`

## Phase 12: Docs, Fixtures, and Completion Audit [W22-P12-DOCS-PROOFS]

- Task 01: Close hosted docs coverage and guide parity
  [W22-P12-T01-DOCS]
  Currently: the hosted surface can still drift from the split reference guides
  unless docs closure is part of the wave exit.
  Expected deliveries: docs coverage across the entire `full.*` tree and parity
  with the named guides under `../language-reference/full/`.
  DoD: every W22 module has a documented public surface consistent with the
  language-reference staging docs.
  Verify: `fuse doc --check stdlib/full/`
- Task 02: Close the hosted regression and proof matrix
  [W22-P12-T02-REGRESSION-MATRIX]
  Currently: broad build success is not enough to prove that the hosted baseline
  is usable as a library set.
  Expected deliveries: a matrix naming the module-level tests and end-to-end
  proofs that close each W22 public surface.
  DoD: W22 exits with explicit proof ownership for every named hosted module,
  not with a single residual umbrella test target.
  Verify: `fuse build stdlib/full/...`

## Wave Closure Phase [W22-PCL-WAVE-CLOSURE]

- Task 01: Retire hosted stubs [W22-PCL-T01-RETIRE]
  Currently: any remaining hosted stub means the Stage 1 stdlib baseline is
  still incomplete.
  Expected deliveries: retired W22 hosted stubs and updated stub history.
  DoD: `STUBS.md` contains no Active row for a hosted/application baseline
  module this wave claimed to ship.
  Verify: `go run tools/checkstubs/main.go -wave W22`
- Task 02: WC022 entry [W22-PCL-T02-CLOSURE-LOG]
  Currently: W22 is the wave that proves whether Stage 1 means "usable" rather
  than "compiler plus thin wrappers", so its closure record must be explicit.
  Expected deliveries: `WC022` learning-log entry naming hosted proofs, retired
  stubs, and any residual constraints carried into W23.
  DoD: `docs/learning-log.md` contains a complete `WC022` entry aligned with
  this wave's proof commands.
  Verify: `grep "WC022" docs/learning-log.md`

