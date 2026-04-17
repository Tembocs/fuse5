package check

import (
	"fmt"
	"strconv"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// checkExpr types a single expression under an expected-type
// context and mutates the expression's TypedBase.Type to the
// resolved TypeId. Returns that TypeId so callers can use it
// directly without re-reading the node.
//
// The `expected` parameter is the context's desired type. Pass
// typetable.NoType when there is no context (e.g., top-level
// expression statements); literal types default to I32/F64 in
// that case.
func (c *checker) checkExpr(modPath string, scope *bodyScope, e hir.Expr, expected typetable.TypeId) typetable.TypeId {
	if e == nil {
		return typetable.NoType
	}
	c.stats.ExprsTyped++
	result := c.inferExpr(modPath, scope, e, expected)
	// Mutate the expression's TypeId in place. Every concrete
	// expression type ultimately embeds TypedBase, so we can set
	// it uniformly via the helper below.
	c.setExprType(e, result)
	return result
}

// inferExpr dispatches per-expression-kind and returns the
// concrete TypeId, without mutating the expression. checkExpr is
// the surface entry that writes the result back.
func (c *checker) inferExpr(modPath string, scope *bodyScope, e hir.Expr, expected typetable.TypeId) typetable.TypeId {
	switch x := e.(type) {
	case *hir.LiteralExpr:
		return c.inferLiteral(x, expected)
	case *hir.PathExpr:
		return c.inferPath(modPath, scope, x, expected)
	case *hir.BinaryExpr:
		return c.inferBinary(modPath, scope, x, expected)
	case *hir.UnaryExpr:
		return c.inferUnary(modPath, scope, x, expected)
	case *hir.AssignExpr:
		c.checkExpr(modPath, scope, x.Lhs, typetable.NoType)
		c.checkExpr(modPath, scope, x.Rhs, x.Lhs.TypeOf())
		return c.tab.Unit()
	case *hir.CastExpr:
		return c.inferCast(modPath, scope, x)
	case *hir.CallExpr:
		return c.inferCall(modPath, scope, x, expected)
	case *hir.Block:
		return c.checkBlock(modPath, scope, x, expected)
	case *hir.IfExpr:
		return c.inferIf(modPath, scope, x, expected)
	case *hir.FieldExpr:
		c.checkExpr(modPath, scope, x.Receiver, typetable.NoType)
		return c.tab.Infer()
	case *hir.TupleExpr:
		elems := make([]typetable.TypeId, 0, len(x.Elements))
		for _, el := range x.Elements {
			elems = append(elems, c.checkExpr(modPath, scope, el, typetable.NoType))
		}
		return c.tab.Tuple(elems)
	case *hir.StructLitExpr:
		return c.inferStructLit(modPath, scope, x, expected)
	case *hir.ReferenceExpr:
		inner := c.checkExpr(modPath, scope, x.Inner, typetable.NoType)
		if x.Mutable {
			return c.tab.Mutref(inner)
		}
		return c.tab.Ref(inner)
	case *hir.UnsafeExpr:
		return c.checkBlock(modPath, scope, x.Body, expected)
	}
	// Fallback for expressions W06 does not yet type (match, loops,
	// closures, spawn, try, index-range). The downstream wave that
	// lowers them is responsible for the actual typing; leaving
	// them as Infer here is the explicit "W06 doesn't touch this"
	// signal, not an Unknown default.
	return c.tab.Infer()
}

// inferLiteral picks the concrete TypeId for a literal. Integer
// and float literals honor the `expected` hint when it is a
// primitive of the matching family; otherwise defaults apply
// (I32 for int, F64 for float).
func (c *checker) inferLiteral(x *hir.LiteralExpr, expected typetable.TypeId) typetable.TypeId {
	switch x.Kind {
	case hir.LitBool:
		return c.tab.Bool()
	case hir.LitChar:
		return c.tab.Char()
	case hir.LitString, hir.LitRawString:
		return c.tab.String_()
	case hir.LitCString:
		return c.tab.CStr()
	case hir.LitInt:
		if expected != typetable.NoType && isIntegerTypeId(c.tab, expected) {
			// Range-check the literal's value against the target.
			if v, err := strconv.ParseInt(x.Text, 0, 64); err == nil {
				if !literalFitsInteger(v, expected, c.tab) {
					c.diagnose(x.Span,
						fmt.Sprintf("integer literal %s does not fit in %s", x.Text, c.typeName(expected)),
						"choose a wider integer type or adjust the literal")
				}
			}
			return expected
		}
		return c.tab.I32()
	case hir.LitFloat:
		if expected != typetable.NoType && isFloatTypeId(c.tab, expected) {
			return expected
		}
		return c.tab.F64()
	case hir.LitNone:
		// `None` is Option-typed at W11; at W06 fall back to Infer
		// so the later wave can resolve it.
		return c.tab.Infer()
	}
	return c.tab.Infer()
}

// inferPath resolves a PathExpr. Single-segment paths are looked
// up in the body scope (parameters, locals), then in the fn-sig
// table, then in primitive names. Multi-segment paths are
// already resolved by the W03 resolver (via PathExpr.Symbol); we
// pull the type of that symbol out of the Program.ItemTypes map.
func (c *checker) inferPath(modPath string, scope *bodyScope, x *hir.PathExpr, expected typetable.TypeId) typetable.TypeId {
	if len(x.Segments) == 1 {
		name := x.Segments[0]
		if name == "self" {
			if scope.selfT != typetable.NoType {
				return scope.selfT
			}
		}
		if t, ok := scope.lookup(name); ok {
			return t
		}
	}
	if x.Symbol != 0 {
		if tid := c.prog.ItemType(x.Symbol); tid != typetable.NoType {
			return tid
		}
	}
	// Unresolved identifier within a body is a type error unless
	// it is clearly a local being seen for the first time. At W06
	// we take a conservative stance: emit a diagnostic.
	c.diagnose(x.Span,
		fmt.Sprintf("unresolved name %q in type-checking", dottedPath(x.Segments)),
		"check that the name is in scope and spelled correctly")
	_ = expected
	return c.tab.Infer()
}

// inferBinary types a binary expression. The checker implements
// the common rules from reference §5.5:
//
//   - Arithmetic operators require numeric operands of the same
//     type; the result type matches the operand type.
//   - Comparison operators (`==`, `!=`, `<`, `<=`, `>`, `>=`)
//     require equal operand types and produce Bool.
//   - Logical operators (`&&`, `||`) require Bool operands and
//     produce Bool.
//   - Bitwise/shift operators require integer operands.
func (c *checker) inferBinary(modPath string, scope *bodyScope, x *hir.BinaryExpr, expected typetable.TypeId) typetable.TypeId {
	switch x.Op {
	case hir.BinLogAnd, hir.BinLogOr:
		c.checkExpr(modPath, scope, x.Lhs, c.tab.Bool())
		c.checkExpr(modPath, scope, x.Rhs, c.tab.Bool())
		return c.tab.Bool()
	case hir.BinEq, hir.BinNe, hir.BinLt, hir.BinLe, hir.BinGt, hir.BinGe:
		lhs := c.checkExpr(modPath, scope, x.Lhs, typetable.NoType)
		c.checkExpr(modPath, scope, x.Rhs, lhs)
		return c.tab.Bool()
	}
	// Arithmetic / bitwise / shift — honor the expected hint when
	// it is a numeric primitive so integer literal defaults align.
	hint := expected
	if !isIntegerTypeId(c.tab, hint) && !isFloatTypeId(c.tab, hint) {
		hint = typetable.NoType
	}
	lhs := c.checkExpr(modPath, scope, x.Lhs, hint)
	rhs := c.checkExpr(modPath, scope, x.Rhs, lhs)
	if lhs != rhs && !c.isWidenable(rhs, lhs) && !c.isWidenable(lhs, rhs) {
		c.diagnose(x.Span,
			fmt.Sprintf("mismatched arithmetic operands: %s vs %s",
				c.typeName(lhs), c.typeName(rhs)),
			"use an explicit `as` cast to widen the narrower side")
	}
	return lhs
}

// inferUnary types a unary expression.
func (c *checker) inferUnary(modPath string, scope *bodyScope, x *hir.UnaryExpr, expected typetable.TypeId) typetable.TypeId {
	switch x.Op {
	case hir.UnNot:
		c.checkExpr(modPath, scope, x.Operand, c.tab.Bool())
		return c.tab.Bool()
	case hir.UnNeg:
		return c.checkExpr(modPath, scope, x.Operand, expected)
	case hir.UnDeref:
		inner := c.checkExpr(modPath, scope, x.Operand, typetable.NoType)
		if t := c.tab.Get(inner); t != nil && (t.Kind == typetable.KindRef || t.Kind == typetable.KindMutref || t.Kind == typetable.KindPtr) {
			if len(t.Children) > 0 {
				return t.Children[0]
			}
		}
		return c.tab.Infer()
	case hir.UnAddr:
		inner := c.checkExpr(modPath, scope, x.Operand, typetable.NoType)
		return c.tab.Ref(inner)
	}
	return c.tab.Infer()
}

// inferCast types an `as` cast. The checker permits primitive-to-
// primitive numeric casts and pointer-to-pointer casts. Anything
// else is diagnosed so wrong casts don't reach codegen (reference
// §28.1).
func (c *checker) inferCast(modPath string, scope *bodyScope, x *hir.CastExpr) typetable.TypeId {
	src := c.checkExpr(modPath, scope, x.Expr, typetable.NoType)
	dst := x.TypeOf()
	if dst == typetable.NoType || dst == c.tab.Infer() {
		c.diagnose(x.Span, "cast target type is unresolved",
			"write the target type explicitly: `expr as T`")
		return c.tab.Infer()
	}
	// Accept: primitive numeric ↔ primitive numeric; ptr ↔ ptr.
	if isNumericKind(c.tab, src) && isNumericKind(c.tab, dst) {
		return dst
	}
	if isPtrKind(c.tab, src) && isPtrKind(c.tab, dst) {
		return dst
	}
	if isIntegerTypeId(c.tab, src) && isPtrKind(c.tab, dst) {
		return dst
	}
	if isPtrKind(c.tab, src) && isIntegerTypeId(c.tab, dst) {
		return dst
	}
	c.diagnose(x.Span,
		fmt.Sprintf("invalid cast: %s as %s", c.typeName(src), c.typeName(dst)),
		"casts are limited to numeric↔numeric, pointer↔pointer, and integer↔pointer")
	return dst
}

// inferCall types a call expression. The callee must be a
// function type; arguments are checked against the parameter
// types; the result is the return type.
func (c *checker) inferCall(modPath string, scope *bodyScope, x *hir.CallExpr, expected typetable.TypeId) typetable.TypeId {
	calleeType := c.checkExpr(modPath, scope, x.Callee, typetable.NoType)
	ft := c.tab.Get(calleeType)
	if ft == nil || ft.Kind != typetable.KindFn {
		// May still be under inference (the bridge sometimes leaves
		// callees as Infer when their symbol is not registered). We
		// emit a diagnostic only when the type is clearly wrong.
		if calleeType != c.tab.Infer() && calleeType != typetable.NoType {
			c.diagnose(x.Span,
				fmt.Sprintf("cannot call value of type %s", c.typeName(calleeType)),
				"the call target must be a function")
		}
		for _, a := range x.Args {
			c.checkExpr(modPath, scope, a, typetable.NoType)
		}
		return c.tab.Infer()
	}
	if !ft.IsVariadic && len(x.Args) != len(ft.Children) {
		c.diagnose(x.Span,
			fmt.Sprintf("arity mismatch: fn takes %d arg(s), got %d", len(ft.Children), len(x.Args)),
			"check the call matches the fn's declared parameters")
	}
	for i, a := range x.Args {
		var paramT typetable.TypeId
		if i < len(ft.Children) {
			paramT = ft.Children[i]
		}
		c.checkExpr(modPath, scope, a, paramT)
	}
	_ = expected
	return ft.Return
}

// inferIf types an if expression. Both arms must produce the same
// type (at W06 we require exact match; full unification is W10
// work when match arms need it).
func (c *checker) inferIf(modPath string, scope *bodyScope, x *hir.IfExpr, expected typetable.TypeId) typetable.TypeId {
	c.checkExpr(modPath, scope, x.Cond, c.tab.Bool())
	thenT := c.checkBlock(modPath, scope, x.Then, expected)
	if x.Else == nil {
		if thenT != c.tab.Unit() {
			c.diagnose(x.Span,
				"`if` without `else` must produce Unit",
				"add an `else` branch that yields the same type")
		}
		return c.tab.Unit()
	}
	elseT := c.checkExpr(modPath, scope, x.Else, thenT)
	if thenT != elseT && !c.isWidenable(elseT, thenT) && !c.isWidenable(thenT, elseT) {
		c.diagnose(x.Span,
			fmt.Sprintf("if arms produce different types: %s vs %s",
				c.typeName(thenT), c.typeName(elseT)),
			"make both branches yield the same type")
	}
	return thenT
}

// inferStructLit types a struct literal. The struct's nominal
// TypeId comes from the bridge via StructLitExpr.StructType; we
// check that each field matches its declared type by consulting
// the struct's declaration in the Program.
func (c *checker) inferStructLit(modPath string, scope *bodyScope, x *hir.StructLitExpr, expected typetable.TypeId) typetable.TypeId {
	structT := x.StructType
	if structT == c.tab.Infer() || structT == typetable.NoType {
		if expected != typetable.NoType {
			structT = expected
			x.StructType = expected
		}
	}
	// Look up the struct declaration to validate fields.
	decl := c.structDeclForType(structT)
	if decl == nil {
		for _, f := range x.Fields {
			if f.Value != nil {
				c.checkExpr(modPath, scope, f.Value, typetable.NoType)
			}
		}
		return structT
	}
	expected2 := map[string]typetable.TypeId{}
	for _, f := range decl.Fields {
		expected2[f.Name] = f.TypeOf()
	}
	for _, f := range x.Fields {
		want, ok := expected2[f.Name]
		if !ok {
			c.diagnose(f.Span,
				fmt.Sprintf("struct %s has no field %q", c.typeName(structT), f.Name),
				"check the field name against the struct declaration")
			continue
		}
		got := c.checkExpr(modPath, scope, f.Value, want)
		if !c.isAssignable(got, want) {
			c.diagnose(f.Span,
				fmt.Sprintf("field %q: expected %s, got %s",
					f.Name, c.typeName(want), c.typeName(got)),
				"use an explicit `as` cast if a widening is intended")
		}
	}
	return structT
}

// structDeclForType finds the HIR StructDecl for a nominal
// struct TypeId. Returns nil for non-struct TypeIds.
func (c *checker) structDeclForType(tid typetable.TypeId) *hir.StructDecl {
	for _, modPath := range c.prog.Order {
		m := c.prog.Modules[modPath]
		for _, it := range m.Items {
			if sd, ok := it.(*hir.StructDecl); ok && sd.TypeID == tid {
				return sd
			}
		}
	}
	return nil
}

// setExprType writes a resolved TypeId back onto the expression
// node. Every Expr implementation embeds TypedBase, so a simple
// type switch covers the set without reflection.
func (c *checker) setExprType(e hir.Expr, tid typetable.TypeId) {
	if tid == typetable.NoType {
		return
	}
	switch x := e.(type) {
	case *hir.LiteralExpr:
		c.assign(&x.Type, tid)
	case *hir.PathExpr:
		c.assign(&x.Type, tid)
	case *hir.BinaryExpr:
		c.assign(&x.Type, tid)
	case *hir.AssignExpr:
		c.assign(&x.Type, tid)
	case *hir.UnaryExpr:
		c.assign(&x.Type, tid)
	case *hir.CastExpr:
		c.assign(&x.Type, tid)
	case *hir.CallExpr:
		c.assign(&x.Type, tid)
	case *hir.FieldExpr:
		c.assign(&x.Type, tid)
	case *hir.OptFieldExpr:
		c.assign(&x.Type, tid)
	case *hir.TryExpr:
		c.assign(&x.Type, tid)
	case *hir.IndexExpr:
		c.assign(&x.Type, tid)
	case *hir.IndexRangeExpr:
		c.assign(&x.Type, tid)
	case *hir.Block:
		c.assign(&x.Type, tid)
	case *hir.IfExpr:
		c.assign(&x.Type, tid)
	case *hir.MatchExpr:
		c.assign(&x.Type, tid)
	case *hir.LoopExpr:
		c.assign(&x.Type, tid)
	case *hir.WhileExpr:
		c.assign(&x.Type, tid)
	case *hir.ForExpr:
		c.assign(&x.Type, tid)
	case *hir.TupleExpr:
		c.assign(&x.Type, tid)
	case *hir.StructLitExpr:
		c.assign(&x.Type, tid)
	case *hir.ClosureExpr:
		c.assign(&x.Type, tid)
	case *hir.SpawnExpr:
		c.assign(&x.Type, tid)
	case *hir.UnsafeExpr:
		c.assign(&x.Type, tid)
	case *hir.ReferenceExpr:
		c.assign(&x.Type, tid)
	}
}

// assign updates a TypedBase.Type slot, tracking InferResolved.
func (c *checker) assign(slot *typetable.TypeId, tid typetable.TypeId) {
	if *slot == c.tab.Infer() && tid != c.tab.Infer() {
		c.stats.InferResolved++
	}
	*slot = tid
}

// --- Small type-predicate helpers ----------------------------------

func isIntegerTypeId(tab *typetable.Table, tid typetable.TypeId) bool {
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

func isFloatTypeId(tab *typetable.Table, tid typetable.TypeId) bool {
	t := tab.Get(tid)
	return t != nil && (t.Kind == typetable.KindF32 || t.Kind == typetable.KindF64)
}

func isNumericKind(tab *typetable.Table, tid typetable.TypeId) bool {
	return isIntegerTypeId(tab, tid) || isFloatTypeId(tab, tid)
}

func isPtrKind(tab *typetable.Table, tid typetable.TypeId) bool {
	t := tab.Get(tid)
	return t != nil && t.Kind == typetable.KindPtr
}

// literalFitsInteger checks whether a parsed int64 literal value
// fits in the target integer TypeId's value range. Used by
// inferLiteral to range-check `let x: I8 = 300` and similar.
func literalFitsInteger(v int64, tid typetable.TypeId, tab *typetable.Table) bool {
	t := tab.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindI8:
		return v >= -1<<7 && v < 1<<7
	case typetable.KindI16:
		return v >= -1<<15 && v < 1<<15
	case typetable.KindI32:
		return v >= -1<<31 && v < 1<<31
	case typetable.KindI64, typetable.KindISize:
		return true
	case typetable.KindU8:
		return v >= 0 && v < 1<<8
	case typetable.KindU16:
		return v >= 0 && v < 1<<16
	case typetable.KindU32:
		return v >= 0 && v < 1<<32
	case typetable.KindU64, typetable.KindUSize:
		return v >= 0
	}
	return true
}

// dottedPath joins a path's segments with dots for diagnostics.
func dottedPath(segs []string) string {
	out := ""
	for i, s := range segs {
		if i > 0 {
			out += "."
		}
		out += s
	}
	return out
}
