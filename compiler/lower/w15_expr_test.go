package lower

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/mir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestPointerCastDiagnostic pins the W15 pointer-cast diagnostic
// established by the 2026-04-18 audit fix. A `ptr as int` shape
// (or the reverse) must emit the STUBS.md-declared text rather
// than fall through to the generic "not supported" message.
// Rule 6.9 — no silent stubs.
func TestPointerCastDiagnostic(t *testing.T) {
	l, _, b, tab := w15TestLowerer(t)
	// Build a cast from Ptr[I32] to I64 (a pointer-to-integer
	// cast the W15 classifier rejects).
	ptrType := tab.Ptr(tab.I32())
	innerReg := b.ConstInt(0)
	_ = innerReg
	// Hand-build a CastExpr whose inner already has a Ptr type.
	cast := &hir.CastExpr{
		TypedBase: hir.TypedBase{Type: tab.I64()},
		Expr:      mkLitInt(tab, "0", ptrType),
	}
	_, ok := l.lowerCastTagged("", b, cast, nil)
	if ok {
		t.Fatalf("pointer-to-integer cast should be rejected")
	}
	if len(l.diags) == 0 {
		t.Fatalf("no diagnostic for pointer cast")
	}
	msg := l.diags[len(l.diags)-1].Message
	if !contains(msg, "pointer-to-integer and integer-to-pointer casts not yet classified") {
		t.Errorf("diagnostic does not match STUBS.md declared text: %q", msg)
	}
}

