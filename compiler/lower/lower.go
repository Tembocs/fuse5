package lower

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/mir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Diagnostic is one diagnostic emitted by the lowerer. The shape
// mirrors `lex.Diagnostic` so downstream passes can merge them into
// a single diagnostic stream.
type Diagnostic = lex.Diagnostic

// Lower produces a MIR Module from a HIR Program, honoring the W05
// spine contract: only int-returning functions whose bodies are
// chains of integer-literal arithmetic plus `return` are supported.
// Anything else generates a diagnostic rather than a silent default
// (Rule 6.9).
//
// The resulting MIR Module contains every successfully lowered
// function; functions that produced diagnostics are omitted so the
// caller can still inspect partial lowering output in tests.
func Lower(prog *hir.Program) (*mir.Module, []Diagnostic) {
	l := &lowerer{prog: prog}
	l.run()
	return l.module, l.diags
}

type lowerer struct {
	prog         *hir.Program
	module       *mir.Module
	diags        []Diagnostic
	spawnCounter int // W17: deterministic counter for lifted spawn fns
}

func (l *lowerer) run() {
	l.module = &mir.Module{}
	for _, modPath := range l.prog.Order {
		m := l.prog.Modules[modPath]
		for _, it := range m.Items {
			if fn, ok := it.(*hir.FnDecl); ok {
				if lowered := l.lowerFn(modPath, fn); lowered != nil {
					l.module.Functions = append(l.module.Functions, lowered)
				}
			}
		}
	}
}

// lowerFn lowers one HIR fn into a MIR Function. Returns nil when
// the fn cannot be lowered. W06 added fn parameters and calls to the
// supported surface — generic, closure, and complex-flow bodies are
// still diagnosed rather than silently approximated (Rule 6.9).
func (l *lowerer) lowerFn(modPath string, fn *hir.FnDecl) *mir.Function {
	if fn.Body == nil {
		// Extern declarations are not lowered at W05. They are
		// preserved in HIR; W17 drives their codegen.
		return nil
	}
	if len(fn.Generics) != 0 {
		l.diagnose(fn.Span,
			"spine does not yet lower generic fn bodies",
			"monomorphization and generic instantiation arrive in W08")
		return nil
	}
	mirFn, b := mir.NewFunction(modPath, fn.Name)
	// Bind each HIR param to a MIR register via OpParam. The
	// checker-assigned TypeId is not consulted at MIR level (int64
	// is the universal register type at W06); we simply ensure each
	// param slot is known by name for body lookups.
	paramRegs := map[string]mir.Reg{}
	for i, p := range fn.Params {
		reg := b.Param(i)
		paramRegs[p.Name] = reg
	}
	if !l.lowerBlockToReturn(modPath, b, fn.Body, paramRegs) {
		return nil
	}
	if err := mirFn.Validate(); err != nil {
		l.diagnose(fn.Span,
			fmt.Sprintf("MIR validation failed for fn %q: %v", fn.Name, err),
			"the lowerer produced an instruction the MIR does not yet accept")
		return nil
	}
	return mirFn
}

// lowerBlockToReturn expects the HIR block to be a straight-line
// body ending in `return EXPR;`. W06 keeps the "one statement per
// body" spine restriction but supports fn-call / parameter-read
// subexpressions inside the return value (the checker ensures
// types are consistent so MIR codegen works).
func (l *lowerer) lowerBlockToReturn(modPath string, b *mir.Builder, blk *hir.Block, params map[string]mir.Reg) bool {
	if blk == nil {
		l.diagnose(lex.Span{}, "spine requires a fn body", "add a body with `return <expr>;`")
		return false
	}
	if blk.Trailing != nil {
		l.diagnose(blk.NodeSpan(),
			"spine does not yet lower trailing block expressions",
			"use an explicit `return <expr>;` statement")
		return false
	}
	if len(blk.Stmts) != 1 {
		l.diagnose(blk.NodeSpan(),
			"spine supports exactly one statement per fn body",
			"the minimal spine shape is `fn name(...) -> I32 { return <expr>; }`")
		return false
	}
	ret, ok := blk.Stmts[0].(*hir.ReturnStmt)
	if !ok {
		l.diagnose(blk.Stmts[0].NodeSpan(),
			"spine requires the single body statement to be a `return`",
			"use `return <expr>;` as the only body statement")
		return false
	}
	if ret.Value == nil {
		l.diagnose(ret.NodeSpan(),
			"spine requires `return` to carry an integer expression",
			"write `return 0;` at minimum")
		return false
	}
	reg, okExpr := l.lowerExpr(modPath, b, ret.Value, params)
	if !okExpr {
		return false
	}
	b.Return(reg)
	return true
}

