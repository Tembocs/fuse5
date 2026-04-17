package typetable

import (
	"fmt"
	"sort"
)

// Table is the interner. One Table instance backs a whole compilation
// unit; TypeIds are unique within that Table and have no meaning
// across tables.
//
// Zero value is not usable; construct via New().
type Table struct {
	types []Type          // indexed by TypeId
	byKey map[string]TypeId
	// Primitive TypeIds are pre-computed at construction so Bool,
	// I32, etc. are always the same numeric value within a Table.
	// Their values are *not* stable across Tables — callers must
	// consult Lookup if they want to compare by kind.
	primitive map[Kind]TypeId
}

// New returns a fresh Table with every primitive kind pre-interned.
// The reserved zero slot (NoType) occupies TypeId 0 so that a zero
// TypeId unambiguously means "not yet assigned" (parallel to
// resolve.NoSymbol).
func New() *Table {
	t := &Table{
		types:     []Type{{}}, // reserved slot for NoType
		byKey:     map[string]TypeId{},
		primitive: map[Kind]TypeId{},
	}
	// Intern every primitive up front in a deterministic order so
	// every Table of this version yields the same layout (Rule 7.1).
	for _, k := range primitiveOrder {
		id := t.intern(Type{Kind: k})
		t.primitive[k] = id
	}
	// Intern the explicit Infer TypeId so callers never need to
	// allocate it on the fly.
	t.primitive[KindInfer] = t.intern(Type{Kind: KindInfer})
	return t
}

// primitiveOrder is the canonical intern order. It drives both the
// determinism of the table layout and the well-known helper methods
// below. Changing this list is a language change.
var primitiveOrder = []Kind{
	KindBool,
	KindI8, KindI16, KindI32, KindI64, KindISize,
	KindU8, KindU16, KindU32, KindU64, KindUSize,
	KindF32, KindF64,
	KindChar, KindString, KindCStr,
	KindUnit, KindNever,
}

// Intern returns the TypeId for t, constructing a new one if no
// equivalent type has been seen. Equivalent means: same key() string.
//
// Intern validates the Type minimally — mostly that required fields
// are set for the kind (e.g. Fn has a Return, Array has a Length).
// Malformed input panics: a malformed Intern is a compiler bug, not a
// user error, so failing loud is correct per Rule 3.6 (backend
// representation is architecture).
func (t *Table) Intern(typ Type) TypeId {
	validate(typ)
	return t.intern(typ)
}

// intern is the unchecked interning core. Used internally for
// primitive pre-interning and by Intern after validation.
func (t *Table) intern(typ Type) TypeId {
	k := typ.key()
	if id, ok := t.byKey[k]; ok {
		return id
	}
	// Defensive copy of slices so callers can recycle their buffer.
	typ.Children = typ.CloneChildren()
	typ.TypeArgs = typ.CloneTypeArgs()
	id := TypeId(len(t.types))
	t.types = append(t.types, typ)
	t.byKey[k] = id
	return id
}

// Get returns the interned Type behind id. Callers must not mutate
// the returned Type; it is a pointer into the Table's backing array
// for efficiency but is logically immutable.
func (t *Table) Get(id TypeId) *Type {
	if id <= 0 || int(id) >= len(t.types) {
		return nil
	}
	return &t.types[id]
}

// Lookup returns the TypeId for the primitive kind k, or NoType if k
// is not a primitive (including KindInfer).
func (t *Table) Lookup(k Kind) TypeId {
	id, ok := t.primitive[k]
	if !ok {
		return NoType
	}
	return id
}

// Len returns the number of interned types (excluding the reserved
// zero slot). Used in tests that assert interning dedupes.
func (t *Table) Len() int { return len(t.types) - 1 }

// --- Convenience constructors -------------------------------------------

// Bool / I32 / F64 / ... methods return pre-interned primitives by
// name, so call sites don't thread the Kind enum through.
func (t *Table) Bool() TypeId   { return t.primitive[KindBool] }
func (t *Table) I8() TypeId     { return t.primitive[KindI8] }
func (t *Table) I16() TypeId    { return t.primitive[KindI16] }
func (t *Table) I32() TypeId    { return t.primitive[KindI32] }
func (t *Table) I64() TypeId    { return t.primitive[KindI64] }
func (t *Table) ISize() TypeId  { return t.primitive[KindISize] }
func (t *Table) U8() TypeId     { return t.primitive[KindU8] }
func (t *Table) U16() TypeId    { return t.primitive[KindU16] }
func (t *Table) U32() TypeId    { return t.primitive[KindU32] }
func (t *Table) U64() TypeId    { return t.primitive[KindU64] }
func (t *Table) USize() TypeId  { return t.primitive[KindUSize] }
func (t *Table) F32() TypeId    { return t.primitive[KindF32] }
func (t *Table) F64() TypeId    { return t.primitive[KindF64] }
func (t *Table) Char() TypeId   { return t.primitive[KindChar] }
func (t *Table) String_() TypeId { return t.primitive[KindString] }
func (t *Table) CStr() TypeId   { return t.primitive[KindCStr] }
func (t *Table) Unit() TypeId   { return t.primitive[KindUnit] }
func (t *Table) Never() TypeId  { return t.primitive[KindNever] }

// Infer returns the explicit "to be resolved by type checker" TypeId.
// Passes that produce this TypeId must also emit a pending-inference
// marker into the HIR so W06 can finish the job (L013).
func (t *Table) Infer() TypeId { return t.primitive[KindInfer] }

// Tuple, Array, Slice, Ptr, Ref, Mutref, Fn, TraitObject, Channel,
// ThreadHandle — each is a thin wrapper around Intern with the
// appropriate Kind set.

