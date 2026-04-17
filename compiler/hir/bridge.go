package hir

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Bridge converts resolved AST into HIR. It consumes:
//
//   - A *typetable.Table for type interning.
//   - The *resolve.Resolved output with modules, symbols, and
//     path-occurrence bindings.
//   - The original []*resolve.SourceFile list so the bridge can
//     descend into the AST items that the resolver filtered.
//
// The bridge enforces the core W04 contract: every HIR Typed node
// ends up with a concrete TypeId or an explicit KindInfer TypeId.
// There is no third state. NoType is rejected by the builders, and
// no code path in this file assigns Unknown or defaults silently
// (reference §2.8, §7.9; L007, L013, L021 defenses).
//
// Types are derived, in priority order:
//
//  1. From a user-written annotation (`let x: I32 = ...`, fn
//     parameter and return types, field types).
//  2. From the resolver binding when the path names an item the
//     bridge has already registered a TypeId for.
//  3. From a literal's lexical kind (integer → I32, etc.).
//  4. Explicit `table.Infer()` for anything the type-checker is
//     required to settle (binary arithmetic on two free operands,
//     closure bodies, etc.).
//
// Rule 4 is the L013 escape hatch: if the bridge cannot honestly
// assign a concrete type, it writes `Infer` so W06 knows to fill
// it in. Passes observing a post-check `Infer` must emit a bug,
// not a user diagnostic.
type Bridge struct {
	Types     *typetable.Table
	Resolved  *resolve.Resolved
	Sources   []*resolve.SourceFile
	Program   *Program
	Builder   *Builder
	Diags     []lex.Diagnostic
	typeOfSym map[int]typetable.TypeId
}

// NewBridge constructs a Bridge with all inputs wired up. The
// returned Bridge is ready to call Run.
func NewBridge(tab *typetable.Table, resolved *resolve.Resolved, sources []*resolve.SourceFile) *Bridge {
	return &Bridge{
		Types:     tab,
		Resolved:  resolved,
		Sources:   sources,
		Program:   NewProgram(tab),
		Builder:   NewBuilder(tab),
		typeOfSym: map[int]typetable.TypeId{},
	}
}

// Run performs the two-phase bridge: phase 1 registers nominal and
// function TypeIds against their defining symbols; phase 2 lowers
// every item to HIR with types propagated. Diagnostics accumulate
// on b.Diags; the returned Program is never nil.
func (b *Bridge) Run() (*Program, []lex.Diagnostic) {
	b.registerItemTypes()
	b.lowerModules()
	return b.Program, b.Diags
}

// --- Phase 1: nominal + fn signature TypeIds -------------------------

// registerItemTypes walks every resolved module and assigns a TypeId
// to every symbol that declares a value or a type. Ordering is
// deterministic (module Order, then source-order items) so the
// intern table ends up identically shaped across runs (Rule 7.1).
func (b *Bridge) registerItemTypes() {
	for _, modPath := range b.Resolved.Graph.Order {
		m := b.Resolved.Graph.Modules[modPath]
		for _, it := range m.Items {
			b.registerItem(modPath, it)
		}
	}
}

// registerItem dispatches on the AST item kind and, for nominal and
// callable items, interns a TypeId that later uses can find via the
// Program.ItemType map.
func (b *Bridge) registerItem(modPath string, it ast.Item) {
	switch x := it.(type) {
	case *ast.StructDecl:
		symID := b.symbolFor(modPath, x.Name)
		if symID == 0 {
			return
		}
		id := b.Types.Nominal(typetable.KindStruct, symID, modPath, x.Name.Name, nil)
		b.typeOfSym[symID] = id
		b.Program.BindItemType(symID, id)
	case *ast.EnumDecl:
		symID := b.symbolFor(modPath, x.Name)
		if symID == 0 {
			return
		}
		id := b.Types.Nominal(typetable.KindEnum, symID, modPath, x.Name.Name, nil)
		b.typeOfSym[symID] = id
		b.Program.BindItemType(symID, id)
		// Enum variants inherit the enum's TypeId; their individual
		// identity comes from the SymEnumVariant symbol.
		for _, v := range x.Variants {
			vID := b.symbolFor(modPath, v.Name)
			if vID != 0 {
				b.Program.BindItemType(vID, id)
				b.typeOfSym[vID] = id
			}
		}
	case *ast.UnionDecl:
		symID := b.symbolFor(modPath, x.Name)
		if symID == 0 {
			return
		}
		id := b.Types.Nominal(typetable.KindUnion, symID, modPath, x.Name.Name, nil)
		b.typeOfSym[symID] = id
		b.Program.BindItemType(symID, id)
	case *ast.TraitDecl:
		symID := b.symbolFor(modPath, x.Name)
		if symID == 0 {
			return
		}
		id := b.Types.Nominal(typetable.KindTrait, symID, modPath, x.Name.Name, nil)
		b.typeOfSym[symID] = id
		b.Program.BindItemType(symID, id)
	case *ast.TypeDecl:
		symID := b.symbolFor(modPath, x.Name)
		if symID == 0 {
			return
		}
		id := b.Types.Nominal(typetable.KindTypeAlias, symID, modPath, x.Name.Name, nil)
		b.typeOfSym[symID] = id
		b.Program.BindItemType(symID, id)
	case *ast.FnDecl:
		b.registerFn(modPath, x)
	case *ast.ConstDecl:
		symID := b.symbolFor(modPath, x.Name)
		if symID == 0 {
			return
		}
		t := b.lowerType(modPath, x.Type)
		b.typeOfSym[symID] = t
		b.Program.BindItemType(symID, t)
	case *ast.StaticDecl:
		symID := b.symbolFor(modPath, x.Name)
		if symID == 0 {
			return
		}
		t := b.lowerType(modPath, x.Type)
		b.typeOfSym[symID] = t
		b.Program.BindItemType(symID, t)
	case *ast.ExternDecl:
		if x.Item != nil {
			b.registerItem(modPath, x.Item)
		}
	}
}

// registerFn computes and stores a Fn TypeId for x. Generics are
// preserved by emitting GenericParam TypeIds for any unresolved path
// that refers to a declared generic — W08 specialises these.
func (b *Bridge) registerFn(modPath string, x *ast.FnDecl) {
	symID := b.symbolFor(modPath, x.Name)
	if symID == 0 {
		return
	}
	params := make([]typetable.TypeId, 0, len(x.Params))
	for _, p := range x.Params {
		params = append(params, b.lowerType(modPath, p.Type))
	}
	ret := b.Types.Unit()
	if x.Return != nil {
		ret = b.lowerType(modPath, x.Return)
	}
	fnType := b.Types.Fn(params, ret, x.Variadic)
	b.typeOfSym[symID] = fnType
	b.Program.BindItemType(symID, fnType)
}

