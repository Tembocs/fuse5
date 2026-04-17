package consteval

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Tembocs/fuse5/compiler/typetable"
)

// ValueKind enumerates the shapes the evaluator can produce. The set
// is closed: every operation yields one of these kinds or fails with
// a diagnostic. No silent fallback to an "unknown" value kind exists
// (Rule 3.9, L013 defense).
type ValueKind int

const (
	// VKInvalid is the zero value. A Value of this kind is never
	// returned to callers; it exists only so the zero `Value{}`
	// is unambiguously uninitialised.
	VKInvalid ValueKind = iota
	VKInt
	VKBool
	VKUnit
	VKTuple
	VKArray
	VKStruct
	VKChar
)

// Value is one evaluated compile-time constant. Field usage depends
// on Kind:
//
//   - VKInt:    Int holds the value; Type identifies the integer type.
//   - VKBool:   Bool holds the value; Type is Bool.
//   - VKUnit:   all zero.
//   - VKTuple:  Elems holds the element values in source order.
//   - VKArray:  Elems holds the element values in index order; Type
//     is the KindArray TypeId.
//   - VKStruct: Fields holds field values keyed by name; FieldOrder
//     records the source-declaration order for
//     deterministic iteration (Rule 7.1).
//   - VKChar:   Int holds the code point; Type is Char.
//
// Int is stored as uint64 regardless of signedness so the evaluator
// does not lose bits on 64-bit operands. Signedness for arithmetic
// is recovered from Type (the TypeTable Kind for integer TypeIds).
type Value struct {
	Kind       ValueKind
	Type       typetable.TypeId
	Int        uint64
	Bool       bool
	Elems      []Value
	Fields     map[string]Value
	FieldOrder []string
}

// IntValue constructs a VKInt Value.
func IntValue(t typetable.TypeId, v uint64) Value {
	return Value{Kind: VKInt, Type: t, Int: v}
}

// BoolValue constructs a VKBool Value.
func BoolValue(t typetable.TypeId, b bool) Value {
	return Value{Kind: VKBool, Type: t, Bool: b}
}

// UnitValue constructs a VKUnit Value.
func UnitValue(t typetable.TypeId) Value { return Value{Kind: VKUnit, Type: t} }

// CharValue constructs a VKChar Value.
func CharValue(t typetable.TypeId, cp uint64) Value {
	return Value{Kind: VKChar, Type: t, Int: cp}
}

// TupleValue constructs a VKTuple Value.
func TupleValue(t typetable.TypeId, elems []Value) Value {
	return Value{Kind: VKTuple, Type: t, Elems: elems}
}

// ArrayValue constructs a VKArray Value.
func ArrayValue(t typetable.TypeId, elems []Value) Value {
	return Value{Kind: VKArray, Type: t, Elems: elems}
}

// StructValue constructs a VKStruct Value. order records field
// declaration order; the map keys must match order exactly.
func StructValue(t typetable.TypeId, order []string, fields map[string]Value) Value {
	return Value{Kind: VKStruct, Type: t, Fields: fields, FieldOrder: order}
}

// IsTruthy reports whether a boolean Value is true. Callers that
// want to branch on integer values must first cast to Bool via a
// comparison operator.
func (v Value) IsTruthy() bool {
	return v.Kind == VKBool && v.Bool
}

// String is the canonical human-readable form used in diagnostics
// and goldens. The format is stable across runs and platforms
// (Rule 7.1).
func (v Value) String() string {
	switch v.Kind {
	case VKInt:
		return strconv.FormatUint(v.Int, 10)
	case VKBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case VKUnit:
		return "()"
	case VKChar:
		return fmt.Sprintf("char(%d)", v.Int)
	case VKTuple:
		var sb strings.Builder
		sb.WriteByte('(')
		for i, e := range v.Elems {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(e.String())
		}
		sb.WriteByte(')')
		return sb.String()
	case VKArray:
		var sb strings.Builder
		sb.WriteByte('[')
		for i, e := range v.Elems {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(e.String())
		}
		sb.WriteByte(']')
		return sb.String()
	case VKStruct:
		var sb strings.Builder
		sb.WriteByte('{')
		for i, name := range v.FieldOrder {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(name)
			sb.WriteString(": ")
			sb.WriteString(v.Fields[name].String())
		}
		sb.WriteByte('}')
		return sb.String()
	}
	return "<invalid>"
}

// SignedInt reinterprets v.Int as a two's-complement signed value of
// the width implied by the type kind. v.Kind must be VKInt.
func (v Value) SignedInt(tab *typetable.Table) int64 {
	bits := integerBits(tab, v.Type)
	if bits >= 64 {
		return int64(v.Int)
	}
	mask := uint64(1) << bits
	sign := uint64(1) << (bits - 1)
	x := v.Int & (mask - 1)
	if x&sign != 0 {
		// sign-extend
		return int64(x) - int64(mask)
	}
	return int64(x)
}

// typeKind looks up the Kind for id, returning KindInvalid when the
// TypeId is unresolvable. This guards against nil-deref panics in
// malformed input while preserving the strict-by-default contract:
// callers check KindInvalid and emit a diagnostic rather than assume.
func typeKind(tab *typetable.Table, id typetable.TypeId) typetable.Kind {
	t := tab.Get(id)
	if t == nil {
		return typetable.KindInvalid
	}
	return t.Kind
}

// integerBits returns the bit width of the integer TypeId, or 64 for
// unknown widths (safe default for overflow math).
func integerBits(tab *typetable.Table, id typetable.TypeId) int {
	switch typeKind(tab, id) {
	case typetable.KindI8, typetable.KindU8:
		return 8
	case typetable.KindI16, typetable.KindU16:
		return 16
	case typetable.KindI32, typetable.KindU32:
		return 32
	case typetable.KindI64, typetable.KindU64:
		return 64
	case typetable.KindISize, typetable.KindUSize:
		return 64
	}
	return 64
}

// isSigned reports whether id is a signed integer kind.
func isSigned(tab *typetable.Table, id typetable.TypeId) bool {
	switch typeKind(tab, id) {
	case typetable.KindI8, typetable.KindI16, typetable.KindI32,
		typetable.KindI64, typetable.KindISize:
		return true
	}
	return false
}

// isIntegerType reports whether id is any integer kind.
func isIntegerType(tab *typetable.Table, id typetable.TypeId) bool {
	switch typeKind(tab, id) {
	case typetable.KindI8, typetable.KindI16, typetable.KindI32, typetable.KindI64,
		typetable.KindISize, typetable.KindU8, typetable.KindU16, typetable.KindU32,
		typetable.KindU64, typetable.KindUSize:
		return true
	}
	return false
}

// maskToWidth clamps v to the low `bits` of a uint64 — used after
// every arithmetic step to keep values canonical for the integer
// type's width.
func maskToWidth(v uint64, bits int) uint64 {
	if bits >= 64 {
		return v
	}
	return v & ((uint64(1) << bits) - 1)
}
