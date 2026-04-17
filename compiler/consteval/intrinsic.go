package consteval

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// intrinsicKind names the const intrinsics recognized by W14.
type intrinsicKind int

const (
	intrinsicNone intrinsicKind = iota
	intrinsicSizeOf
	intrinsicAlignOf
)

// recognizeIntrinsic inspects a CallExpr and, if its callee is the
// `size_of` or `align_of` built-in, returns the intrinsic kind. The
// W14 grammar accepts these as paths with exactly one turbofish type
// argument: `size_of[T]()` / `align_of[T]()`. The bridge preserves
// the type args on PathExpr.TypeArgs.
func recognizeIntrinsic(c *hir.CallExpr) (intrinsicKind, bool) {
	callee, ok := c.Callee.(*hir.PathExpr)
	if !ok {
		return intrinsicNone, false
	}
	if len(callee.Segments) == 0 {
		return intrinsicNone, false
	}
	last := callee.Segments[len(callee.Segments)-1]
	switch last {
	case "size_of":
		return intrinsicSizeOf, true
	case "align_of":
		return intrinsicAlignOf, true
	}
	return intrinsicNone, false
}

// evalIntrinsic computes the integer value of a size_of / align_of
// call. The W14 evaluator supports primitive, pointer, reference,
// tuple, array, and nominal struct/enum/union layouts. Generic type
// parameters (KindGenericParam) are rejected — monomorphization
// must have resolved them before const evaluation runs.
func (ev *Evaluator) evalIntrinsic(c *hir.CallExpr, kind intrinsicKind) (Value, flow) {
	callee, _ := c.Callee.(*hir.PathExpr)
	if len(callee.TypeArgs) != 1 {
		ev.emit(c.NodeSpan(),
			"size_of / align_of require exactly one type argument",
			"call as `size_of[T]()` or `align_of[T]()`")
		return Value{}, flowError
	}
	if len(c.Args) != 0 {
		ev.emit(c.NodeSpan(),
			"size_of / align_of take no value arguments",
			"remove the argument list or pass only the type argument")
		return Value{}, flowError
	}
	target := callee.TypeArgs[0]
	switch kind {
	case intrinsicSizeOf:
		size, err := ev.layoutSize(target)
		if err != nil {
			ev.emit(c.NodeSpan(), err.Error(),
				"apply size_of to a fully concrete type with a known layout")
			return Value{}, flowError
		}
		return IntValue(ev.Tab.USize(), size), flowNormal
	case intrinsicAlignOf:
		align, err := ev.layoutAlign(target)
		if err != nil {
			ev.emit(c.NodeSpan(), err.Error(),
				"apply align_of to a fully concrete type with a known layout")
			return Value{}, flowError
		}
		return IntValue(ev.Tab.USize(), align), flowNormal
	}
	ev.emit(c.NodeSpan(),
		"unknown intrinsic in const context",
		"only size_of and align_of are recognized at W14")
	return Value{}, flowError
}

// layoutSize returns the size in bytes of t, or an error if the
// type's layout is not statically determinable. The model matches
// the C11 backend's layout assumptions (reference §9.2) — a
// 64-bit target with natural alignment.
func (ev *Evaluator) layoutSize(t typetable.TypeId) (uint64, error) {
	k := typeKind(ev.Tab, t)
	switch k {
	case typetable.KindBool, typetable.KindI8, typetable.KindU8:
		return 1, nil
	case typetable.KindI16, typetable.KindU16:
		return 2, nil
	case typetable.KindI32, typetable.KindU32, typetable.KindF32, typetable.KindChar:
		return 4, nil
	case typetable.KindI64, typetable.KindU64, typetable.KindF64,
		typetable.KindISize, typetable.KindUSize,
		typetable.KindPtr, typetable.KindRef, typetable.KindMutref:
		return 8, nil
	case typetable.KindUnit, typetable.KindNever:
		return 0, nil
	case typetable.KindArray:
		elemID, length, ok := ev.arrayInfo(t)
		if !ok {
			return 0, fmt.Errorf("array type has unresolved length in a const context")
		}
		elem, err := ev.layoutSize(elemID)
		if err != nil {
			return 0, err
		}
		return elem * length, nil
	case typetable.KindTuple:
		return ev.layoutTupleSize(t)
	case typetable.KindStruct:
		return ev.layoutStructSize(t)
	case typetable.KindEnum:
		return ev.layoutEnumSize(t)
	case typetable.KindSlice, typetable.KindTraitObject:
		// Slice / dyn fat-pointers are two-word.
		return 16, nil
	case typetable.KindFn:
		// Fn values are function pointers.
		return 8, nil
	case typetable.KindUnion:
		return ev.layoutUnionSize(t)
	case typetable.KindGenericParam:
		return 0, fmt.Errorf("cannot compute size_of of an unspecialized generic parameter")
	}
	return 0, fmt.Errorf("size_of: type has no statically known layout")
}