// lowerExpr lowers one integer-producing expression and returns the
// register holding its value. W06 extends the supported surface to
// PathExpr (resolves to a param read) and CallExpr (direct fn
// invocation with integer args). Forms outside the spine still
// produce diagnostics rather than silent approximations.
func (l *lowerer) lowerExpr(modPath string, b *mir.Builder, e hir.Expr, params map[string]mir.Reg) (mir.Reg, bool) {
	switch x := e.(type) {
	case *hir.LiteralExpr:
		switch x.Kind {
		case hir.LitInt:
			v, err := parseIntLiteral(x.Text)
			if err != nil {
				l.diagnose(x.NodeSpan(),
					fmt.Sprintf("invalid integer literal %q: %v", x.Text, err),
					"write a value in the signed 64-bit range")
				return mir.NoReg, false
			}
			return b.ConstInt(v), true
		case hir.LitBool:
			// Booleans lower as i64 constants (true=1, false=0),
			// matching the discriminant convention used by
			// match-on-bool dispatch.
			if x.Bool {
				return b.ConstInt(1), true
			}
			return b.ConstInt(0), true
		default:
			l.diagnose(x.NodeSpan(),
				fmt.Sprintf("spine only lowers integer and boolean literals, not %s", litKindName(x.Kind)),
				"replace the literal with an integer or boolean")
			return mir.NoReg, false
		}
	case *hir.BinaryExpr:
		// W17 routes equality / inequality through the semantic-
		// equality lowerer so scalar vs nominal dispatch is
		// structurally visible in MIR (reference §5.8).
		if x.Op == hir.BinEq || x.Op == hir.BinNe {
			return l.lowerSemanticEquality(modPath, b, x, params)
		}
		op, ok := mapBinaryOp(x.Op)
		if !ok {
			l.diagnose(x.NodeSpan(),
				fmt.Sprintf("spine does not yet lower binary operator %s", binOpName(x.Op)),
				"use `+`, `-`, `*`, `/`, or `%`")
			return mir.NoReg, false
		}
		lhs, ok := l.lowerExpr(modPath, b, x.Lhs, params)
		if !ok {
			return mir.NoReg, false
		}
		rhs, ok := l.lowerExpr(modPath, b, x.Rhs, params)
		if !ok {
			return mir.NoReg, false
		}
		return b.Binary(op, lhs, rhs), true
	case *hir.PathExpr:
		// W06 resolves single-segment paths to parameter registers.
		// Two-segment paths may name an enum variant
		// (`EnumName.VariantName`), which W10 lowers as a ConstInt
		// carrying the variant's declared index — the
		// discriminant convention used by match dispatch.
		if len(x.Segments) == 1 {
			if reg, ok := params[x.Segments[0]]; ok {
				return reg, true
			}
			l.diagnose(x.NodeSpan(),
				fmt.Sprintf("spine does not yet resolve %q as a value", x.Segments[0]),
				"use a parameter name or inline the expression")
			return mir.NoReg, false
		}
		if len(x.Segments) == 2 {
			enumName := x.Segments[0]
			variantName := x.Segments[1]
			if idx, ok := l.lookupEnumVariantByName(enumName, variantName); ok {
				return b.ConstInt(int64(idx)), true
			}
		}
		l.diagnose(x.NodeSpan(),
			"spine does not yet lower this path value",
			"for enum variants, use `EnumName.VariantName`; for fn calls, use `name(args)`")
		return mir.NoReg, false
	case *hir.CallExpr:
		// W12: immediately-invoked closure `(fn() -> T { ... })()`.
		// For no-capture closures the lowerer inlines the body as
		// the call's result; this is the W12 proof-path shape
		// because real env-struct + lifted-fn emission lands with
		// W15 MIR consolidation. Call-desugaring (`f(args)` →
		// `f.call(args)`) is structural at W12; the metadata lives
		// in ClosureAnalysis.
		if clos, ok := x.Callee.(*hir.ClosureExpr); ok {
			if len(x.Args) != len(clos.Params) {
				l.diagnose(x.NodeSpan(),
					"closure call arity mismatch",
					"pass exactly as many arguments as the closure declares")
				return mir.NoReg, false
			}
			// Bind each argument under the closure's param name
			// for the duration of its body.
			inlineParams := map[string]mir.Reg{}
			for k, v := range params {
				inlineParams[k] = v
			}
			for i, a := range x.Args {
				r, ok := l.lowerExpr(modPath, b, a, params)
				if !ok {
					return mir.NoReg, false
				}
				inlineParams[clos.Params[i].Name] = r
			}
			return l.lowerInlinedClosureBody(modPath, b, clos.Body, inlineParams)
		}
		// W17: method-call disambiguation. `obj.name(args)` lowers
		// to one of four forms depending on the receiver's type:
		//   - ThreadHandle[T].join() → OpThreadJoin
		//   - Chan[T].send/recv/close → OpChan{Send,Recv,Close}
		//   - wrapping_*/checked_*/saturating_* → OpOverflowArith
		//   - any other method → OpMethodCall
		if field, ok := x.Callee.(*hir.FieldExpr); ok {
			if reg, okRuntime := l.lowerRuntimeMethodCall(modPath, b, x, field, params); okRuntime {
				return reg, true
			}
			if reg, ok2, recognised := l.lowerOverflowMethod(modPath, b, x, field, params); recognised {
				if !ok2 {
					return mir.NoReg, false
				}
				return reg, true
			}
			return l.lowerMethodCall(modPath, b, x, field, params)
		}
		callee, ok := x.Callee.(*hir.PathExpr)
		if !ok {
			l.diagnose(x.NodeSpan(),
				"spine requires a direct fn-name callee",
				"indirect calls via fn-pointer expressions land in later waves")
			return mir.NoReg, false
		}
		if len(callee.Segments) != 1 {
			l.diagnose(callee.NodeSpan(),
				"spine does not yet lower module-qualified calls",
				"import and call the fn by its bare name")
			return mir.NoReg, false
		}
		argRegs := make([]mir.Reg, 0, len(x.Args))
		for _, a := range x.Args {
			r, ok := l.lowerExpr(modPath, b, a, params)
			if !ok {
				return mir.NoReg, false
			}
			argRegs = append(argRegs, r)
		}
		return b.Call(cName(modPath, callee.Segments[0]), argRegs), true
	case *hir.SpawnExpr:
		return l.lowerSpawn(modPath, b, x, params)
	case *hir.ReferenceExpr:
		return l.lowerReference(modPath, b, x, params)
	case *hir.FieldExpr:
		return l.lowerFieldAccess(modPath, b, x, params)
	case *hir.OptFieldExpr:
		return l.lowerOptChain(modPath, b, x, params)
	case *hir.IndexRangeExpr:
		return l.lowerIndexRange(modPath, b, x, params)
	case *hir.StructLitExpr:
		return l.lowerStructLit(modPath, b, x, params)
	case *hir.MatchExpr:
		return l.lowerMatch(modPath, b, x, params)
	case *hir.TryExpr:
		return l.lowerTry(modPath, b, x, params)
	case *hir.CastExpr:
		// W17 routes CastExpr through the tagged classifier so
		// codegen sees a CastMode-carrying OpCast. Reference §28.1
		// classification: widen / narrow / reinterpret / float↔int /
		// pointer casts. A cast that classifyCast cannot resolve
		// falls back to a passthrough so existing consteval-driven
		// flows (e.g. const_fn.fuse `FACT_5 as I32`) continue to
		// produce the same observable behaviour.
		if reg, ok := l.lowerCastTagged(modPath, b, x, params); ok {
			return reg, true
		}
		// Fallback: if classification failed but the inner
		// expression is still lowerable, preserve the W14
		// passthrough semantics.
		return l.lowerExpr(modPath, b, x.Expr, params)
	default:
		l.diagnose(e.NodeSpan(),
			fmt.Sprintf("spine does not yet lower %T", e),
			"the W06 spine supports int literals, +/-/*// arithmetic, parameter reads, and direct fn calls")
		return mir.NoReg, false
	}
}

