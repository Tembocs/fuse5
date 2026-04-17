package consteval

import (
	"fmt"
	"strings"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// CheckRestrictions walks every `const fn` in prog and reports any
// body construct that violates §46.1. The pass is separate from the
// evaluator so uncalled const fns still surface illegal shapes
// (defense in depth — the evaluator only visits called bodies).
//
// Diagnostics emitted:
//
//   - calling an FFI (extern) fn in a const body
//   - calling a non-const fn in a const body
//   - `spawn` expression in a const body
//   - `unsafe { ... }` block in a const body
//   - `&mut` reference in a const body (aliased mutation is not
//     deterministic at compile time)
//   - any interior-mutability type reference — recognized by the
//     built-in nominal name set (`Cell`, `RefCell`, `Atomic*`). W14
//     has no stdlib types yet; this list is future-proofed via
//     InteriorMutableNames.
//
// The pass returns diagnostics in source-order walk order so goldens
// are stable (Rule 7.1).
func CheckRestrictions(prog *hir.Program) []Diagnostic {
	c := &restrictChecker{prog: prog}
	// Build a symbol → FnDecl index so call-site resolution is O(1).
	for _, modPath := range prog.Order {
		mod := prog.Modules[modPath]
		for _, it := range mod.Items {
			c.indexItem(it)
		}
	}
	// Walk const fn bodies (free-standing and inside impl blocks).
	for _, modPath := range prog.Order {
		mod := prog.Modules[modPath]
		for _, it := range mod.Items {
			c.walkItem(it)
		}
	}
	return c.diags
}

// InteriorMutableNames is the set of nominal names whose reference
// in a const fn body is rejected by §46.1 as interior mutability.
// Exported so other packages (stdlib authors, future waves) can
// extend the list without forking this file.
var InteriorMutableNames = map[string]struct{}{
	"Cell":    {},
	"RefCell": {},
	"Mutex":   {},
	"RwLock":  {},
	// Atomic* families:
	"AtomicBool": {},
	"AtomicI8":   {},
	"AtomicI16":  {},
	"AtomicI32":  {},
	"AtomicI64":  {},
	"AtomicU8":   {},
	"AtomicU16":  {},
	"AtomicU32":  {},
	"AtomicU64":  {},
	"AtomicPtr":  {},
}

// AllocationFnNames is the closed list of allocator entry points
// rejected in const contexts. W14 has no heap-allocating stdlib; this
// list still matters because FFI headers or extern modules may
// introduce them under familiar names.
var AllocationFnNames = map[string]struct{}{
	"malloc":         {},
	"calloc":         {},
	"realloc":        {},
	"free":           {},
	"Box_new":        {},
	"Vec_with_cap":   {},
	"String_reserve": {},
}

// ThreadingFnNames is the closed list of threading/concurrency
// entry points rejected in const contexts.
var ThreadingFnNames = map[string]struct{}{
	"spawn":        {},
	"thread_start": {},
	"channel_send": {},
	"channel_recv": {},
}

type restrictChecker struct {
	prog    *hir.Program
	fnBySym map[int]*hir.FnDecl
	diags   []Diagnostic
}

func (c *restrictChecker) indexItem(it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		if c.fnBySym == nil {
			c.fnBySym = map[int]*hir.FnDecl{}
		}
		if x.SymID != 0 {
			c.fnBySym[x.SymID] = x
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			c.indexItem(sub)
		}
	}
}

func (c *restrictChecker) walkItem(it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		if x.IsConst && x.Body != nil {
			c.walkBlock(x.Body, x.Name)
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			c.walkItem(sub)
		}
	}
}

func (c *restrictChecker) walkBlock(b *hir.Block, fnName string) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		c.walkStmt(s, fnName)
	}
	if b.Trailing != nil {
		c.walkExpr(b.Trailing, fnName)
	}
}

func (c *restrictChecker) walkStmt(s hir.Stmt, fnName string) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.Value != nil {
			c.walkExpr(x.Value, fnName)
		}
	case *hir.VarStmt:
		if x.Value != nil {
			c.walkExpr(x.Value, fnName)
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			c.walkExpr(x.Value, fnName)
		}
	case *hir.BreakStmt:
		if x.Value != nil {
			c.walkExpr(x.Value, fnName)
		}
	case *hir.ExprStmt:
		c.walkExpr(x.Expr, fnName)
	}
}

