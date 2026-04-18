package mir

// W15 MIR consolidation ops.
//
// These opcodes extend the W05/W09 instruction set with the lowering
// forms that Wave 15 consolidates: casts, references, function
// pointers, slice ranges, field/struct operations, method dispatch,
// semantic equality, and overflow-policy arithmetic.
//
// Each op is accepted by Function.Validate (see w15_validate.go) and
// emitted by a dedicated Builder method. Codegen (W17) is the
// scheduled retirer of these ops — until then, if an op produced by
// the lowerer reaches codegen, emitInst's default case reports
// "unsupported MIR op" which is exactly the expected W17-stub
// diagnostic behavior (Rule 6.9).
//
// The op values start immediately after OpDrop. Adding further ops
// must happen at the end of this block; inserting in the middle
// would shift every later constant's numeric value and break any
// serialized MIR in flight.

const (
	// OpCast is `Dst = Lhs as T`. Mode carries the cast flavor
	// (CastWiden / CastNarrow / CastReinterpret / etc.) per
	// reference §28.1. The lowerer classifies; codegen picks
	// the right C11 narrowing/widening form.
	OpCast Op = iota + 1000 // leave a numeric gap so future W05-W09 ops can be appended without renumbering W15

	// OpBorrow is `Dst = &Lhs` (Flag=false) or `Dst = &mut Lhs`
	// (Flag=true). Produces a register holding a reference to
	// the target; ownership analysis already validated the
	// shape at W09.
	OpBorrow

	// OpFnPtr loads the address of a named fn into Dst. CallName
	// is the mangled target. Reference §29.1: function pointer
	// values are not closures and carry no captured state.
	OpFnPtr

	// OpCallIndirect calls a function through a function-pointer
	// register. Lhs holds the fn pointer, CallArgs the arguments,
	// Dst receives the return value. Distinct from OpCall so
	// codegen can emit a C11 indirect-call dispatch.
	OpCallIndirect

	// OpSliceNew constructs a slice descriptor `{ ptr: Lhs + Rhs,
	// len: Extra - Rhs }` borrowing from the base. Flag is true
	// for inclusive ranges (`..=`) — lowering adds 1 to the
	// length at codegen. Reference §32.1.
	OpSliceNew

	// OpFieldRead reads `Lhs.FieldName` into Dst. The lowerer
	// resolves field offsets; codegen renders `r_Dst = r_Lhs.f;`.
	// Reference §9.4: `obj.name` NOT in callee position is a
	// field read; in callee position it is a method call.
	OpFieldRead

	// OpStructNew allocates and zero-initialises a fresh struct
	// value of the type named by CallName. Subsequent OpFieldWrite
	// instructions populate the fields. Decomposes struct
	// literals into atomic ops.
	OpStructNew

	// OpStructCopy copies Lhs (a base struct value) into Dst.
	// Used by struct-update `Path { ..base }` lowering: copy
	// base, then overwrite the explicit fields with OpFieldWrite.
	// Reference §45.1: explicit-field precedence over base.
	OpStructCopy

	// OpFieldWrite stores Rhs into `Lhs.FieldName`. No Dst. Used
	// for struct-lit field population and struct-update overrides.
	OpFieldWrite

	// OpMethodCall is a method-dispatched call. CallName is the
	// mangled method symbol; CallArgs[0] is the receiver (`self`).
	// Tagged distinctly from OpCall so codegen can use a
	// method-specific lookup (important for trait objects when
	// combined with W13 vtable dispatch).
	OpMethodCall

	// OpEqScalar computes Dst = (Lhs == Rhs) using primitive
	// equality. Emitted for int/bool/char/primitive comparisons.
	// Reference §5.8: scalar equality is type-specific.
	OpEqScalar

	// OpEqCall dispatches `Dst = <CallName>(Lhs, Rhs)` where
	// CallName is the mangled `PartialEq::eq` impl. Emitted for
	// equality on non-scalar nominals.
	OpEqCall

	// OpWrappingAdd/Sub/Mul compute arithmetic with two's
	// complement wrap-around regardless of build profile.
	// Reference §33.1.
	OpWrappingAdd
	OpWrappingSub
	OpWrappingMul

	// OpCheckedAdd/Sub/Mul produce a (result, overflow) pair —
	// lowered to distinct MIR ops so codegen can emit the
	// platform intrinsic (e.g. `__builtin_add_overflow`) and
	// branch on overflow. Reference §33.1.
	OpCheckedAdd
	OpCheckedSub
	OpCheckedMul

	// OpSaturatingAdd/Sub/Mul clamp the result at the integer
	// type's min/max boundary instead of wrapping. Reference §33.1.
	OpSaturatingAdd
	OpSaturatingSub
	OpSaturatingMul
)

// CastMode classifies an OpCast instruction per reference §28.1.
type CastMode int

const (
	CastInvalid CastMode = iota
	// CastWiden: numeric type to a wider numeric type. Signed
	// sources sign-extend; unsigned sources zero-extend.
	CastWiden
	// CastNarrow: numeric type to a narrower numeric type. The
	// bit pattern is truncated; codegen emits the explicit C11
	// cast so narrowing is observable, not UB.
	CastNarrow
	// CastReinterpret: pointer-to-pointer or same-width numeric
	// reinterpretation. Does not participate in ownership
	// analysis.
	CastReinterpret
	// CastFloatToInt: float-to-int truncation toward zero. NaN /
	// out-of-range sources produce a platform-defined but
	// deterministic result recorded in CI goldens.
	CastFloatToInt
	// CastIntToFloat: int-to-float conversion; precision loss is
	// possible for large magnitudes.
	CastIntToFloat
	// CastPtrToInt: pointer-to-integer (USize/ISize representation).
	// Legal in safe code per reference §28.1.
	CastPtrToInt
	// CastIntToPtr: integer-to-pointer; legal only in `unsafe`
	// contexts. The checker has already confirmed the unsafe
	// boundary; the lowerer tags the op for codegen.
	CastIntToPtr
)

// String renders a cast mode for diagnostics and test assertions.
func (m CastMode) String() string {
	switch m {
	case CastWiden:
		return "widen"
	case CastNarrow:
		return "narrow"
	case CastReinterpret:
		return "reinterpret"
	case CastFloatToInt:
		return "float_to_int"
	case CastIntToFloat:
		return "int_to_float"
	case CastPtrToInt:
		return "ptr_to_int"
	case CastIntToPtr:
		return "int_to_ptr"
	}
	return "invalid"
}

// StructField pairs a field name with the register holding that
// field's value during struct-lit construction. Carried on an Inst
// via the Fields slice for OpStructNew / OpStructCopy bookkeeping —
// the actual field stores are emitted as OpFieldWrite instructions.
type StructField struct {
	Name  string
	Value Reg
}
