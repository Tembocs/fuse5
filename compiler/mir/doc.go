// Package mir owns the mid-level IR used by Fuse backends (Rule 3.1 —
// disjoint IR type families; reference §57 for the C11 emitter contract
// it serves during bootstrap).
//
// At W05 the MIR is deliberately minimal — it is exactly expressive
// enough to lower an integer-returning `main` and no more. Any HIR
// construct the W05 lowerer encounters but cannot map to this set
// produces a diagnostic rather than a silent approximation (Rule 6.9).
//
// Supported instructions at W05:
//
//   - ConstInt:   set a register to an integer constant.
//   - BinaryAdd/Sub/Mul/Div/Mod: two-operand arithmetic on i64 registers.
//   - Return:     terminate the basic block with an integer result.
//
// Later waves extend this set. W06 adds comparisons and branches. W07
// adds the concurrency instructions. W11 adds error propagation. W12
// adds closure-capture markers. Each extension is additive; older
// backends must keep compiling against the same MIR layout.
//
// The MIR is function-granular: one `Function` per Fuse fn declaration,
// with parameters, locals, and a list of `Block`s. Every block ends in
// exactly one terminator (a `Return` at W05; branches at W06). This
// invariant is enforced by `Function.Validate`.
package mir