// symbolFor returns the resolve.SymbolID (as int) for the name
// declared by an item in modPath, or 0 when resolution did not
// register it. The resolver already produced the symbol in P02, so a
// zero return here means the item was @cfg-filtered or malformed.
func (b *Bridge) symbolFor(modPath string, name ast.Ident) int {
	m := b.Resolved.Graph.Modules[modPath]
	if m == nil {
		return 0
	}
	return int(m.Scope.LookupLocal(name.Name))
}

// --- Phase 2: module and item lowering ------------------------------

func (b *Bridge) lowerModules() {
	for _, modPath := range b.Resolved.Graph.Order {
		rm := b.Resolved.Graph.Modules[modPath]
		hm := &Module{
			Base: Base{ID: ItemID(modPath, ""), Span: lex.Span{File: rm.Source.File.Filename}},
			Path: modPath,
		}
		for _, it := range rm.Items {
			if lowered := b.lowerItem(modPath, it); lowered != nil {
				hm.Items = append(hm.Items, lowered)
			}
		}
		b.Program.RegisterModule(hm)
	}
}

// lowerItem is the top-level dispatch: AST item → HIR item (or nil
// when the item is stored purely via type registration, like a plain
// type alias with no body).
func (b *Bridge) lowerItem(modPath string, it ast.Item) Item {
	switch x := it.(type) {
	case *ast.FnDecl:
		return b.lowerFn(modPath, x)
	case *ast.StructDecl:
		return b.lowerStruct(modPath, x)
	case *ast.EnumDecl:
		return b.lowerEnum(modPath, x)
	case *ast.ConstDecl:
		return b.lowerConst(modPath, x)
	case *ast.StaticDecl:
		return b.lowerStatic(modPath, x)
	case *ast.TypeDecl:
		return b.lowerTypeAlias(modPath, x)
	case *ast.UnionDecl:
		return b.lowerUnion(modPath, x)
	case *ast.TraitDecl:
		return b.lowerTrait(modPath, x)
	case *ast.ImplDecl:
		return b.lowerImpl(modPath, x)
	case *ast.ExternDecl:
		if x.Item != nil {
			return b.lowerItem(modPath, x.Item)
		}
	}
	return nil
}

func (b *Bridge) lowerFn(modPath string, x *ast.FnDecl) *FnDecl {
	symID := b.symbolFor(modPath, x.Name)
	fnType, ok := b.typeOfSym[symID]
	if !ok {
		fnType = b.Types.Fn(nil, b.Types.Unit(), false)
	}
	idb := NewIdBuilder(modPath, x.Name.Name)
	idb.Push("params")
	params := make([]*Param, 0, len(x.Params))
	for i, p := range x.Params {
		paramID := idb.Child(fmt.Sprintf("%d", i))
		paramType := b.lowerType(modPath, p.Type)
		params = append(params, b.Builder.NewParam(
			paramID, p.NodeSpan(), p.Name.Name, paramType, toOwnership(p.Ownership)))
	}
	idb.Pop()

	ret := b.Types.Unit()
	if x.Return != nil {
		ret = b.lowerType(modPath, x.Return)
	}

	var body *Block
	if x.Body != nil {
		idb.Push("body")
		body = b.lowerBlock(modPath, idb, x.Body, ret)
		idb.Pop()
	}

	return b.Builder.NewFn(
		ItemID(modPath, x.Name.Name),
		x.NodeSpan(),
		x.Name.Name,
		fnType,
		params,
		ret,
		body,
	)
}

func (b *Bridge) lowerStruct(modPath string, x *ast.StructDecl) *StructDecl {
	symID := b.symbolFor(modPath, x.Name)
	typeID := b.typeOfSym[symID]
	idb := NewIdBuilder(modPath, x.Name.Name)
	var fields []*Field
	var tuple []typetable.TypeId
	for i, f := range x.Fields {
		fID := idb.Child(fmt.Sprintf("field.%d", i))
		fields = append(fields, b.Builder.NewField(
			fID, f.NodeSpan(), f.Name.Name, b.lowerType(modPath, f.Type)))
	}
	for _, t := range x.Tuple {
		tuple = append(tuple, b.lowerType(modPath, t))
	}
	return b.Builder.NewStruct(
		ItemID(modPath, x.Name.Name),
		x.NodeSpan(),
		x.Name.Name,
		typeID,
		fields,
		tuple,
		x.IsUnit,
	)
}

func (b *Bridge) lowerEnum(modPath string, x *ast.EnumDecl) *EnumDecl {
	symID := b.symbolFor(modPath, x.Name)
	typeID := b.typeOfSym[symID]
	variants := make([]*Variant, 0, len(x.Variants))
	for _, v := range x.Variants {
		vt := make([]typetable.TypeId, 0, len(v.Tuple))
		for _, t := range v.Tuple {
			vt = append(vt, b.lowerType(modPath, t))
		}
		var vfields []*Field
		for i, f := range v.Fields {
			fID := ItemID(modPath, x.Name.Name+"."+v.Name.Name+".field."+fmt.Sprintf("%d", i))
			vfields = append(vfields, b.Builder.NewField(
				fID, f.NodeSpan(), f.Name.Name, b.lowerType(modPath, f.Type)))
		}
		variants = append(variants, &Variant{
			Base:   Base{ID: ItemID(modPath, x.Name.Name+"."+v.Name.Name), Span: v.NodeSpan()},
			Name:   v.Name.Name,
			TypeID: typeID,
			Tuple:  vt,
			Fields: vfields,
			IsUnit: len(v.Tuple) == 0 && len(v.Fields) == 0,
		})
	}
	return b.Builder.NewEnum(
		ItemID(modPath, x.Name.Name),
		x.NodeSpan(),
		x.Name.Name,
		typeID,
		variants,
	)
}

func (b *Bridge) lowerConst(modPath string, x *ast.ConstDecl) *ConstDecl {
	declType := b.lowerType(modPath, x.Type)
	idb := NewIdBuilder(modPath, x.Name.Name)
	idb.Push("value")
	value := b.lowerExpr(modPath, idb, x.Value, declType)
	idb.Pop()
	return &ConstDecl{
		Base:  Base{ID: ItemID(modPath, x.Name.Name), Span: x.NodeSpan()},
		Name:  x.Name.Name,
		Type:  declType,
		Value: value,
	}
}

