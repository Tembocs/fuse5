package lower

// W15 lowering forms: references, method/field disambiguation,
// semantic equality, optional chaining, casts, function pointers,
// slice ranges, struct literals with update, and overflow-policy
// arithmetic.
//
// Each form is in a dedicated method on *lowerer so the production
// dispatch in lowerExpr can route to it without becoming a 300-line
// switch. The split also makes each form independently testable by
// constructing a small HIR fragment and calling the method directly.

import (
	"fmt"
	"strings"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/mir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// lowerReference emits an OpBorrow for `&expr` / `&mut expr`.
// Reference §14 (borrows) — the checker has already validated
// aliasing; lowering produces the address-of MIR op.
func (l *lowerer) lowerReference(modPath string, b *mir.Builder, x *hir.ReferenceExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	inner, ok := l.lowerExpr(modPath, b, x.Inner, params)
	if !ok {
		return mir.NoReg, false
	}
	return b.Borrow(inner, x.Mutable), true
}

// lowerFieldAccess emits an OpFieldRead for `receiver.name` when
// used as a value (reference §9.4). Method-vs-field disambiguation
// happens at the CallExpr site, not here; by the time lowerExpr
// reaches a bare FieldExpr, we know it is a field read.
func (l *lowerer) lowerFieldAccess(modPath string, b *mir.Builder, x *hir.FieldExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	recv, ok := l.lowerExpr(modPath, b, x.Receiver, params)
	if !ok {
		return mir.NoReg, false
	}
	if x.Name == "" {
		l.diagnose(x.NodeSpan(),
			"field access with empty field name",
			"this is a bridge bug, not a user error")
		return mir.NoReg, false
	}
	return b.FieldRead(recv, x.Name), true
}

// lowerMethodCall produces an OpMethodCall for `receiver.name(args...)`.
// The caller (lowerCallExpr) has already verified the callee is a
// FieldExpr and is responsible for this dispatch branch per
// reference §9.4. The mangled method name is built from the
// receiver's declared nominal type so codegen can locate the
// concrete impl.
func (l *lowerer) lowerMethodCall(modPath string, b *mir.Builder, call *hir.CallExpr, field *hir.FieldExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	recv, ok := l.lowerExpr(modPath, b, field.Receiver, params)
	if !ok {
		return mir.NoReg, false
	}
	argRegs := make([]mir.Reg, 0, len(call.Args))
	for _, a := range call.Args {
		r, ok := l.lowerExpr(modPath, b, a, params)
		if !ok {
			return mir.NoReg, false
		}
		argRegs = append(argRegs, r)
	}
	method := l.methodMangledName(field.Receiver, field.Name)
	return b.MethodCall(method, recv, argRegs), true
}

// methodMangledName derives a stable C-safe name for a method. The
// scheme is `<TypeName>__<MethodName>` so that `Counter::get` becomes
// `Counter__get`. Unresolved receiver types fall back to a best-
// effort name that codegen will diagnose at emission; this matches
// Rule 6.9 (silent stubs forbidden — codegen errors out clearly).
func (l *lowerer) methodMangledName(receiver hir.Expr, method string) string {
	recvType := receiver.TypeOf()
	if t := l.prog.Types.Get(recvType); t != nil && t.Name != "" {
		return sanitizeCName(t.Name) + "__" + sanitizeCName(method)
	}
	return "unknown_type__" + sanitizeCName(method)
}

// sanitizeCName turns a Fuse name into a C-safe identifier.
func sanitizeCName(s string) string {
	if s == "" {
		return "_"
	}
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			out.WriteByte(c)
		} else {
			out.WriteByte('_')
		}
	}
	return out.String()
}

