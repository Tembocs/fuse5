# STUBS

> Status: normative root-level infrastructure for Fuse.
>
> This file is the live registry of every compiler stub and the append-only
> log of stub lifecycle events. It is governed by
> [docs/rules.md](docs/rules.md) §6.9–§6.16 and
> [docs/repository-layout.md](docs/repository-layout.md) §2.
>
> Every stub listed in the Active stubs table below must emit the declared
> diagnostic when the corresponding language feature is used. Silent stubs
> are forbidden (Rule 6.9). A stub without a concrete retiring wave is a
> defect (Rule 2.5).
>
> The Stub history section is append-only (Rule 10.1, Rule 6.16). Existing
> entries are never edited; new wave-closure blocks are appended at wave
> boundaries.

## Active stubs

At Wave 00 the entire compiler is a forward-declared stub. Each row below
names a language-feature group, the package that will own it, the
placeholder behavior (all present packages are empty doc-only stubs), the
diagnostic the compiler must emit when the feature is used, and the wave
that retires the stub. File:Line is of the form `package/ (not yet
created)` until real code exists; the retiring wave fills in the line
number when it lands the feature.

| Stub | File:Line | Current behavior | Diagnostic emitted | Retiring wave |
|---|---|---|---|---|
| Lexer and token model | compiler/lex/ (empty) | no tokens emitted | "lexer not yet implemented" | W01 |
| Parser and AST | compiler/parse/ (empty) | no AST produced | "parser not yet implemented" | W02 |
| Module resolver | compiler/resolve/ (empty) | no resolution performed | "resolver not yet implemented" | W03 |
| HIR and TypeTable | compiler/hir/ (empty) | no HIR constructed | "HIR/TypeTable not yet implemented" | W04 |
| Minimal end-to-end spine | compiler/driver/ (empty) | no binary produced | "Stage 1 driver not yet implemented" | W05 |
| Type checker | compiler/check/ (empty) | no types checked | "type checker not yet implemented" | W06 |
| Concurrency checker (Send/Sync/Chan/spawn/@rank) | compiler/check/ (empty) | no concurrency enforcement | "concurrency checker not yet implemented" | W07 |
| Monomorphization | compiler/monomorph/ (empty) | no generic specialization | "monomorphization not yet implemented" | W08 |
| Ownership, liveness, borrow rules, drop codegen | compiler/liveness/ (empty) | no ownership enforcement | "ownership/liveness not yet implemented" | W09 |
| Pattern matching dispatch and exhaustiveness | compiler/check/ (empty) | no match dispatch | "pattern matching not yet implemented" | W10 |
| Error propagation (`?` operator) | compiler/lower/ (empty) | no `?` lowering | "error propagation not yet implemented" | W11 |
| Closures, capture, `move` prefix, Fn/FnMut/FnOnce | compiler/lower/ (empty) | no closure lifting | "closures not yet implemented" | W12 |
| Trait objects (`dyn Trait`, vtables, object safety) | compiler/codegen/ (empty) | no dynamic dispatch | "trait objects not yet implemented" | W13 |
| Compile-time evaluation (`const fn`, `size_of`, `align_of`) | compiler/ (not yet created consteval/) | no const evaluation | "const fn not yet implemented" | W14 |
| MIR consolidation (casts, fn pointers, slice range, struct update, overflow arithmetic) | compiler/lower/ (empty) | no MIR produced | "MIR lowering not yet implemented" | W15 |
| Runtime ABI (threads, channels, panic, IO) | runtime/src/ (empty) | no runtime | "runtime not yet implemented" | W16 |
| Codegen C11 hardening (`@repr`, `@align`, `@inline`, intrinsics, variadic, debug info, perf baseline) | compiler/codegen/ (empty) | no C emitted | "C11 codegen not yet implemented" | W17 |
| CLI, diagnostics, `fuse fmt/doc/repl`, incremental driver, Rule 6.17 audit | compiler/driver/ (empty) | Stage 1 CLI only knows `version` and `help` | "subcommand not yet implemented" | W18 |
| Language server (LSP 3.17) | compiler/ (not yet created lsp/) | no LSP server | "fuse lsp not yet implemented" | W19 |
| Stdlib core (traits, primitives, strings, collections, Cell/RefCell, Ptr.null, overflow methods) | stdlib/core/ (empty) | no stdlib | "stdlib core not yet implemented" | W20 |
| Custom allocators (Allocator trait, parameterized collections) | stdlib/core/alloc/ (not yet created) | no allocator trait | "custom allocators not yet implemented" | W21 |
| Stdlib hosted (IO, fs, os, time, thread, sync, channels, network) | stdlib/full/ (empty) | no hosted stdlib | "stdlib hosted not yet implemented" | W22 |
| Package management (manifest, lockfile, resolver, fetcher, registry protocol) | compiler/ (not yet created pkg/) | no package manager | "package manager not yet implemented" | W23 |
| Stub clearance gate | n/a — gating wave | clearance happens at wave entry | n/a — policy wave | W24 |
| Stage 2 self-hosting | stage2/src/ (empty) | no stage2 compiler | "stage 2 compiler not yet ported" | W25 |
| Native backend with DWARF | compiler/ (not yet created codegen/native/) | no native backend | "native backend not yet implemented" | W26 |
| Performance gate (runtime ratios, compile-time budgets, code-size, memory footprint) | tests/perf/ (empty) | no perf gate | "perf gate not yet implemented" | W27 |
| Retirement of Go and C from active path | compiler/ & runtime/ | bootstrap stack active | n/a — retirement wave | W28 |
| Target matrix and `stdlib/ext/` | stdlib/ext/ (empty) | no ext stdlib | "stdlib ext not yet implemented" | W29 |
| Ecosystem documentation (tutorial, book, migration guides, site) | docs/ (tutorial/book/migration/ not yet created) | no user-facing docs | n/a — documentation wave | W30 |

