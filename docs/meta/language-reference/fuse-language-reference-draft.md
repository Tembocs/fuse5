# Fuse Language Reference Draft

> Status: meta draft for Attempt 6.
>
> This document is the working replacement for
> [../../fuse-language-reference.md](../../fuse-language-reference.md) while
> the existing reference remains intact.
>
> Unless this draft changes or extends them, sections 1 through 57 of the
> existing reference are inherited here unchanged. This draft exists because
> the previous reference did not explicitly own the full Stage 1 standard
> library promise, which left major user-visible libraries out of scope while
> still allowing Stage 1 to look structurally complete.

## 58. Standard Library Baseline

> Implementation status: SPECIFIED - Stage 1 baseline draft. Concrete wave
> assignments must be added in the implementation plan before this draft is
> promoted into the main reference.

### 58.1 Stdlib is part of the language contract

The Fuse language reference is not complete if it only specifies syntax,
typing, ownership, lowering, and runtime bridges. The public standard library
surface that Stage 1 is expected to ship is part of the language contract.

If Fuse claims a module is part of the standard library, that module must be:

- named explicitly in the reference
- given a public module identity
- described by a minimum behavioral contract
- scheduled in the implementation plan
- validated with fixtures and proof programs where behavior is executable

The standard library is therefore not a post-reference convenience layer. It is
part of what "complete Stage 1" means.

### 58.2 Public namespace and tier model

The public standard library namespace is tiered:

- `core.*` for OS-free modules
- `full.*` for hosted modules that are still part of the non-optional Stage 1
  baseline
- `ext.*` for genuinely optional extensions

The tier split is architectural. It must not be used to move baseline libraries
out of the Stage 1 promise. Any module named in sections 59 through 61 belongs
to the required bootstrap-era standard library and must not exist only under
`ext.*`.

### 58.3 Relationship to existing reference sections

Some standard library behavior is already specified elsewhere in the inherited
reference. Those sections remain in force here and are elevated into the
explicit Stage 1 stdlib inventory:

| Public module | Existing section(s) |
| --- | --- |
| `core.option` | section 16 |
| `core.result` | section 16 |
| `full.chan` | section 17 |
| `full.thread` | section 39 |
| `core.string` | section 42 |
| `core.fmt` | section 43 |
| `full.sync` | sections 17, 39 |

This draft does not remove or weaken those earlier sections. It makes the
stdlib inventory explicit so that missing modules cannot remain invisible.

### 58.4 Import stability

The canonical public import paths for the modules in this draft use the leaf
names listed below under the `core.*` or `full.*` namespace. The on-disk
directory layout under `stdlib/` may group files differently, but the public
module names named here are normative.

For example:

```fuse
import core.string.String;
import core.result.Result;
import full.json.Value;
import full.http.Client;
import full.http_server.Server;
```

### 58.5 Validation contract

Each module or user-visible subfeature named in this draft must eventually own:

- a feature fixture in `tests/features/`
- structural coverage in `tests/stdlib/` where appropriate
- executable proof programs in `tests/e2e/` when the module has observable
  runtime behavior
- documentation coverage checks under `fuse doc --check`

Blocking, synchronous APIs are acceptable for the Stage 1 baseline unless a
section below explicitly says otherwise. Async, event-loop, or coroutine-based
expansions are later features, not reasons to omit the baseline modules.

## 59. Core Foundation Modules

> Implementation status: SPECIFIED - Stage 1 baseline

These modules define the language-facing library surface for primitives, core
traits, formatting, and core value types. Some wrap language primitives that
already exist in syntax or type checking; that does not make the modules
optional. The module documentation and helper surface are still part of the
standard library contract.

### 59.1 Primitive support and numeric modules

| Public module | Minimum required surface |
| --- | --- |
| `core.bool` | Parsing, rendering, and helper surface for `Bool`; documentation for the primitive boolean type and its module-level utilities. |
| `core.int` | Platform-default signed integer helpers, limits, parsing, formatting, and conversions documented at module scope. |
| `core.int8` | Typed constants, parsing, formatting, limits, and bit-width-specific helpers for `I8`. |
| `core.int32` | Typed constants, parsing, formatting, limits, and bit-width-specific helpers for `I32`. |
| `core.uint8` | Typed constants, parsing, formatting, limits, and bit-width-specific helpers for `U8`. |
| `core.uint32` | Typed constants, parsing, formatting, limits, and bit-width-specific helpers for `U32`. |
| `core.uint64` | Typed constants, parsing, formatting, limits, and bit-width-specific helpers for `U64`. |
| `core.float` | Default floating-point helpers, parsing, formatting, rounding, classification, and constants. |
| `core.float32` | `F32`-specific constants, parsing, formatting, rounding, and classification helpers. |
| `core.math` | Common numeric utilities such as `min`, `max`, `clamp`, `abs`, powers, roots, and trigonometric helpers where promised by v1. |

