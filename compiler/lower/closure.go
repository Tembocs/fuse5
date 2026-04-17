package lower

import (
	"sort"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// W12 closure analysis lives here because capture classification,
// environment-struct shape, and lifted-body generation are
// lowering concerns. The W12 contracts (reference §15):
//
//   - CaptureMode classifies each outer binding a closure references
//     (Copy / Ref / Mutref / Owned) using the tightest mode the body
//     requires. `move` closures override every capture to Owned
//     (§15.3).
//   - EscapeClass marks a closure as non-escaping when its
//     environment struct carries any `ref T` / `mutref T` field; the
//     W09-P01-T06 enforcement consumes this at use sites.
//   - EnvShape describes the synthetic environment struct — one
//     field per captured binding, ordered by name for determinism.
//   - LiftedShape summarises what codegen needs to emit: the env
//     struct type, the standalone fn name (`<fnname>_closure_N`),
//     and the parameter list (captures first as `env`, then the
//     closure's own params).
//
// These are pure predicates at W12: they compute metadata from
// the HIR without mutating it. The bridge carries `ClosureExpr`
// as-is; emission of the env struct and the lifted fn is
// scheduled for W15 MIR consolidation when MIR grows struct
// layout support. The checker uses the predicates today to
// attribute Fn/FnMut/FnOnce auto-impls.

// CaptureMode enumerates the four classifications §15.7 permits
// for a captured outer binding.
type CaptureMode int

const (
	CaptureCopy   CaptureMode = iota // read-only use of a Copy type
	CaptureRef                       // read-only use of a non-Copy type
	CaptureMutref                    // write through the binding
	CaptureOwned                     // `move` / `owned` use in the body
)

// String returns the human-readable spelling of the mode.
func (m CaptureMode) String() string {
	switch m {
	case CaptureCopy:
		return "copy"
	case CaptureRef:
		return "ref"
	case CaptureMutref:
		return "mutref"
	case CaptureOwned:
		return "owned"
	}
	return "unknown"
}

// Capture records one captured outer binding with its source
// name and classified mode.
type Capture struct {
	Name string
	Mode CaptureMode
}

// EscapeClass records whether a closure may outlive its
// defining scope.
type EscapeClass int

const (
	EscapeEscaping   EscapeClass = iota // env has no ref/mutref captures
	EscapeNonEscape                     // env has at least one borrow capture
)

// String returns the human-readable spelling.
func (e EscapeClass) String() string {
	switch e {
	case EscapeEscaping:
		return "escaping"
	case EscapeNonEscape:
		return "non-escaping"
	}
	return "unknown"
}

// CallableTrait enumerates the three callable traits a closure
// may auto-implement. The tightest trait a closure satisfies
// depends on its capture set: Owned / Copy captures → Fn; any
// Mutref capture → FnMut (and Fn is not implemented); any Owned
// capture that is consumed in the body → FnOnce only.
type CallableTrait int

const (
	CallableFn CallableTrait = iota
	CallableFnMut
	CallableFnOnce
)

// String returns the trait's declared name.
func (t CallableTrait) String() string {
	switch t {
	case CallableFn:
		return "Fn"
	case CallableFnMut:
		return "FnMut"
	case CallableFnOnce:
		return "FnOnce"
	}
	return "unknown"
}

// EnvShape is the flat description of a closure's synthetic
// environment struct. One StructField entry per captured binding,
// sorted by name for determinism (Rule 7.1).
type EnvShape struct {
	Fields []EnvField
}

// EnvField is one entry in the env struct.
type EnvField struct {
	Name string
	Mode CaptureMode
}

// LiftedShape summarises what W15 MIR consolidation will emit for
// a lifted closure body: the env struct (as an EnvShape), the
// closure's own parameters, and a synthesised fn name.
type LiftedShape struct {
	Env        EnvShape
	ParamNames []string
	FnName     string
}

// ClosureAnalysis bundles every per-closure metadata the W12
// contracts require.
type ClosureAnalysis struct {
	Captures    []Capture
	Escape      EscapeClass
	Traits      []CallableTrait
	Env         EnvShape
	Lifted      LiftedShape
	IsMove      bool
}

// AnalyzeClosure computes the W12 metadata for a single
// ClosureExpr. The outer scope's name→TypeId map tells the
// analyzer which free variables belong to the enclosing scope
// (anything else is either a parameter of the closure itself, a
// built-in, or a compiler error the checker has already caught).
//
// anchorName is a stable identifier for the enclosing fn; the
// lifted closure's fn name is derived from it.
func AnalyzeClosure(c *hir.ClosureExpr, anchorName string, outerScope map[string]typetable.TypeId, tab *typetable.Table) ClosureAnalysis {
	a := ClosureAnalysis{IsMove: c.IsMove}
	rawCaptures := collectFreeVariables(c, outerScope)
	a.Captures = classifyCaptures(rawCaptures, outerScope, tab, c.IsMove)
	a.Env = buildEnvShape(a.Captures)
	a.Escape = classifyEscape(a.Captures)
	a.Traits = classifyCallable(a.Captures)
	a.Lifted = LiftedShape{
		Env:        a.Env,
		ParamNames: collectParamNames(c),
		FnName:     anchorName + "_closure",
	}
	return a
}

// collectFreeVariables walks the closure body and returns the
// set of identifier names that are (a) referenced in the body,
// (b) not bound by the closure's own parameter list, and
// (c) present in the outer scope. Returns a sorted slice for
// determinism.
func collectFreeVariables(c *hir.ClosureExpr, outerScope map[string]typetable.TypeId) map[string]captureUse {
	// Build the set of names the closure binds locally (its
	// parameters). Any reference to one of these names is not a
	// free variable.
	locals := map[string]bool{}
	for _, p := range c.Params {
		locals[p.Name] = true
	}
	out := map[string]captureUse{}
	walkClosureBody(c.Body, locals, outerScope, out)
	return out
}

// captureUse tracks how a free variable is used inside the
// closure body so classifyCaptures can pick the tightest mode.
type captureUse struct {
	read     bool
	write    bool
	moved    bool
}

// walkClosureBody recursively visits every expression under the
// closure's body, filling `out` with an entry per referenced
// outer binding. Writes set `write`; reads set `read`; explicit
// `move x` expressions set `moved`.
func walkClosureBody(b *hir.Block, locals map[string]bool, outerScope map[string]typetable.TypeId, out map[string]captureUse) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		walkClosureStmt(s, locals, outerScope, out)
	}
	if b.Trailing != nil {
		walkClosureExpr(b.Trailing, locals, outerScope, out, false /* written */)
	}
}