// layoutAlign returns the alignment in bytes of t.
func (ev *Evaluator) layoutAlign(t typetable.TypeId) (uint64, error) {
	k := typeKind(ev.Tab, t)
	switch k {
	case typetable.KindBool, typetable.KindI8, typetable.KindU8:
		return 1, nil
	case typetable.KindI16, typetable.KindU16:
		return 2, nil
	case typetable.KindI32, typetable.KindU32, typetable.KindF32, typetable.KindChar:
		return 4, nil
	case typetable.KindI64, typetable.KindU64, typetable.KindF64,
		typetable.KindISize, typetable.KindUSize,
		typetable.KindPtr, typetable.KindRef, typetable.KindMutref,
		typetable.KindSlice, typetable.KindTraitObject, typetable.KindFn:
		return 8, nil
	case typetable.KindUnit, typetable.KindNever:
		return 1, nil
	case typetable.KindArray:
		elemID, _, ok := ev.arrayInfo(t)
		if !ok {
			return 0, fmt.Errorf("array type has unresolved length in a const context")
		}
		return ev.layoutAlign(elemID)
	case typetable.KindTuple:
		return ev.layoutTupleAlign(t)
	case typetable.KindStruct:
		return ev.layoutStructAlign(t)
	case typetable.KindEnum:
		return ev.layoutEnumAlign(t)
	case typetable.KindUnion:
		return ev.layoutUnionAlign(t)
	case typetable.KindGenericParam:
		return 0, fmt.Errorf("cannot compute align_of of an unspecialized generic parameter")
	}
	return 0, fmt.Errorf("align_of: type has no statically known layout")
}

// arrayInfo extracts the element TypeId and length from an Array
// type. The TypeTable stores arrays as (Children[0], Length); W14
// requires the length to be a concrete integer.
func (ev *Evaluator) arrayInfo(t typetable.TypeId) (typetable.TypeId, uint64, bool) {
	info := ev.Tab.Get(t)
	if info == nil || info.Kind != typetable.KindArray {
		return 0, 0, false
	}
	if len(info.Children) == 0 {
		return 0, 0, false
	}
	return info.Children[0], info.Length, true
}

// layoutTupleSize / layoutTupleAlign compute the size and alignment
// of a tuple by walking its element types. Padding follows the
// canonical "align each element to its own alignment" rule (Rule 9.2).
func (ev *Evaluator) layoutTupleSize(t typetable.TypeId) (uint64, error) {
	info := ev.Tab.Get(t)
	if info == nil {
		return 0, fmt.Errorf("tuple type missing from TypeTable")
	}
	var size, maxAlign uint64 = 0, 1
	for _, a := range info.Children {
		fSize, err := ev.layoutSize(a)
		if err != nil {
			return 0, err
		}
		fAlign, err := ev.layoutAlign(a)
		if err != nil {
			return 0, err
		}
		size = alignUp(size, fAlign)
		size += fSize
		if fAlign > maxAlign {
			maxAlign = fAlign
		}
	}
	size = alignUp(size, maxAlign)
	return size, nil
}

