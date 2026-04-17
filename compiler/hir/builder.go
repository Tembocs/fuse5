package hir

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Builders allocate HIR nodes with their required metadata populated.
// A builder method panics if a required invariant is violated —
// missing NodeID, missing Type when Type is mandatory, nil child
// where the grammar requires non-nil. These panics are compiler bugs,
// never user errors (the bridge validates user input before it gets
// here).
//
// Invariants enforced:
//
//   - Every Node has a non-empty NodeID. EmptyNodeID is rejected.
//   - Every Typed node has a non-NoType TypeId. If the bridge can't
//     determine the type, it must pass the explicit KindInfer TypeId
//     (table.Infer()), not NoType (L013 defense).
//   - Structural children that the grammar makes non-optional (e.g.
//     IfExpr.Then, WhileExpr.Body) are non-nil.
//
// Builders are therefore the single forcing function for W04's
// TestBuilderEnforcement and TestMetadataFields tests.

// Builder constructs HIR nodes. Use one Builder per Program so NodeID
// generation is anchored against a known module/item context.
type Builder struct {
	Types *typetable.Table
}

// NewBuilder returns a Builder bound to tab. A Program can share one
// Builder across all of its modules.
func NewBuilder(tab *typetable.Table) *Builder {
	if tab == nil {
		panic("hir.NewBuilder: nil TypeTable")
	}
	return &Builder{Types: tab}
}

// --- Item builders ----------------------------------------------------

// NewFn constructs a FnDecl. typeID must be a KindFn TypeId; the
// builder validates it to catch bridge bugs early.
func (b *Builder) NewFn(id NodeID, span lex.Span, name string, typeID typetable.TypeId, params []*Param, ret typetable.TypeId, body *Block) *FnDecl {
	b.requireID(id, "FnDecl")
	b.requireName(name, "FnDecl")
	b.requireKind(typeID, typetable.KindFn, "FnDecl.TypeID")
	if ret == typetable.NoType {
		panic("hir.NewFn: return TypeId is NoType (use Unit for no return)")
	}
	return &FnDecl{
		Base:   Base{ID: id, Span: span},
		Name:   name,
		TypeID: typeID,
		Params: params,
		Return: ret,
		Body:   body,
	}
}

// NewStruct constructs a StructDecl. typeID must be a KindStruct
// TypeId pointing at this struct's nominal identity.
func (b *Builder) NewStruct(id NodeID, span lex.Span, name string, typeID typetable.TypeId, fields []*Field, tuple []typetable.TypeId, isUnit bool) *StructDecl {
	b.requireID(id, "StructDecl")
	b.requireName(name, "StructDecl")
	b.requireKind(typeID, typetable.KindStruct, "StructDecl.TypeID")
	return &StructDecl{
		Base:        Base{ID: id, Span: span},
		Name:        name,
		TypeID:      typeID,
		Fields:      fields,
		TupleFields: tuple,
		IsUnit:      isUnit,
	}
}

// NewEnum constructs an EnumDecl.
func (b *Builder) NewEnum(id NodeID, span lex.Span, name string, typeID typetable.TypeId, variants []*Variant) *EnumDecl {
	b.requireID(id, "EnumDecl")
	b.requireName(name, "EnumDecl")
	b.requireKind(typeID, typetable.KindEnum, "EnumDecl.TypeID")
	return &EnumDecl{
		Base:     Base{ID: id, Span: span},
		Name:     name,
		TypeID:   typeID,
		Variants: variants,
	}
}

// NewField constructs a Field (struct member).
func (b *Builder) NewField(id NodeID, span lex.Span, name string, typ typetable.TypeId) *Field {
	b.requireID(id, "Field")
	b.requireName(name, "Field")
	b.requireType(typ, "Field")
	return &Field{
		TypedBase: TypedBase{
			Base: Base{ID: id, Span: span},
			Type: typ,
		},
		Name: name,
	}
}

// NewParam constructs a parameter node.
func (b *Builder) NewParam(id NodeID, span lex.Span, name string, typ typetable.TypeId, ownership Ownership) *Param {
	b.requireID(id, "Param")
	b.requireName(name, "Param")
	b.requireType(typ, "Param")
	return &Param{
		TypedBase: TypedBase{
			Base: Base{ID: id, Span: span},
			Type: typ,
		},
		Name:      name,
		Ownership: ownership,
	}
}

// --- Expression builders ---------------------------------------------

// NewLiteral constructs a LiteralExpr.
func (b *Builder) NewLiteral(id NodeID, span lex.Span, kind LitKind, text string, typ typetable.TypeId) *LiteralExpr {
	b.requireID(id, "LiteralExpr")
	b.requireType(typ, "LiteralExpr")
	return &LiteralExpr{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Kind:      kind,
		Text:      text,
	}
}