func (b *Bridge) lowerStatic(modPath string, x *ast.StaticDecl) *StaticDecl {
	declType := b.lowerType(modPath, x.Type)
	var value Expr
	if x.Value != nil {
		idb := NewIdBuilder(modPath, x.Name.Name)
		idb.Push("value")
		value = b.lowerExpr(modPath, idb, x.Value, declType)
		idb.Pop()
	}
	return &StaticDecl{
		Base:     Base{ID: ItemID(modPath, x.Name.Name), Span: x.NodeSpan()},
		Name:     x.Name.Name,
		Type:     declType,
		Value:    value,
		IsExtern: x.IsExtern,
	}
}

func (b *Bridge) lowerTypeAlias(modPath string, x *ast.TypeDecl) *TypeAliasDecl {
	symID := b.symbolFor(modPath, x.Name)
	typeID := b.typeOfSym[symID]
	target := b.lowerType(modPath, x.Target)
	return &TypeAliasDecl{
		Base:   Base{ID: ItemID(modPath, x.Name.Name), Span: x.NodeSpan()},
		Name:   x.Name.Name,
		TypeID: typeID,
		Target: target,
	}
}

func (b *Bridge) lowerUnion(modPath string, x *ast.UnionDecl) *UnionDecl {
	symID := b.symbolFor(modPath, x.Name)
	typeID := b.typeOfSym[symID]
	var fields []*Field
	idb := NewIdBuilder(modPath, x.Name.Name)
	for i, f := range x.Fields {
		fID := idb.Child(fmt.Sprintf("field.%d", i))
		fields = append(fields, b.Builder.NewField(
			fID, f.NodeSpan(), f.Name.Name, b.lowerType(modPath, f.Type)))
	}
	return &UnionDecl{
		Base:   Base{ID: ItemID(modPath, x.Name.Name), Span: x.NodeSpan()},
		Name:   x.Name.Name,
		TypeID: typeID,
		Fields: fields,
	}
}

func (b *Bridge) lowerTrait(modPath string, x *ast.TraitDecl) *TraitDecl {
	symID := b.symbolFor(modPath, x.Name)
	typeID := b.typeOfSym[symID]
	var supers []typetable.TypeId
	for _, s := range x.Supertrs {
		supers = append(supers, b.lowerType(modPath, s))
	}
	// Trait body items are lowered but — at W04 — not fully typed;
	// type-checker integration lands in W06.
	var items []Item
	for _, it := range x.Items {
		if li := b.lowerItem(modPath, it); li != nil {
			items = append(items, li)
		}
	}
	return &TraitDecl{
		Base:     Base{ID: ItemID(modPath, x.Name.Name), Span: x.NodeSpan()},
		Name:     x.Name.Name,
		TypeID:   typeID,
		Supertrs: supers,
		Items:    items,
	}
}

func (b *Bridge) lowerImpl(modPath string, x *ast.ImplDecl) *ImplDecl {
	target := b.lowerType(modPath, x.Target)
	var traitID typetable.TypeId
	if x.Trait != nil {
		traitID = b.lowerType(modPath, x.Trait)
	}
	anchor := "impl_" + typeTextForIdentity(b.Types, target)
	_ = anchor // reserved for nested item lowering once W06 drives it
	var items []Item
	for _, it := range x.Items {
		if li := b.lowerItem(modPath, it); li != nil {
			items = append(items, li)
		}
	}
	return &ImplDecl{
		Base:   Base{ID: NodeID(fmt.Sprintf("%s::impl:%d", modPath, target)), Span: x.NodeSpan()},
		Target: target,
		Trait:  traitID,
		Items:  items,
	}
}

// --- Type lowering --------------------------------------------------

// lowerType converts an AST type expression to a TypeId. Unresolved
// type paths fall through to KindInfer (explicit pending inference),
// never Unknown (L013).
func (b *Bridge) lowerType(modPath string, t ast.Type) typetable.TypeId {
	if t == nil {
		return b.Types.Unit()
	}
	switch x := t.(type) {
	case *ast.PathType:
		return b.lowerPathType(modPath, x)
	case *ast.TupleType:
		elems := make([]typetable.TypeId, 0, len(x.Elements))
		for _, el := range x.Elements {
			elems = append(elems, b.lowerType(modPath, el))
		}
		return b.Types.Tuple(elems)
	case *ast.ArrayType:
		elem := b.lowerType(modPath, x.Element)
		// Length is a const expression; at W04 we record 0 when it
		// is not a literal integer. W14 (consteval) settles this.
		return b.Types.Array(elem, literalUint(x.Length))
	case *ast.SliceType:
		return b.Types.Slice(b.lowerType(modPath, x.Element))
	case *ast.PtrType:
		return b.Types.Ptr(b.lowerType(modPath, x.Pointee))
	case *ast.FnType:
		params := make([]typetable.TypeId, 0, len(x.Params))
		for _, p := range x.Params {
			params = append(params, b.lowerType(modPath, p))
		}
		ret := b.Types.Unit()
		if x.Return != nil {
			ret = b.lowerType(modPath, x.Return)
		}
		return b.Types.Fn(params, ret, false)
	case *ast.DynType:
		bounds := make([]typetable.TypeId, 0, len(x.Traits))
		for _, tr := range x.Traits {
			bounds = append(bounds, b.lowerType(modPath, tr))
		}
		return b.Types.TraitObject(bounds)
	case *ast.UnitType:
		return b.Types.Unit()
	case *ast.ImplType:
		if x.Trait != nil {
			return b.lowerType(modPath, x.Trait)
		}
	}
	return b.Types.Infer()
}

// lowerPathType resolves a PathType to a TypeId via: (a) primitive
// name lookup; (b) resolve.Bindings for the path's span; (c)
// fallback to Infer.
func (b *Bridge) lowerPathType(modPath string, x *ast.PathType) typetable.TypeId {
	if len(x.Segments) == 1 {
		if tid := primitiveKindLookup(b.Types, x.Segments[0].Name); tid != typetable.NoType {
			return tid
		}
	}
	if sym, ok := b.Resolved.Bindings[resolve.SiteKey{Module: modPath, Span: x.NodeSpan()}]; ok {
		if tid, ok := b.typeOfSym[int(sym)]; ok {
			return tid
		}
	}
	// Channel and ThreadHandle are spelled as generic-looking PathTypes
	// `Chan[T]` / `ThreadHandle[T]`. Recognise them so W07 has a stable
	// TypeId to attach to.
	if len(x.Segments) == 1 {
		switch x.Segments[0].Name {
		case "Chan":
			if len(x.Args) == 1 {
				return b.Types.Channel(b.lowerType(modPath, x.Args[0]))
			}
		case "ThreadHandle":
			if len(x.Args) == 1 {
				return b.Types.ThreadHandle(b.lowerType(modPath, x.Args[0]))
			}
		}
	}
	return b.Types.Infer()
}