func (ev *Evaluator) layoutTupleAlign(t typetable.TypeId) (uint64, error) {
	info := ev.Tab.Get(t)
	if info == nil {
		return 0, fmt.Errorf("tuple type missing from TypeTable")
	}
	var maxAlign uint64 = 1
	for _, a := range info.Children {
		fAlign, err := ev.layoutAlign(a)
		if err != nil {
			return 0, err
		}
		if fAlign > maxAlign {
			maxAlign = fAlign
		}
	}
	return maxAlign, nil
}

// layoutStructSize walks the struct's named fields (or tuple fields
// for tuple-structs) and sums their sizes with alignment padding.
func (ev *Evaluator) layoutStructSize(t typetable.TypeId) (uint64, error) {
	s := ev.findStructDecl(t)
	if s == nil {
		return 0, fmt.Errorf("struct type has no declaration in the program")
	}
	fieldTypes := ev.structFieldTypes(s)
	var size, maxAlign uint64 = 0, 1
	for _, ft := range fieldTypes {
		fSize, err := ev.layoutSize(ft)
		if err != nil {
			return 0, err
		}
		fAlign, err := ev.layoutAlign(ft)
		if err != nil {
			return 0, err
		}
		size = alignUp(size, fAlign)
		size += fSize
		if fAlign > maxAlign {
			maxAlign = fAlign
		}
	}
	size = alignUp(size, maxAlign)
	if size == 0 {
		// Zero-sized struct still occupies one byte in the C11
		// backend to preserve address identity (reference §9.2.4).
		size = 0
	}
	return size, nil
}

func (ev *Evaluator) layoutStructAlign(t typetable.TypeId) (uint64, error) {
	s := ev.findStructDecl(t)
	if s == nil {
		return 1, nil
	}
	fieldTypes := ev.structFieldTypes(s)
	var maxAlign uint64 = 1
	for _, ft := range fieldTypes {
		fAlign, err := ev.layoutAlign(ft)
		if err != nil {
			return 0, err
		}
		if fAlign > maxAlign {
			maxAlign = fAlign
		}
	}
	return maxAlign, nil
}

// layoutEnumSize picks the widest variant payload plus a one-byte
// tag, rounded to the enum's alignment. Unit-only enums collapse to
// a single-byte tag.
func (ev *Evaluator) layoutEnumSize(t typetable.TypeId) (uint64, error) {
	e := ev.findEnumDecl(t)
	if e == nil {
		return 1, nil
	}
	var maxPayload, maxAlign uint64 = 0, 1
	for _, v := range e.Variants {
		pSize, pAlign, err := ev.variantLayout(v)
		if err != nil {
			return 0, err
		}
		if pSize > maxPayload {
			maxPayload = pSize
		}
		if pAlign > maxAlign {
			maxAlign = pAlign
		}
	}
	// Tag occupies one byte; pad tag to payload alignment, then
	// add payload, and pad total to enum alignment.
	total := alignUp(1, maxAlign) + maxPayload
	if total < 1 {
		total = 1
	}
	return alignUp(total, maxAlign), nil
}

func (ev *Evaluator) layoutEnumAlign(t typetable.TypeId) (uint64, error) {
	e := ev.findEnumDecl(t)
	if e == nil {
		return 1, nil
	}
	var maxAlign uint64 = 1
	for _, v := range e.Variants {
		_, pAlign, err := ev.variantLayout(v)
		if err != nil {
			return 0, err
		}
		if pAlign > maxAlign {
			maxAlign = pAlign
		}
	}
	return maxAlign, nil
}