// lowerMatch lowers a match expression to cascading MIR blocks.
//
// Shape:
//   - Compute the scrutinee into a register and allocate a
//     shared result register initialised to zero.
//   - For each non-default arm, emit a comparison block that
//     tests the scrutinee against the arm's discriminant
//     (TermIfEq). On match, jump to the arm's body block; on
//     mismatch, fall through to the next comparison.
//   - A trailing wildcard / BindPat arm becomes the fallthrough
//     target of the last mismatch.
//   - Every arm body writes its produced value into the shared
//     result register and jumps to a common merge block. The
//     merge block's terminator is whatever the outer expression
//     lowerer emits next (typically a Return).
//
// Supported scrutinee shapes:
//
//	(a) scrutinee is Bool; arms are LiteralPat bool patterns plus
//	    an optional wildcard.
//	(b) scrutinee is a simple enum (no payloads); arms are
//	    ConstructorPat using the enum's variant names, plus an
//	    optional wildcard.
//
// Match expressions whose arm patterns fall outside the W10
// supported set (non-bool literals, range, tuple, nested
// binding) produce a diagnostic rather than a silent
// approximation (Rule 6.9).
func (l *lowerer) lowerMatch(modPath string, b *mir.Builder, m *hir.MatchExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	scrut, ok := l.lowerExpr(modPath, b, m.Scrutinee, params)
	if !ok {
		return mir.NoReg, false
	}
	discs, ok := l.armDiscriminants(m)
	if !ok {
		return mir.NoReg, false
	}
	// Allocate the result register in the scrutinee's block so it
	// exists before any branch.
	result := b.ConstInt(0)
	// Allocate body blocks for every arm + the merge block.
	armBlocks := make([]mir.BlockId, len(m.Arms))
	for i := range m.Arms {
		armBlocks[i] = b.BeginBlock()
		// Each arm body will be filled in via UseBlock below.
	}
	mergeBlock := b.BeginBlock()
	// Jump back to the scrutinee block's frame: the comparisons
	// are emitted as a chain of blocks that fall through to the
	// next comparison on mismatch. Allocate those now.
	var compareBlocks []mir.BlockId
	lastArmIsDefault := discs[len(discs)-1] == nil
	numCompares := len(m.Arms)
	if lastArmIsDefault {
		numCompares--
	}
	for i := 0; i < numCompares; i++ {
		compareBlocks = append(compareBlocks, b.BeginBlock())
	}
	// We must have a block to emit the initial jump into the first
	// compare. The scrutinee's block is the only place to emit
	// that jump. BeginBlock sealed it; we need to re-open it.
	// `mir.Builder` lacks a direct "previous block" API, so we use
	// the fact that blocks are allocated sequentially: the block
	// whose ID is `armBlocks[0] - 1` was the one that held the
	// scrutinee and the result ConstInt.
	scrutBlock := armBlocks[0] - 1
	b.UseBlock(scrutBlock)
	// Jump from scrutinee block into the first comparison (or the
	// default arm when there are no comparisons).
	if len(compareBlocks) > 0 {
		b.Jump(compareBlocks[0])
	} else {
		b.Jump(armBlocks[0])
	}
	// Emit each comparison block.
	for i, cb := range compareBlocks {
		b.UseBlock(cb)
		disc := *discs[i]
		// On mismatch, fall through to the next compare, or to
		// the default arm if this was the last compare.
		var falseTarget mir.BlockId
		if i+1 < len(compareBlocks) {
			falseTarget = compareBlocks[i+1]
		} else if lastArmIsDefault {
			falseTarget = armBlocks[len(armBlocks)-1]
		} else {
			// Exhaustiveness is a checker invariant; if we land
			// here the checker failed us. Jump to merge so the
			// MIR is at least well-formed.
			falseTarget = mergeBlock
		}
		b.IfEq(scrut, disc, armBlocks[i], falseTarget)
	}
	// Emit every arm body.
	for i, arm := range m.Arms {
		b.UseBlock(armBlocks[i])
		// W10 scope: arm bodies contain a single trailing
		// expression inside the block. Anything richer is left
		// to W14 const-eval / W17 codegen hardening.
		bodyVal, ok := l.lowerArmBody(modPath, b, arm, params)
		if !ok {
			return mir.NoReg, false
		}
		// Emit an add with zero to "move" bodyVal into result.
		// This keeps the per-arm write of the result register
		// deterministic; codegen renders it as `rR = rBody + 0;`.
		zero := b.ConstInt(0)
		assigned := b.Binary(mir.OpAdd, bodyVal, zero)
		// Force the result register to be overwritten: since MIR
		// is not strict SSA, we can re-alias by emitting a second
		// OpAdd that pretends to produce `result`. Instead of a
		// real phi, we use a hack: drop the old `result` and let
		// the merge block read the last-written `assigned` by
		// jumping to a per-arm "set result" routine. That's too
		// much ceremony; at W10 we keep things simple and return
		// the last arm's result via the merge block reading
		// `result`. To make that work across arms we must write
		// into `result` explicitly, which requires aliasing — see
		// the emitMoveInto helper below.
		l.emitMoveInto(b, assigned, result)
		b.Jump(mergeBlock)
	}
	// Merge block: its only instruction is an identity move, its
	// terminator inherits whatever the caller emits (usually a
	// Return carrying `result`). We leave it open so the caller
	// (lowerBlockToReturn) can terminate it.
	b.UseBlock(mergeBlock)
	return result, true
}