// primitiveKindLookup returns the TypeId for a bare primitive name or
// NoType if name is not a primitive spelling.
func primitiveKindLookup(tab *typetable.Table, name string) typetable.TypeId {
	switch name {
	case "Bool":
		return tab.Bool()
	case "I8":
		return tab.I8()
	case "I16":
		return tab.I16()
	case "I32":
		return tab.I32()
	case "I64":
		return tab.I64()
	case "ISize":
		return tab.ISize()
	case "U8":
		return tab.U8()
	case "U16":
		return tab.U16()
	case "U32":
		return tab.U32()
	case "U64":
		return tab.U64()
	case "USize":
		return tab.USize()
	case "F32":
		return tab.F32()
	case "F64":
		return tab.F64()
	case "Char":
		return tab.Char()
	case "String":
		return tab.String_()
	case "CStr":
		return tab.CStr()
	case "Unit":
		return tab.Unit()
	case "Never":
		return tab.Never()
	}
	return typetable.NoType
}

// literalUint extracts the uint64 value of a literal-int expression or
// 0 when the expression is anything else. Array length types that
// aren't literal integers must go through W14 const evaluation; at W04
// we accept zero as a placeholder and mark the array's length field as
// opaque to downstream uses.
func literalUint(e ast.Expr) uint64 {
	if lit, ok := e.(*ast.LiteralExpr); ok && lit.Kind == ast.LitInt {
		var v uint64
		for _, c := range lit.Text {
			if c < '0' || c > '9' {
				return 0
			}
			v = v*10 + uint64(c-'0')
		}
		return v
	}
	return 0
}

// toOwnership maps ast.Ownership to hir.Ownership. They are
// intentionally two disjoint enums (Rule 3.1) so renaming one never
// accidentally changes the other.
func toOwnership(o ast.Ownership) Ownership {
	switch o {
	case ast.OwnRef:
		return OwnRef
	case ast.OwnMutref:
		return OwnMutref
	case ast.OwnOwned:
		return OwnOwned
	}
	return OwnNone
}

// typeTextForIdentity returns a stable string spelling of a TypeId
// for use in node-ID construction where the TypeId participates in
// identity. The spelling is not a full pretty-printer; it is just
// deterministic enough that impl block IDs don't collide.
func typeTextForIdentity(tab *typetable.Table, tid typetable.TypeId) string {
	t := tab.Get(tid)
	if t == nil {
		return "NoType"
	}
	switch t.Kind {
	case typetable.KindStruct, typetable.KindEnum, typetable.KindUnion,
		typetable.KindTrait, typetable.KindTypeAlias:
		return t.Name
	}
	return t.Kind.String()
}

// --- Expression lowering -------------------------------------------

// lowerExpr dispatches AST expressions to HIR. The hint parameter is
// a "wanted type" the caller can pass when context is known (e.g.
// `let x: I32 = <expr>` wants I32). The bridge uses the hint for
// literals whose concrete primitive kind is ambiguous (integer-kind
// literals default to I32 otherwise).
func (b *Bridge) lowerExpr(modPath string, idb *IdBuilder, e ast.Expr, hint typetable.TypeId) Expr {
	if e == nil {
		return b.Builder.NewLiteral(idb.Child("null"), lex.Span{}, LitNone, "", b.Types.Unit())
	}
	switch x := e.(type) {
	case *ast.LiteralExpr:
		return b.lowerLiteral(idb, x, hint)
	case *ast.PathExpr:
		return b.lowerPathExpr(modPath, idb, x)
	case *ast.BinaryExpr:
		return b.lowerBinary(modPath, idb, x)
	case *ast.UnaryExpr:
		return b.lowerUnary(modPath, idb, x)
	case *ast.AssignExpr:
		return b.lowerAssign(modPath, idb, x)
	case *ast.CastExpr:
		return b.lowerCast(modPath, idb, x)
	case *ast.CallExpr:
		return b.lowerCall(modPath, idb, x)
	case *ast.FieldExpr:
		return b.lowerField(modPath, idb, x)
	case *ast.OptFieldExpr:
		return b.lowerOptField(modPath, idb, x)
	case *ast.TryExpr:
		return b.lowerTry(modPath, idb, x)
	case *ast.IndexExpr:
		return b.lowerIndex(modPath, idb, x)
	case *ast.IndexRangeExpr:
		return b.lowerIndexRange(modPath, idb, x)
	case *ast.BlockExpr:
		return b.lowerBlock(modPath, idb, x, hint)
	case *ast.IfExpr:
		return b.lowerIf(modPath, idb, x, hint)
	case *ast.MatchExpr:
		return b.lowerMatch(modPath, idb, x, hint)
	case *ast.LoopExpr:
		return b.lowerLoop(modPath, idb, x)
	case *ast.WhileExpr:
		return b.lowerWhile(modPath, idb, x)
	case *ast.ForExpr:
		return b.lowerFor(modPath, idb, x)
	case *ast.TupleExpr:
		return b.lowerTuple(modPath, idb, x)
	case *ast.StructLitExpr:
		return b.lowerStructLit(modPath, idb, x)
	case *ast.ParenExpr:
		return b.lowerExpr(modPath, idb, x.Inner, hint)
	case *ast.ClosureExpr:
		return b.lowerClosure(modPath, idb, x)
	case *ast.SpawnExpr:
		return b.lowerSpawn(modPath, idb, x)
	case *ast.UnsafeExpr:
		return b.lowerUnsafe(modPath, idb, x, hint)
	}
	// Unknown AST form — bridge bug, not user input. Fail loudly.
	panic(fmt.Sprintf("hir.Bridge.lowerExpr: unhandled AST expression %T", e))
}

func (b *Bridge) lowerLiteral(idb *IdBuilder, x *ast.LiteralExpr, hint typetable.TypeId) *LiteralExpr {
	var (
		kind LitKind
		typ  typetable.TypeId
	)
	switch x.Kind {
	case ast.LitInt:
		kind = LitInt
		typ = b.Types.I32()
		if hint != typetable.NoType && isIntegerPrimitive(b.Types, hint) {
			typ = hint
		}
	case ast.LitFloat:
		kind = LitFloat
		typ = b.Types.F64()
		if hint != typetable.NoType && isFloatPrimitive(b.Types, hint) {
			typ = hint
		}
	case ast.LitString:
		kind, typ = LitString, b.Types.String_()
	case ast.LitRawString:
		kind, typ = LitRawString, b.Types.String_()
	case ast.LitCString:
		kind, typ = LitCString, b.Types.CStr()
	case ast.LitChar:
		kind, typ = LitChar, b.Types.Char()
	case ast.LitBool:
		kind, typ = LitBool, b.Types.Bool()
	case ast.LitNone:
		kind, typ = LitNone, b.Types.Infer()
	}
	lit := b.Builder.NewLiteral(idb.Child("lit"), x.NodeSpan(), kind, x.Text, typ)
	lit.Bool = x.Value
	return lit
}

