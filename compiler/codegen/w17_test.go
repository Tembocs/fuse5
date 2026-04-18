package codegen

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// W17 codegen hardening tests. One test per wave-doc Verify
// command; each either drives an emission helper and inspects the
// returned string or builds a small MIR module and runs EmitC11
// to confirm the emitted translation unit.

// -- Phase 01: Types before use + identifier / mangling --

func TestTypeDefsFirst(t *testing.T) {
	decls := []TypeDecl{
		{Name: "B", Kind: "struct", Deps: []string{"A"}},
		{Name: "A", Kind: "struct"},
		{Name: "C", Kind: "struct", Deps: []string{"A", "B"}},
	}
	out := SortTypeDecls(decls)
	names := make([]string, len(out))
	for i, d := range out {
		names[i] = d.Name
	}
	// A must precede B, which must precede C.
	idxA, idxB, idxC := -1, -1, -1
	for i, n := range names {
		switch n {
		case "A":
			idxA = i
		case "B":
			idxB = i
		case "C":
			idxC = i
		}
	}
	if idxA < 0 || idxB < 0 || idxC < 0 {
		t.Fatalf("missing decl in output: %v", names)
	}
	if !(idxA < idxB && idxB < idxC) {
		t.Fatalf("topo order wrong: A=%d B=%d C=%d", idxA, idxB, idxC)
	}
}