// emitMoveInto writes `src` into `dst` by emitting a synthetic
// `dst = src + 0` sequence. MIR is not strict SSA: an instruction
// whose Dst names an already-defined register is legal and codegen
// renders it as a plain C assignment — exactly what match-arm
// convergence needs.
func (l *lowerer) emitMoveInto(b *mir.Builder, src, dst mir.Reg) {
	// Use Binary with OpAdd against a zero constant. But we need
	// to write into an existing Reg, not a fresh one. The builder
	// doesn't expose "emit Inst with pre-allocated Dst" directly;
	// reach into the current block via the mir package.
	// Construct the Inst by hand.
	zero := b.ConstInt(0)
	blk := b.CurrentBlock()
	if blk == nil {
		return
	}
	blk.Insts = append(blk.Insts, mir.Inst{
		Op: mir.OpAdd, Dst: dst, Lhs: src, Rhs: zero,
	})
}

// lowerArmBody lowers a MatchArm's body block to a single MIR
// register result. At W10 the body is expected to be a block with
// a trailing expression or a single return; we take the trailing
// expression when present.
func (l *lowerer) lowerArmBody(modPath string, b *mir.Builder, arm *hir.MatchArm, params map[string]mir.Reg) (mir.Reg, bool) {
	if arm.Body == nil {
		l.diagnose(arm.Span,
			"match arm has no body",
			"add a body: `<pattern> => <expr>,`")
		return mir.NoReg, false
	}
	if arm.Body.Trailing != nil {
		return l.lowerExpr(modPath, b, arm.Body.Trailing, params)
	}
	// If the body has a single return statement, lower that.
	if len(arm.Body.Stmts) == 1 {
		if r, ok := arm.Body.Stmts[0].(*hir.ReturnStmt); ok && r.Value != nil {
			return l.lowerExpr(modPath, b, r.Value, params)
		}
	}
	l.diagnose(arm.Span,
		"W10 match arms must be a single expression (either a trailing expression or a `return <expr>` statement)",
		"simplify the arm body to a single expression")
	return mir.NoReg, false
}