func (b *Bridge) lowerPathExpr(modPath string, idb *IdBuilder, x *ast.PathExpr) *PathExpr {
	segs := make([]string, len(x.Segments))
	for i, s := range x.Segments {
		segs[i] = s.Name
	}
	symID := 0
	typ := b.Types.Infer()
	if sym, ok := b.Resolved.Bindings[resolve.SiteKey{Module: modPath, Span: x.NodeSpan()}]; ok {
		symID = int(sym)
		if tid, ok := b.typeOfSym[symID]; ok {
			typ = tid
		}
	}
	return b.Builder.NewPath(idb.Child("path:"+strings.Join(segs, ".")), x.NodeSpan(), symID, segs, typ)
}

func (b *Bridge) lowerBinary(modPath string, idb *IdBuilder, x *ast.BinaryExpr) *BinaryExpr {
	lhs := b.lowerExpr(modPath, idb.Push("lhs"), x.Lhs, typetable.NoType)
	idb.Pop()
	rhs := b.lowerExpr(modPath, idb.Push("rhs"), x.Rhs, typetable.NoType)
	idb.Pop()
	// Type: Bool for comparison/logical operators; Infer for
	// arithmetic until W06 unifies operand types.
	typ := b.Types.Infer()
	if isBoolOp(mapBinOp(x.Op)) {
		typ = b.Types.Bool()
	}
	return b.Builder.NewBinary(idb.Child("bin"), x.NodeSpan(), mapBinOp(x.Op), lhs, rhs, typ)
}

func (b *Bridge) lowerUnary(modPath string, idb *IdBuilder, x *ast.UnaryExpr) *UnaryExpr {
	inner := b.lowerExpr(modPath, idb.Push("unary"), x.Operand, typetable.NoType)
	idb.Pop()
	typ := b.Types.Infer()
	op := mapUnaryOp(x.Op)
	if op == UnNot {
		typ = b.Types.Bool()
	}
	u := &UnaryExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("un"), Span: x.NodeSpan()}, Type: typ},
		Op:        op,
		Operand:   inner,
	}
	return u
}

func (b *Bridge) lowerAssign(modPath string, idb *IdBuilder, x *ast.AssignExpr) *AssignExpr {
	lhs := b.lowerExpr(modPath, idb.Push("lhs"), x.Lhs, typetable.NoType)
	idb.Pop()
	rhs := b.lowerExpr(modPath, idb.Push("rhs"), x.Rhs, typetable.NoType)
	idb.Pop()
	return &AssignExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("assign"), Span: x.NodeSpan()}, Type: b.Types.Unit()},
		Op:        mapAssignOp(x.Op),
		Lhs:       lhs,
		Rhs:       rhs,
	}
}

func (b *Bridge) lowerCast(modPath string, idb *IdBuilder, x *ast.CastExpr) *CastExpr {
	inner := b.lowerExpr(modPath, idb.Push("cast"), x.Expr, typetable.NoType)
	idb.Pop()
	typ := b.lowerType(modPath, x.Type)
	return &CastExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("cast"), Span: x.NodeSpan()}, Type: typ},
		Expr:      inner,
	}
}

func (b *Bridge) lowerCall(modPath string, idb *IdBuilder, x *ast.CallExpr) *CallExpr {
	callee := b.lowerExpr(modPath, idb.Push("callee"), x.Callee, typetable.NoType)
	idb.Pop()
	args := make([]Expr, 0, len(x.Args))
	for i, a := range x.Args {
		idb.Push(fmt.Sprintf("arg.%d", i))
		args = append(args, b.lowerExpr(modPath, idb, a, typetable.NoType))
		idb.Pop()
	}
	// Result type: if callee's type is a Fn, use its Return; else Infer.
	typ := b.Types.Infer()
	if cT := b.Types.Get(callee.TypeOf()); cT != nil && cT.Kind == typetable.KindFn {
		typ = cT.Return
	}
	return b.Builder.NewCall(idb.Child("call"), x.NodeSpan(), callee, args, typ)
}

func (b *Bridge) lowerField(modPath string, idb *IdBuilder, x *ast.FieldExpr) *FieldExpr {
	recv := b.lowerExpr(modPath, idb.Push("recv"), x.Receiver, typetable.NoType)
	idb.Pop()
	return &FieldExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("field:" + x.Name.Name), Span: x.NodeSpan()}, Type: b.Types.Infer()},
		Receiver:  recv,
		Name:      x.Name.Name,
	}
}

func (b *Bridge) lowerOptField(modPath string, idb *IdBuilder, x *ast.OptFieldExpr) *OptFieldExpr {
	recv := b.lowerExpr(modPath, idb.Push("recv"), x.Receiver, typetable.NoType)
	idb.Pop()
	return &OptFieldExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("optfield:" + x.Name.Name), Span: x.NodeSpan()}, Type: b.Types.Infer()},
		Receiver:  recv,
		Name:      x.Name.Name,
	}
}

func (b *Bridge) lowerTry(modPath string, idb *IdBuilder, x *ast.TryExpr) *TryExpr {
	recv := b.lowerExpr(modPath, idb.Push("try"), x.Receiver, typetable.NoType)
	idb.Pop()
	return &TryExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("try"), Span: x.NodeSpan()}, Type: b.Types.Infer()},
		Receiver:  recv,
	}
}

func (b *Bridge) lowerIndex(modPath string, idb *IdBuilder, x *ast.IndexExpr) *IndexExpr {
	recv := b.lowerExpr(modPath, idb.Push("recv"), x.Receiver, typetable.NoType)
	idb.Pop()
	idx := b.lowerExpr(modPath, idb.Push("idx"), x.Index, b.Types.USize())
	idb.Pop()
	return &IndexExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("index"), Span: x.NodeSpan()}, Type: b.Types.Infer()},
		Receiver:  recv,
		Index:     idx,
	}
}

