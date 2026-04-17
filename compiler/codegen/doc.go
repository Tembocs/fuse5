// Package codegen owns the Fuse bootstrap backend — the C11 emitter
// that turns MIR into a freestanding `.c` file the host toolchain can
// compile to a native binary (reference §57).
//
// At W05 the emitter supports the W05 MIR subset only: integer
// constants, signed 64-bit arithmetic, and block-level returns from
// a `main` function that becomes the C `main`. Anything beyond that
// is a diagnostic (Rule 6.9). Later waves (W13 trait objects, W17
// C11 hardening) extend the emitter additively.
//
// Output contract:
//
//   - Emits ISO C11 with no reliance on compiler extensions.
//   - Uses `int64_t` for register values and `<stdint.h>` for the
//     width-exact integer types.
//   - Emits exactly one `main` whose return type is `int`; the MIR
//     return value is narrowed to `int` with a static cast that
//     preserves the low 32 bits.
//   - No reliance on the C runtime beyond `<stdint.h>`; that keeps
//     W05 builds trivially portable across Linux, macOS, and
//     Windows before the W16 runtime lands.
package codegen
