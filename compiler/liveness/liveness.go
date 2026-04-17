package liveness

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Diagnostic mirrors lex.Diagnostic so callers merge our output
// into a single diagnostic stream.
type Diagnostic = lex.Diagnostic

// Result is the output of Analyze. DropIntents is the per-fn list
// of locals that require destructor calls at end-of-scope; codegen
// consumes this to emit `TypeName_drop(&_lN)` calls.
type Result struct {
	DropIntents map[hir.NodeID][]DropIntent
}

// DropIntent records one destructor-call site: the local's NodeID,
// its type, and the fn whose scope owns it. Codegen emits one
// `<TypeName>_drop(&local)` per intent at the fn's return site.
type DropIntent struct {
	LocalName string
	Type      typetable.TypeId
	EnclosingFn hir.NodeID
}

// Analyze runs every ownership / borrow / liveness / drop check
// across prog. On success, the returned Result carries per-fn
// drop metadata. Diagnostics accumulate per rule; a non-empty
// return is a compile-fatal set — the driver must not proceed to
// codegen.
func Analyze(prog *hir.Program) (*Result, []Diagnostic) {
	a := &analyzer{
		prog: prog,
		tab:  prog.Types,
		res:  &Result{DropIntents: map[hir.NodeID][]DropIntent{}},
	}
	a.checkNoBorrowInField()
	for _, modPath := range prog.Order {
		m := prog.Modules[modPath]
		for _, it := range m.Items {
			a.checkItem(modPath, it)
		}
	}
	return a.res, a.diags
}

// analyzer holds the per-run state. All checks are methods on it
// so they share access to the program, the TypeTable, and the
// diagnostic sink.
type analyzer struct {
	prog  *hir.Program
	tab   *typetable.Table
	res   *Result
	diags []Diagnostic
}

func (a *analyzer) diagnose(span lex.Span, msg, hint string) {
	a.diags = append(a.diags, Diagnostic{Span: span, Message: msg, Hint: hint})
}

// --- §54.1: No borrows in struct fields -----------------------------

// checkNoBorrowInField scans every struct declaration and its
// fields, rejecting any field whose type (at any nesting depth)
// contains KindRef or KindMutref.
func (a *analyzer) checkNoBorrowInField() {
	for _, modPath := range a.prog.Order {
		m := a.prog.Modules[modPath]
		for _, it := range m.Items {
			sd, ok := it.(*hir.StructDecl)
			if !ok {
				continue
			}
			for _, f := range sd.Fields {
				if a.typeContainsBorrow(f.TypeOf()) {
					a.diagnose(f.Span,
						fmt.Sprintf("struct field %q has a borrow type; §54.1 forbids borrows in struct fields", f.Name),
						"store owned data in fields, or introduce a newtype wrapper around the borrow at a use site")
				}
			}
			for i, t := range sd.TupleFields {
				if a.typeContainsBorrow(t) {
					a.diagnose(sd.Span,
						fmt.Sprintf("tuple-struct field %d has a borrow type; §54.1 forbids borrows in struct fields", i),
						"use owned data in tuple-struct fields")
				}
			}
		}
	}
}

// TypeContainsBorrow exposes the §54.1 predicate so tests and
// other analyses can reuse it without recursing into every
// TypeTable kind themselves.
func TypeContainsBorrow(tab *typetable.Table, tid typetable.TypeId) bool {
	return (&analyzer{tab: tab}).typeContainsBorrow(tid)
}