The numeric modules above do not introduce new primitive types. They are the
documented stdlib faces for the primitive types already defined by the core
language.

### 59.2 Trait, formatting, and hashing modules

| Public module | Minimum required surface |
| --- | --- |
| `core.traits` | Canonical re-export surface for the core traits that users are expected to import directly. |
| `core.comparable` | Ordering and comparison traits plus comparison helper functions. |
| `core.equatable` | Equality trait surface and equality-related helper contracts. |
| `core.hashable` | Trait surface for values that participate in hashing. |
| `core.hash` | Hasher state, hash-combine/update helpers, and the library surface consumed by maps and sets. |
| `core.printable` | User-facing printing trait or re-export surface used by simple output APIs. |
| `core.debuggable` | Debug-format trait or re-export surface used by debug-oriented formatting. |
| `core.fmt` | `Formatter`, formatting traits, format-string support, and the `format!`, `print!`, and `eprint!` surfaces described in inherited section 43. |

The exact trait names may be implemented through re-exports or aliases, but the
public module identities listed above must exist and be documented as part of
the baseline stdlib.

### 59.3 Core values and collections

| Public module | Minimum required surface |
| --- | --- |
| `core.string` | Owned UTF-8 string type, byte views, construction, concatenation, parsing helpers, and conversions described in inherited section 42. |
| `core.option` | `Option[T]`, its constructors, combinators, unwrap/expect behavior, and propagation support described in inherited section 16. |
| `core.result` | `Result[T, E]`, its constructors, combinators, unwrap/expect behavior, and propagation support described in inherited section 16. |
| `core.list` | Owned growable sequence with creation, indexing, iteration, append/remove operations, and formatting/equality/hash behavior when element types permit it. |
| `core.map` | Key/value associative container with insertion, lookup, removal, iteration, and stable behavior around key equality and hashing. |
| `core.set` | Unique-value collection with insertion, membership, removal, iteration, and hashing/equality requirements. |

### 59.4 Existing core modules retained

This draft adds missing core modules but does not remove the existing baseline
core modules already implied by the repository and inherited reference,
including `core.cell`, `core.alloc`, `core.ptr`, `core.marker`,
`core.overflow`, and `core.rt_bridge`.

## 60. Hosted Runtime and System Modules

> Implementation status: SPECIFIED - Stage 1 baseline

These modules depend on the hosted runtime and operating-system boundary, but
they are still part of the required Stage 1 standard library rather than
optional post-bootstrap extras.

### 60.1 Filesystem, OS, and process boundary

| Public module | Minimum required surface |
| --- | --- |
| `full.io` | Reader/writer traits, standard streams, text and byte output, and common I/O error reporting. |
| `full.fs` | Files, directories, metadata, path traversal, and file-system mutation operations. |
| `full.os` | OS identity, exit/status conventions, platform-level handles, and runtime-facing operating-system helpers. |
| `full.env` | Environment variables, process arguments, current working directory, home/temp discovery, and related process environment queries. |
| `full.path` | Path join/split/normalize helpers, basename/dirname operations, extension handling, and platform-aware separator rules. |
| `full.process` | Child process spawning, waiting, exit status inspection, redirection, and pipe integration. |
| `full.sys` | Low-level system bridge types and intentionally exposed host/platform queries that are below `os` but still part of the safe documented surface. |

### 60.2 Time, concurrency, and randomness

| Public module | Minimum required surface |
| --- | --- |
| `full.time` | `Instant`, `Duration`, wall-clock access, monotonic time, and sleep operations. |
| `full.timer` | One-shot and repeating timer helpers, deadline/timeout handling, and timer handles or cancellation where promised. |
| `full.thread` | Thread creation, handles, joining, detach behavior, and thread-result propagation described in inherited section 39. |
| `full.sync` | Mutexes, reader/writer locks, condition variables, once-only initialization, and related synchronization primitives. |
| `full.shared` | Focused shared-state surface for `Shared[T]` and associated helpers; this may layer on `full.sync` but must exist as its own documented module if promised publicly. |
| `full.chan` | Channels and send/receive operations described in inherited section 17. |
| `full.random` | Deterministic pseudo-random generation, system randomness, random bytes, and typed random helpers for the common primitive families. |