func walkClosureStmt(s hir.Stmt, locals map[string]bool, outerScope map[string]typetable.TypeId, out map[string]captureUse) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if bp, ok := x.Pattern.(*hir.BindPat); ok {
			locals[bp.Name] = true
		}
		if x.Value != nil {
			walkClosureExpr(x.Value, locals, outerScope, out, false)
		}
	case *hir.VarStmt:
		locals[x.Name] = true
		if x.Value != nil {
			walkClosureExpr(x.Value, locals, outerScope, out, false)
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			walkClosureExpr(x.Value, locals, outerScope, out, false)
		}
	case *hir.ExprStmt:
		if x.Expr != nil {
			walkClosureExpr(x.Expr, locals, outerScope, out, false)
		}
	}
}

func walkClosureExpr(e hir.Expr, locals map[string]bool, outerScope map[string]typetable.TypeId, out map[string]captureUse, writeContext bool) {
	if e == nil {
		return
	}
	switch x := e.(type) {
	case *hir.PathExpr:
		if len(x.Segments) == 1 {
			name := x.Segments[0]
			if locals[name] {
				return
			}
			if _, inOuter := outerScope[name]; !inOuter {
				return
			}
			u := out[name]
			if writeContext {
				u.write = true
			} else {
				u.read = true
			}
			out[name] = u
		}
	case *hir.AssignExpr:
		// LHS is a write; RHS is a read.
		walkClosureExpr(x.Lhs, locals, outerScope, out, true)
		walkClosureExpr(x.Rhs, locals, outerScope, out, false)
	case *hir.BinaryExpr:
		walkClosureExpr(x.Lhs, locals, outerScope, out, false)
		walkClosureExpr(x.Rhs, locals, outerScope, out, false)
	case *hir.UnaryExpr:
		// UnDeref / UnAddr / UnNeg / UnNot — all read.
		walkClosureExpr(x.Operand, locals, outerScope, out, false)
	case *hir.CallExpr:
		walkClosureExpr(x.Callee, locals, outerScope, out, false)
		for _, a := range x.Args {
			walkClosureExpr(a, locals, outerScope, out, false)
		}
	case *hir.FieldExpr:
		walkClosureExpr(x.Receiver, locals, outerScope, out, false)
	case *hir.Block:
		walkClosureBody(x, locals, outerScope, out)
	case *hir.IfExpr:
		walkClosureExpr(x.Cond, locals, outerScope, out, false)
		walkClosureBody(x.Then, locals, outerScope, out)
		if x.Else != nil {
			walkClosureExpr(x.Else, locals, outerScope, out, false)
		}
	case *hir.TupleExpr:
		for _, el := range x.Elements {
			walkClosureExpr(el, locals, outerScope, out, false)
		}
	case *hir.MatchExpr:
		walkClosureExpr(x.Scrutinee, locals, outerScope, out, false)
		for _, arm := range x.Arms {
			walkClosureBody(arm.Body, locals, outerScope, out)
		}
	case *hir.TryExpr:
		walkClosureExpr(x.Receiver, locals, outerScope, out, false)
	case *hir.ReferenceExpr:
		// &x is a read of x; &mut x is a write-context read.
		walkClosureExpr(x.Inner, locals, outerScope, out, x.Mutable)
	case *hir.ClosureExpr:
		// Nested closures — their own free-variable set is a
		// separate analysis, but at the outer level we still
		// need to identify any outer name they reference.
		walkClosureBody(x.Body, mergeLocals(locals, x.Params), outerScope, out)
	}
}

