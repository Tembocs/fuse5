package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// compilerIntrinsicPrefix is the stable callee-name prefix the
// lowerer uses to mark an OpCall whose callee is a compiler
// intrinsic rather than a user function. Routing by prefix keeps
// the MIR op set small (no new opcodes) while still giving the
// emitter a deterministic decision point. Format:
//
//	__fuse_intrinsic_<name>[__<payload>]
//
// Examples:
//
//	__fuse_intrinsic_unreachable
//	__fuse_intrinsic_likely
//	__fuse_intrinsic_ptr_null__int64_t
//	__fuse_intrinsic_size_of__8
const compilerIntrinsicPrefix = "__fuse_intrinsic_"

// tryEmitCompilerIntrinsic inspects `in.CallName` and, if it names
// a compiler intrinsic, returns the C expression the OpCall site
// should assign to in.Dst plus ok=true. A non-intrinsic name
// returns ok=false so the caller falls through to the plain
// `name(args...)` emission.
//
// The function is deliberately total: every currently-recognised
// intrinsic has a helper in this file and every unrecognised name
// just returns false.
func tryEmitCompilerIntrinsic(in mir.Inst) (string, bool) {
	name := in.CallName
	if !strings.HasPrefix(name, compilerIntrinsicPrefix) {
		return "", false
	}
	rest := strings.TrimPrefix(name, compilerIntrinsicPrefix)
	// Split the intrinsic name from any `__payload` (e.g. the type
	// for ptr_null, the byte count for size_of). The payload is
	// opaque to this function — each arm interprets its own.
	op, payload, hasPayload := cutOnce(rest, "__")
	regExprs := make([]string, len(in.CallArgs))
	for i, a := range in.CallArgs {
		regExprs[i] = fmt.Sprintf("r%d", a)
	}
	switch op {
	case "unreachable", "likely", "unlikely", "assume", "fence", "prefetch":
		return EmitIntrinsic(op, regExprs), true
	case "ptr_null":
		target := ""
		if hasPayload {
			target = payload
		}
		return EmitPtrNull(target), true
	case "size_of":
		bytes := uint64(0)
		if hasPayload {
			if n, err := strconv.ParseUint(payload, 10, 64); err == nil {
				bytes = n
			}
		}
		return EmitSizeOf(bytes), true
	}
	return "", false
}

// cutOnce is strings.Cut from Go 1.18+; replicated locally to
// avoid pulling the import graph wider just for one call site.
func cutOnce(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// EmitIntrinsic renders the C-level expansion for a Fuse compiler
// intrinsic. Reference §57 enumerates the intrinsic set; W17 maps
// each to the portable gcc / clang builtin. Unknown names return
// an explanatory comment.
//
// Arg count contracts:
//   - unreachable, fence: 0 args
//   - likely, unlikely, assume: 1 arg (the expression)
//   - prefetch: 1–3 args (address, rw, locality)
func EmitIntrinsic(name string, args []string) string {
	switch name {
	case "unreachable":
		return "__builtin_unreachable()"
	case "likely":
		if len(args) != 1 {
			return fmt.Sprintf("/* likely: expected 1 arg, got %d */", len(args))
		}
		return fmt.Sprintf("__builtin_expect((long)(%s), 1)", args[0])
	case "unlikely":
		if len(args) != 1 {
			return fmt.Sprintf("/* unlikely: expected 1 arg, got %d */", len(args))
		}
		return fmt.Sprintf("__builtin_expect((long)(%s), 0)", args[0])
	case "assume":
		if len(args) != 1 {
			return fmt.Sprintf("/* assume: expected 1 arg, got %d */", len(args))
		}
		// Portable spelling: `if (!cond) __builtin_unreachable();`
		return fmt.Sprintf("do { if (!(%s)) __builtin_unreachable(); } while (0)", args[0])
	case "fence":
		// Acquire-release fence. Reference §17 concurrency-fences.
		return "__atomic_thread_fence(__ATOMIC_SEQ_CST)"
	case "prefetch":
		// Prefetch accepts address plus optional (rw, locality).
		// Fill defaults (read, high locality) for missing args.
		addr := ""
		rw := "0"
		locality := "3"
		if len(args) >= 1 {
			addr = args[0]
		}
		if len(args) >= 2 {
			rw = args[1]
		}
		if len(args) >= 3 {
			locality = args[2]
		}
		return fmt.Sprintf("__builtin_prefetch((const void*)(%s), %s, %s)", addr, rw, locality)
	}
	return fmt.Sprintf("/* intrinsic %q — unknown */", name)
}

// EmitVariadicCall renders a call to an extern variadic C function
// with per-argument default promotions applied: float → double,
// short / char → int, everything else passed through unchanged.
//
// The caller supplies the function name, the raw argument
// expressions, and the parallel argTypes (one kind per arg). When
// argTypes is empty we fall back to a no-promotion call — the
// checker is the authoritative source of "which args need
// promotion"; the emitter just obeys.
func EmitVariadicCall(fnName string, args []string, argTypes []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s(", fnName)
	for i, a := range args {
		if i > 0 {
			sb.WriteString(", ")
		}
		kind := ""
		if i < len(argTypes) {
			kind = argTypes[i]
		}
		sb.WriteString(promoteVariadicArg(a, kind))
	}
	sb.WriteByte(')')
	return sb.String()
}

// promoteVariadicArg wraps the arg expression in the explicit C
// cast that the variadic default-argument-promotion rules require.
// Reference C11 §6.5.2.2p6–7.
func promoteVariadicArg(expr, kind string) string {
	switch kind {
	case "f32", "f64", "float":
		return fmt.Sprintf("(double)(%s)", expr)
	case "i8", "i16", "I8", "I16":
		return fmt.Sprintf("(int)(%s)", expr)
	case "u8", "u16", "U8", "U16":
		return fmt.Sprintf("(unsigned int)(%s)", expr)
	}
	// Bool / int / long / pointer-typed args need no promotion;
	// pass them through unchanged.
	return expr
}

// EmitPtrNull returns the C expression for `Ptr.null[T]()`. The
// target type is baked into the cast so the resulting expression
// is addressable through the same type as an ordinary Ptr[T].
func EmitPtrNull(targetType string) string {
	if targetType == "" {
		return "((void*)0)"
	}
	return fmt.Sprintf("((%s*)0)", targetType)
}

// EmitSizeOf / EmitAlignOf return the C literal for a `size_of[T]()`
// / `align_of[T]()` invocation in runtime position. The W14
// consteval pass rewrites compile-time uses to a literal TypeId-
// keyed integer; these helpers emit the same value for positions
// that the evaluator cannot fold at compile time (for example,
// a dynamically-selected type id in future waves).
func EmitSizeOf(bytes uint64) string {
	return fmt.Sprintf("((uint64_t)%d)", bytes)
}

func EmitAlignOf(bytes uint64) string {
	return fmt.Sprintf("((uint64_t)%d)", bytes)
}

// EmitSizeOfVal returns the C expression for `size_of_val(ref v)`.
// The result is the byte-size of the referenced value, obtained
// via sizeof-of-dereference on the pointer the ref is represented
// as.
func EmitSizeOfVal(refExpr string) string {
	return fmt.Sprintf("((uint64_t)sizeof(*(%s)))", refExpr)
}
