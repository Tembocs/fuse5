# Wave 23: Package Management

> Part of the [Fuse implementation plan](../implementation-plan.md).

Goal: deliver a first-class package manager — manifest format, resolver,
lockfile, fetcher, registry protocol, and driver integration — so that
Fuse ships with a coherent dependency story from v1. A systems language
with no package manager forces every team to invent one; the project will
not ship that obligation onto its users.

Entry criterion: W22 done. Phase 00 confirms no overdue stubs. Hosted
stdlib (IO, fs, os, time, network, hashing, tls) is required because the
package manager depends on it.

State on entry: `compiler/pkg/` and `cmd/fuse/pkg.go` are empty stubs.
No `fuse.toml` schema exists. No registry protocol exists.

Exit criteria:

- `fuse.toml` manifest format specified and parsed
- `fuse.lock` lockfile format specified and produced on resolve
- dependency resolver produces a deterministic, minimal solution or a
  specific conflict diagnostic
- fetcher retrieves source-only packages over HTTPS with SHA-256 integrity
  checking; no code executes during fetch
- a minimal registry protocol is specified: index format, package
  metadata format, publishing model. The reference registry need not be
  hosted to ship v1, but its protocol must be frozen.
- `fuse build` and `fuse check` honor dependencies: resolve on first
  run, then load from the lockfile on subsequent runs
- `fuse add`, `fuse remove`, `fuse update` mutate the manifest and
  lockfile coherently
- vendoring (`fuse vendor`) produces a self-contained source tree that
  builds without network access
- every error path (unknown name, unsatisfiable version, checksum mismatch,
  cycle) has a diagnostic compliant with Rule 6.17
- proof program: a two-crate project resolves, builds, runs, and returns
  a computed value from the dependency

Proof of completion:

```
go test ./compiler/pkg/... -v
go test ./compiler/pkg/... -run TestManifestParse -v
go test ./compiler/pkg/... -run TestResolverDeterministic -count=3 -v
go test ./compiler/pkg/... -run TestLockfileRoundTrip -v
go test ./compiler/pkg/... -run TestFetcherIntegrity -v
go test ./cmd/fuse/... -run TestPkgSubcommands -v
go test ./tests/e2e/... -run TestTwoCrateProject -v
```

## Phase 00: Stub Audit [W23-P00-STUB-AUDIT]

