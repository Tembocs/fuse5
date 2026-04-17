package lower

import (
	"fmt"
	"strconv"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/mir"
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
	prog   *hir.Program
	module *mir.Module
	diags  []Diagnostic
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
		if x.Kind != hir.LitInt {
			l.diagnose(x.NodeSpan(),
				fmt.Sprintf("spine only lowers integer literals, not %s", litKindName(x.Kind)),
				"replace the literal with an integer")
			return mir.NoReg, false
		}
		v, err := strconv.ParseInt(x.Text, 0, 64)
		if err != nil {
			l.diagnose(x.NodeSpan(),
				fmt.Sprintf("invalid integer literal %q: %v", x.Text, err),
				"write a value in the signed 64-bit range")
			return mir.NoReg, false
		}
		return b.ConstInt(v), true
	case *hir.BinaryExpr:
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
		// Multi-segment paths (module-qualified items) are only
		// meaningful as callees; on their own they cannot produce
		// an integer value yet, so we diagnose.
		if len(x.Segments) == 1 {
			if reg, ok := params[x.Segments[0]]; ok {
				return reg, true
			}
			l.diagnose(x.NodeSpan(),
				fmt.Sprintf("spine does not yet resolve %q as a value", x.Segments[0]),
				"use a parameter name or inline the expression")
			return mir.NoReg, false
		}
		l.diagnose(x.NodeSpan(),
			"spine does not yet lower module-qualified path values",
			"call the fn directly via `name(args)`")
		return mir.NoReg, false
	case *hir.CallExpr:
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
	default:
		l.diagnose(e.NodeSpan(),
			fmt.Sprintf("spine does not yet lower %T", e),
			"the W06 spine supports int literals, +/-/*// arithmetic, parameter reads, and direct fn calls")
		return mir.NoReg, false
	}
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