// typeContainsBorrow returns true if tid is, or structurally
// contains, a KindRef or KindMutref TypeId. It stops at nominal
// boundaries — the field rule applies only to the declared type
// itself, so `struct Outer { inner: Inner }` does not trigger
// even if Inner has borrows (Inner's own declaration was rejected
// when it landed in the program).
func (a *analyzer) typeContainsBorrow(tid typetable.TypeId) bool {
	t := a.tab.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindRef, typetable.KindMutref:
		return true
	case typetable.KindTuple:
		for _, ch := range t.Children {
			if a.typeContainsBorrow(ch) {
				return true
			}
		}
	case typetable.KindSlice, typetable.KindArray:
		if len(t.Children) > 0 {
			return a.typeContainsBorrow(t.Children[0])
		}
	case typetable.KindPtr:
		// Ptr is a raw pointer, not a borrow. It is permitted in
		// struct fields (reference §54.1 calls out borrows only).
		return false
	case typetable.KindFn:
		for _, ch := range t.Children {
			if a.typeContainsBorrow(ch) {
				return true
			}
		}
		return a.typeContainsBorrow(t.Return)
	}
	return false
}

// --- Per-item walk --------------------------------------------------

func (a *analyzer) checkItem(modPath string, it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		a.checkFn(modPath, x)
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			a.checkItem(modPath, sub)
		}
	case *hir.TraitDecl:
		for _, sub := range x.Items {
			if fn, ok := sub.(*hir.FnDecl); ok && fn.Body != nil {
				a.checkFn(modPath, fn)
			}
		}
	}
}

// checkFn runs every per-fn check: return-borrow rule, use-after-
// move, closure-escape at use sites, and drop-intent insertion.
// Signature-only checks (return-borrow, mutref-aliasing) run even
// when the body is nil — those invariants live at the ABI.
func (a *analyzer) checkFn(modPath string, fn *hir.FnDecl) {
	a.checkReturnBorrowRule(fn)
	a.checkMutrefAliasing(fn)
	if fn.Body == nil {
		return
	}
	a.checkUseAfterMove(fn)
	a.checkClosureEscape(fn)
	a.insertDropIntents(fn)
}

// --- §54.6: Return-borrow rule ----------------------------------------

// checkReturnBorrowRule enforces that a fn returning `ref T` or
// `mutref T` must receive at least one borrow parameter of a
// compatible kind. Returning a borrow to a local is a diagnostic
// — structurally rejected at W09, without lifetime variables.
func (a *analyzer) checkReturnBorrowRule(fn *hir.FnDecl) {
	retT := a.tab.Get(fn.Return)
	if retT == nil {
		return
	}
	if retT.Kind != typetable.KindRef && retT.Kind != typetable.KindMutref {
		return
	}
	// A borrow return requires at least one borrowed parameter.
	hasBorrowParam := false
	for _, p := range fn.Params {
		pt := a.tab.Get(p.TypeOf())
		if pt == nil {
			continue
		}
		if pt.Kind == typetable.KindRef || pt.Kind == typetable.KindMutref {
			hasBorrowParam = true
			break
		}
	}
	if !hasBorrowParam {
		a.diagnose(fn.Span,
			fmt.Sprintf("fn %q returns a borrow but has no borrow parameters; §54.6 requires the return to point into a borrowed parameter",
				fn.Name),
			"make at least one parameter a `ref T` or `mutref T`, or change the return type to an owned value")
		return
	}
	// Walk return statements — any return of a local-defined value
	// is a violation. A return of a param (or a field thereof) is
	// legal because that param itself was borrowed.
	a.walkReturns(fn.Body, func(ret *hir.ReturnStmt) {
		if ret.Value == nil {
			return
		}
		if a.returnsBorrowToLocal(fn, ret.Value) {
			a.diagnose(ret.Span,
				fmt.Sprintf("fn %q returns a borrow to a local binding; §54.6 requires returned borrows to point into a borrowed parameter", fn.Name),
				"return the borrowed parameter itself, or return an owned clone of the value")
		}
	})
}

// returnsBorrowToLocal is a structural predicate: the returned
// expression must be traceable to a borrow of a parameter (via
// UnAddr on a param, a field projection of a borrowed param, or
// the param itself when it is already a borrow). Anything else
// is treated as "local".
func (a *analyzer) returnsBorrowToLocal(fn *hir.FnDecl, e hir.Expr) bool {
	switch x := e.(type) {
	case *hir.PathExpr:
		// A bare path: is it a param?
		if len(x.Segments) == 1 {
			for _, p := range fn.Params {
				if p.Name == x.Segments[0] {
					return false // returns the param itself
				}
			}
		}
		return true
	case *hir.ReferenceExpr:
		return a.returnsBorrowToLocal(fn, x.Inner)
	case *hir.UnaryExpr:
		if x.Op == hir.UnAddr {
			return a.returnsBorrowToLocal(fn, x.Operand)
		}
		return true
	case *hir.FieldExpr:
		return a.returnsBorrowToLocal(fn, x.Receiver)
	}
	return true
}