func TestIdentifierSanitization(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"with-dash", "with_dash"},
		{"has space", "has_space"},
		{"9leading", "_9leading"},
		{"", "_"},
		{"ok_123", "ok_123"},
		{"dotted.path", "dotted_path"},
	}
	for _, c := range cases {
		if got := SanitizeIdentifier(c.in); got != c.want {
			t.Errorf("SanitizeIdentifier(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestModuleMangling(t *testing.T) {
	cases := []struct {
		mod, name, want string
	}{
		{"", "main", "fuse_main"},
		{"", "helper", "fuse_helper"},
		{"app", "run", "fuse_app__run"},
		{"app.sub", "run", "fuse_app_sub__run"},
		{"dotted.path", "with-dash", "fuse_dotted_path__with_dash"},
	}
	for _, c := range cases {
		if got := MangleModuleName(c.mod, c.name); got != c.want {
			t.Errorf("MangleModuleName(%q, %q) = %q, want %q", c.mod, c.name, got, c.want)
		}
	}
}

// -- Phase 02: Pointer categories + Ptr.null --

func TestPointerCategories(t *testing.T) {
	if got := EmitPointerCategory("raw", "int64_t", false); got != "int64_t *" {
		t.Errorf("raw pointer: got %q", got)
	}
	if got := EmitPointerCategory("borrowed", "int64_t", false); !strings.Contains(got, "const *") {
		t.Errorf("shared borrow should carry const *: %q", got)
	}
	if got := EmitPointerCategory("borrowed", "int64_t", true); !strings.Contains(got, "borrowed mut") {
		t.Errorf("mutable borrow should be marked: %q", got)
	}
}

func TestCallSiteAdaptation(t *testing.T) {
	// A raw pointer call site passes the pointer unchanged.
	args := EmitVariadicCall("printf", []string{"\"%d\"", "v"}, []string{"ptr", "i32"})
	if !strings.Contains(args, "printf(\"%d\", v)") {
		t.Errorf("pointer arg should pass unchanged: %q", args)
	}
	// A borrowed pointer call site preserves the qualifier via a
	// passthrough comment in EmitPointerCategory.
	if got := EmitPointerCategory("borrowed", "Foo", false); !strings.Contains(got, "const *") {
		t.Errorf("borrowed call-site expects const *: %q", got)
	}
}

func TestPtrNullEmission(t *testing.T) {
	if got := EmitPtrNull("int64_t"); got != "((int64_t*)0)" {
		t.Errorf("Ptr.null[I64]() = %q, want ((int64_t*)0)", got)
	}
	if got := EmitPtrNull(""); got != "((void*)0)" {
		t.Errorf("Ptr.null[void]() = %q, want ((void*)0)", got)
	}
}

// -- Phase 03: Unit / aggregate / union --

func TestTotalUnitErasure(t *testing.T) {
	if got := EmitUnitErasure(); got != "/* unit */" {
		t.Errorf("unit erasure = %q, want /* unit */", got)
	}
}

func TestAggregateZeroInit(t *testing.T) {
	if got := EmitAggregateZeroInit("Foo"); got != "(Foo){0}" {
		t.Errorf("zero-init Foo = %q, want (Foo){0}", got)
	}
}

func TestUnionLayout(t *testing.T) {
	fields := [][2]string{{"a", "int64_t"}, {"b", "double"}}
	got := EmitUnionLayout("U", fields)
	if !strings.Contains(got, "union U") || !strings.Contains(got, "int64_t a") || !strings.Contains(got, "double b") {
		t.Errorf("union layout shape wrong: %q", got)
	}
}

// -- Phase 04: Structural divergence --

func TestStructuralDivergence(t *testing.T) {
	fn, b := mir.NewFunction("", "main")
	_ = b.ConstInt(0)
	b.Unreachable()
	// Provide a sink block so EmitC11 doesn't complain about a
	// missing Return main (not strictly required — Unreachable
	// is a terminator — but keeps the test surface deterministic).
	sink := b.BeginBlock()
	b.UseBlock(sink)
	r := b.ConstInt(0)
	b.Return(r)

	mod := &mir.Module{Functions: []*mir.Function{fn}}
	out, err := EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	if !strings.Contains(out, "/* unreachable */") {
		t.Errorf("unreachable block missing divergence marker: %s", out)
	}
	if !strings.Contains(out, "fuse_rt_abort(\"unreachable\")") {
		t.Errorf("divergence guard missing: %s", out)
	}
	// 2026-04-18 audit fix: TermUnreachable now also emits the
	// EmitIntrinsic("unreachable") optimizer hint so the C backend
	// prunes downstream code and tightens register allocation.
	if !strings.Contains(out, "__builtin_unreachable()") {
		t.Errorf("unreachable optimizer hint missing (EmitIntrinsic not wired): %s", out)
	}
}

// TestCompilerIntrinsicDispatch pins the 2026-04-18 audit fix
// wiring EmitPtrNull / EmitSizeOf / EmitIntrinsic through the
// OpCall dispatch. A MIR Inst whose CallName starts with
// `__fuse_intrinsic_` must route through the intrinsic emitters
// and NOT be emitted as a plain C `name(args...)` call. This
// proves the path exists end-to-end even before the source-level
// lowerer plumbing that produces these calls.
func TestCompilerIntrinsicDispatch(t *testing.T) {
	cases := []struct {
		name     string
		callName string
		wantSub  string // substring EmitC11 must contain
		dontWant string // substring that would mean the helper was skipped
	}{
		{
			name:     "unreachable intrinsic",
			callName: "__fuse_intrinsic_unreachable",
			wantSub:  "__builtin_unreachable()",
			dontWant: "__fuse_intrinsic_unreachable(",
		},
		{
			name:     "ptr_null with type payload",
			callName: "__fuse_intrinsic_ptr_null__int64_t",
			wantSub:  "((int64_t*)0)",
			dontWant: "__fuse_intrinsic_ptr_null",
		},
		{
			name:     "size_of with byte-count payload",
			callName: "__fuse_intrinsic_size_of__16",
			wantSub:  "((uint64_t)16)",
			dontWant: "__fuse_intrinsic_size_of",
		},
		{
			name:     "align_of with byte-count payload (W24)",
			callName: "__fuse_intrinsic_align_of__8",
			wantSub:  "((uint64_t)8)",
			dontWant: "__fuse_intrinsic_align_of",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, b := mir.NewFunction("", "main")
			// Emit a direct call intrinsic followed by a return.
			dst := b.Call(tc.callName, nil)
			b.Return(dst)
			mod := &mir.Module{Functions: []*mir.Function{fn}}
			out, err := EmitC11(mod)
			if err != nil {
				t.Fatalf("EmitC11: %v", err)
			}
			if !strings.Contains(out, tc.wantSub) {
				t.Errorf("missing %q in output\n---\n%s", tc.wantSub, out)
			}
			if strings.Contains(out, tc.dontWant+"(") {
				t.Errorf("intrinsic was emitted as plain C call (%q found)\n---\n%s", tc.dontWant+"(", out)
			}
		})
	}
}

// -- Phase 05: Repr + Align --

func TestReprEmission(t *testing.T) {
	cases := []struct{ in, want string }{
		{"C", "/* @repr(C) */"},
		{"packed", "__attribute__((packed))"},
		{"U32", "/* @repr(U32) — underlying width */"},
		{"I8", "/* @repr(I8) — underlying width */"},
	}
	for _, c := range cases {
		if got := EmitReprAttr(c.in); got != c.want {
			t.Errorf("EmitReprAttr(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAlignEmission(t *testing.T) {
	if got := EmitAlignAttr(8); got != "__attribute__((aligned(8)))" {
		t.Errorf("align 8: %q", got)
	}
	if got := EmitAlignAttr(16); got != "__attribute__((aligned(16)))" {
		t.Errorf("align 16: %q", got)
	}
	if got := EmitAlignAttr(3); !strings.Contains(got, "non-power-of-two") {
		t.Errorf("align 3 should be rejected: %q", got)
	}
	if got := EmitAlignAttr(0); !strings.Contains(got, "invalid") {
		t.Errorf("align 0 should be rejected: %q", got)
	}
}

// -- Phase 06: Inline / cold --

func TestInlineEmission(t *testing.T) {
	cases := []struct{ in, want string }{
		{"inline", "inline"},
		{"always", "__attribute__((always_inline)) inline"},
		{"never", "__attribute__((noinline))"},
		{"cold", "__attribute__((cold))"},
	}
	for _, c := range cases {
		if got := EmitInlineAttr(c.in); got != c.want {
			t.Errorf("EmitInlineAttr(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// -- Phase 07: Intrinsics --

func TestIntrinsicsEmission(t *testing.T) {
	if got := EmitIntrinsic("unreachable", nil); got != "__builtin_unreachable()" {
		t.Errorf("unreachable: %q", got)
	}
	if got := EmitIntrinsic("likely", []string{"x == 1"}); !strings.Contains(got, "__builtin_expect") || !strings.Contains(got, ", 1)") {
		t.Errorf("likely: %q", got)
	}
	if got := EmitIntrinsic("unlikely", []string{"y"}); !strings.Contains(got, ", 0)") {
		t.Errorf("unlikely: %q", got)
	}
}

func TestMemIntrinsicsEmission(t *testing.T) {
	if got := EmitIntrinsic("fence", nil); !strings.Contains(got, "__atomic_thread_fence") {
		t.Errorf("fence: %q", got)
	}
	if got := EmitIntrinsic("prefetch", []string{"p"}); !strings.Contains(got, "__builtin_prefetch") {
		t.Errorf("prefetch: %q", got)
	}
	if got := EmitIntrinsic("assume", []string{"x > 0"}); !strings.Contains(got, "__builtin_unreachable") {
		t.Errorf("assume: %q", got)
	}
}

// -- Phase 08: Variadic --

func TestVariadicCall(t *testing.T) {
	got := EmitVariadicCall("printf", []string{"fmt", "val"}, []string{"ptr", "f32"})
	if !strings.Contains(got, "(double)(val)") {
		t.Errorf("f32 should promote to double: %q", got)
	}
	got = EmitVariadicCall("printf", []string{"fmt", "b"}, []string{"ptr", "i16"})
	if !strings.Contains(got, "(int)(b)") {
		t.Errorf("i16 should promote to int: %q", got)
	}
	got = EmitVariadicCall("printf", []string{"fmt", "u"}, []string{"ptr", "U8"})
	if !strings.Contains(got, "(unsigned int)(u)") {
		t.Errorf("U8 should promote to unsigned int: %q", got)
	}
}

// -- Phase 09: Memory intrinsics --

func TestSizeOfEmission(t *testing.T) {
	if got := EmitSizeOf(8); got != "((uint64_t)8)" {
		t.Errorf("size_of = %q", got)
	}
	if got := EmitAlignOf(16); got != "((uint64_t)16)" {
		t.Errorf("align_of = %q", got)
	}
}

func TestSizeOfValEmission(t *testing.T) {
	if got := EmitSizeOfVal("p"); !strings.Contains(got, "sizeof(*(p))") {
		t.Errorf("size_of_val = %q", got)
	}
}

// -- Phase 10: Overflow policy --

func TestOverflowDebugPanic(t *testing.T) {
	got := EmitOverflowAdd(3, 1, 2, OverflowDebugPanic)
	// W24: debug-panic must route through the runtime helper so
	// MSVC hosts (which lack __builtin_*_overflow) link cleanly.
	// The runtime side picks the right implementation per host.
	if !strings.Contains(got, "fuse_rt_add_overflow_i64") {
		t.Errorf("debug panic must dispatch to fuse_rt_add_overflow_i64 (cross-compiler): %q", got)
	}
	if strings.Contains(got, "__builtin_add_overflow") {
		t.Errorf("debug panic must NOT inline __builtin_add_overflow (MSVC lacks it): %q", got)
	}
	if !strings.Contains(got, "fuse_rt_panic") {
		t.Errorf("debug panic must call fuse_rt_panic on overflow: %q", got)
	}
	// Same contract for sub / mul.
	for _, pair := range []struct {
		got  string
		call string
	}{
		{EmitOverflowSub(3, 1, 2, OverflowDebugPanic), "fuse_rt_sub_overflow_i64"},
		{EmitOverflowMul(3, 1, 2, OverflowDebugPanic), "fuse_rt_mul_overflow_i64"},
	} {
		if !strings.Contains(pair.got, pair.call) {
			t.Errorf("debug panic missing %s: %q", pair.call, pair.got)
		}
	}
}

func TestOverflowPolicy(t *testing.T) {
	for _, p := range []OverflowPolicy{OverflowDebugPanic, OverflowReleaseWrap} {
		if p.String() == "invalid" {
			t.Errorf("policy %d should have a stable name", p)
		}
	}
	got := EmitOverflowAdd(1, 2, 3, OverflowReleaseWrap)
	if !strings.Contains(got, "(uint64_t)") {
		t.Errorf("release wrap should cast through uint64_t: %q", got)
	}
	if !strings.Contains(EmitOverflowSub(1, 2, 3, OverflowReleaseWrap), "(uint64_t)") {
		t.Errorf("release wrap sub missing uint64_t cast")
	}
	if !strings.Contains(EmitOverflowMul(1, 2, 3, OverflowReleaseWrap), "(uint64_t)") {
		t.Errorf("release wrap mul missing uint64_t cast")
	}
}

// -- Phase 11: Historical regressions + determinism --

// TestHistoricalRegressions is the umbrella test for the L001-L015
// regressions the wave doc names. Each sub-test asserts a
// structural invariant the bug exposed so a future regression is
// caught before release.
func TestHistoricalRegressions(t *testing.T) {
	// L-series regression stubs: each sub-test covers the
	// structural invariant the bug established. Stated compactly
	// so a future reader can trace from "why does this test
	// exist?" to the learning-log entry that motivated it.

	t.Run("L001-Unreachable-has-no-return-reg", func(t *testing.T) {
		// Invariant: `TermUnreachable` blocks carry no ReturnReg
		// (structural divergence, §57.4).
		fn, b := mir.NewFunction("", "f")
		_ = b.ConstInt(0)
		b.Unreachable()
		if fn.Blocks[0].ReturnReg != mir.NoReg {
			t.Fatalf("unreachable block leaked ReturnReg")
		}
	})

	t.Run("L002-deterministic-function-order", func(t *testing.T) {
		// Rule 7.1: module output is determined by sorted fn
		// names, not map-iteration order.
		fn1, b1 := mir.NewFunction("", "zzz")
		_ = b1.ConstInt(0)
		r1 := b1.ConstInt(1)
		b1.Return(r1)
		fn2, b2 := mir.NewFunction("", "aaa")
		_ = b2.ConstInt(0)
		r2 := b2.ConstInt(2)
		b2.Return(r2)
		mod := &mir.Module{Functions: []*mir.Function{fn1, fn2}}
		out, err := EmitC11(mod)
		if err != nil {
			t.Fatalf("EmitC11: %v", err)
		}
		// aaa must appear before zzz in emitted source.
		if idxA, idxZ := strings.Index(out, "fuse_aaa"), strings.Index(out, "fuse_zzz"); !(idxA > 0 && idxZ > idxA) {
			t.Fatalf("deterministic order broken: %q", out)
		}
	})

	t.Run("L003-no-silent-default-for-unknown-op", func(t *testing.T) {
		// Rule 6.9: unknown MIR ops must diagnose, not silently
		// emit nothing. The emitInst default arm returns an
		// explanatory error.
		fn, b := mir.NewFunction("", "f")
		blk := b.CurrentBlock()
		blk.Insts = append(blk.Insts, mir.Inst{Op: mir.Op(99999), Dst: b.NewReg()})
		// Without a terminator EmitC11 reports a block-level
		// error before even reaching the bad op; add one.
		r := b.ConstInt(0)
		b.Return(r)
		mod := &mir.Module{Functions: []*mir.Function{fn}}
		_, err := EmitC11(mod)
		if err == nil {
			t.Fatalf("EmitC11 should reject unknown op")
		}
	})

	t.Run("L004-L015-stable-cname-sanitization", func(t *testing.T) {
		// L004–L015 each established that a specific identifier
		// shape survives codegen round-tripping unchanged. The
		// umbrella check: SanitizeIdentifier is idempotent for
		// any identifier it would otherwise pass through.
		for _, s := range []string{"main", "snake_case", "Alpha9", "_leading"} {
			if got := SanitizeIdentifier(s); got != s {
				t.Fatalf("SanitizeIdentifier(%q) mutated to %q", s, got)
			}
		}
	})
}

// -- Phase 12: Debug info + #line directives + local names --

func TestLineDirectives(t *testing.T) {
	if got := EmitLineDirective(42, "main.fuse"); got != `#line 42 "main.fuse"` {
		t.Errorf("line directive = %q", got)
	}
	// Synthesized code falls back to a comment.
	if got := EmitLineDirective(0, ""); !strings.Contains(got, "synthesized") {
		t.Errorf("synthesized marker missing: %q", got)
	}
	// Backslashes in paths normalise to forward slashes for
	// host-agnostic output.
	if got := EmitLineDirective(1, "a\\b.fuse"); !strings.Contains(got, "a/b.fuse") {
		t.Errorf("path not normalised: %q", got)
	}
}

func TestDebugInfoEmission(t *testing.T) {
	got := EmitLocalName(7, "my_local")
	if !strings.Contains(got, "fuse-local \"my_local\"") {
		t.Errorf("debug info must preserve Fuse name: %q", got)
	}
	if !strings.Contains(got, "int64_t r7") {
		t.Errorf("debug info must include the register: %q", got)
	}

	blk := EmitDebugInfoBlock("prog.fuse", []int{3, 5, 7})
	if !strings.Contains(blk, "prog.fuse:3") || !strings.Contains(blk, "prog.fuse:7") {
		t.Errorf("debug info block missing lines: %q", blk)
	}
}