// NewPath constructs a resolved PathExpr.
func (b *Builder) NewPath(id NodeID, span lex.Span, symbol int, segments []string, typ typetable.TypeId) *PathExpr {
	b.requireID(id, "PathExpr")
	b.requireType(typ, "PathExpr")
	if len(segments) == 0 {
		panic("hir.NewPath: segments must not be empty")
	}
	return &PathExpr{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Symbol:    symbol,
		Segments:  segments,
	}
}

// NewBinary constructs a BinaryExpr.
func (b *Builder) NewBinary(id NodeID, span lex.Span, op BinaryOp, lhs, rhs Expr, typ typetable.TypeId) *BinaryExpr {
	b.requireID(id, "BinaryExpr")
	b.requireType(typ, "BinaryExpr")
	b.requireExpr(lhs, "BinaryExpr.Lhs")
	b.requireExpr(rhs, "BinaryExpr.Rhs")
	return &BinaryExpr{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Op:        op,
		Lhs:       lhs,
		Rhs:       rhs,
	}
}

// NewCall constructs a CallExpr.
func (b *Builder) NewCall(id NodeID, span lex.Span, callee Expr, args []Expr, typ typetable.TypeId) *CallExpr {
	b.requireID(id, "CallExpr")
	b.requireType(typ, "CallExpr")
	b.requireExpr(callee, "CallExpr.Callee")
	return &CallExpr{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Callee:    callee,
		Args:      args,
	}
}

// NewBlock constructs a Block.
func (b *Builder) NewBlock(id NodeID, span lex.Span, stmts []Stmt, trailing Expr, typ typetable.TypeId) *Block {
	b.requireID(id, "Block")
	b.requireType(typ, "Block")
	return &Block{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Stmts:     stmts,
		Trailing:  trailing,
	}
}

// NewIf constructs an IfExpr.
func (b *Builder) NewIf(id NodeID, span lex.Span, cond Expr, then *Block, els Expr, typ typetable.TypeId) *IfExpr {
	b.requireID(id, "IfExpr")
	b.requireType(typ, "IfExpr")
	b.requireExpr(cond, "IfExpr.Cond")
	if then == nil {
		panic("hir.NewIf: Then block is nil")
	}
	return &IfExpr{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Cond:      cond,
		Then:      then,
		Else:      els,
	}
}

// NewMatch constructs a MatchExpr.
func (b *Builder) NewMatch(id NodeID, span lex.Span, scrut Expr, arms []*MatchArm, typ typetable.TypeId) *MatchExpr {
	b.requireID(id, "MatchExpr")
	b.requireType(typ, "MatchExpr")
	b.requireExpr(scrut, "MatchExpr.Scrutinee")
	for i, arm := range arms {
		if arm == nil || arm.Pattern == nil || arm.Body == nil {
			panic(fmt.Sprintf("hir.NewMatch: arm %d missing pattern or body", i))
		}
	}
	return &MatchExpr{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Scrutinee: scrut,
		Arms:      arms,
	}
}

// --- Pattern builders ------------------------------------------------

// NewLiteralPat constructs a LiteralPat.
func (b *Builder) NewLiteralPat(id NodeID, span lex.Span, kind LitKind, text string, typ typetable.TypeId) *LiteralPat {
	b.requireID(id, "LiteralPat")
	b.requireType(typ, "LiteralPat")
	return &LiteralPat{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Kind:      kind,
		Text:      text,
	}
}

// NewBindPat constructs a BindPat.
func (b *Builder) NewBindPat(id NodeID, span lex.Span, name string, typ typetable.TypeId) *BindPat {
	b.requireID(id, "BindPat")
	b.requireName(name, "BindPat")
	b.requireType(typ, "BindPat")
	return &BindPat{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Name:      name,
	}
}

// NewWildcardPat constructs a WildcardPat. Wildcards still carry a
// TypeId — normally the scrutinee's type — so exhaustiveness
// checking in W10 can reason about them.
func (b *Builder) NewWildcardPat(id NodeID, span lex.Span, typ typetable.TypeId) *WildcardPat {
	b.requireID(id, "WildcardPat")
	b.requireType(typ, "WildcardPat")
	return &WildcardPat{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
	}
}