// TestCastLowering verifies the W15 cast-classification ladder
// (reference §28.1): every (source, target) pair produces a
// specific CastMode that feeds OpCast. This is the structural
// consolidation work — codegen hardening lands in W17.
func TestCastLowering(t *testing.T) {
	tab := typetable.New()
	l := &lowerer{
		prog:   &hir.Program{Types: tab, Modules: map[string]*hir.Module{"": {Path: ""}}, Order: []string{""}},
		module: &mir.Module{},
	}
	cases := []struct {
		name string
		src  typetable.TypeId
		dst  typetable.TypeId
		want mir.CastMode
	}{
		{"i8-to-i64-widen", tab.Intern(typetable.Type{Kind: typetable.KindI8}), tab.I64(), mir.CastWiden},
		{"i64-to-i32-narrow", tab.I64(), tab.I32(), mir.CastNarrow},
		{"i32-to-f32-int-to-float", tab.I32(), tab.Intern(typetable.Type{Kind: typetable.KindF32}), mir.CastIntToFloat},
		{"f64-to-i64-float-to-int", tab.Intern(typetable.Type{Kind: typetable.KindF64}), tab.I64(), mir.CastFloatToInt},
		{"i32-to-i32-reinterpret", tab.I32(), tab.I32(), mir.CastReinterpret},
		{"u32-to-i32-same-width-reinterpret",
			tab.Intern(typetable.Type{Kind: typetable.KindU32}), tab.I32(), mir.CastReinterpret},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := l.classifyCast(tc.src, tc.dst)
			if got != tc.want {
				t.Fatalf("classifyCast(%s, %s) = %s, want %s",
					tab.Get(tc.src).Kind, tab.Get(tc.dst).Kind, got, tc.want)
			}
		})
	}

	// End-to-end through the tagged lowerer.
	t.Run("tagged-lowering-emits-op-cast", func(t *testing.T) {
		l2, fn, b, tab2 := w15TestLowerer(t)
		inner := mkLitInt(tab2, "42", tab2.I32())
		cast := &hir.CastExpr{
			TypedBase: hir.TypedBase{Type: tab2.I64()},
			Expr:      inner,
		}
		reg, ok := l2.lowerCastTagged("", b, cast, nil)
		if !ok {
			t.Fatalf("lowerCastTagged returned ok=false; diags=%v", l2.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		c := findInst(fn, mir.OpCast)
		if c == nil {
			t.Fatalf("no OpCast in function")
		}
		if mir.CastMode(c.Mode) != mir.CastWiden {
			t.Fatalf("OpCast.Mode = %s, want widen", mir.CastMode(c.Mode))
		}
	})
}

// TestFnPointerLowering verifies that loading a fn address through
// lowerFnPointer emits OpFnPtr with the expected mangled name.
// Reference §29.1: function pointer values carry no captured state.
func TestFnPointerLowering(t *testing.T) {
	l, fn, b, _ := w15TestLowerer(t)
	reg := l.lowerFnPointer(b, "mymod", "increment")
	b.Return(reg)
	if err := fn.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	fp := findInst(fn, mir.OpFnPtr)
	if fp == nil {
		t.Fatalf("no OpFnPtr in function")
	}
	if fp.CallName != "fuse_mymod__increment" {
		t.Fatalf("OpFnPtr.CallName = %q, want %q", fp.CallName, "fuse_mymod__increment")
	}

	t.Run("indirect-call-through-fn-ptr", func(t *testing.T) {
		l2, fn2, b2, tab := w15TestLowerer(t)
		fnPtr := l2.lowerFnPointer(b2, "", "apply")
		arg := b2.ConstInt(7)
		_ = tab
		result := b2.CallIndirect(fnPtr, []mir.Reg{arg})
		b2.Return(result)
		if err := fn2.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if findInst(fn2, mir.OpCallIndirect) == nil {
			t.Fatalf("no OpCallIndirect in function")
		}
	})
}

// TestSliceRangeLowering verifies that `arr[low..high]` emits a
// single OpSliceNew with base, low, high, and the inclusive flag.
// Reference §32.1: slice descriptor `{ ptr: base + a, len: b - a }`.
func TestSliceRangeLowering(t *testing.T) {
	cases := []struct {
		name       string
		low, high  bool // whether to include the endpoints
		inclusive  bool
	}{
		{"closed-exclusive", true, true, false},
		{"closed-inclusive", true, true, true},
		{"open-low", false, true, false},
		{"open-high", true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, fn, b, tab := w15TestLowerer(t)
			arrType := tab.Array(tab.I32(), 10)
			base := mkLitInt(tab, "0", arrType)
			var low, high hir.Expr
			if tc.low {
				low = mkLitInt(tab, "2", tab.I32())
			}
			if tc.high {
				high = mkLitInt(tab, "7", tab.I32())
			}
			rng := &hir.IndexRangeExpr{
				TypedBase: hir.TypedBase{Type: tab.Slice(tab.I32())},
				Receiver:  base,
				Low:       low,
				High:      high,
				Inclusive: tc.inclusive,
			}
			reg, ok := l.lowerIndexRange("", b, rng, nil)
			if !ok {
				t.Fatalf("lowerIndexRange returned ok=false; diags=%v", l.diags)
			}
			b.Return(reg)
			if err := fn.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			sl := findInst(fn, mir.OpSliceNew)
			if sl == nil {
				t.Fatalf("no OpSliceNew in function")
			}
			if sl.Flag != tc.inclusive {
				t.Fatalf("OpSliceNew.Flag = %v, want inclusive=%v", sl.Flag, tc.inclusive)
			}
			if tc.low {
				if sl.Rhs == mir.NoReg {
					t.Fatalf("closed low endpoint lowered to NoReg")
				}
			} else {
				if sl.Rhs != mir.NoReg {
					t.Fatalf("open-low endpoint lowered to register %d; expected NoReg", sl.Rhs)
				}
			}
			if tc.high {
				if sl.Extra == mir.NoReg {
					t.Fatalf("closed high endpoint lowered to NoReg")
				}
			} else {
				if sl.Extra != mir.NoReg {
					t.Fatalf("open-high endpoint lowered to register %d; expected NoReg", sl.Extra)
				}
			}
		})
	}
}

