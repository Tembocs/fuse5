// Package driver owns end-to-end orchestration of the Fuse Stage 1
// compiler pipeline.
//
// At W05 the driver knows exactly one operation — `fuse build` for a
// single-file, integer-returning `main` — and wires together:
//
//  1. compiler/parse       (lex + parse into AST)
//  2. compiler/resolve     (module graph, symbols, visibility)
//  3. compiler/hir         (AST → HIR bridge with TypeTable)
//  4. compiler/lower       (HIR → MIR for the W05 spine)
//  5. compiler/codegen     (MIR → C11 source)
//  6. compiler/cc          (C11 source → native binary)
//
// The pipeline is intentionally linear at W05; the W04 Manifest
// infrastructure is in place for W18's incremental driver to
// consume but is not yet used here. Later waves wire passes through
// the manifest so incremental recompilation works without reshaping
// the driver.
//
// Diagnostics from any stage propagate out as the returned error
// accompanied by a slice of `lex.Diagnostic`. A caller that wants
// to print them decides the formatting (the CLI does that at
// cmd/fuse).
package driver