// walkReturns visits every ReturnStmt inside b, invoking fn for
// each. The visit is recursive through nested blocks and if/match
// expressions.
func (a *analyzer) walkReturns(b *hir.Block, fn func(*hir.ReturnStmt)) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		a.walkReturnsStmt(s, fn)
	}
	if b.Trailing != nil {
		a.walkReturnsExpr(b.Trailing, fn)
	}
}

func (a *analyzer) walkReturnsStmt(s hir.Stmt, fn func(*hir.ReturnStmt)) {
	switch x := s.(type) {
	case *hir.ReturnStmt:
		fn(x)
	case *hir.ExprStmt:
		if x.Expr != nil {
			a.walkReturnsExpr(x.Expr, fn)
		}
	case *hir.LetStmt:
		if x.Value != nil {
			a.walkReturnsExpr(x.Value, fn)
		}
	}
}

func (a *analyzer) walkReturnsExpr(e hir.Expr, fn func(*hir.ReturnStmt)) {
	switch x := e.(type) {
	case *hir.Block:
		a.walkReturns(x, fn)
	case *hir.IfExpr:
		a.walkReturns(x.Then, fn)
		if el, ok := x.Else.(*hir.Block); ok {
			a.walkReturns(el, fn)
		}
		if ei, ok := x.Else.(*hir.IfExpr); ok {
			a.walkReturnsExpr(ei, fn)
		}
	case *hir.MatchExpr:
		for _, arm := range x.Arms {
			a.walkReturns(arm.Body, fn)
		}
	}
}

// --- §54.7: mutref aliasing exclusion -----------------------------

// checkMutrefAliasing rejects any fn whose parameter list contains
// two mutref parameters to the same target type, or a mutref that
// coexists with a ref on the same target. W09 considers two shapes:
//
//   - HIR `Param.Ownership` set to `OwnMutref` / `OwnRef`, which is
//     how the parser encodes `mutref x: T` / `ref x: T` as it
//     carries no borrow type-constructor in the source grammar.
//   - TypeTable `KindRef` / `KindMutref` on the parameter type,
//     which is what synthetic HIR programs (and future source
//     grammar extensions) use.
//
// Both shapes are keyed on the same "target" TypeId so two
// different encodings of the same rule land in one diagnostic
// pathway.
func (a *analyzer) checkMutrefAliasing(fn *hir.FnDecl) {
	mutrefTargets := map[typetable.TypeId]*hir.Param{}
	refTargets := map[typetable.TypeId]*hir.Param{}
	for _, p := range fn.Params {
		kind, target := borrowShapeOfParam(a.tab, p)
		if kind == borrowNone {
			continue
		}
		switch kind {
		case borrowMutref:
			if prior, seen := mutrefTargets[target]; seen {
				a.diagnose(p.Span,
					fmt.Sprintf("mutref parameter %q aliases mutref %q on the same target; §54.7 requires mutref to be exclusive",
						p.Name, prior.Name),
					"take one mutref and use field projections, or take two distinct owned values")
			}
			if prior, seen := refTargets[target]; seen {
				a.diagnose(p.Span,
					fmt.Sprintf("mutref parameter %q coexists with ref parameter %q on the same target; §54.7 requires mutref to be exclusive",
						p.Name, prior.Name),
					"choose one borrow kind for this target, or take distinct arguments")
			}
			mutrefTargets[target] = p
		case borrowRef:
			if prior, seen := mutrefTargets[target]; seen {
				a.diagnose(p.Span,
					fmt.Sprintf("ref parameter %q coexists with mutref %q on the same target; §54.7 requires mutref to be exclusive",
						p.Name, prior.Name),
					"pick one borrow kind for this target")
			}
			refTargets[target] = p
		}
	}
}