// lowerSemanticEquality emits OpEqScalar for primitive operand
// types and OpEqCall for nominals (reference §5.8). Lhs and Rhs
// must have already been type-checked to the same TypeId by W06.
//
// This is called from lowerBinary when the operator is BinEq or
// BinNe. BinNe additionally negates the result (added as a
// post-processing XOR with 1); the surface differs only in the
// final polarity.
func (l *lowerer) lowerSemanticEquality(modPath string, b *mir.Builder, x *hir.BinaryExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	if x.Op != hir.BinEq && x.Op != hir.BinNe {
		return mir.NoReg, false
	}
	lhs, ok := l.lowerExpr(modPath, b, x.Lhs, params)
	if !ok {
		return mir.NoReg, false
	}
	rhs, ok := l.lowerExpr(modPath, b, x.Rhs, params)
	if !ok {
		return mir.NoReg, false
	}
	operandType := x.Lhs.TypeOf()
	var eqReg mir.Reg
	if l.isScalarType(operandType) {
		eqReg = b.EqScalar(lhs, rhs)
	} else {
		eqName := l.equalityMangledName(operandType)
		eqReg = b.EqCall(eqName, lhs, rhs)
	}
	if x.Op == hir.BinNe {
		// Negate: result = eqReg XOR 1. Fall back to arithmetic
		// since the W05 MIR has no dedicated not op; codegen emits
		// `!r` from the bit pattern 0/1. OpSub(1, eqReg) produces
		// the same 0/1 flip at codegen without a new op.
		one := b.ConstInt(1)
		return b.Binary(mir.OpSub, one, eqReg), true
	}
	return eqReg, true
}

// isScalarType reports whether the given TypeId is a primitive
// numeric, bool, or char — the shapes that support direct `==`
// without a trait dispatch.
func (l *lowerer) isScalarType(tid typetable.TypeId) bool {
	t := l.prog.Types.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindI8, typetable.KindI16, typetable.KindI32, typetable.KindI64,
		typetable.KindU8, typetable.KindU16, typetable.KindU32, typetable.KindU64,
		typetable.KindISize, typetable.KindUSize,
		typetable.KindF32, typetable.KindF64,
		typetable.KindBool, typetable.KindChar:
		return true
	}
	return false
}

// equalityMangledName returns the C-safe symbol used by codegen to
// resolve a `PartialEq::eq` impl for the given operand type. The
// canonical scheme is `<TypeName>__eq`; codegen must emit or
// externally link that symbol. If the type is anonymous, the
// fallback name makes the bug explicit at codegen rather than
// silently miscompiling (Rule 6.9).
func (l *lowerer) equalityMangledName(tid typetable.TypeId) string {
	if t := l.prog.Types.Get(tid); t != nil && t.Name != "" {
		return sanitizeCName(t.Name) + "__eq"
	}
	return "unknown_type__eq"
}

// lowerOptChain emits the conditional-read lowering for
// `receiver?.field` (reference §5.6, §5.8). The shape is:
//
//   - Compute the receiver into R.
//   - Compare R against the None/Err discriminant K of the
//     enclosing option/result type.
//   - On R == K, return R from the enclosing fn (propagate).
//   - Otherwise, emit OpFieldRead(R, field) and continue with
//     that value as the chain's result.
//
// §5.8 explicitly forbids "`?` applied to `expr` followed by
// field access on the result"; the lowering below is a single
// structural unit, not two composed pieces.
func (l *lowerer) lowerOptChain(modPath string, b *mir.Builder, x *hir.OptFieldExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	recv, ok := l.lowerExpr(modPath, b, x.Receiver, params)
	if !ok {
		return mir.NoReg, false
	}
	recvType := x.Receiver.TypeOf()
	errIdx, ok := l.errorVariantIndex(recvType)
	if !ok {
		l.diagnose(x.NodeSpan(),
			"`?.` receiver must resolve to an option- or result-shaped enum",
			"declare the absent / error case as `None` or `Err`")
		return mir.NoReg, false
	}
	// Allocate the continuation blocks.
	propagateBlock := b.BeginBlock()
	okBlock := b.BeginBlock()
	// Re-enter the block that computed recv so we can emit the branch.
	recvBlock := propagateBlock - 1
	b.UseBlock(recvBlock)
	b.IfEq(recv, int64(errIdx), propagateBlock, okBlock)
	// Propagate: return recv unchanged.
	b.UseBlock(propagateBlock)
	b.Return(recv)
	// Continuation: read the field.
	b.UseBlock(okBlock)
	if x.Name == "" {
		l.diagnose(x.NodeSpan(),
			"optional chain with empty field name",
			"this is a bridge bug, not a user error")
		return mir.NoReg, false
	}
	return b.FieldRead(recv, x.Name), true
}

