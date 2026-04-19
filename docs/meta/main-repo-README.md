# Fuse

> A compiled systems programming language focused on memory safety,
> concurrency safety, and developer experience as a first-class constraint.

This file is the staged root `README.md` for Fuse. It is the short entry point
for new readers of the repository and points to the core governance,
specification, and delivery documents.

Fuse is a statically typed systems language designed for programs that need
predictable behavior, explicit control, and readable semantics. It provides
memory safety without a garbage collector, concurrency safety without a borrow
checker, and APIs whose important effects remain visible at the call site.

Fuse uses ownership, deterministic destruction, explicit borrowing, and
structured control over mutation and error propagation. There is no hidden
runtime model for ordinary code, no tracing collector, and no requirement to
understand invisible effects before reading a function call.

## What Fuse Emphasizes

- memory safety through ownership analysis and deterministic destruction
- concurrency through channels, ranked synchronization, and explicit thread
  creation
- explicit mutation through `mutref` and related ownership forms
- explicit error propagation through `Result`, `Option`, and `?`
- explicit unsafe boundaries through `unsafe {}` and raw pointer types

The language is intended to make the cost and effect of an operation easier to
see from the code itself. Mutation, fallibility, and unsafe behavior should not
be hidden behind ordinary-looking calls.

## Language Shape

Fuse is compiled ahead of time to native code. The language includes:

- structs, enums, traits, and impl blocks
- generic functions and generic types
- deterministic ownership and borrowing
- pattern matching and expression-oriented control flow
- channels and shared synchronization primitives for concurrency
- a small explicit unsafe boundary for FFI and raw pointer work

The current compiler architecture uses a bootstrap path in which a Go compiler
lowers Fuse through C11 before producing native binaries. The long-term goal is
a self-hosted Fuse compiler that no longer depends on that bootstrap backend.

## Project Discipline

Earlier Fuse work failed less because the language direction was weak and more
because planning, proof, and implementation drifted apart. This repository is
structured to keep the language reference, the rules, the implementation plan,
the learning log, `STUBS.md`, and the end-to-end proof corpus aligned so that
the same failure mode does not recur quietly.

That discipline also applies to Stage 1 scope. A Stage 1 compiler is not
considered complete if it only ships core types plus thin systems-facing
wrappers. If Fuse claims a practical baseline standard library, the project is
expected to specify, schedule, implement, and prove that baseline before Stage
1 is called complete.

## Project Documents

The core documents live under `docs/` and govern both the language and the
implementation discipline. If the implementation and the documents disagree,
the documents are the place to start.

| Document | Purpose |
|---|---|
| `docs/language-reference/fuse-language-reference.md` | Language specification and implementation contracts |
| `docs/implementation-plan.md` | Build plan from bootstrap to self-hosting |
| `docs/rules.md` | Contributor and agent discipline rules |
| `docs/learning-log.md` | Append-only lessons from bugs, design gaps, and fixes |

There are two additional normative files at the repository root that are not
ordinary documentation:

| File | Purpose |
|---|---|
| `STUBS.md` | Live registry of every compiler stub, its diagnostic, and the wave that retires it |
| `tests/e2e/README.md` | Registry of every end-to-end proof program and its expected output |

`STUBS.md` tracks features that parse and may type-check but are not fully
implemented. Every such feature must emit a diagnostic rather than silently
producing wrong output. The file is updated at every wave boundary and is
verified by CI.

## Getting Started

If you are approaching the project for the first time:

1. Read `docs/rules.md`.
2. Read `docs/implementation-plan.md`.
3. Read `docs/language-reference/fuse-language-reference.md`.
4. Check `STUBS.md`.
5. Read `docs/learning-log.md` for the project's accumulated lessons.

## Status

Fuse is pre-1.0 and should be treated as an active language and compiler effort,
not a stable production platform. The language direction is deliberate, but the
implementation is still evolving and the compiler remains under active
construction.