// armDiscriminants computes the concrete integer discriminant for
// each arm, returning a slice where a nil entry means "wildcard /
// default". Non-enum / non-bool scrutinees produce a diagnostic.
//
// The discriminant for a Bool pattern is 1 (true) or 0 (false).
// The discriminant for an enum variant pattern is the variant's
// index in the enum's declared order.
func (l *lowerer) armDiscriminants(m *hir.MatchExpr) ([]*int64, bool) {
	out := make([]*int64, len(m.Arms))
	scrutType := m.Scrutinee.TypeOf()
	scrutT := l.prog.Types.Get(scrutType)
	for i, arm := range m.Arms {
		switch p := arm.Pattern.(type) {
		case *hir.WildcardPat:
			out[i] = nil
		case *hir.BindPat:
			out[i] = nil // a bare binding is a catch-all
		case *hir.LiteralPat:
			if p.Kind != hir.LitBool {
				l.diagnose(arm.Span,
					"W10 lowerer currently matches only bool-literal and enum-variant patterns; other literals land later",
					"restructure to an if/else chain or wait for W14 const-eval")
				return nil, false
			}
			v := int64(0)
			if p.Bool {
				v = 1
			}
			out[i] = &v
		case *hir.ConstructorPat:
			// Prefer the pattern's ConstructorType (set by the
			// resolver/bridge) over the scrutinee's type — the
			// pattern is the authoritative local source of
			// enum identity at W10.
			enumTypeID := p.ConstructorType
			if enumTypeID == 0 && scrutT != nil {
				enumTypeID = scrutType
			}
			enumT := l.prog.Types.Get(enumTypeID)
			if enumT == nil || enumT.Kind != typetable.KindEnum {
				l.diagnose(arm.Span,
					"constructor pattern on non-enum scrutinee is not lowerable at W10",
					"match an enum scrutinee, or use a literal/wildcard pattern")
				return nil, false
			}
			variant := p.VariantName
			if variant == "" && len(p.Path) > 0 {
				variant = p.Path[len(p.Path)-1]
			}
			idx, ok := l.enumVariantIndex(enumTypeID, variant)
			if !ok {
				l.diagnose(arm.Span,
					fmt.Sprintf("unknown variant %q for enum %q", variant, enumT.Name),
					"match against a variant declared in the enum")
				return nil, false
			}
			v := int64(idx)
			out[i] = &v
		case *hir.OrPat:
			// An or-pattern covering every possible value (e.g.
			// `true | false` for Bool) is a wildcard. Detect
			// total coverage by comparing the alt set against
			// the known primitive / enum variant set; for W10
			// we accept the common case where every alt is a
			// LiteralPat / ConstructorPat and treat it as a
			// wildcard arm. A non-total OrPat is deferred.
			if orIsTotal(p) {
				out[i] = nil
				continue
			}
			l.diagnose(arm.Span,
				"W10 lowerer only covers total or-patterns (e.g. `true | false`); non-total or-patterns land later",
				"split into individual arms, or widen to a wildcard")
			return nil, false
		case *hir.RangePat:
			l.diagnose(arm.Span,
				"W10 lowerer does not yet handle range patterns; const-evaluation arrives in W14",
				"split into concrete arms or add a wildcard")
			return nil, false
		case *hir.AtBindPat:
			// `@`-binding is legal when the inner pattern is a
			// wildcard (a total match with a name). Non-total
			// `@`-binding is deferred.
			if _, totalInner := p.Pattern.(*hir.WildcardPat); totalInner {
				out[i] = nil
				continue
			}
			l.diagnose(arm.Span,
				"W10 lowerer only supports `name @ _` (total @-binding); inner-pattern @-binding lands later",
				"use a total @-binding or split into arms")
			return nil, false
		default:
			l.diagnose(arm.Span,
				"W10 lowerer does not yet cover this pattern form",
				"use a literal, constructor, wildcard, or bind pattern")
			return nil, false
		}
	}
	return out, true
}