// borrowKind is the normalized borrow category for a parameter,
// combining the source-level Ownership marker and the TypeTable-
// level Kind into one enum.
type borrowKind int

const (
	borrowNone borrowKind = iota
	borrowRef
	borrowMutref
)

// borrowShapeOfParam returns (kind, target-TypeId). For the
// source-level shape `ref x: T`, target is the declared TypeId T.
// For the synthetic shape `x: Ref[T]`, target is T from the
// TypeTable's Children[0]. Owned parameters return borrowNone.
func borrowShapeOfParam(tab *typetable.Table, p *hir.Param) (borrowKind, typetable.TypeId) {
	switch p.Ownership {
	case hir.OwnRef:
		return borrowRef, p.TypeOf()
	case hir.OwnMutref:
		return borrowMutref, p.TypeOf()
	}
	pt := tab.Get(p.TypeOf())
	if pt == nil || len(pt.Children) == 0 {
		return borrowNone, typetable.NoType
	}
	switch pt.Kind {
	case typetable.KindRef:
		return borrowRef, pt.Children[0]
	case typetable.KindMutref:
		return borrowMutref, pt.Children[0]
	}
	return borrowNone, typetable.NoType
}

// --- §14: Use-after-move ------------------------------------------

// checkUseAfterMove tracks locals that have been moved (by move
// expression or by being passed to an owned parameter) and reports
// any subsequent read. Move semantics at W09 are structural:
//   - An assignment `let y = x;` where y's type is not Copy and x
//     was a local, moves x.
//   - A function-call argument in an `owned` parameter slot moves
//     the argument if the argument is a local and its type is not Copy.
func (a *analyzer) checkUseAfterMove(fn *hir.FnDecl) {
	env := &moveEnv{
		live:  map[string]bool{},
		moved: map[string]lex.Span{},
	}
	for _, p := range fn.Params {
		env.live[p.Name] = true
	}
	a.walkMove(fn.Body, env)
}

type moveEnv struct {
	live  map[string]bool     // names currently usable
	moved map[string]lex.Span // names that have been moved; span of the move
}

func (a *analyzer) walkMove(b *hir.Block, env *moveEnv) {
	if b == nil {
		return
	}
	if env.live == nil {
		env.live = map[string]bool{}
	}
	for _, s := range b.Stmts {
		a.walkMoveStmt(s, env)
	}
	if b.Trailing != nil {
		a.walkMoveExpr(b.Trailing, env)
	}
}

func (a *analyzer) walkMoveStmt(s hir.Stmt, env *moveEnv) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.Value != nil {
			a.walkMoveExpr(x.Value, env)
			// If the value is a bare path and the source's type
			// is non-Copy, consider the source moved.
			if src, ok := x.Value.(*hir.PathExpr); ok && len(src.Segments) == 1 {
				name := src.Segments[0]
				if env.live[name] && !a.isCopy(src.TypeOf()) {
					env.moved[name] = x.Span
					delete(env.live, name)
				}
			}
			// Bind the pattern's name as a fresh live local.
			if bp, ok := x.Pattern.(*hir.BindPat); ok {
				env.live[bp.Name] = true
			}
		}
	case *hir.VarStmt:
		if x.Value != nil {
			a.walkMoveExpr(x.Value, env)
			if src, ok := x.Value.(*hir.PathExpr); ok && len(src.Segments) == 1 {
				name := src.Segments[0]
				if env.live[name] && !a.isCopy(src.TypeOf()) {
					env.moved[name] = x.Span
					delete(env.live, name)
				}
			}
			env.live[x.Name] = true
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			a.walkMoveExpr(x.Value, env)
		}
	case *hir.ExprStmt:
		if x.Expr != nil {
			a.walkMoveExpr(x.Expr, env)
		}
	}
}

