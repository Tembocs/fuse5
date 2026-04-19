# Implementation Tree

> Status: canonical implementation planning tree for the Attempt 6 meta docs.

This directory stages the future `docs/implementation/` tree for Attempt 6.
It owns delivery sequencing, wave closure, proof expectations, and
feature-to-wave ownership. It is not a second language reference; it is the
execution plan that retires the language reference honestly.

## Planning contract

- no user-visible feature spans waves
- every wave names entry state, phases, expected deliveries, proof commands,
  and closure artifacts
- every wave must advance at least one of Fuse's three pillars without
  regressing another pillar
- every stdlib module named in the language reference has explicit wave and
  phase ownership in this tree

## The three pillars

Fuse is planned and implemented against three first-class constraints:

- memory safety without a garbage collector
- concurrency safety without a Rust-style borrow-checker learning cliff
- developer experience as a first-class constraint

The implementation plan must preserve all three. A wave may emphasize one
pillar more heavily than the others, but it may not quietly trade one pillar
away to make another easier.

## Entry points

- [../implementation-plan.md](../implementation-plan.md) — canonical
  implementation-plan entry point for the meta tree
- [phase-model.md](phase-model.md) — required phase, task, verification, and
  closure model used by every wave document

The per-wave documents in this directory are the detailed delivery contracts for
W00 through W30.

## Stdlib ownership in the meta plan

The expanded Attempt 6 language reference now names the baseline standard
library explicitly. The implementation tree therefore assigns ownership by wave
rather than leaving modules implied by broad labels.

| Wave | Scope | Owned public surface |
| --- | --- | --- |
| W20 | core foundations | `core.bool`, `core.comparable`, `core.debuggable`, `core.equatable`, `core.float`, `core.float32`, `core.fmt`, `core.hash`, `core.hashable`, `core.int`, `core.int32`, `core.int8`, `core.list`, `core.map`, `core.math`, `core.option`, `core.printable`, `core.result`, `core.set`, `core.string`, `core.traits`, `core.uint32`, `core.uint64`, `core.uint8` |
| W21 | allocatorization of core | allocator trait, system/global allocator policy, allocator-parameterized heap-owning core containers |
| W22 | hosted and application baseline | `full.io`, `full.fs`, `full.os`, `full.env`, `full.path`, `full.process`, `full.sys`, `full.random`, `full.simd`, `full.time`, `full.timer`, `full.thread`, `full.sync`, `full.shared`, `full.chan`, `full.net`, `full.http`, `full.http_server`, `full.json`, `full.yaml`, `full.toml`, `full.json_schema`, `full.uri`, `full.regex`, `full.crypto`, `full.jsonrpc`, `full.argparse`, `full.log`, `full.test` |
| W23 | package acquisition | manifests, lockfiles, resolver, fetcher, registry protocol |
| W30 | ecosystem documentation | tutorials, guides, published documentation site, migration and ecosystem material |

## Reference pairing

The implementation tree stays in lockstep with the staged language reference.
Wave docs should cite the relevant guides under [../language-reference/](../language-reference/),
especially the per-module `core/` and `full/` guides that define the stdlib
surface W20-W22 are expected to retire.