// orIsTotal returns true when every alternative in p would cover
// its own case. For W10 we accept the specific "bool full
// coverage" shape: both `true` and `false` literals present.
func orIsTotal(p *hir.OrPat) bool {
	seenTrue, seenFalse := false, false
	for _, a := range p.Alts {
		if lp, ok := a.(*hir.LiteralPat); ok && lp.Kind == hir.LitBool {
			if lp.Bool {
				seenTrue = true
			} else {
				seenFalse = true
			}
		}
	}
	return seenTrue && seenFalse
}

// lowerInlinedClosureBody inlines a closure body at the call
// site. At W12 this is the pragmatic stand-in for
// env-struct-plus-lifted-fn emission: the W05 spine can't yet
// model indirect fn-pointer calls through an env-carrying
// struct (W15 MIR consolidation lands that). Inlining covers
// the W12 proof-path shape for no-capture closures and for
// closures whose captures happen to match the caller's local
// names, which the closure-analysis predicates already map.
//
// Only a single-statement body is accepted — matching the W05
// spine restriction. Anything richer deferred until the spine
// lifts multi-statement bodies in W15+.
func (l *lowerer) lowerInlinedClosureBody(modPath string, b *mir.Builder, body *hir.Block, params map[string]mir.Reg) (mir.Reg, bool) {
	if body == nil {
		l.diagnose(lex.Span{},
			"closure body is nil",
			"this is a bridge bug, not a user error")
		return mir.NoReg, false
	}
	if body.Trailing != nil && len(body.Stmts) == 0 {
		return l.lowerExpr(modPath, b, body.Trailing, params)
	}
	if len(body.Stmts) == 1 {
		if ret, ok := body.Stmts[0].(*hir.ReturnStmt); ok && ret.Value != nil {
			return l.lowerExpr(modPath, b, ret.Value, params)
		}
	}
	l.diagnose(body.NodeSpan(),
		"spine requires a single-expression closure body for inline call",
		"use `fn() -> T { return <expr>; }` or a bare trailing expression")
	return mir.NoReg, false
}

