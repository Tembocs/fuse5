# Wave 30: Ecosystem Documentation

> Part of the [Fuse implementation plan](../implementation-plan.md).

Goal: ship the user-facing documentation that makes Fuse approachable to
programmers who did not build it. The language reference is correct but
written for implementers; the plan is correct but written for
contributors. This wave produces the material users need: a tutorial, a
book-form language guide, idiomatic-patterns reference, and a migration
handbook from adjacent languages.

Entry criterion: W29 done. The language surface, toolchain, package
manager, and target matrix are all stable enough that a writer can
document them without chasing moving targets.

State on entry: `docs/book/`, `docs/tutorial/`, and `docs/migration/`
are empty directories. `docs/ecosystem-guide.md` does not exist.

Exit criteria:

- `docs/tutorial/` contains a hands-on tutorial that takes a reader from
  "hello world" to a non-trivial program using generics, traits, error
  propagation, concurrency, and packages
- `docs/book/` contains a book-form language guide organized for
  first-time readers, derived from but not replacing the language
  reference
- `docs/migration/from-rust.md`, `from-go.md`, `from-c.md` explain the
  idioms and differences for users arriving from each language
- `docs/ecosystem-guide.md` covers: how to structure a project, how to
  publish a package, conventions for versioning, testing, and
  documentation
- `docs/book/idiomatic-patterns.md` collects the standard patterns
  (builder, newtype, typestate where it applies, channel pipelines,
  Drop-based RAII)
- every document passes `fuse doc --check` where applicable, and
  examples in every document compile under the current compiler
- cross-links between the tutorial, book, reference, and ecosystem
  guide are validated (no dangling internal links)
- documentation site built from these sources is published and linked
  from the README

Proof of completion:

```
go run tools/checkdocs/main.go -book-structure
go run tools/checkdocs/main.go -tutorial-structure
go run tools/checkdocs/main.go -cross-links
go test ./tools/docexamples/... -run TestBookExamplesCompile -v
go test ./tools/docexamples/... -run TestTutorialExamplesCompile -v
```

## Phase 00: Stub Audit [W30-P00-STUB-AUDIT]

- Task 01: Documentation audit [W30-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W30 -phase P00`

## Phase 01: Tutorial [W30-P01-TUTORIAL]

- Task 01: Hands-on tutorial [W30-P01-T01-TUTORIAL]
  DoD: `docs/tutorial/` contains a sequenced tutorial with runnable code
  at every step. Each example has a matching file under
  `tests/e2e/tutorial/` that must compile and run.
  Verify: `go test ./tools/docexamples/... -run TestTutorialExamplesCompile -v`

## Phase 02: Book-Form Language Guide [W30-P02-BOOK]

- Task 01: Book outline and chapter list [W30-P02-T01-OUTLINE]
  DoD: `docs/book/outline.md` lists every chapter, its audience, and
  what it teaches. Chapters are sequenced by prerequisite, not by
  language-reference section number.
  Verify: `test -f docs/book/outline.md && go run tools/checkdocs/main.go -book-structure`
- Task 02: Chapter drafts [W30-P02-T02-CHAPTERS]
  DoD: every outlined chapter has a draft under `docs/book/`. Examples
  in drafts compile.
  Verify: `go test ./tools/docexamples/... -run TestBookExamplesCompile -v`
- Task 03: Idiomatic patterns [W30-P02-T03-PATTERNS]
  DoD: `docs/book/idiomatic-patterns.md` covers the canonical patterns
  with runnable examples and trade-off notes.
  Verify: `go test ./tools/docexamples/... -run TestIdiomExamplesCompile -v`

## Phase 03: Migration Guides [W30-P03-MIGRATION]

- Task 01: From Rust [W30-P03-T01-FROM-RUST]
  DoD: `docs/migration/from-rust.md` covers ownership differences (no
  lifetimes, no-borrow-in-field), trait-object syntax, concurrency
  model, package-manager mapping, and the common pitfalls.
  Verify: `test -f docs/migration/from-rust.md && go run tools/checkdocs/main.go -migration-rust`
- Task 02: From Go [W30-P03-T02-FROM-GO]
  DoD: `docs/migration/from-go.md` covers ownership vs GC, errors via
  `Result[T, E]` and `?` vs multi-return, generics, interfaces vs
  traits, and goroutines vs `spawn` + channels.
  Verify: `test -f docs/migration/from-go.md && go run tools/checkdocs/main.go -migration-go`
- Task 03: From C [W30-P03-T03-FROM-C]
  DoD: `docs/migration/from-c.md` covers `unsafe`, `Ptr[T]`, FFI,
  ownership instead of manual free, and the escape hatch for systems
  code that genuinely needs it.
  Verify: `test -f docs/migration/from-c.md && go run tools/checkdocs/main.go -migration-c`

## Phase 04: Ecosystem Guide [W30-P04-ECOSYSTEM]

- Task 01: Project structure and conventions [W30-P04-T01-STRUCTURE]
  DoD: `docs/ecosystem-guide.md` documents project layout, versioning,
  testing conventions, documentation conventions, and CI recipes.
  Verify: `test -f docs/ecosystem-guide.md`
- Task 02: Publishing workflow [W30-P04-T02-PUBLISHING]
  DoD: the guide explains how to publish a package using the W23
  registry protocol, including the authentication and integrity model.
  Verify: `go run tools/checkdocs/main.go -ecosystem-publishing`

## Phase 05: Integration and Publishing [W30-P05-PUBLISH]

- Task 01: Cross-link validation [W30-P05-T01-CROSS-LINKS]
  DoD: internal links between tutorial, book, migration guides,
  ecosystem guide, reference, and plan are validated in CI.
  Verify: `go run tools/checkdocs/main.go -cross-links`
- Task 02: Documentation site [W30-P05-T02-SITE]
  DoD: a site generator (committed under `tools/docsite/`) builds the
  documentation from `docs/` into a navigable static site. The site
  builds in CI on every commit.
  Verify: `go run tools/docsite/main.go -build && go run tools/docsite/main.go -verify`

## Wave Closure Phase [W30-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W30-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W30`
- Task 02: WC030 entry [W30-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC030" docs/learning-log.md`
