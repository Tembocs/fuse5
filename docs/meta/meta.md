# Fuse Meta

> This directory stages the future documentation root for Fuse.
> Its purpose is to state, plainly, what this documentation set is trying to
> correct from earlier Fuse work.

Earlier Fuse work did not fail because the language direction was empty. It
failed because planning, proof, and implementation drifted apart. This staged
documentation set uses the accumulated lessons in the learning log to tighten
the reference, the rules, the planning model, and the test infrastructure so
that the same classes of failure do not quietly recur.

Another repeated failure is stdlib scope drift. "Complete Stage 1" was treated
as if a compiler plus a thin systems-facing library layer were enough. That is
not the bar. A Stage 1 compiler is not meaningfully complete if the project
still lacks the baseline application-facing standard library users reasonably
expect, such as structured-data support (`json`, `yaml`) and network and
application services (`http` client/server). If Fuse intends to claim those as
stdlib, they must be specified, scheduled, implemented, and proven before
Stage 1 is called complete.

That expectation was concrete, not abstract. The missing baseline library
inventory people were actually looking for included at least:

- core and data foundations: `bool`, `comparable`, `debuggable`, `equatable`,
  `float`, `float32`, `fmt`, `hash`, `hashable`, `int`, `int32`, `int8`,
  `list`, `map`, `math`, `option`, `printable`, `result`, `set`, `string`,
  `traits`, `uint32`, `uint64`, `uint8`
- hosted runtime and systems modules: `chan`, `env`, `http`, `io`, `json`,
  `net`, `os`, `path`, `process`, `random`, `shared`, `simd`, `sys`, `time`,
  `timer`
- utility and protocol libraries: `argparse`, `crypto`, `http_server`,
  `json_schema`, `jsonrpc`, `log`, `regex`, `test`, `toml`, `uri`, `yaml`

## Current goals

1. Restructure the language reference into a cleaner feature inventory.
Each user-visible feature should be easy to identify, easy to schedule, and
easy to prove. That includes the standard library surface: broad composite
sections that hide multiple independent language or stdlib features should be
split into smaller atomic features where needed.

2. Strengthen the rules with guardrails derived from the learning log.
The rules should encode the hard lessons from earlier Fuse work directly: no
silent stubs, no unverifiable completions, no feature retirement without proof,
no overdue stub drift, and no user-visible feature spanning waves as a
half-implemented pipeline.

3. Reshape the implementation plan so feature ownership is explicit.
Waves and phases should be granular enough that the actual work is visible, but
the retirement rule stays strict: a user-visible feature retires in one wave.
If a feature is too large, the plan must either expand the wave or split the
feature into smaller user-visible units. Forgotten features in the language
reference or the expected stdlib baseline must be added to the plan explicitly
rather than absorbed implicitly into broad wave labels.

4. Build a spec-linked Fuse fixture corpus.
Create a dedicated `tests/features/` corpus containing Fuse source files for
every specified feature, even before the feature retires. These fixtures should
live with the testing infrastructure rather than inside implementation
packages. Each fixture should record its reference section, expected outcome,
and current status: parse-pass, check-pass, run-pass, compile-fail, or
stub-diagnostic. When a feature retires, the relevant fixture graduates into an
adversarial proof in `tests/e2e/` with committed expected behavior.

5. Keep the Go-based Stage 1 compiler honest.
By the time the Go compiler is declared feature-complete, every language and
stdlib feature in the reference must already exist end to end. Later waves may
change the implementation path - self-hosting, native backend, performance
gates, target expansion, ecosystem documentation - but they must not be used to
finish unfinished language semantics. That requirement includes a practical
baseline stdlib, not just core types and host/runtime wrappers. A complete
Stage 1 should already have the standard application-facing libraries the
project promises, rather than leaving JSON, YAML, HTTP, or similar baseline
facilities to a later rescue wave or to `stdlib/ext`. The concrete baseline
inventory above must therefore be treated as planning input, not as optional
wishlist material.

## Practical consequence

This repository is not just "build the compiler again". It is:

- rebuild the specification so features are schedulable
- rebuild the rules so failure modes are blocked earlier
- rebuild the plan so feature retirement is honest
- rebuild the stdlib promise so Stage 1 means usable
- rebuild the tests so the spec is executable

If those five stay aligned, Fuse has a chance to stop repeating the same
architectural failures that invalidated earlier Fuse work.

## Draft docs tree

`docs/meta/` now stages the future documentation root for Fuse. When this tree
is copied into a new repository and renamed to `docs/`, the intended top-level
split is:

- `language-reference/` — the canonical normative reference tree
- `implementation/` — the canonical delivery-plan tree
- `rules.md` — contributor and agent discipline rules
- `learning-log.md` — append-only lessons and wave-closure log
- `README.md` — root repository entry point

`meta.md` remains the repository-level direction note that explains why this
restructure exists.