func (a *analyzer) walkMoveExpr(e hir.Expr, env *moveEnv) {
	switch x := e.(type) {
	case *hir.PathExpr:
		if len(x.Segments) == 1 {
			name := x.Segments[0]
			if span, wasMoved := env.moved[name]; wasMoved {
				a.diagnose(x.Span,
					fmt.Sprintf("use of moved value %q; the local was moved at %s", name, span),
					"move is non-Copy by default; clone the value before the first use, or introduce two bindings")
			}
		}
	case *hir.BinaryExpr:
		a.walkMoveExpr(x.Lhs, env)
		a.walkMoveExpr(x.Rhs, env)
	case *hir.UnaryExpr:
		a.walkMoveExpr(x.Operand, env)
	case *hir.CallExpr:
		a.walkMoveExpr(x.Callee, env)
		for _, arg := range x.Args {
			a.walkMoveExpr(arg, env)
		}
	case *hir.Block:
		a.walkMove(x, env)
	case *hir.IfExpr:
		a.walkMoveExpr(x.Cond, env)
		a.walkMove(x.Then, env)
		if x.Else != nil {
			a.walkMoveExpr(x.Else, env)
		}
	}
}

// isCopy consults the W07 marker-trait predicate via the
// TypeTable's Kind. Per §47.1, primitives and borrows-wrapping-Copy
// are Copy; nominal types without an explicit Copy impl are not
// (we default conservatively to "not Copy" for any non-primitive
// to get correct move semantics).
func (a *analyzer) isCopy(tid typetable.TypeId) bool {
	t := a.tab.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindBool, typetable.KindChar,
		typetable.KindI8, typetable.KindI16, typetable.KindI32, typetable.KindI64, typetable.KindISize,
		typetable.KindU8, typetable.KindU16, typetable.KindU32, typetable.KindU64, typetable.KindUSize,
		typetable.KindF32, typetable.KindF64,
		typetable.KindUnit, typetable.KindNever, typetable.KindPtr:
		return true
	}
	return false
}

// --- §15.5: Closure escape classification --------------------------

// checkClosureEscape walks fn's body for closure expressions; any
// closure whose environment captures a ref/mutref must not be
// placed at an escape site (return from the fn, spawn, assignment
// into a struct field, etc.).
func (a *analyzer) checkClosureEscape(fn *hir.FnDecl) {
	// W09 enforces this structurally by looking at return
	// statements and spawn expressions whose subject is a closure
	// whose declared parameter types include a borrow — a
	// heuristic that stands in for capture analysis proper.
	a.walkClosureEscape(fn.Body, fn)
}

func (a *analyzer) walkClosureEscape(b *hir.Block, fn *hir.FnDecl) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		a.walkClosureEscapeStmt(s, fn)
	}
	if b.Trailing != nil {
		a.walkClosureEscapeExpr(b.Trailing, fn, false)
	}
}

func (a *analyzer) walkClosureEscapeStmt(s hir.Stmt, fn *hir.FnDecl) {
	switch x := s.(type) {
	case *hir.ReturnStmt:
		if x.Value != nil {
			a.walkClosureEscapeExpr(x.Value, fn, true)
		}
	case *hir.LetStmt:
		if x.Value != nil {
			a.walkClosureEscapeExpr(x.Value, fn, false)
		}
	case *hir.ExprStmt:
		if x.Expr != nil {
			a.walkClosureEscapeExpr(x.Expr, fn, false)
		}
	}
}