func (b *Bridge) lowerIndexRange(modPath string, idb *IdBuilder, x *ast.IndexRangeExpr) *IndexRangeExpr {
	recv := b.lowerExpr(modPath, idb.Push("recv"), x.Receiver, typetable.NoType)
	idb.Pop()
	var lo, hi Expr
	if x.Low != nil {
		lo = b.lowerExpr(modPath, idb.Push("lo"), x.Low, b.Types.USize())
		idb.Pop()
	}
	if x.High != nil {
		hi = b.lowerExpr(modPath, idb.Push("hi"), x.High, b.Types.USize())
		idb.Pop()
	}
	return &IndexRangeExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("range"), Span: x.NodeSpan()}, Type: b.Types.Infer()},
		Receiver:  recv,
		Low:       lo,
		High:      hi,
		Inclusive: x.Inclusive,
	}
}

// lowerBlock converts an AST BlockExpr. expectedType is the context's
// wanted type; the block's own Type is the hint (or Infer when the
// trailing is absent / the caller doesn't care).
func (b *Bridge) lowerBlock(modPath string, idb *IdBuilder, x *ast.BlockExpr, expectedType typetable.TypeId) *Block {
	idb.Push("block")
	defer idb.Pop()
	stmts := make([]Stmt, 0, len(x.Stmts))
	for i, s := range x.Stmts {
		idb.Push(fmt.Sprintf("stmt.%d", i))
		if ls := b.lowerStmt(modPath, idb, s); ls != nil {
			stmts = append(stmts, ls)
		}
		idb.Pop()
	}
	var trailing Expr
	typ := b.Types.Unit()
	if x.Trailing != nil {
		idb.Push("trailing")
		trailing = b.lowerExpr(modPath, idb, x.Trailing, expectedType)
		idb.Pop()
		typ = trailing.TypeOf()
	}
	return b.Builder.NewBlock(idb.Here(), x.NodeSpan(), stmts, trailing, typ)
}

func (b *Bridge) lowerIf(modPath string, idb *IdBuilder, x *ast.IfExpr, hint typetable.TypeId) *IfExpr {
	cond := b.lowerExpr(modPath, idb.Push("cond"), x.Cond, b.Types.Bool())
	idb.Pop()
	thenBlk := b.lowerBlock(modPath, idb.Push("then"), x.Then, hint)
	idb.Pop()
	var els Expr
	if x.Else != nil {
		els = b.lowerExpr(modPath, idb.Push("else"), x.Else, hint)
		idb.Pop()
	}
	typ := b.Types.Unit()
	if thenBlk.Type != typetable.NoType {
		typ = thenBlk.Type
	}
	return b.Builder.NewIf(idb.Child("if"), x.NodeSpan(), cond, thenBlk, els, typ)
}

func (b *Bridge) lowerMatch(modPath string, idb *IdBuilder, x *ast.MatchExpr, hint typetable.TypeId) *MatchExpr {
	scrut := b.lowerExpr(modPath, idb.Push("scrut"), x.Scrutinee, typetable.NoType)
	idb.Pop()
	arms := make([]*MatchArm, 0, len(x.Arms))
	for i, a := range x.Arms {
		idb.Push(fmt.Sprintf("arm.%d", i))
		pat := b.lowerPat(modPath, idb.Push("pat"), a.Pattern, scrut.TypeOf())
		idb.Pop()
		var guard Expr
		if a.Guard != nil {
			guard = b.lowerExpr(modPath, idb.Push("guard"), a.Guard, b.Types.Bool())
			idb.Pop()
		}
		body := b.lowerBlock(modPath, idb.Push("body"), a.Body, hint)
		idb.Pop()
		arms = append(arms, &MatchArm{
			Base:    Base{ID: idb.Child("arm"), Span: a.NodeSpan()},
			Pattern: pat,
			Guard:   guard,
			Body:    body,
		})
		idb.Pop()
	}
	typ := hint
	if typ == typetable.NoType {
		typ = b.Types.Infer()
	}
	return b.Builder.NewMatch(idb.Child("match"), x.NodeSpan(), scrut, arms, typ)
}

func (b *Bridge) lowerLoop(modPath string, idb *IdBuilder, x *ast.LoopExpr) *LoopExpr {
	body := b.lowerBlock(modPath, idb.Push("body"), x.Body, typetable.NoType)
	idb.Pop()
	return &LoopExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("loop"), Span: x.NodeSpan()}, Type: b.Types.Infer()},
		Body:      body,
	}
}

func (b *Bridge) lowerWhile(modPath string, idb *IdBuilder, x *ast.WhileExpr) *WhileExpr {
	cond := b.lowerExpr(modPath, idb.Push("cond"), x.Cond, b.Types.Bool())
	idb.Pop()
	body := b.lowerBlock(modPath, idb.Push("body"), x.Body, typetable.NoType)
	idb.Pop()
	return &WhileExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("while"), Span: x.NodeSpan()}, Type: b.Types.Unit()},
		Cond:      cond,
		Body:      body,
	}
}

func (b *Bridge) lowerFor(modPath string, idb *IdBuilder, x *ast.ForExpr) *ForExpr {
	pat := b.lowerPat(modPath, idb.Push("pat"), x.Pattern, typetable.NoType)
	idb.Pop()
	iter := b.lowerExpr(modPath, idb.Push("iter"), x.Iter, typetable.NoType)
	idb.Pop()
	body := b.lowerBlock(modPath, idb.Push("body"), x.Body, typetable.NoType)
	idb.Pop()
	return &ForExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("for"), Span: x.NodeSpan()}, Type: b.Types.Unit()},
		Pattern:   pat,
		Iter:      iter,
		Body:      body,
	}
}

func (b *Bridge) lowerTuple(modPath string, idb *IdBuilder, x *ast.TupleExpr) *TupleExpr {
	elems := make([]Expr, 0, len(x.Elements))
	elemTypes := make([]typetable.TypeId, 0, len(x.Elements))
	for i, el := range x.Elements {
		idb.Push(fmt.Sprintf("elem.%d", i))
		e := b.lowerExpr(modPath, idb, el, typetable.NoType)
		idb.Pop()
		elems = append(elems, e)
		elemTypes = append(elemTypes, e.TypeOf())
	}
	return &TupleExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("tuple"), Span: x.NodeSpan()}, Type: b.Types.Tuple(elemTypes)},
		Elements:  elems,
	}
}