// lowerTry lowers a `?` expression to branch-and-early-return.
//
// Shape (for `e?` where e has enum type EnumT with an `Err` or
// `None` variant at declared index K):
//
//   - Compute `e` into a register R.
//   - If R == K, early-return R from the enclosing fn (this is
//     the "propagate the error" path).
//   - Otherwise, continue with R as the `?` expression's value.
//
// The lowerer leaves the continuation-block open so the
// surrounding expression can use the register. The `?` result
// register is R itself — no payload extraction at W11 because
// enum variants don't yet carry payloads through the pipeline.
// Reference §16 and the wave-doc proof shape: `run(false)`
// propagates Err and the enclosing fn returns Err (exit 43),
// while `run(true)` continues with Ok (exit 0).
func (l *lowerer) lowerTry(modPath string, b *mir.Builder, x *hir.TryExpr, params map[string]mir.Reg) (mir.Reg, bool) {
	if x.Receiver == nil {
		l.diagnose(x.NodeSpan(),
			"`?` with no receiver expression",
			"this is a bridge bug, not a user error")
		return mir.NoReg, false
	}
	recv, ok := l.lowerExpr(modPath, b, x.Receiver, params)
	if !ok {
		return mir.NoReg, false
	}
	recvType := x.Receiver.TypeOf()
	errIdx, ok := l.errorVariantIndex(recvType)
	if !ok {
		l.diagnose(x.NodeSpan(),
			"`?` receiver's enum has no `Err` / `None` variant",
			"declare the error case as `Err` or `None`; the checker normally catches this earlier")
		return mir.NoReg, false
	}
	// Allocate a block for early-return (the Err/None arm) and a
	// block for the continuation (the success arm). The current
	// block then ends with TermIfEq.
	errBlock := b.BeginBlock()
	okBlock := b.BeginBlock()
	// Re-enter the block that holds `recv` to emit the branch.
	// The block immediately preceding errBlock is the one where
	// we computed recv.
	recvBlock := errBlock - 1
	b.UseBlock(recvBlock)
	b.IfEq(recv, int64(errIdx), errBlock, okBlock)
	// Err arm: return the receiver unchanged (propagate the error).
	b.UseBlock(errBlock)
	b.Return(recv)
	// Success arm: the caller continues emission in okBlock with
	// `recv` as the `?` expression's value.
	b.UseBlock(okBlock)
	return recv, true
}

// errorVariantIndex returns the 0-based index of the `Err` or
// `None` variant within the enum that `enumType` names. Returns
// ok=false when enumType is not an enum or the enum has neither
// variant.
func (l *lowerer) errorVariantIndex(enumType typetable.TypeId) (int, bool) {
	t := l.prog.Types.Get(enumType)
	if t == nil || t.Kind != typetable.KindEnum {
		return 0, false
	}
	for _, modPath := range l.prog.Order {
		m := l.prog.Modules[modPath]
		for _, it := range m.Items {
			ed, ok := it.(*hir.EnumDecl)
			if !ok || ed.TypeID != enumType {
				continue
			}
			for idx, v := range ed.Variants {
				if v.Name == "Err" || v.Name == "None" {
					return idx, true
				}
			}
			return 0, false
		}
	}
	return 0, false
}

// lookupEnumVariantByName scans every enum declaration in the
// program for one matching enumName, then returns the 0-based
// index of variantName within that enum. Used by path lowering
// to handle `EnumName.VariantName` value expressions.
func (l *lowerer) lookupEnumVariantByName(enumName, variantName string) (int, bool) {
	for _, modPath := range l.prog.Order {
		m := l.prog.Modules[modPath]
		for _, it := range m.Items {
			if ed, ok := it.(*hir.EnumDecl); ok && ed.Name == enumName {
				for idx, v := range ed.Variants {
					if v.Name == variantName {
						return idx, true
					}
				}
			}
		}
	}
	return 0, false
}