### 60.3 Networking and machine-oriented facilities

| Public module | Minimum required surface |
| --- | --- |
| `full.net` | IP addresses, socket addresses, streams, listeners, connection lifecycle, and DNS/basic name resolution. |
| `full.http` | Request/response types, headers, status codes, body reading/writing, and at least one baseline blocking HTTP client surface. |
| `full.http_server` | Listener-driven blocking server surface built on `full.net` and `full.http`, including request handlers and response emission. |
| `full.simd` | Target-gated vector types and operations, capability checks or `@cfg` integration, and documented fallback/gating rules. |

For Stage 1, the `full.http` and `full.http_server` baseline may be blocking and
thread-based. The lack of an async runtime is not grounds for omitting those
modules from the standard library reference.

## 61. Structured Data, Text, Tooling, and Protocol Modules

> Implementation status: SPECIFIED - Stage 1 baseline

These modules cover the application-facing standard library surface that was
previously missing from the reference even though it is part of the expected
stdlib baseline.

### 61.1 Structured data formats

| Public module | Minimum required surface |
| --- | --- |
| `full.json` | JSON value model, parse, stringify, encoder/decoder helpers, and Result-based error reporting for invalid input. |
| `full.yaml` | YAML document parse/emit support, scalar/sequence/mapping model, and Result-based parse/validation errors. |
| `full.toml` | TOML document parse/emit support and config-oriented table/array/scalar handling. |
| `full.json_schema` | Schema representation and validation over `full.json` values. |

### 61.2 Text processing and identifiers

| Public module | Minimum required surface |
| --- | --- |
| `full.uri` | URI parsing, component access, normalization, percent-encoding helpers, and relative resolution. |
| `full.regex` | Regex compilation, match/search APIs, capture groups, replacement helpers, and compilation errors reported through `Result`. |

### 61.3 Application, tooling, and protocol helpers

| Public module | Minimum required surface |
| --- | --- |
| `full.argparse` | Command-line parsing, flags, positional arguments, subcommands, help/usage generation, and parse errors. |
| `full.crypto` | Baseline cryptographic surface such as hash functions, HMAC or MAC helpers, secure random bytes, and any smaller v1-safe crypto subset the project commits to. |
| `full.jsonrpc` | JSON-RPC request, response, notification, and error envelope types on top of `full.json`, independent of transport. |
| `full.log` | Leveled or structured logging facade with formatter integration and pluggable sinks/targets. |
| `full.test` | Assertions, expected-failure helpers, fixture/golden support, and other standard testing utilities exposed to Fuse code. |

### 61.4 Error discipline for data and protocol modules

Modules in this section must use `Result`-based error reporting for invalid
external data. Parsing user input, network payloads, or configuration files is
not allowed to rely on panics as the primary failure mechanism.

## 62. Standard Library Promotion Rules

> Implementation status: SPECIFIED - governance contract for the draft

### 62.1 Leaf modules are user-visible features

Each named public module in sections 59 through 61 is a user-visible feature.
It may internally be implemented through submodules or re-exports, but the leaf
module named in the reference must exist as a documented public surface.

For example, `full.http_server` may internally depend on `full.http`,
`full.net`, and `full.io`, but it is still its own user-visible module and must
be planned, tested, and retired honestly.

### 62.2 No baseline module may hide in `ext.*`

Any module listed in sections 59 through 61 is part of the Stage 1 promise. It
must not be moved to `ext.*`, left unscheduled, or treated as optional merely
because it was forgotten in an earlier implementation plan.

### 62.3 Proof requirements

At minimum, the following proof obligations apply before a named module is
considered complete:

- structural build coverage for the module itself
- documentation coverage for its public items
- feature fixtures covering nominal success cases and representative failures
- executable end-to-end proofs for runtime-observable behavior such as process
  spawning, networking, serialization round-trips, HTTP client/server
  interaction, timers, and regex matching

### 62.4 Promotion into the main reference

This draft should not stay permanently separate. Promotion into the main
reference requires:

1. splitting the module tables into smaller atomic language-reference features
2. assigning each feature to concrete waves in the implementation plan
3. wiring every feature to fixtures and proof programs
4. merging the resulting material back into `docs/fuse-language-reference.md`

Until that promotion happens, this file is the authoritative statement of what
was missing from the original reference's standard library coverage.