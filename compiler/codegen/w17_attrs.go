package codegen

import "fmt"

// EmitReprAttr returns the C-level attribute spelling for a Fuse
// `@repr(...)` decorator (reference §57.10, §31). Each form maps
// to a specific C attribute or struct-member-by-member layout
// directive on gcc / clang / MSVC. Unknown repr kinds return an
// explanatory comment so the emitted C is still well-formed and
// the test surface can pin the expected text.
//
//   - "C"          → __attribute__((__packed__)) removed; plain
//                    struct layout follows the C ABI (no attr)
//   - "packed"     → __attribute__((packed))
//   - "Ixx" / "Uxx"→ enum underlying-type directive comment so
//                    downstream type emission pins the
//                    representation width
func EmitReprAttr(kind string) string {
	switch kind {
	case "C":
		// Plain C layout is the default when no attribute is
		// attached; we still emit a marker comment so readers of
		// the emitted source can see the intent.
		return "/* @repr(C) */"
	case "packed":
		return "__attribute__((packed))"
	case "I8", "I16", "I32", "I64", "U8", "U16", "U32", "U64":
		return fmt.Sprintf("/* @repr(%s) — underlying width */", kind)
	}
	return fmt.Sprintf("/* @repr(%s) — unknown */", kind)
}

// EmitAlignAttr returns the C alignment attribute for `@align(N)`.
// N must be a positive power of two; callers that reach here with
// a bad value get a self-documenting comment so downstream
// consumers can tell that alignment intent was present but
// ill-formed.
func EmitAlignAttr(n int) string {
	if n <= 0 {
		return "/* @align(<=0) — invalid */"
	}
	if (n & (n - 1)) != 0 {
		return fmt.Sprintf("/* @align(%d) — non-power-of-two, rejected at W06 */", n)
	}
	return fmt.Sprintf("__attribute__((aligned(%d)))", n)
}

// EmitInlineAttr returns the C inlining directive for `@inline`,
// `@inline(always)`, `@inline(never)`, or `@cold`. MSVC spelling
// uses `__forceinline` / `__declspec(noinline)` — we emit the
// gcc / clang portable form and rely on a host probe in W18 to
// pick the MSVC equivalent.
func EmitInlineAttr(kind string) string {
	switch kind {
	case "inline":
		return "inline"
	case "always":
		return "__attribute__((always_inline)) inline"
	case "never":
		return "__attribute__((noinline))"
	case "cold":
		return "__attribute__((cold))"
	}
	return fmt.Sprintf("/* @%s — unknown inline directive */", kind)
}