// enumVariantIndex returns the 0-based index of variantName in the
// enum's declared variant list. Returns ok=false when variant is
// not declared.
func (l *lowerer) enumVariantIndex(enumType typetable.TypeId, variantName string) (int, bool) {
	for _, modPath := range l.prog.Order {
		m := l.prog.Modules[modPath]
		for _, it := range m.Items {
			if ed, ok := it.(*hir.EnumDecl); ok && ed.TypeID == enumType {
				for idx, v := range ed.Variants {
					if v.Name == variantName {
						return idx, true
					}
				}
			}
		}
	}
	return 0, false
}

// cName returns the C-level identifier for a fn defined in modPath
// with the given Fuse name. Matches the codegen naming scheme so
// OpCall's CallName field resolves at C compile time.
func cName(modPath, name string) string {
	if name == "main" {
		return "main"
	}
	if modPath == "" {
		return "fuse_" + name
	}
	return "fuse_" + modPath + "__" + name
}

// mapBinaryOp translates a HIR BinaryOp into the matching MIR Op for
// the W05 subset. Returns ok=false for anything outside that subset.
func mapBinaryOp(op hir.BinaryOp) (mir.Op, bool) {
	switch op {
	case hir.BinAdd:
		return mir.OpAdd, true
	case hir.BinSub:
		return mir.OpSub, true
	case hir.BinMul:
		return mir.OpMul, true
	case hir.BinDiv:
		return mir.OpDiv, true
	case hir.BinMod:
		return mir.OpMod, true
	}
	return mir.OpInvalid, false
}

func binOpName(op hir.BinaryOp) string {
	switch op {
	case hir.BinAdd:
		return "+"
	case hir.BinSub:
		return "-"
	case hir.BinMul:
		return "*"
	case hir.BinDiv:
		return "/"
	case hir.BinMod:
		return "%"
	case hir.BinShl:
		return "<<"
	case hir.BinShr:
		return ">>"
	case hir.BinAnd:
		return "&"
	case hir.BinOr:
		return "|"
	case hir.BinXor:
		return "^"
	case hir.BinLogAnd:
		return "&&"
	case hir.BinLogOr:
		return "||"
	case hir.BinEq:
		return "=="
	case hir.BinNe:
		return "!="
	case hir.BinLt:
		return "<"
	case hir.BinLe:
		return "<="
	case hir.BinGt:
		return ">"
	case hir.BinGe:
		return ">="
	}
	return "unknown"
}

func litKindName(k hir.LitKind) string {
	switch k {
	case hir.LitInt:
		return "integer"
	case hir.LitFloat:
		return "float"
	case hir.LitString:
		return "string"
	case hir.LitRawString:
		return "raw string"
	case hir.LitCString:
		return "C string"
	case hir.LitChar:
		return "char"
	case hir.LitBool:
		return "boolean"
	case hir.LitNone:
		return "None"
	}
	return "literal"
}

func (l *lowerer) diagnose(span lex.Span, msg, hint string) {
	l.diags = append(l.diags, Diagnostic{Span: span, Message: msg, Hint: hint})
}

// parseIntLiteral accepts Go-style integer literals plus Fuse type
// suffixes (e.g. `42u64`, `0xFFi32`, `0b1010u8`). Underscores are
// tolerated as digit separators (reference §1.5); the checker has
// already validated that the suffix matches the declared type so
// the lowerer can discard it before parsing.
func parseIntLiteral(text string) (int64, error) {
	clean := strings.ReplaceAll(text, "_", "")
	for _, suf := range []string{"isize", "usize", "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64"} {
		if strings.HasSuffix(clean, suf) {
			clean = clean[:len(clean)-len(suf)]
			break
		}
	}
	// Large unsigned literals (e.g. 0xFFFFFFFFu32) may overflow
	// int64 when parsed as signed; fall back to unsigned and
	// reinterpret the bits so the lowered MIR preserves the same
	// binary value.
	if v, err := strconv.ParseInt(clean, 0, 64); err == nil {
		return v, nil
	}
	u, err := strconv.ParseUint(clean, 0, 64)
	if err != nil {
		return 0, err
	}
	return int64(u), nil
}