// mergeLocals produces a locals map that shadows with the
// nested closure's own params. Used when we recurse into a
// nested closure's body from the outer analysis.
func mergeLocals(outer map[string]bool, params []*hir.Param) map[string]bool {
	m := map[string]bool{}
	for k, v := range outer {
		m[k] = v
	}
	for _, p := range params {
		m[p.Name] = true
	}
	return m
}

// collectParamNames returns the closure's own parameter names
// (not captures). Kept separate so LiftedShape can carry both
// lists without ambiguity.
func collectParamNames(c *hir.ClosureExpr) []string {
	out := make([]string, 0, len(c.Params))
	for _, p := range c.Params {
		out = append(out, p.Name)
	}
	return out
}

// classifyCaptures applies the W12-P01-T01 rule: tightest mode
// per capture. Reads of a Copy type → Copy; reads of non-Copy →
// Ref; writes → Mutref; move / owned uses → Owned. A `move`
// closure prefix overrides everything to Owned.
func classifyCaptures(uses map[string]captureUse, outerScope map[string]typetable.TypeId, tab *typetable.Table, moveClosure bool) []Capture {
	names := make([]string, 0, len(uses))
	for n := range uses {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Capture, 0, len(names))
	for _, name := range names {
		u := uses[name]
		var mode CaptureMode
		switch {
		case moveClosure || u.moved:
			mode = CaptureOwned
		case u.write:
			mode = CaptureMutref
		default:
			if isCopyTypeId(tab, outerScope[name]) {
				mode = CaptureCopy
			} else {
				mode = CaptureRef
			}
		}
		out = append(out, Capture{Name: name, Mode: mode})
	}
	return out
}

// isCopyTypeId is a local version of the W07 IsCopy predicate —
// duplicated here to avoid a liveness/check import cycle. The
// W07 definition is the source of truth; this helper mirrors it
// for the W12 classification step.
func isCopyTypeId(tab *typetable.Table, tid typetable.TypeId) bool {
	t := tab.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindBool, typetable.KindChar,
		typetable.KindI8, typetable.KindI16, typetable.KindI32,
		typetable.KindI64, typetable.KindISize,
		typetable.KindU8, typetable.KindU16, typetable.KindU32,
		typetable.KindU64, typetable.KindUSize,
		typetable.KindF32, typetable.KindF64,
		typetable.KindUnit, typetable.KindNever, typetable.KindPtr:
		return true
	}
	return false
}

// buildEnvShape flattens the capture list into an EnvShape by
// promoting CaptureMode → EnvField.Mode 1:1.
func buildEnvShape(captures []Capture) EnvShape {
	fields := make([]EnvField, 0, len(captures))
	for _, c := range captures {
		fields = append(fields, EnvField{Name: c.Name, Mode: c.Mode})
	}
	return EnvShape{Fields: fields}
}

// classifyEscape computes the reference-§15.5 rule: if the env
// struct contains any Ref or Mutref capture, the closure is
// non-escaping. Otherwise escaping.
func classifyEscape(captures []Capture) EscapeClass {
	for _, c := range captures {
		if c.Mode == CaptureRef || c.Mode == CaptureMutref {
			return EscapeNonEscape
		}
	}
	return EscapeEscaping
}

// classifyCallable computes the set of callable traits a closure
// auto-implements:
//
//   - Fn: no Mutref or consumed-Owned captures. Can be called
//     any number of times without mutating the env.
//   - FnMut: no consumed-Owned captures but has at least one
//     Mutref. Can be called repeatedly; mutates the env.
//   - FnOnce: has at least one Owned capture. Can only be
//     called once because calling consumes the env.
//
// A closure may auto-impl multiple traits (Fn ⟹ FnMut ⟹ FnOnce
// by the trait hierarchy §15.6). The returned slice is the
// tightest set; dispatch can widen at use sites.
func classifyCallable(captures []Capture) []CallableTrait {
	hasMutref := false
	hasOwned := false
	for _, c := range captures {
		if c.Mode == CaptureMutref {
			hasMutref = true
		}
		if c.Mode == CaptureOwned {
			hasOwned = true
		}
	}
	switch {
	case hasOwned:
		// Owned captures are consumed on call → FnOnce only.
		return []CallableTrait{CallableFnOnce}
	case hasMutref:
		// Mutref captures allow repeated calls but mutate →
		// FnMut + FnOnce (by hierarchy).
		return []CallableTrait{CallableFnMut, CallableFnOnce}
	default:
		// Pure reads → Fn + FnMut + FnOnce.
		return []CallableTrait{CallableFn, CallableFnMut, CallableFnOnce}
	}
}

// DesugarCall returns the callable-trait method name the lowerer
// should dispatch through for a call site where the receiver
// implements the given tightest trait. At W12 this is a
// structural mapping — code emission for the call still goes
// through the ordinary CallExpr path until W15 MIR consolidation
// wires actual vtable dispatch.
//
//   Fn     → "call"
//   FnMut  → "call_mut"
//   FnOnce → "call_once"
func DesugarCall(t CallableTrait) string {
	switch t {
	case CallableFn:
		return "call"
	case CallableFnMut:
		return "call_mut"
	case CallableFnOnce:
		return "call_once"
	}
	return ""
}