func (c *restrictChecker) walkExpr(e hir.Expr, fnName string) {
	switch x := e.(type) {
	case *hir.SpawnExpr:
		c.emit(e.NodeSpan(),
			fmt.Sprintf("const fn %q cannot contain `spawn` (reference §46.1)", fnName),
			"remove the spawn or move it to a non-const fn")
		if x.Closure != nil && x.Closure.Body != nil {
			c.walkBlock(x.Closure.Body, fnName)
		}
	case *hir.UnsafeExpr:
		c.emit(e.NodeSpan(),
			fmt.Sprintf("const fn %q cannot contain an `unsafe` block (reference §46.1)", fnName),
			"remove the unsafe block or hoist the call out of the const fn")
		if x.Body != nil {
			c.walkBlock(x.Body, fnName)
		}
	case *hir.ReferenceExpr:
		if x.Mutable {
			c.emit(e.NodeSpan(),
				fmt.Sprintf("const fn %q cannot take `&mut` references (interior mutability is forbidden)", fnName),
				"drop the `mut` modifier or refactor to return a new value")
		}
		c.walkExpr(x.Inner, fnName)
	case *hir.CallExpr:
		c.checkCallTarget(x, fnName)
		if x.Callee != nil {
			c.walkExpr(x.Callee, fnName)
		}
		for _, a := range x.Args {
			c.walkExpr(a, fnName)
		}
	case *hir.PathExpr:
		c.checkPathName(x, fnName)
	case *hir.BinaryExpr:
		c.walkExpr(x.Lhs, fnName)
		c.walkExpr(x.Rhs, fnName)
	case *hir.UnaryExpr:
		c.walkExpr(x.Operand, fnName)
	case *hir.CastExpr:
		c.walkExpr(x.Expr, fnName)
	case *hir.IfExpr:
		c.walkExpr(x.Cond, fnName)
		c.walkBlock(x.Then, fnName)
		if x.Else != nil {
			c.walkExpr(x.Else, fnName)
		}
	case *hir.Block:
		c.walkBlock(x, fnName)
	case *hir.LoopExpr:
		c.walkBlock(x.Body, fnName)
	case *hir.WhileExpr:
		c.walkExpr(x.Cond, fnName)
		c.walkBlock(x.Body, fnName)
	case *hir.ForExpr:
		c.walkExpr(x.Iter, fnName)
		c.walkBlock(x.Body, fnName)
	case *hir.MatchExpr:
		c.walkExpr(x.Scrutinee, fnName)
		for _, arm := range x.Arms {
			if arm.Guard != nil {
				c.walkExpr(arm.Guard, fnName)
			}
			c.walkBlock(arm.Body, fnName)
		}
	case *hir.TupleExpr:
		for _, el := range x.Elements {
			c.walkExpr(el, fnName)
		}
	case *hir.StructLitExpr:
		if nm := structNameFromType(c.prog, x.StructType); isInteriorMutableName(nm) {
			c.emit(e.NodeSpan(),
				fmt.Sprintf("const fn %q cannot construct interior-mutable type %q (reference §46.1)", fnName, nm),
				"construct a plain value instead or defer the construction to a non-const context")
		}
		for _, f := range x.Fields {
			c.walkExpr(f.Value, fnName)
		}
		if x.Base != nil {
			c.walkExpr(x.Base, fnName)
		}
	case *hir.FieldExpr:
		c.walkExpr(x.Receiver, fnName)
	case *hir.OptFieldExpr:
		c.walkExpr(x.Receiver, fnName)
	case *hir.TryExpr:
		c.walkExpr(x.Receiver, fnName)
	case *hir.IndexExpr:
		c.walkExpr(x.Receiver, fnName)
		c.walkExpr(x.Index, fnName)
	case *hir.IndexRangeExpr:
		c.walkExpr(x.Receiver, fnName)
		if x.Low != nil {
			c.walkExpr(x.Low, fnName)
		}
		if x.High != nil {
			c.walkExpr(x.High, fnName)
		}
	case *hir.AssignExpr:
		// Assignment in const fns is legal only on locals the
		// evaluator introduced via var. The evaluator enforces
		// this dynamically; the restriction checker leaves it.
		c.walkExpr(x.Lhs, fnName)
		c.walkExpr(x.Rhs, fnName)
	case *hir.ClosureExpr:
		// Closures are allowed in a const fn body only when they
		// themselves never escape; the evaluator does not evaluate
		// them. Flag the construction as non-const.
		c.emit(e.NodeSpan(),
			fmt.Sprintf("const fn %q cannot construct closures", fnName),
			"hoist the closure into a named const fn or defer to a non-const context")
	}
}

