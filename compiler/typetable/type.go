package typetable

import (
	"fmt"
	"strings"
)

// TypeId is an opaque handle into the Table. TypeId equality is the
// canonical type equality relation in Fuse after interning
// (Rule 7.2 — equality is integer comparison).
type TypeId int

// NoType is the zero TypeId. A node carrying NoType has not yet been
// assigned a type; it must be either filled with a concrete TypeId or
// explicitly set to the `Infer` TypeId before HIR leaves the bridge.
// Passes that observe NoType mid-pipeline must emit a diagnostic
// rather than default to Unknown (L013).
const NoType TypeId = 0

// Type is the interned description behind a TypeId. Callers never
// construct a Type directly at use sites; they call Table.Intern with
// a key/description and receive a TypeId.
//
// Field usage by Kind:
//
//   - Primitive kinds: all fields zero.
//   - Tuple: Children holds element TypeIds in order.
//   - Array: Children[0] is element; Length is the literal length.
//   - Slice / Ptr / Ref / Mutref: Children[0] is element/pointee.
//   - Fn: Children[0..N-1] are parameter types; Return is the return
//     type; IsVariadic marks a trailing `...`.
//   - TraitObject: Children holds the bound trait TypeIds in source
//     order.
//   - Nominal (Struct/Enum/Union/Trait/TypeAlias): Symbol names the
//     defining symbol (reference §2.8), Module records the declaring
//     module path for diagnostics, Name is the declared identifier.
//     TypeArgs holds the generic substitution for specialized
//     instances; an unspecialized nominal type has len(TypeArgs) == 0.
//   - GenericParam: Name holds the parameter spelling (`T`), Symbol
//     names the generic parameter's defining symbol (the FnDecl or
//     StructDecl that introduced it), Module records the declaring
//     module.
//   - Channel / ThreadHandle: Children[0] is the element/return type.
//   - Infer: all fields zero; the type-checker in W06 is responsible
//     for resolving this TypeId.
type Type struct {
	Kind       Kind
	Children   []TypeId // element, parameter, or bound children
	Return     TypeId   // Fn only
	Length     uint64   // Array only
	IsVariadic bool     // Fn only

	// Symbol, Module, and Name together establish nominal identity
	// (§2.8) for KindStruct / KindEnum / KindUnion / KindTrait /
	// KindTypeAlias and anchor KindGenericParam.
	Symbol int    // resolve.SymbolID value (kept as int to avoid import cycle)
	Module string // declaring module path, for diagnostics
	Name   string // declared identifier

	// TypeArgs holds generic arguments for a specialized nominal type
	// such as `Vec[I32]`. Unspecialized forms have len == 0. The
	// W04 interner keys nominal types by (Symbol, TypeArgs); an
	// unspecialized generic and its specializations are therefore
	// distinct TypeIds.
	TypeArgs []TypeId
}

// key returns a canonical, deterministic key that the Table uses to
// intern a Type. Equal keys ⇔ equal types. Keys are strings so that
// the map-based interner is simple and test output is readable.
//
// Determinism (Rule 7.1) is the driving requirement: two runs of the
// compiler over the same source must produce byte-identical keys for
// equivalent types. The format is therefore fully spelled out —
// numeric fields, Kind name, and child TypeIds in fixed order — with
// no reliance on Go's map iteration order or allocator behavior.
func (t Type) key() string {
	var sb strings.Builder
	sb.WriteString(t.Kind.String())
	if t.Kind.IsPrimitive() || t.Kind == KindInfer {
		return sb.String()
	}
	sb.WriteByte('{')
	switch t.Kind {
	case KindArray:
		fmt.Fprintf(&sb, "len=%d;elem=%d", t.Length, childAt(t.Children, 0))
	case KindFn:
		sb.WriteString("params=[")
		for i, c := range t.Children {
			if i > 0 {
				sb.WriteByte(',')
			}
			fmt.Fprintf(&sb, "%d", c)
		}
		fmt.Fprintf(&sb, "];ret=%d;var=%t", t.Return, t.IsVariadic)
	case KindTuple, KindTraitObject:
		sb.WriteString("children=[")
		for i, c := range t.Children {
			if i > 0 {
				sb.WriteByte(',')
			}
			fmt.Fprintf(&sb, "%d", c)
		}
		sb.WriteByte(']')
	case KindSlice, KindPtr, KindRef, KindMutref, KindChannel, KindThreadHandle:
		fmt.Fprintf(&sb, "elem=%d", childAt(t.Children, 0))
	case KindStruct, KindEnum, KindUnion, KindTrait, KindTypeAlias:
		// Nominal identity: defining symbol + declaring module + generic args.
		fmt.Fprintf(&sb, "sym=%d;mod=%q;name=%q", t.Symbol, t.Module, t.Name)
		if len(t.TypeArgs) > 0 {
			sb.WriteString(";args=[")
			for i, a := range t.TypeArgs {
				if i > 0 {
					sb.WriteByte(',')
				}
				fmt.Fprintf(&sb, "%d", a)
			}
			sb.WriteByte(']')
		}
	case KindGenericParam:
		fmt.Fprintf(&sb, "sym=%d;mod=%q;name=%q", t.Symbol, t.Module, t.Name)
	}
	sb.WriteByte('}')
	return sb.String()
}

// childAt returns the i-th child TypeId or NoType when i is out of
// range. The helper lets key() stay tolerant of slightly malformed
// Types without panicking during hashing; the malformation itself
// surfaces through Intern's validator in Table.Intern.
func childAt(cs []TypeId, i int) TypeId {
	if i < 0 || i >= len(cs) {
		return NoType
	}
	return cs[i]
}

// CloneChildren returns a defensive copy of the Children slice so
// callers inspecting an interned Type cannot mutate the table's
// internal state. Used primarily by tests.
func (t Type) CloneChildren() []TypeId {
	out := make([]TypeId, len(t.Children))
	copy(out, t.Children)
	return out
}

// CloneTypeArgs returns a defensive copy of TypeArgs.
func (t Type) CloneTypeArgs() []TypeId {
	out := make([]TypeId, len(t.TypeArgs))
	copy(out, t.TypeArgs)
	return out
}