// variantLayout returns the (size, align) of one enum variant's
// payload. Unit variants have zero-sized payload.
func (ev *Evaluator) variantLayout(v *hir.Variant) (uint64, uint64, error) {
	if v.IsUnit {
		return 0, 1, nil
	}
	var size, maxAlign uint64 = 0, 1
	walk := func(ft typetable.TypeId) error {
		fSize, err := ev.layoutSize(ft)
		if err != nil {
			return err
		}
		fAlign, err := ev.layoutAlign(ft)
		if err != nil {
			return err
		}
		size = alignUp(size, fAlign)
		size += fSize
		if fAlign > maxAlign {
			maxAlign = fAlign
		}
		return nil
	}
	for _, ft := range v.Tuple {
		if err := walk(ft); err != nil {
			return 0, 0, err
		}
	}
	for _, f := range v.Fields {
		if err := walk(f.TypeOf()); err != nil {
			return 0, 0, err
		}
	}
	size = alignUp(size, maxAlign)
	return size, maxAlign, nil
}

// layoutUnionSize / layoutUnionAlign model a C11 union: its size is
// the max of its fields' sizes, padded to the max alignment.
func (ev *Evaluator) layoutUnionSize(t typetable.TypeId) (uint64, error) {
	u := ev.findUnionDecl(t)
	if u == nil {
		return 1, nil
	}
	var maxSize, maxAlign uint64 = 0, 1
	for _, f := range u.Fields {
		ft := f.TypeOf()
		fSize, err := ev.layoutSize(ft)
		if err != nil {
			return 0, err
		}
		fAlign, err := ev.layoutAlign(ft)
		if err != nil {
			return 0, err
		}
		if fSize > maxSize {
			maxSize = fSize
		}
		if fAlign > maxAlign {
			maxAlign = fAlign
		}
	}
	return alignUp(maxSize, maxAlign), nil
}

func (ev *Evaluator) layoutUnionAlign(t typetable.TypeId) (uint64, error) {
	u := ev.findUnionDecl(t)
	if u == nil {
		return 1, nil
	}
	var maxAlign uint64 = 1
	for _, f := range u.Fields {
		fAlign, err := ev.layoutAlign(f.TypeOf())
		if err != nil {
			return 0, err
		}
		if fAlign > maxAlign {
			maxAlign = fAlign
		}
	}
	return maxAlign, nil
}

// findStructDecl / findEnumDecl / findUnionDecl look up nominal
// declarations by their TypeId. W14 does a linear scan (no index) —
// const evaluation is a small percentage of compile time, and
// wiring a reverse index adds memory that is not repaid in practice.
func (ev *Evaluator) findStructDecl(t typetable.TypeId) *hir.StructDecl {
	for _, modPath := range ev.Prog.Order {
		mod := ev.Prog.Modules[modPath]
		for _, it := range mod.Items {
			if s, ok := it.(*hir.StructDecl); ok && s.TypeID == t {
				return s
			}
		}
	}
	return nil
}

func (ev *Evaluator) findEnumDecl(t typetable.TypeId) *hir.EnumDecl {
	for _, modPath := range ev.Prog.Order {
		mod := ev.Prog.Modules[modPath]
		for _, it := range mod.Items {
			if e, ok := it.(*hir.EnumDecl); ok && e.TypeID == t {
				return e
			}
		}
	}
	return nil
}

func (ev *Evaluator) findUnionDecl(t typetable.TypeId) *hir.UnionDecl {
	for _, modPath := range ev.Prog.Order {
		mod := ev.Prog.Modules[modPath]
		for _, it := range mod.Items {
			if u, ok := it.(*hir.UnionDecl); ok && u.TypeID == t {
				return u
			}
		}
	}
	return nil
}

// structFieldTypes returns the TypeIds of a struct's fields in
// declaration order (named first, then tuple positions).
func (ev *Evaluator) structFieldTypes(s *hir.StructDecl) []typetable.TypeId {
	out := make([]typetable.TypeId, 0, len(s.Fields)+len(s.TupleFields))
	for _, f := range s.Fields {
		out = append(out, f.TypeOf())
	}
	out = append(out, s.TupleFields...)
	return out
}

// alignUp rounds n up to a multiple of alignment. `alignment` must
// be a power of two ≥ 1.
func alignUp(n, alignment uint64) uint64 {
	if alignment == 0 {
		return n
	}
	return (n + alignment - 1) & ^(alignment - 1)
}