// Tuple returns the TypeId for a tuple of the given element TypeIds.
func (t *Table) Tuple(elems []TypeId) TypeId {
	return t.Intern(Type{Kind: KindTuple, Children: elems})
}

// Array returns the TypeId for `[T; N]`.
func (t *Table) Array(elem TypeId, length uint64) TypeId {
	return t.Intern(Type{Kind: KindArray, Children: []TypeId{elem}, Length: length})
}

// Slice returns the TypeId for `[T]`.
func (t *Table) Slice(elem TypeId) TypeId {
	return t.Intern(Type{Kind: KindSlice, Children: []TypeId{elem}})
}

// Ptr returns the TypeId for `Ptr[T]`.
func (t *Table) Ptr(pointee TypeId) TypeId {
	return t.Intern(Type{Kind: KindPtr, Children: []TypeId{pointee}})
}

// Ref returns the TypeId for a shared reference `&T`.
func (t *Table) Ref(target TypeId) TypeId {
	return t.Intern(Type{Kind: KindRef, Children: []TypeId{target}})
}

// Mutref returns the TypeId for a mutable reference `&mut T`.
func (t *Table) Mutref(target TypeId) TypeId {
	return t.Intern(Type{Kind: KindMutref, Children: []TypeId{target}})
}

// Fn returns the TypeId for `fn(params...) -> ret`.
func (t *Table) Fn(params []TypeId, ret TypeId, variadic bool) TypeId {
	return t.Intern(Type{
		Kind:       KindFn,
		Children:   params,
		Return:     ret,
		IsVariadic: variadic,
	})
}

// TraitObject returns the TypeId for `dyn Trait1 + Trait2 + ...`.
func (t *Table) TraitObject(traits []TypeId) TypeId {
	out := make([]TypeId, len(traits))
	copy(out, traits)
	// Sort bounds so `dyn A + B` and `dyn B + A` share a TypeId.
	// Bounds are semantically a set, not a list (reference §13).
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return t.Intern(Type{Kind: KindTraitObject, Children: out})
}

// Channel returns the TypeId for `Chan[T]` (reference §17.2).
//
// At W04 the kind exists in the table but does not yet participate in
// concurrency checking; W07 integrates it. The intern is available so
// that later waves do not need to retrofit a new TypeId layout.
func (t *Table) Channel(elem TypeId) TypeId {
	return t.Intern(Type{Kind: KindChannel, Children: []TypeId{elem}})
}

// ThreadHandle returns the TypeId for `ThreadHandle[T]`
// (reference §39). Same W04/W07 split as Channel.
func (t *Table) ThreadHandle(result TypeId) TypeId {
	return t.Intern(Type{Kind: KindThreadHandle, Children: []TypeId{result}})
}

// Nominal returns a TypeId for a nominal type (struct/enum/union/
// trait/type-alias). The (symbol, module, name) triple establishes
// §2.8 nominal identity: two calls with the same Symbol are the same
// type, even if Name or Module differ (they must not, but identity is
// Symbol-driven); two calls with different Symbols produce different
// TypeIds regardless of Name collision.
//
// TypeArgs specialises a generic nominal like `Vec[I32]` — the
// unspecialised form is a separate TypeId with len(TypeArgs) == 0.
func (t *Table) Nominal(kind Kind, symbol int, module, name string, typeArgs []TypeId) TypeId {
	if !kind.IsNominal() {
		panic(fmt.Sprintf("typetable.Nominal: kind %s is not nominal", kind))
	}
	return t.Intern(Type{
		Kind:     kind,
		Symbol:   symbol,
		Module:   module,
		Name:     name,
		TypeArgs: typeArgs,
	})
}

// GenericParam returns a TypeId for a generic parameter like `T`. The
// (symbol, name) pair uniquely identifies the parameter within the
// build.
func (t *Table) GenericParam(symbol int, module, name string) TypeId {
	return t.Intern(Type{
		Kind:   KindGenericParam,
		Symbol: symbol,
		Module: module,
		Name:   name,
	})
}

// --- Validation ---------------------------------------------------------

// validate enforces the minimal structural invariants for each Kind.
// A validate panic is a compiler bug; user input is already rejected
// by the parser, resolver, or type checker before reaching Intern.
func validate(t Type) {
	switch t.Kind {
	case KindInvalid:
		panic("typetable.Intern: KindInvalid (zero Type{}) is not internable")
	case KindArray:
		if len(t.Children) != 1 {
			panic(fmt.Sprintf("typetable.Intern: Array requires 1 element, got %d", len(t.Children)))
		}
	case KindSlice, KindPtr, KindRef, KindMutref, KindChannel, KindThreadHandle:
		if len(t.Children) != 1 {
			panic(fmt.Sprintf("typetable.Intern: %s requires 1 element, got %d", t.Kind, len(t.Children)))
		}
	case KindFn:
		// Fn may have zero parameters; Return must be set to a valid
		// TypeId (Unit for `fn() {}`). NoType Return is illegal.
		if t.Return == NoType {
			panic("typetable.Intern: Fn requires a Return TypeId (use Unit for no return)")
		}
	case KindStruct, KindEnum, KindUnion, KindTrait, KindTypeAlias:
		if t.Symbol == 0 {
			panic(fmt.Sprintf("typetable.Intern: nominal %s requires a defining Symbol (reference §2.8)", t.Kind))
		}
		if t.Name == "" {
			panic(fmt.Sprintf("typetable.Intern: nominal %s requires a Name", t.Kind))
		}
	case KindGenericParam:
		if t.Symbol == 0 || t.Name == "" {
			panic("typetable.Intern: GenericParam requires Symbol and Name")
		}
	}
}