## Stub history

The Stub history is append-only. Each wave closure appends a block naming
the stubs added this wave, the stubs retired this wave (with the proof
program or test that confirmed retirement), and any stubs rescheduled
(with reason). Entries are never edited in place (Rule 6.16).

### W00 — Governance and Phase Model

Added:
- Lexer and token model (compiler/lex/ empty) — emits "lexer not yet
  implemented" — retires W01
- Parser and AST (compiler/parse/ empty) — emits "parser not yet
  implemented" — retires W02
- Module resolver (compiler/resolve/ empty) — emits "resolver not yet
  implemented" — retires W03
- HIR and TypeTable (compiler/hir/ empty) — emits "HIR/TypeTable not yet
  implemented" — retires W04
- Minimal end-to-end spine (compiler/driver/ empty) — emits "Stage 1
  driver not yet implemented" — retires W05
- Type checker (compiler/check/ empty) — emits "type checker not yet
  implemented" — retires W06
- Concurrency checker (compiler/check/ empty) — emits "concurrency
  checker not yet implemented" — retires W07
- Monomorphization (compiler/monomorph/ empty) — emits "monomorphization
  not yet implemented" — retires W08
- Ownership, liveness, borrow rules, drop codegen (compiler/liveness/
  empty) — emits "ownership/liveness not yet implemented" — retires W09
- Pattern matching (compiler/check/ empty) — emits "pattern matching not
  yet implemented" — retires W10
- Error propagation (compiler/lower/ empty) — emits "error propagation
  not yet implemented" — retires W11
- Closures (compiler/lower/ empty) — emits "closures not yet implemented"
  — retires W12
- Trait objects (compiler/codegen/ empty) — emits "trait objects not yet
  implemented" — retires W13
- Compile-time evaluation (consteval package not yet created) — emits
  "const fn not yet implemented" — retires W14
- MIR consolidation (compiler/lower/ empty) — emits "MIR lowering not yet
  implemented" — retires W15
- Runtime ABI (runtime/src/ empty) — emits "runtime not yet implemented"
  — retires W16
- Codegen C11 hardening (compiler/codegen/ empty) — emits "C11 codegen
  not yet implemented" — retires W17
- CLI and diagnostics (compiler/driver/ empty) — emits "subcommand not
  yet implemented" — retires W18
- Language server (lsp package not yet created) — emits "fuse lsp not yet
  implemented" — retires W19
- Stdlib core (stdlib/core/ empty) — emits "stdlib core not yet
  implemented" — retires W20
- Custom allocators (stdlib/core/alloc/ not yet created) — emits "custom
  allocators not yet implemented" — retires W21
- Stdlib hosted (stdlib/full/ empty) — emits "stdlib hosted not yet
  implemented" — retires W22
- Package management (pkg package not yet created) — emits "package
  manager not yet implemented" — retires W23
- Stub clearance gate — policy wave, no stub — retires W24
- Stage 2 self-hosting (stage2/src/ empty) — emits "stage 2 compiler not
  yet ported" — retires W25
- Native backend with DWARF (compiler/codegen/native/ not yet created) —
  emits "native backend not yet implemented" — retires W26
- Performance gate (tests/perf/ empty) — emits "perf gate not yet
  implemented" — retires W27
- Retirement of Go and C — retirement wave, no stub — retires W28
- Target matrix and stdlib/ext (stdlib/ext/ empty) — emits "stdlib ext
  not yet implemented" — retires W29
- Ecosystem documentation (docs/book /tutorial /migration not yet
  created) — documentation wave, no stub — retires W30

Retired: (none this wave — W00 is the seeding wave)

Rescheduled: (none this wave)
