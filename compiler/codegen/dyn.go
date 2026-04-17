package codegen

import (
	"fmt"
	"strings"

	"github.com/Tembocs/fuse5/compiler/lower"
)

// W13 vtable emission. Codegen takes a VtableLayout and
// produces a single C static table with the layout described
// in §57.8: the three-word header (size, align, drop_fn) and
// then one slot per method in trait-declaration order.
//
// The C form:
//
//   static const struct <VtableStructName> <VtableName> = {
//       .size    = sizeof(<ConcreteName>),
//       .align   = _Alignof(<ConcreteName>),
//       .drop_fn = <ConcreteName>_drop,
//       .method1 = <ConcreteName>_method1,
//       ...
//   };
//
// The struct type itself is emitted once per (trait, method
// set) pair. At W13 we inline everything into a single string
// builder so the test surface can inspect the emitted text
// directly; production codegen will hoist the struct types
// into a translation-unit header when multiple vtables share
// a method layout.

// EmitVtable renders a full C definition (struct + static
// table) for the given VtableLayout. The returned string is a
// well-formed C11 fragment.
func EmitVtable(l lower.VtableLayout) string {
	var sb strings.Builder
	sb.WriteString("/* vtable for ")
	sb.WriteString(l.ConcreteName)
	sb.WriteString(" as ")
	sb.WriteString(l.TraitName)
	sb.WriteString(" */\n")
	sb.WriteString("struct ")
	sb.WriteString(vtableStructName(l))
	sb.WriteString(" {\n")
	for _, e := range l.Entries {
		fmt.Fprintf(&sb, "    %s %s;\n", slotCType(e.Kind), e.Name)
	}
	sb.WriteString("};\n")
	sb.WriteString("static const struct ")
	sb.WriteString(vtableStructName(l))
	sb.WriteString(" ")
	sb.WriteString(l.VtableName())
	sb.WriteString(" = {\n")
	for _, e := range l.Entries {
		fmt.Fprintf(&sb, "    .%s = %s,\n", e.Name, slotInitExpr(l, e))
	}
	sb.WriteString("};\n")
	return sb.String()
}

// EmitFatPointerStruct emits the C definition of the fat
// pointer for a given dyn-trait shape. The struct name is
// `DynPtr_<TraitName>`; its fields are `data` (void*) and
// `vtable` (const struct <VtableStructName>*). Callers that
// build fat pointers use this type; codegen for method
// dispatch reads through it.
func EmitFatPointerStruct(traitName string) string {
	return fmt.Sprintf(
		"/* fat pointer for dyn %s */\n"+
			"struct DynPtr_%s {\n"+
			"    void *data;\n"+
			"    const void *vtable;\n"+
			"};\n",
		traitName, safeIdent(traitName),
	)
}

// EmitMethodDispatch renders a single method-dispatch call
// through a fat pointer: loads the method slot from the
// vtable and invokes it with the data pointer as receiver.
//
// The call shape (for a method of signature `fn(self, i32)
// -> i32`):
//
//   r<result> = ((int64_t (*)(void*, int64_t))
//                 ((const struct <VtableStructName>*)r<fat>.vtable)->method)(
//                     r<fat>.data, r<arg>);
//
// At W13 we return the text snippet for inspection. The real
// lowerer/codegen wiring (CallExpr → MIR → C emission) lands
// with W15 MIR consolidation and W17 codegen hardening; this
// helper is the W13 shape contract tests key off.
func EmitMethodDispatch(fatPointerReg string, method string, vtableStruct string, argRegs []string) string {
	var sb strings.Builder
	sb.WriteString("((const struct ")
	sb.WriteString(vtableStruct)
	sb.WriteString("*)")
	sb.WriteString(fatPointerReg)
	sb.WriteString(".vtable)->")
	sb.WriteString(method)
	sb.WriteString("(")
	sb.WriteString(fatPointerReg)
	sb.WriteString(".data")
	for _, a := range argRegs {
		sb.WriteString(", ")
		sb.WriteString(a)
	}
	sb.WriteString(")")
	return sb.String()
}

// vtableStructName returns the C struct-type name associated
// with a vtable layout. Different concrete impls of the same
// trait share the same struct layout; we key the struct name
// on the method set so combined vtables (`dyn A + B`) produce
// a distinct struct.
func vtableStructName(l lower.VtableLayout) string {
	return "VtableLayout_" + safeIdent(l.TraitName)
}

// slotCType returns the C type a vtable slot should be
// declared as. Size/align are `size_t` per C11; drop_fn and
// method pointers use an opaque `void(*)(void*)` to avoid
// needing the full signature at the struct definition site
// (callers cast to the correct signature at call-time).
func slotCType(k lower.VtableSlotKind) string {
	switch k {
	case lower.SlotSize, lower.SlotAlign:
		return "size_t"
	case lower.SlotDropFn, lower.SlotMethod:
		return "void (*)(void*)"
	}
	return "void *"
}

// slotInitExpr returns the initializer expression for a slot,
// as it appears in the `.slot = <init>` form of the static
// table.
func slotInitExpr(l lower.VtableLayout, e lower.VtableEntry) string {
	switch e.Kind {
	case lower.SlotSize:
		return "sizeof(" + safeIdent(l.ConcreteName) + ")"
	case lower.SlotAlign:
		return "_Alignof(" + safeIdent(l.ConcreteName) + ")"
	case lower.SlotDropFn:
		return safeIdent(l.ConcreteName) + "_drop"
	case lower.SlotMethod:
		return "(void (*)(void*))" + safeIdent(l.ConcreteName) + "_" + safeIdent(e.Name)
	}
	return "0"
}

// safeIdent mirrors the lower package helper so codegen can
// run the same deterministic name mangling without importing
// private functions.
func safeIdent(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_' {
			b = append(b, c)
		} else {
			b = append(b, '_')
		}
	}
	return string(b)
}