// lowerCastTagged emits an OpCast with a computed CastMode.
// Replaces the W14 passthrough (which still lives in lowerExpr
// for e2e-compatibility until the tagged op fully flows through
// W17 codegen).
func (l *lowerer) lowerCastTagged(modPath string, b *mir.Builder, x *hir.CastExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	src, ok := l.lowerExpr(modPath, b, x.Expr, params)
	if !ok {
		return mir.NoReg, false
	}
	mode := l.classifyCast(x.Expr.TypeOf(), x.TypeOf())
	if mode == mir.CastInvalid {
		l.diagnose(x.NodeSpan(),
			"cast between source and target types is not supported",
			"numeric / pointer / reinterpret casts are the only kinds W15 classifies; wrap the value through a constructor instead")
		return mir.NoReg, false
	}
	return b.Cast(src, mode), true
}

// classifyCast walks the reference §28.1 classification ladder and
// returns the most specific CastMode for a given (source, target)
// TypeId pair. CastInvalid means the cast is not expressible as a
// direct MIR op — codegen would need a constructor call instead.
func (l *lowerer) classifyCast(src, dst typetable.TypeId) mir.CastMode {
	sT := l.prog.Types.Get(src)
	dT := l.prog.Types.Get(dst)
	if sT == nil || dT == nil {
		return mir.CastInvalid
	}
	// Same type → reinterpret (no-op at C-level, but structurally
	// a CastReinterpret so downstream passes see a classified op).
	if src == dst {
		return mir.CastReinterpret
	}
	sScalar := l.isScalarType(src)
	dScalar := l.isScalarType(dst)
	if sScalar && dScalar {
		sBits := scalarBits(sT.Kind)
		dBits := scalarBits(dT.Kind)
		sFloat := isFloatKind(sT.Kind)
		dFloat := isFloatKind(dT.Kind)
		switch {
		case sFloat && !dFloat:
			return mir.CastFloatToInt
		case !sFloat && dFloat:
			return mir.CastIntToFloat
		case sBits == dBits:
			return mir.CastReinterpret
		case sBits < dBits:
			return mir.CastWiden
		case sBits > dBits:
			return mir.CastNarrow
		}
	}
	// Pointer / integer cast forms are classified by kind, but the
	// W15 scope limits pointer handling to reinterpret until W16
	// brings the raw-pointer surface online.
	return mir.CastInvalid
}

// scalarBits returns the bit-width of a primitive numeric Kind.
func scalarBits(k typetable.Kind) int {
	switch k {
	case typetable.KindI8, typetable.KindU8:
		return 8
	case typetable.KindI16, typetable.KindU16:
		return 16
	case typetable.KindI32, typetable.KindU32, typetable.KindF32:
		return 32
	case typetable.KindI64, typetable.KindU64, typetable.KindF64:
		return 64
	case typetable.KindISize, typetable.KindUSize:
		return 64 // 64-bit assumption for the stage-1 target
	case typetable.KindBool:
		return 8
	case typetable.KindChar:
		return 32
	}
	return 0
}

// isFloatKind reports whether k is a floating-point primitive.
func isFloatKind(k typetable.Kind) bool {
	return k == typetable.KindF32 || k == typetable.KindF64
}

// lowerFnPointer emits an OpFnPtr when a PathExpr that resolves to
// a function symbol is used as a value (reference §29.1). This is
// only triggered when the callee-detection logic in lowerCallExpr
// determines the expression is not a direct call — e.g., the
// programmer wrote `let f: fn(I32) -> I32 = increment;`.
func (l *lowerer) lowerFnPointer(b *mir.Builder, modPath, fnName string) mir.Reg {
	return b.FnPtr(cName(modPath, fnName))
}

// lowerIndexRange emits an OpSliceNew from `arr[low..high]`
// (reference §32.1). Open endpoints lower to NoReg — codegen will
// substitute 0 / arr.len at emission.
func (l *lowerer) lowerIndexRange(modPath string, b *mir.Builder, x *hir.IndexRangeExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	if x.Receiver == nil {
		l.diagnose(x.NodeSpan(),
			"slice range without receiver",
			"this is a bridge bug, not a user error")
		return mir.NoReg, false
	}
	base, ok := l.lowerExpr(modPath, b, x.Receiver, params)
	if !ok {
		return mir.NoReg, false
	}
	var low, high mir.Reg
	if x.Low != nil {
		lr, ok := l.lowerExpr(modPath, b, x.Low, params)
		if !ok {
			return mir.NoReg, false
		}
		low = lr
	}
	if x.High != nil {
		hr, ok := l.lowerExpr(modPath, b, x.High, params)
		if !ok {
			return mir.NoReg, false
		}
		high = hr
	}
	return b.SliceNew(base, low, high, x.Inclusive), true
}

