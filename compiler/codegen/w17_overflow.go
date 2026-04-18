package codegen

import "fmt"

// OverflowPolicy names the default overflow behaviour for plain
// `+ - *` on integer types per reference §33.1. Debug builds panic
// on overflow; release builds wrap (the deterministic choice the
// W17 CI golden records).
type OverflowPolicy int

const (
	OverflowInvalid OverflowPolicy = iota
	OverflowDebugPanic
	OverflowReleaseWrap
)

// String renders the policy for diagnostics and golden comparison.
func (p OverflowPolicy) String() string {
	switch p {
	case OverflowDebugPanic:
		return "debug_panic"
	case OverflowReleaseWrap:
		return "release_wrap"
	}
	return "invalid"
}

// EmitOverflowAdd returns the C statement for a plain `+` on two
// int64 registers under the given policy.
//
//   - debug_panic: dispatches to fuse_rt_add_overflow_i64 and calls
//     fuse_rt_panic on overflow before observing the result. The
//     runtime routes to __builtin_add_overflow on GCC / Clang and
//     to a portable pure-C fallback on MSVC (W24 retires the MSVC
//     overflow-fallback STUBS row).
//   - release_wrap: wraps via uint64 cast, matching reference
//     §33.1's deterministic release default. Wrap arithmetic does
//     not depend on any builtin.
func EmitOverflowAdd(dst, lhs, rhs int, policy OverflowPolicy) string {
	switch policy {
	case OverflowDebugPanic:
		return fmt.Sprintf(
			"    if (fuse_rt_add_overflow_i64(r%d, r%d, &r%d)) fuse_rt_panic(\"arithmetic overflow\");\n",
			lhs, rhs, dst)
	case OverflowReleaseWrap:
		return fmt.Sprintf(
			"    r%d = (int64_t)((uint64_t)r%d + (uint64_t)r%d);\n",
			dst, lhs, rhs)
	}
	return fmt.Sprintf("    /* overflow policy invalid: dst=r%d lhs=r%d rhs=r%d */\n", dst, lhs, rhs)
}

// EmitOverflowSub / EmitOverflowMul mirror EmitOverflowAdd for the
// other two arithmetic ops. The policy choice at W17 applies
// uniformly to +, -, * per §33.1. Like EmitOverflowAdd, the debug-
// panic path routes through runtime helpers so MSVC hosts link
// cleanly.
func EmitOverflowSub(dst, lhs, rhs int, policy OverflowPolicy) string {
	switch policy {
	case OverflowDebugPanic:
		return fmt.Sprintf(
			"    if (fuse_rt_sub_overflow_i64(r%d, r%d, &r%d)) fuse_rt_panic(\"arithmetic overflow\");\n",
			lhs, rhs, dst)
	case OverflowReleaseWrap:
		return fmt.Sprintf(
			"    r%d = (int64_t)((uint64_t)r%d - (uint64_t)r%d);\n",
			dst, lhs, rhs)
	}
	return fmt.Sprintf("    /* overflow policy invalid: dst=r%d lhs=r%d rhs=r%d */\n", dst, lhs, rhs)
}

func EmitOverflowMul(dst, lhs, rhs int, policy OverflowPolicy) string {
	switch policy {
	case OverflowDebugPanic:
		return fmt.Sprintf(
			"    if (fuse_rt_mul_overflow_i64(r%d, r%d, &r%d)) fuse_rt_panic(\"arithmetic overflow\");\n",
			lhs, rhs, dst)
	case OverflowReleaseWrap:
		return fmt.Sprintf(
			"    r%d = (int64_t)((uint64_t)r%d * (uint64_t)r%d);\n",
			dst, lhs, rhs)
	}
	return fmt.Sprintf("    /* overflow policy invalid: dst=r%d lhs=r%d rhs=r%d */\n", dst, lhs, rhs)
}