// NewConstructorPat constructs a ConstructorPat.
func (b *Builder) NewConstructorPat(id NodeID, span lex.Span, ctorType typetable.TypeId, variant string, path []string, tuple []Pat, fields []*FieldPat, hasRest bool, typ typetable.TypeId) *ConstructorPat {
	b.requireID(id, "ConstructorPat")
	b.requireType(typ, "ConstructorPat")
	if ctorType == typetable.NoType {
		panic("hir.NewConstructorPat: ConstructorType is NoType")
	}
	if len(path) == 0 {
		panic("hir.NewConstructorPat: path must not be empty")
	}
	return &ConstructorPat{
		TypedBase:       TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		ConstructorType: ctorType,
		VariantName:     variant,
		Path:            path,
		Tuple:           tuple,
		Fields:          fields,
		HasRest:         hasRest,
	}
}

// NewOrPat constructs an OrPat.
func (b *Builder) NewOrPat(id NodeID, span lex.Span, alts []Pat, typ typetable.TypeId) *OrPat {
	b.requireID(id, "OrPat")
	b.requireType(typ, "OrPat")
	if len(alts) < 2 {
		panic("hir.NewOrPat: OrPat requires at least 2 alternatives")
	}
	return &OrPat{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Alts:      alts,
	}
}

// NewRangePat constructs a RangePat.
func (b *Builder) NewRangePat(id NodeID, span lex.Span, lo, hi Expr, inclusive bool, typ typetable.TypeId) *RangePat {
	b.requireID(id, "RangePat")
	b.requireType(typ, "RangePat")
	if lo == nil || hi == nil {
		panic("hir.NewRangePat: both bounds are required")
	}
	return &RangePat{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Lo:        lo,
		Hi:        hi,
		Inclusive: inclusive,
	}
}

// NewAtBindPat constructs an AtBindPat.
func (b *Builder) NewAtBindPat(id NodeID, span lex.Span, name string, inner Pat, typ typetable.TypeId) *AtBindPat {
	b.requireID(id, "AtBindPat")
	b.requireName(name, "AtBindPat")
	b.requireType(typ, "AtBindPat")
	if inner == nil {
		panic("hir.NewAtBindPat: inner pattern is required")
	}
	return &AtBindPat{
		TypedBase: TypedBase{Base: Base{ID: id, Span: span}, Type: typ},
		Name:      name,
		Pattern:   inner,
	}
}

// --- Statement builders ----------------------------------------------

// NewLet constructs a LetStmt. declared may be table.Infer() when the
// source omitted an annotation — but never NoType (L013).
func (b *Builder) NewLet(id NodeID, span lex.Span, pat Pat, declared typetable.TypeId, value Expr) *LetStmt {
	b.requireID(id, "LetStmt")
	b.requireType(declared, "LetStmt.DeclaredType")
	if pat == nil {
		panic("hir.NewLet: pattern is required")
	}
	return &LetStmt{
		Base:         Base{ID: id, Span: span},
		Pattern:      pat,
		DeclaredType: declared,
		Value:        value,
	}
}

// NewExprStmt wraps an expression in ExprStmt.
func (b *Builder) NewExprStmt(id NodeID, span lex.Span, e Expr) *ExprStmt {
	b.requireID(id, "ExprStmt")
	b.requireExpr(e, "ExprStmt")
	return &ExprStmt{
		Base: Base{ID: id, Span: span},
		Expr: e,
	}
}

// NewReturn constructs a ReturnStmt.
func (b *Builder) NewReturn(id NodeID, span lex.Span, value Expr) *ReturnStmt {
	b.requireID(id, "ReturnStmt")
	return &ReturnStmt{
		Base:  Base{ID: id, Span: span},
		Value: value,
	}
}

// --- Invariant helpers ------------------------------------------------

func (b *Builder) requireID(id NodeID, what string) {
	if id == EmptyNodeID {
		panic(fmt.Sprintf("hir.Builder: %s constructed with empty NodeID", what))
	}
}

func (b *Builder) requireName(name, what string) {
	if name == "" {
		panic(fmt.Sprintf("hir.Builder: %s constructed with empty Name", what))
	}
}

func (b *Builder) requireType(t typetable.TypeId, what string) {
	if t == typetable.NoType {
		panic(fmt.Sprintf("hir.Builder: %s constructed with NoType (use table.Infer() explicitly if inference is pending)", what))
	}
}

func (b *Builder) requireKind(t typetable.TypeId, want typetable.Kind, what string) {
	got := b.Types.Get(t)
	if got == nil {
		panic(fmt.Sprintf("hir.Builder: %s TypeId %d not in table", what, t))
	}
	if got.Kind != want {
		panic(fmt.Sprintf("hir.Builder: %s expected %s, got %s", what, want, got.Kind))
	}
}

func (b *Builder) requireExpr(e Expr, what string) {
	if e == nil {
		panic(fmt.Sprintf("hir.Builder: %s is nil", what))
	}
}