// lowerStructLit handles `Path { f1: v1, ..base }` — the reference
// §45.1 form with explicit-field precedence over base moves.
//
// When Base is nil: OpStructNew + OpFieldWrite per explicit field.
// When Base is set: OpStructCopy, then OpFieldWrite for each
// explicit field (explicit overrides base; unassigned fields
// already hold base's values after the copy).
func (l *lowerer) lowerStructLit(modPath string, b *mir.Builder, x *hir.StructLitExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	t := l.prog.Types.Get(x.StructType)
	if t == nil {
		l.diagnose(x.NodeSpan(),
			"struct literal has no resolved struct type",
			"this is a checker bug, not a user error")
		return mir.NoReg, false
	}
	typeName := sanitizeCName(t.Name)
	// Lower every explicit field first so later passes see a stable
	// evaluation order (reference §45.1 implies left-to-right).
	fieldVals := make([]mir.StructField, 0, len(x.Fields))
	for _, fld := range x.Fields {
		val, ok := l.lowerExpr(modPath, b, fld.Value, params)
		if !ok {
			return mir.NoReg, false
		}
		fieldVals = append(fieldVals, mir.StructField{Name: fld.Name, Value: val})
	}
	var dst mir.Reg
	if x.Base == nil {
		dst = b.StructNew(typeName, fieldVals)
	} else {
		baseReg, ok := l.lowerExpr(modPath, b, x.Base, params)
		if !ok {
			return mir.NoReg, false
		}
		dst = b.StructCopy(baseReg)
	}
	// Emit explicit field writes after the allocation / copy so
	// reference §45.1 "explicit-field precedence" is structurally
	// guaranteed — the last write wins at the C assignment level.
	for _, fv := range fieldVals {
		b.FieldWrite(dst, fv.Name, fv.Value)
	}
	return dst, true
}

// lowerOverflowMethod recognizes `recv.wrapping_add(arg)` /
// `checked_*` / `saturating_*` method calls and emits the dedicated
// MIR op. Returns (NoReg, false, false) when the method is not a
// recognized overflow form — the caller then falls back to the
// generic method-call lowering.
//
// Reference §33.1: each explicit-overflow method lowers to a
// distinct MIR op so codegen emits policy-appropriate instruction
// sequences (plain wrap, `__builtin_*_overflow`, saturating
// clamp).
func (l *lowerer) lowerOverflowMethod(modPath string, b *mir.Builder, call *hir.CallExpr, field *hir.FieldExpr, params map[string]mir.Reg) (mir.Reg, bool, bool) {
	op, ok := classifyOverflowMethod(field.Name)
	if !ok {
		return mir.NoReg, false, false
	}
	if len(call.Args) != 1 {
		l.diagnose(call.NodeSpan(),
			fmt.Sprintf("overflow-policy method %q takes exactly one argument", field.Name),
			"pass a single operand of the same integer type")
		return mir.NoReg, false, true
	}
	lhs, okLhs := l.lowerExpr(modPath, b, field.Receiver, params)
	if !okLhs {
		return mir.NoReg, false, true
	}
	rhs, okRhs := l.lowerExpr(modPath, b, call.Args[0], params)
	if !okRhs {
		return mir.NoReg, false, true
	}
	return b.OverflowArith(op, lhs, rhs), true, true
}

// classifyOverflowMethod maps a Fuse overflow-policy method name to
// its MIR op. Returns ok=false for names that are not overflow
// methods.
func classifyOverflowMethod(name string) (mir.Op, bool) {
	switch name {
	case "wrapping_add":
		return mir.OpWrappingAdd, true
	case "wrapping_sub":
		return mir.OpWrappingSub, true
	case "wrapping_mul":
		return mir.OpWrappingMul, true
	case "checked_add":
		return mir.OpCheckedAdd, true
	case "checked_sub":
		return mir.OpCheckedSub, true
	case "checked_mul":
		return mir.OpCheckedMul, true
	case "saturating_add":
		return mir.OpSaturatingAdd, true
	case "saturating_sub":
		return mir.OpSaturatingSub, true
	case "saturating_mul":
		return mir.OpSaturatingMul, true
	}
	return mir.OpInvalid, false
}
