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
// the fn cannot be lowered under the W05 contract.
func (l *lowerer) lowerFn(modPath string, fn *hir.FnDecl) *mir.Function {
	if fn.Body == nil {
		// Extern declarations are not lowered at W05. They are
		// preserved in HIR; W17 drives their codegen.
		return nil
	}
	if len(fn.Params) != 0 {
		l.diagnose(fn.Span,
			"W05 spine does not yet lower fn parameters",
			"remove parameters or declare a zero-arg entry point")
		return nil
	}
	mirFn, b := mir.NewFunction(modPath, fn.Name)
	if !l.lowerBlockToReturn(b, fn.Body) {
		return nil
	}
	if err := mirFn.Validate(); err != nil {
		l.diagnose(fn.Span,
			fmt.Sprintf("MIR validation failed for fn %q: %v", fn.Name, err),
			"the W05 lowerer produced an instruction the MIR does not yet accept")
		return nil
	}
	return mirFn
}

// lowerBlockToReturn expects the HIR block to be a straight-line
// body ending in `return EXPR;`. At W05, trailing expressions,
// multiple statements, and locals are all rejected.
func (l *lowerer) lowerBlockToReturn(b *mir.Builder, blk *hir.Block) bool {
	if blk == nil {
		l.diagnose(lex.Span{}, "W05 spine requires a fn body", "add a body with `return <expr>;`")
		return false
	}
	if blk.Trailing != nil {
		l.diagnose(blk.NodeSpan(),
			"W05 spine does not yet lower trailing block expressions",
			"use an explicit `return <expr>;` statement")
		return false
	}
	if len(blk.Stmts) != 1 {
		l.diagnose(blk.NodeSpan(),
			"W05 spine supports exactly one statement per fn body",
			"the minimal spine shape is `fn main() -> I32 { return <expr>; }`")
		return false
	}
	ret, ok := blk.Stmts[0].(*hir.ReturnStmt)
	if !ok {
		l.diagnose(blk.Stmts[0].NodeSpan(),
			"W05 spine requires the single body statement to be a `return`",
			"use `return <expr>;` as the only body statement")
		return false
	}
	if ret.Value == nil {
		l.diagnose(ret.NodeSpan(),
			"W05 spine requires `return` to carry an integer expression",
			"write `return 0;` at minimum")
		return false
	}
	reg, okExpr := l.lowerExpr(b, ret.Value)
	if !okExpr {
		return false
	}
	b.Return(reg)
	return true
}

// lowerExpr lowers one integer-producing expression and returns the
// register holding its value. Returns ok=false (after emitting a
// diagnostic) when the expression uses any form the W05 spine does
// not yet support.
func (l *lowerer) lowerExpr(b *mir.Builder, e hir.Expr) (mir.Reg, bool) {
	switch x := e.(type) {
	case *hir.LiteralExpr:
		if x.Kind != hir.LitInt {
			l.diagnose(x.NodeSpan(),
				fmt.Sprintf("W05 spine only lowers integer literals, not %s", litKindName(x.Kind)),
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
				fmt.Sprintf("W05 spine does not yet lower binary operator %s", binOpName(x.Op)),
				"use `+`, `-`, `*`, `/`, or `%`")
			return mir.NoReg, false
		}
		lhs, ok := l.lowerExpr(b, x.Lhs)
		if !ok {
			return mir.NoReg, false
		}
		rhs, ok := l.lowerExpr(b, x.Rhs)
		if !ok {
			return mir.NoReg, false
		}
		return b.Binary(op, lhs, rhs), true
	default:
		l.diagnose(e.NodeSpan(),
			fmt.Sprintf("W05 spine does not yet lower %T", e),
			"the minimal spine supports integer literals and +/-/*// arithmetic only")
		return mir.NoReg, false
	}
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
