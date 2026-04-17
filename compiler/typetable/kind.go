package typetable

// Kind identifies the structural category of a type. The set is fixed —
// adding a Kind is a language change (reference §2). No `Unknown`
// member exists by design: the bridge is forbidden from synthesizing
// an unknown default (L013, L021). Genuinely un-inferred types carry
// `KindInfer`, which W06 (type checking) must resolve or reject.
type Kind int

const (
	// KindInvalid is the zero value; no interned Type should ever have
	// this kind. It exists only so that a zero Type{} is unambiguously
	// uninitialized.
	KindInvalid Kind = iota

	// Primitive kinds (reference §2.1–§2.7).
	KindBool
	KindI8
	KindI16
	KindI32
	KindI64
	KindISize
	KindU8
	KindU16
	KindU32
	KindU64
	KindUSize
	KindF32
	KindF64
	KindChar
	KindString
	KindCStr
	KindUnit
	KindNever

	// Structural kinds (reference §3).
	KindTuple
	KindArray
	KindSlice
	KindPtr
	KindRef
	KindMutref
	KindFn
	KindTraitObject // `dyn Trait`

	// Nominal kinds (reference §10–§12). Two nominal types are equal
	// iff they share the same declared name AND the same defining
	// symbol (§2.8).
	KindStruct
	KindEnum
	KindUnion
	KindTrait
	KindTypeAlias

	// Generic placeholders.
	KindGenericParam // e.g. `T` inside `fn id[T](x: T)`

	// Concurrency kinds (stubbed for W07 integration).
	KindChannel      // `Chan[T]` — reference §17.2
	KindThreadHandle // `ThreadHandle[T]` — reference §39

	// KindInfer marks a type the bridge could not determine on its
	// own. Unlike `Unknown`, which is a silent fallback, `KindInfer`
	// is an explicit promise: the type checker in W06 must resolve it
	// or reject the program. A KindInfer type that survives checking
	// is a compiler bug, not a user error.
	KindInfer
)

// String returns a stable, human-readable name for the kind. Used by
// diagnostics, golden tests, and fingerprinting. The strings are
// intentionally short and language-reference-aligned so that output
// reads uniformly across the compiler.
func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "invalid"
	case KindBool:
		return "Bool"
	case KindI8:
		return "I8"
	case KindI16:
		return "I16"
	case KindI32:
		return "I32"
	case KindI64:
		return "I64"
	case KindISize:
		return "ISize"
	case KindU8:
		return "U8"
	case KindU16:
		return "U16"
	case KindU32:
		return "U32"
	case KindU64:
		return "U64"
	case KindUSize:
		return "USize"
	case KindF32:
		return "F32"
	case KindF64:
		return "F64"
	case KindChar:
		return "Char"
	case KindString:
		return "String"
	case KindCStr:
		return "CStr"
	case KindUnit:
		return "Unit"
	case KindNever:
		return "Never"
	case KindTuple:
		return "Tuple"
	case KindArray:
		return "Array"
	case KindSlice:
		return "Slice"
	case KindPtr:
		return "Ptr"
	case KindRef:
		return "Ref"
	case KindMutref:
		return "Mutref"
	case KindFn:
		return "Fn"
	case KindTraitObject:
		return "TraitObject"
	case KindStruct:
		return "Struct"
	case KindEnum:
		return "Enum"
	case KindUnion:
		return "Union"
	case KindTrait:
		return "Trait"
	case KindTypeAlias:
		return "TypeAlias"
	case KindGenericParam:
		return "GenericParam"
	case KindChannel:
		return "Chan"
	case KindThreadHandle:
		return "ThreadHandle"
	case KindInfer:
		return "Infer"
	}
	return "unknown"
}

// IsPrimitive reports whether k is one of the reference §2 primitive
// kinds. Primitives are pre-interned with stable TypeIds at table
// construction.
func (k Kind) IsPrimitive() bool {
	switch k {
	case KindBool, KindI8, KindI16, KindI32, KindI64, KindISize,
		KindU8, KindU16, KindU32, KindU64, KindUSize,
		KindF32, KindF64, KindChar, KindString, KindCStr,
		KindUnit, KindNever:
		return true
	}
	return false
}

// IsNominal reports whether k's identity is by defining symbol
// (§2.8). Two nominal types with the same name from different
// modules are distinct even if structurally identical.
func (k Kind) IsNominal() bool {
	switch k {
	case KindStruct, KindEnum, KindUnion, KindTrait, KindTypeAlias:
		return true
	}
	return false
}