func (b *Bridge) lowerStructLit(modPath string, idb *IdBuilder, x *ast.StructLitExpr) *StructLitExpr {
	structType := b.Types.Infer()
	if x.Path != nil {
		if sym, ok := b.Resolved.Bindings[resolve.SiteKey{Module: modPath, Span: x.Path.NodeSpan()}]; ok {
			if tid, ok := b.typeOfSym[int(sym)]; ok {
				structType = tid
			}
		}
	}
	var fields []*StructLitField
	for i, f := range x.Fields {
		idb.Push(fmt.Sprintf("field.%d", i))
		var val Expr
		if f.Shorthand {
			val = b.Builder.NewPath(
				idb.Child("shorthand:"+f.Name.Name),
				f.NodeSpan(), 0, []string{f.Name.Name}, b.Types.Infer())
		} else {
			val = b.lowerExpr(modPath, idb, f.Value, typetable.NoType)
		}
		idb.Pop()
		fields = append(fields, &StructLitField{
			Base:  Base{ID: idb.Child("field:" + f.Name.Name), Span: f.NodeSpan()},
			Name:  f.Name.Name,
			Value: val,
		})
	}
	var base Expr
	if x.Base != nil {
		base = b.lowerExpr(modPath, idb.Push("base"), x.Base, structType)
		idb.Pop()
	}
	return &StructLitExpr{
		TypedBase:  TypedBase{Base: Base{ID: idb.Child("slit"), Span: x.NodeSpan()}, Type: structType},
		StructType: structType,
		Fields:     fields,
		Base:       base,
	}
}

func (b *Bridge) lowerClosure(modPath string, idb *IdBuilder, x *ast.ClosureExpr) *ClosureExpr {
	idb.Push("closure")
	defer idb.Pop()
	params := make([]*Param, 0, len(x.Params))
	paramTypes := make([]typetable.TypeId, 0, len(x.Params))
	for i, p := range x.Params {
		pt := b.lowerType(modPath, p.Type)
		paramTypes = append(paramTypes, pt)
		params = append(params, b.Builder.NewParam(
			idb.Child(fmt.Sprintf("param.%d", i)),
			p.NodeSpan(), p.Name.Name, pt, toOwnership(p.Ownership)))
	}
	ret := b.Types.Unit()
	if x.Return != nil {
		ret = b.lowerType(modPath, x.Return)
	}
	var body *Block
	if x.Body != nil {
		body = b.lowerBlock(modPath, idb.Push("body"), x.Body, ret)
		idb.Pop()
	}
	return &ClosureExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Here(), Span: x.NodeSpan()}, Type: b.Types.Fn(paramTypes, ret, false)},
		IsMove:    x.IsMove,
		Params:    params,
		Return:    ret,
		Body:      body,
	}
}

func (b *Bridge) lowerSpawn(modPath string, idb *IdBuilder, x *ast.SpawnExpr) *SpawnExpr {
	idb.Push("spawn")
	defer idb.Pop()
	inner := b.lowerClosure(modPath, idb, x.Inner)
	// The spawn expression's TypeId is ThreadHandle[inner-return].
	ret := b.Types.Unit()
	if tt := b.Types.Get(inner.TypeOf()); tt != nil && tt.Kind == typetable.KindFn {
		ret = tt.Return
	}
	return &SpawnExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Here(), Span: x.NodeSpan()}, Type: b.Types.ThreadHandle(ret)},
		Closure:   inner,
	}
}

func (b *Bridge) lowerUnsafe(modPath string, idb *IdBuilder, x *ast.UnsafeExpr, hint typetable.TypeId) *UnsafeExpr {
	body := b.lowerBlock(modPath, idb.Push("unsafe"), x.Body, hint)
	idb.Pop()
	return &UnsafeExpr{
		TypedBase: TypedBase{Base: Base{ID: idb.Child("unsafe"), Span: x.NodeSpan()}, Type: body.TypeOf()},
		Body:      body,
	}
}

// --- Statement lowering --------------------------------------------

func (b *Bridge) lowerStmt(modPath string, idb *IdBuilder, s ast.Stmt) Stmt {
	if s == nil {
		return nil
	}
	switch x := s.(type) {
	case *ast.LetStmt:
		declared := b.Types.Infer()
		if x.Type != nil {
			declared = b.lowerType(modPath, x.Type)
		}
		pat := b.lowerPat(modPath, idb.Push("pat"), x.Pattern, declared)
		idb.Pop()
		var val Expr
		if x.Value != nil {
			val = b.lowerExpr(modPath, idb.Push("val"), x.Value, declared)
			idb.Pop()
		}
		return b.Builder.NewLet(idb.Child("let"), x.NodeSpan(), pat, declared, val)
	case *ast.VarStmt:
		declared := b.Types.Infer()
		if x.Type != nil {
			declared = b.lowerType(modPath, x.Type)
		}
		val := b.lowerExpr(modPath, idb.Push("val"), x.Value, declared)
		idb.Pop()
		return &VarStmt{
			Base:         Base{ID: idb.Child("var:" + x.Name.Name), Span: x.NodeSpan()},
			Name:         x.Name.Name,
			DeclaredType: declared,
			Value:        val,
		}
	case *ast.ReturnStmt:
		var val Expr
		if x.Value != nil {
			val = b.lowerExpr(modPath, idb.Push("ret"), x.Value, typetable.NoType)
			idb.Pop()
		}
		return b.Builder.NewReturn(idb.Child("return"), x.NodeSpan(), val)
	case *ast.BreakStmt:
		var val Expr
		if x.Value != nil {
			val = b.lowerExpr(modPath, idb.Push("brk"), x.Value, typetable.NoType)
			idb.Pop()
		}
		return &BreakStmt{Base: Base{ID: idb.Child("break"), Span: x.NodeSpan()}, Value: val}
	case *ast.ContinueStmt:
		return &ContinueStmt{Base: Base{ID: idb.Child("continue"), Span: x.NodeSpan()}}
	case *ast.ExprStmt:
		e := b.lowerExpr(modPath, idb.Push("expr"), x.Expr, typetable.NoType)
		idb.Pop()
		return b.Builder.NewExprStmt(idb.Child("exprstmt"), x.NodeSpan(), e)
	case *ast.ItemStmt:
		inner := b.lowerItem(modPath, x.Item)
		if inner == nil {
			return nil
		}
		return &ItemStmt{Base: Base{ID: idb.Child("item"), Span: x.NodeSpan()}, Item: inner}
	}
	return nil
}

// --- Pattern lowering ----------------------------------------------