- Task 01: Package-manager audit [W23-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W23 -phase P00`

## Phase 01: Manifest and Lockfile Formats [W23-P01-FORMATS]

- Task 01: `fuse.toml` schema [W23-P01-T01-MANIFEST]
  DoD: schema covers package identity (name, version), dependencies with
  version ranges, feature flags (for `@cfg(feature = "...")`), target-
  specific dependencies, workspace membership. Unknown keys are a parse
  diagnostic, not a silent skip.
  Verify: `go test ./compiler/pkg/... -run TestManifestParse -v`
- Task 02: `fuse.lock` schema [W23-P01-T02-LOCKFILE]
  DoD: lockfile records the resolved version, source URL, and SHA-256 of
  every transitive dependency. Byte-stable across machines and runs.
  Verify: `go test ./compiler/pkg/... -run TestLockfileRoundTrip -v`

## Phase 02: Resolver [W23-P02-RESOLVER]

- Task 01: Version-range algebra [W23-P02-T01-VERSION-ALGEBRA]
  DoD: semver-compatible range semantics with a precise written
  specification. Tie-breaks and conflict cases are documented, not
  emergent.
  Verify: `go test ./compiler/pkg/... -run TestVersionAlgebra -v`
- Task 02: Deterministic resolver [W23-P02-T02-RESOLVER]
  DoD: given the same registry state and same manifest, the resolver
  produces byte-identical lockfiles across three runs. Conflicts produce
  a specific diagnostic naming the offending constraints and offering a
  suggestion.
  Verify: `go test ./compiler/pkg/... -run TestResolverDeterministic -count=3 -v`
- Task 03: Cycle detection [W23-P02-T03-CYCLES]
  DoD: dependency cycles produce a named diagnostic citing the cycle path.
  Verify: `go test ./compiler/pkg/... -run TestResolverCycles -v`

## Phase 03: Fetcher and Integrity [W23-P03-FETCHER]

- Task 01: HTTPS source fetcher [W23-P03-T01-HTTPS]
  DoD: packages are retrieved as source tarballs over HTTPS. No build
  scripts execute during fetch; Fuse does not have pre-install hooks.
  Verify: `go test ./compiler/pkg/... -run TestFetcherHttps -v`
- Task 02: SHA-256 integrity [W23-P03-T02-INTEGRITY]
  DoD: every fetched artifact's hash is verified against the lockfile
  entry before the source is unpacked; mismatches abort with a specific
  diagnostic and never write partial content into the cache.
  Verify: `go test ./compiler/pkg/... -run TestFetcherIntegrity -v`
- Task 03: Offline mode and vendoring [W23-P03-T03-OFFLINE]
  DoD: `fuse build --offline` fails fast if a required dependency is not
  already in the local cache. `fuse vendor` writes every transitive
  dependency's source under `vendor/` such that the project builds
  without network access.
  Verify: `go test ./compiler/pkg/... -run TestFetcherOffline -v`

## Phase 04: Registry Protocol [W23-P04-REGISTRY]

- Task 01: Index format [W23-P04-T01-INDEX]
  DoD: registry index is a deterministic, line-oriented file listing
  every published (name, version, url, sha256, dependencies) tuple.
  Specified in [../registry-protocol.md](../registry-protocol.md).
  Verify: `test -f docs/registry-protocol.md && go test ./compiler/pkg/... -run TestRegistryIndexParse -v`
- Task 02: Package metadata schema [W23-P04-T02-METADATA]
  DoD: metadata returned for a given (name, version) is JSON with a
  frozen schema; schema version is explicit in the payload.
  Verify: `go test ./compiler/pkg/... -run TestRegistryMetadata -v`
- Task 03: Publishing model (spec only) [W23-P04-T03-PUBLISH-SPEC]
  DoD: the publishing flow (authentication, tarball upload, index
  append) is specified in [../registry-protocol.md](../registry-protocol.md). Implementation of
  a hosted reference registry is out of scope for v1; the spec is frozen.
  Verify: `go run tools/checkdocs/main.go -registry-protocol-frozen`

## Phase 05: Driver Integration [W23-P05-DRIVER]

- Task 01: `fuse build` and `fuse check` honor dependencies
  [W23-P05-T01-BUILD-INTEGRATION]
  DoD: the driver resolves on first invocation, writes the lockfile, and
  uses the lockfile on subsequent runs. `fuse check` does not fetch.
  Verify: `go test ./compiler/driver/... -run TestDriverPkgIntegration -v`
- Task 02: `fuse add` / `remove` / `update` [W23-P05-T02-SUBCOMMANDS]
  DoD: subcommands mutate manifest and lockfile atomically; partial
  writes leave both files in a consistent prior state.
  Verify: `go test ./cmd/fuse/... -run TestPkgSubcommands -v`

## Phase 06: End-to-End Package Proof [W23-P06-PROOF]

- Task 01: Two-crate project [W23-P06-T01-TWO-CRATES]
  DoD: `tests/e2e/two_crate_project/` contains a root crate that depends
  on a local path-dependency crate. Both resolve, build, run, and return
  a value computed in the dependency through the root's `main`.
  Verify: `go test ./tests/e2e/... -run TestTwoCrateProject -v`

## Wave Closure Phase [W23-PCL-WAVE-CLOSURE]

- Task 01: Retire package-manager stubs [W23-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W23`
- Task 02: WC023 entry [W23-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC023" docs/learning-log.md`