// checkCallTarget emits a diagnostic when the callee is a non-const
// or extern fn. Intrinsics (size_of / align_of) bypass the check.
func (c *restrictChecker) checkCallTarget(call *hir.CallExpr, fnName string) {
	if _, ok := recognizeIntrinsic(call); ok {
		return
	}
	callee, ok := call.Callee.(*hir.PathExpr)
	if !ok {
		c.emit(call.NodeSpan(),
			fmt.Sprintf("const fn %q cannot use indirect call targets", fnName),
			"call a `const fn` through a direct path")
		return
	}
	last := ""
	if len(callee.Segments) > 0 {
		last = callee.Segments[len(callee.Segments)-1]
	}
	if _, bad := AllocationFnNames[last]; bad {
		c.emit(call.NodeSpan(),
			fmt.Sprintf("const fn %q cannot call allocator %q (reference §46.1)", fnName, last),
			"remove the allocation or defer to a non-const fn")
		return
	}
	if _, bad := ThreadingFnNames[last]; bad {
		c.emit(call.NodeSpan(),
			fmt.Sprintf("const fn %q cannot call threading primitive %q (reference §46.1)", fnName, last),
			"remove the threading call or defer to a non-const fn")
		return
	}
	if callee.Symbol == 0 {
		return
	}
	target := c.fnBySym[callee.Symbol]
	if target == nil {
		return
	}
	if target.IsExtern {
		c.emit(call.NodeSpan(),
			fmt.Sprintf("const fn %q cannot call FFI fn %q (reference §46.1)", fnName, target.Name),
			"const fns cannot call extern declarations")
		return
	}
	if !target.IsConst {
		c.emit(call.NodeSpan(),
			fmt.Sprintf("const fn %q cannot call non-const fn %q (reference §46.1)", fnName, target.Name),
			fmt.Sprintf("declare %q as `const fn` or inline its body", target.Name))
	}
}

// checkPathName flags any path whose last segment names an interior
// mutability primitive. Conservative — the checker cannot fully
// resolve type identity at HIR boundary, so it blocks by name.
func (c *restrictChecker) checkPathName(p *hir.PathExpr, fnName string) {
	if len(p.Segments) == 0 {
		return
	}
	last := p.Segments[len(p.Segments)-1]
	if isInteriorMutableName(last) {
		c.emit(p.NodeSpan(),
			fmt.Sprintf("const fn %q cannot reference interior-mutable type %q (reference §46.1)", fnName, last),
			"use a plain value; interior mutability breaks determinism")
	}
}

func (c *restrictChecker) emit(span lex.Span, msg, hint string) {
	c.diags = append(c.diags, Diagnostic{Span: span, Message: msg, Hint: hint})
}

func isInteriorMutableName(name string) bool {
	if _, ok := InteriorMutableNames[name]; ok {
		return true
	}
	// Any name starting with `Atomic` is treated as interior-mutable.
	return strings.HasPrefix(name, "Atomic")
}

// structNameFromType pulls the declared name for a nominal struct
// TypeId out of the program's HIR. Returns empty string when the
// TypeId is not a nominal struct in this program.
func structNameFromType(prog *hir.Program, t typetable.TypeId) string {
	info := prog.Types.Get(t)
	if info == nil {
		return ""
	}
	return info.Name
}