// TestStructUpdateLowering verifies the `Path { f: v, ..base }` form
// (reference §45.1 — explicit-field precedence). Two shapes:
//
//   - no base: OpStructNew + OpFieldWrite per explicit field
//   - with base: OpStructCopy, then OpFieldWrite per override
//
// In both cases the final writes correspond to explicit fields, so
// §45.1 "explicit-field precedence over spread" holds structurally.
func TestStructUpdateLowering(t *testing.T) {
	t.Run("plain-literal-emits-struct-new", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		configType := tab.Nominal(typetable.KindStruct, 7, "", "Config", nil)
		lit := &hir.StructLitExpr{
			TypedBase:  hir.TypedBase{Type: configType},
			StructType: configType,
			Fields: []*hir.StructLitField{
				{Name: "width", Value: mkLitInt(tab, "640", tab.I32())},
				{Name: "height", Value: mkLitInt(tab, "480", tab.I32())},
			},
			Base: nil,
		}
		reg, ok := l.lowerStructLit("", b, lit, nil)
		if !ok {
			t.Fatalf("lowerStructLit returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		sn := findInst(fn, mir.OpStructNew)
		if sn == nil {
			t.Fatalf("no OpStructNew in function")
		}
		if sn.CallName != "Config" {
			t.Fatalf("OpStructNew.CallName = %q, want %q", sn.CallName, "Config")
		}
		if countInsts(fn, mir.OpFieldWrite) != 2 {
			t.Fatalf("expected 2 OpFieldWrite insts, got %d", countInsts(fn, mir.OpFieldWrite))
		}
	})
	t.Run("with-base-emits-struct-copy-then-overrides", func(t *testing.T) {
		l, fn, b, tab := w15TestLowerer(t)
		configType := tab.Nominal(typetable.KindStruct, 8, "", "Config", nil)
		base := mkLitInt(tab, "0", configType)
		lit := &hir.StructLitExpr{
			TypedBase:  hir.TypedBase{Type: configType},
			StructType: configType,
			Fields: []*hir.StructLitField{
				{Name: "width", Value: mkLitInt(tab, "800", tab.I32())},
			},
			Base: base,
		}
		reg, ok := l.lowerStructLit("", b, lit, nil)
		if !ok {
			t.Fatalf("lowerStructLit returned ok=false; diags=%v", l.diags)
		}
		b.Return(reg)
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if findInst(fn, mir.OpStructCopy) == nil {
			t.Fatalf("no OpStructCopy in function")
		}
		// The one explicit field must produce exactly one OpFieldWrite.
		if countInsts(fn, mir.OpFieldWrite) != 1 {
			t.Fatalf("expected 1 OpFieldWrite (explicit override), got %d", countInsts(fn, mir.OpFieldWrite))
		}
		// And no OpStructNew (the base carries the type).
		if findInst(fn, mir.OpStructNew) != nil {
			t.Fatalf("base-update form should not emit OpStructNew")
		}
	})
	t.Run("explicit-field-comes-after-copy", func(t *testing.T) {
		// Reference §45.1: explicit assignments take precedence.
		// At MIR level this must manifest as "the copy precedes
		// every explicit write", so the final state reflects the
		// explicit value, not the base's value.
		l, fn, b, tab := w15TestLowerer(t)
		configType := tab.Nominal(typetable.KindStruct, 9, "", "Config", nil)
		base := mkLitInt(tab, "0", configType)
		lit := &hir.StructLitExpr{
			TypedBase:  hir.TypedBase{Type: configType},
			StructType: configType,
			Fields: []*hir.StructLitField{
				{Name: "width", Value: mkLitInt(tab, "99", tab.I32())},
			},
			Base: base,
		}
		_, ok := l.lowerStructLit("", b, lit, nil)
		if !ok {
			t.Fatalf("lowerStructLit returned ok=false; diags=%v", l.diags)
		}
		// Walk the insts of the only block and find the first
		// OpStructCopy and first OpFieldWrite; the copy must precede.
		var copyIdx, writeIdx int = -1, -1
		blk := fn.Blocks[0]
		for i, in := range blk.Insts {
			if copyIdx == -1 && in.Op == mir.OpStructCopy {
				copyIdx = i
			}
			if writeIdx == -1 && in.Op == mir.OpFieldWrite {
				writeIdx = i
			}
		}
		if copyIdx == -1 || writeIdx == -1 {
			t.Fatalf("missing OpStructCopy or OpFieldWrite in block: %+v", blk.Insts)
		}
		if copyIdx >= writeIdx {
			t.Fatalf("OpStructCopy at %d must precede OpFieldWrite at %d", copyIdx, writeIdx)
		}
	})
}

// TestOverflowArithmeticLowering verifies that each
// `wrapping_*` / `checked_*` / `saturating_*` method call lowers
// to its dedicated MIR op (reference §33.1). Each op is picked up
// later by W17 codegen for policy-specific emission.
//
// Bound by the wave-doc Verify command:
//
//	go test ./compiler/lower/... -run TestOverflowArithmeticLowering -v
func TestOverflowArithmeticLowering(t *testing.T) {
	cases := []struct {
		method string
		op     mir.Op
	}{
		{"wrapping_add", mir.OpWrappingAdd},
		{"wrapping_sub", mir.OpWrappingSub},
		{"wrapping_mul", mir.OpWrappingMul},
		{"checked_add", mir.OpCheckedAdd},
		{"checked_sub", mir.OpCheckedSub},
		{"checked_mul", mir.OpCheckedMul},
		{"saturating_add", mir.OpSaturatingAdd},
		{"saturating_sub", mir.OpSaturatingSub},
		{"saturating_mul", mir.OpSaturatingMul},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			l, fn, b, tab := w15TestLowerer(t)
			// `lhs.method(rhs)` → FieldExpr + CallExpr.
			lhs := mkLitInt(tab, "1", tab.I32())
			rhs := mkLitInt(tab, "2", tab.I32())
			field := &hir.FieldExpr{
				TypedBase: hir.TypedBase{Type: tab.I32()},
				Receiver:  lhs,
				Name:      tc.method,
			}
			call := &hir.CallExpr{
				TypedBase: hir.TypedBase{Type: tab.I32()},
				Callee:    field,
				Args:      []hir.Expr{rhs},
			}
			reg, ok, recognized := l.lowerOverflowMethod("", b, call, field, nil)
			if !recognized {
				t.Fatalf("lowerOverflowMethod did not recognize %q", tc.method)
			}
			if !ok {
				t.Fatalf("lowerOverflowMethod returned ok=false; diags=%v", l.diags)
			}
			b.Return(reg)
			if err := fn.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if findInst(fn, tc.op) == nil {
				t.Fatalf("%q did not emit %s; insts=%+v", tc.method, tc.op, fn.Blocks[0].Insts)
			}
		})
	}

	t.Run("non-overflow-method-not-recognized", func(t *testing.T) {
		l, _, b, tab := w15TestLowerer(t)
		lhs := mkLitInt(tab, "1", tab.I32())
		rhs := mkLitInt(tab, "2", tab.I32())
		field := &hir.FieldExpr{
			TypedBase: hir.TypedBase{Type: tab.I32()},
			Receiver:  lhs,
			Name:      "into_string",
		}
		call := &hir.CallExpr{
			TypedBase: hir.TypedBase{Type: tab.I32()},
			Callee:    field,
			Args:      []hir.Expr{rhs},
		}
		_, _, recognized := l.lowerOverflowMethod("", b, call, field, nil)
		if recognized {
			t.Fatalf("lowerOverflowMethod recognized a non-overflow method %q", field.Name)
		}
	})
}

// silence unused-import check for lex when cases rearrange.
var _ = lex.Span{}