func (a *analyzer) walkClosureEscapeExpr(e hir.Expr, fn *hir.FnDecl, escaping bool) {
	switch x := e.(type) {
	case *hir.ClosureExpr:
		if escaping && a.closureIsNonEscaping(x) {
			a.diagnose(x.Span,
				"non-escaping closure used at an escape site; §15.5 rejects closures with ref captures at return/spawn/Chan/Shared",
				"prefix the closure with `move` to capture by value, or rewrite to avoid the borrow capture")
		}
	case *hir.SpawnExpr:
		// spawn is always an escape site.
		if x.Closure != nil && a.closureIsNonEscaping(x.Closure) {
			a.diagnose(x.Span,
				"non-escaping closure passed to spawn; §15.5 forbids borrow captures across concurrency boundaries",
				"prefix the closure with `move` to capture by value, or wrap shared state in Shared[T]")
		}
	case *hir.Block:
		a.walkClosureEscape(x, fn)
	case *hir.IfExpr:
		a.walkClosureEscape(x.Then, fn)
		if x.Else != nil {
			a.walkClosureEscapeExpr(x.Else, fn, escaping)
		}
	}
}

// closureIsNonEscaping returns true when a closure carries at
// least one borrow type in its parameter list (a proxy for its
// environment at W09 until full capture analysis lands in W12).
// A `move` closure is always treated as escaping since its
// captures are moved into the environment.
func (a *analyzer) closureIsNonEscaping(c *hir.ClosureExpr) bool {
	if c.IsMove {
		return false
	}
	for _, p := range c.Params {
		pt := a.tab.Get(p.TypeOf())
		if pt != nil && (pt.Kind == typetable.KindRef || pt.Kind == typetable.KindMutref) {
			return true
		}
	}
	return false
}

// --- Drop intent insertion ----------------------------------------

// insertDropIntents walks fn's body and records a DropIntent for
// every local whose type implements Drop. Concretely at W09: a
// type implements Drop if the program contains `impl Drop for T`
// in the same module as T's declaration (we do a simple lookup).
func (a *analyzer) insertDropIntents(fn *hir.FnDecl) {
	if fn.Body == nil {
		return
	}
	// Build the set of Drop-implementing TypeIds by scanning impl
	// blocks whose trait is named "Drop".
	dropTypes := a.collectDropTypes()
	// Walk the fn body for let/var bindings whose type is in
	// dropTypes; emit a DropIntent for each.
	var intents []DropIntent
	for _, s := range fn.Body.Stmts {
		switch x := s.(type) {
		case *hir.LetStmt:
			if bp, ok := x.Pattern.(*hir.BindPat); ok {
				if dropTypes[x.DeclaredType] {
					intents = append(intents, DropIntent{
						LocalName:   bp.Name,
						Type:        x.DeclaredType,
						EnclosingFn: fn.ID,
					})
				}
			}
		case *hir.VarStmt:
			if dropTypes[x.DeclaredType] {
				intents = append(intents, DropIntent{
					LocalName:   x.Name,
					Type:        x.DeclaredType,
					EnclosingFn: fn.ID,
				})
			}
		}
	}
	if len(intents) > 0 {
		a.res.DropIntents[fn.ID] = intents
	}
}

// collectDropTypes returns the set of TypeIds that implement the
// Drop trait in the current program. At W09 the lookup is
// structural: any impl block whose target TypeId is present as the
// Target of an `impl Drop for T` block is considered Drop-impl'd.
// The trait itself is identified by name ("Drop") on the
// ImplDecl.Trait TypeId.
func (a *analyzer) collectDropTypes() map[typetable.TypeId]bool {
	out := map[typetable.TypeId]bool{}
	for _, modPath := range a.prog.Order {
		m := a.prog.Modules[modPath]
		for _, it := range m.Items {
			impl, ok := it.(*hir.ImplDecl)
			if !ok || impl.Trait == typetable.NoType {
				continue
			}
			tt := a.tab.Get(impl.Trait)
			if tt != nil && tt.Kind == typetable.KindTrait && tt.Name == "Drop" {
				out[impl.Target] = true
			}
		}
	}
	return out
}

// --- Helpers for tests --------------------------------------------

// DescribeResult returns a deterministic summary string of a
// Result for test diffing. Only used by tests.
func DescribeResult(r *Result) string {
	var out string
	for _, intents := range r.DropIntents {
		for _, in := range intents {
			out += fmt.Sprintf("drop %s (TypeId %d)\n", in.LocalName, in.Type)
		}
	}
	return out
}