func (b *Bridge) lowerPat(modPath string, idb *IdBuilder, p ast.Pat, typ typetable.TypeId) Pat {
	if typ == typetable.NoType {
		typ = b.Types.Infer()
	}
	switch x := p.(type) {
	case *ast.LiteralPat:
		kind := LitInt
		switch x.Value.Kind {
		case ast.LitFloat:
			kind = LitFloat
		case ast.LitString:
			kind = LitString
		case ast.LitRawString:
			kind = LitRawString
		case ast.LitCString:
			kind = LitCString
		case ast.LitChar:
			kind = LitChar
		case ast.LitBool:
			kind = LitBool
		case ast.LitNone:
			kind = LitNone
		}
		lp := b.Builder.NewLiteralPat(idb.Child("litpat"), x.NodeSpan(), kind, x.Value.Text, typ)
		lp.Bool = x.Value.Value
		return lp
	case *ast.WildcardPat:
		return b.Builder.NewWildcardPat(idb.Child("wild"), x.NodeSpan(), typ)
	case *ast.BindPat:
		return b.Builder.NewBindPat(idb.Child("bind:"+x.Name.Name), x.NodeSpan(), x.Name.Name, typ)
	case *ast.CtorPat:
		return b.lowerCtorPat(modPath, idb, x, typ)
	case *ast.TuplePat:
		// TuplePat in AST has no struct/ctor path; lower as an
		// unnamed ConstructorPat with an Infer constructor so the
		// builder accepts it. This is a structural pattern per L007;
		// the constructor is a tuple type.
		elems := make([]Pat, 0, len(x.Elements))
		for i, el := range x.Elements {
			idb.Push(fmt.Sprintf("t.%d", i))
			elems = append(elems, b.lowerPat(modPath, idb, el, b.Types.Infer()))
			idb.Pop()
		}
		return b.Builder.NewConstructorPat(
			idb.Child("tuplepat"), x.NodeSpan(),
			typ, "", []string{"(tuple)"}, elems, nil, false, typ)
	case *ast.OrPat:
		alts := make([]Pat, 0, len(x.Alts))
		for i, a := range x.Alts {
			idb.Push(fmt.Sprintf("or.%d", i))
			alts = append(alts, b.lowerPat(modPath, idb, a, typ))
			idb.Pop()
		}
		if len(alts) < 2 {
			// An OrPat with fewer than two alternatives is malformed
			// syntax that W02 already rejects; defensive fallback.
			return alts[0]
		}
		return b.Builder.NewOrPat(idb.Child("orpat"), x.NodeSpan(), alts, typ)
	case *ast.RangePat:
		lo := b.lowerExpr(modPath, idb.Push("lo"), x.Lo, typ)
		idb.Pop()
		hi := b.lowerExpr(modPath, idb.Push("hi"), x.Hi, typ)
		idb.Pop()
		return b.Builder.NewRangePat(idb.Child("rangepat"), x.NodeSpan(), lo, hi, x.Inclusive, typ)
	case *ast.AtPat:
		inner := b.lowerPat(modPath, idb.Push("at"), x.Pattern, typ)
		idb.Pop()
		return b.Builder.NewAtBindPat(idb.Child("at:"+x.Name.Name), x.NodeSpan(), x.Name.Name, inner, typ)
	}
	// Unknown pattern form — compiler bug.
	panic(fmt.Sprintf("hir.Bridge.lowerPat: unhandled AST pattern %T", p))
}

func (b *Bridge) lowerCtorPat(modPath string, idb *IdBuilder, x *ast.CtorPat, typ typetable.TypeId) *ConstructorPat {
	path := make([]string, len(x.Path))
	for i, s := range x.Path {
		path[i] = s.Name
	}
	ctorType := typ
	variant := ""
	if sym, ok := b.Resolved.Bindings[resolve.SiteKey{Module: modPath, Span: x.NodeSpan()}]; ok {
		if tid, ok := b.typeOfSym[int(sym)]; ok {
			ctorType = tid
		}
		if s := b.Resolved.Symbols.Get(sym); s != nil && s.Kind == resolve.SymEnumVariant {
			variant = s.Name
		}
	}
	var tuple []Pat
	for i, sp := range x.Tuple {
		idb.Push(fmt.Sprintf("ctor.%d", i))
		tuple = append(tuple, b.lowerPat(modPath, idb, sp, b.Types.Infer()))
		idb.Pop()
	}
	var fields []*FieldPat
	for i, f := range x.Struct {
		var inner Pat
		if f.Pattern != nil {
			idb.Push(fmt.Sprintf("field.%d", i))
			inner = b.lowerPat(modPath, idb, f.Pattern, b.Types.Infer())
			idb.Pop()
		} else {
			inner = b.Builder.NewBindPat(idb.Child("shorthand:"+f.Name.Name), f.NodeSpan(), f.Name.Name, b.Types.Infer())
		}
		fields = append(fields, &FieldPat{
			Base:    Base{ID: idb.Child("f:" + f.Name.Name), Span: f.NodeSpan()},
			Name:    f.Name.Name,
			Pattern: inner,
		})
	}
	return b.Builder.NewConstructorPat(
		idb.Child("ctor:"+strings.Join(path, ".")),
		x.NodeSpan(),
		ctorType,
		variant,
		path,
		tuple,
		fields,
		x.HasRest,
		ctorType,
	)
}

// --- Small helpers --------------------------------------------------

func mapBinOp(op ast.BinaryOp) BinaryOp {
	// The order of constants in ast and hir matches intentionally;
	// the direct cast is safe.
	return BinaryOp(op)
}

func mapUnaryOp(op ast.UnaryOp) UnaryOp { return UnaryOp(op) }

func mapAssignOp(op ast.AssignOp) AssignOp { return AssignOp(op) }

func isBoolOp(op BinaryOp) bool {
	switch op {
	case BinEq, BinNe, BinLt, BinLe, BinGt, BinGe, BinLogAnd, BinLogOr:
		return true
	}
	return false
}

func isIntegerPrimitive(tab *typetable.Table, tid typetable.TypeId) bool {
	t := tab.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindI8, typetable.KindI16, typetable.KindI32, typetable.KindI64, typetable.KindISize,
		typetable.KindU8, typetable.KindU16, typetable.KindU32, typetable.KindU64, typetable.KindUSize:
		return true
	}
	return false
}

func isFloatPrimitive(tab *typetable.Table, tid typetable.TypeId) bool {
	t := tab.Get(tid)
	if t == nil {
		return false
	}
	return t.Kind == typetable.KindF32 || t.Kind == typetable.KindF64
}

// sortedModuleOrder returns the Resolved graph's module Order list
// as a clean copy so downstream iteration never modifies it in place.
// Kept here so passes outside the bridge have a single source of the
// deterministic iteration contract.
func sortedModuleOrder(r *resolve.Resolved) []string {
	out := make([]string, len(r.Graph.Order))
	copy(out, r.Graph.Order)
	sort.Strings(out)
	return out
}
