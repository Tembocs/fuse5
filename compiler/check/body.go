package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// checkBodies is pass 2. It walks every function's body with the
// fn's signature context and types every expression it finds. The
// expected-type parameter carries the context's "wanted" type down
// the recursion (reference §5.8 contextual inference).
func (c *checker) checkBodies() {
	for _, modPath := range c.prog.Order {
		m := c.prog.Modules[modPath]
		for _, it := range m.Items {
			c.checkBodyOfItem(modPath, it)
		}
	}
}

func (c *checker) checkBodyOfItem(modPath string, it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		c.checkFnBody(modPath, x)
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			if fn, ok := sub.(*hir.FnDecl); ok {
				c.checkFnBody(modPath, fn)
			}
		}
	case *hir.TraitDecl:
		// Trait bodies are signatures only at W06; default method
		// implementations in traits (when they land) will also
		// flow through checkFnBody here.
		for _, sub := range x.Items {
			if fn, ok := sub.(*hir.FnDecl); ok && fn.Body != nil {
				c.checkFnBody(modPath, fn)
			}
		}
	case *hir.ConstDecl:
		if x.Value != nil {
			c.checkExpr(modPath, &bodyScope{module: modPath}, x.Value, x.Type)
		}
	case *hir.StaticDecl:
		if x.Value != nil {
			c.checkExpr(modPath, &bodyScope{module: modPath}, x.Value, x.Type)
		}
	}
}

// bodyScope is the lexical scope inside a function body. It chains
// to a parent so that nested blocks introduce their own bindings.
// At W06 we only track locals introduced by `let`/`var` and by
// function parameters; full shadowing and capture analysis come
// with W12 closures.
type bodyScope struct {
	parent  *bodyScope
	module  string
	locals  map[string]typetable.TypeId
	retType typetable.TypeId // the enclosing fn's return type
	selfT   typetable.TypeId // Self TypeId inside impl/trait bodies
}

func newBodyScope(parent *bodyScope, module string, retType, selfT typetable.TypeId) *bodyScope {
	return &bodyScope{
		parent:  parent,
		module:  module,
		locals:  map[string]typetable.TypeId{},
		retType: retType,
		selfT:   selfT,
	}
}

// bind registers a local name with its TypeId. Duplicate names in
// the same scope shadow the prior binding (matching §6 let/var
// shadowing rules).
func (s *bodyScope) bind(name string, tid typetable.TypeId) {
	if s.locals == nil {
		s.locals = map[string]typetable.TypeId{}
	}
	s.locals[name] = tid
}

// lookup walks the parent chain to find a local binding.
func (s *bodyScope) lookup(name string) (typetable.TypeId, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if t, ok := cur.locals[name]; ok {
			return t, true
		}
	}
	return typetable.NoType, false
}

// returnType returns the enclosing fn's return TypeId by walking
// parents until one is found. A missing retType means the checker
// is operating on a const/static initializer — callers check for
// NoType before using it.
func (s *bodyScope) returnType() typetable.TypeId {
	for cur := s; cur != nil; cur = cur.parent {
		if cur.retType != typetable.NoType {
			return cur.retType
		}
	}
	return typetable.NoType
}

// checkFnBody types the parameters and body of fn.
func (c *checker) checkFnBody(modPath string, fn *hir.FnDecl) {
	c.stats.FunctionsChecked++
	if fn.Body == nil {
		return // extern / trait signature-only
	}
	scope := newBodyScope(nil, modPath, fn.Return, typetable.NoType)
	for _, p := range fn.Params {
		if p.TypeOf() == c.tab.Infer() || p.TypeOf() == typetable.NoType {
			c.diagnose(p.Span, fmt.Sprintf("parameter %q has no declared type", p.Name),
				"add a type annotation: `name: T`")
			continue
		}
		scope.bind(p.Name, p.TypeOf())
	}
	c.checkBlock(modPath, scope, fn.Body, fn.Return)
}

// checkBlock types a Block against an expected type. Returns the
// block's concrete TypeId after checking.
func (c *checker) checkBlock(modPath string, scope *bodyScope, blk *hir.Block, expected typetable.TypeId) typetable.TypeId {
	if blk == nil {
		return c.tab.Unit()
	}
	for _, s := range blk.Stmts {
		c.checkStmt(modPath, scope, s)
	}
	trail := c.tab.Unit()
	if blk.Trailing != nil {
		trail = c.checkExpr(modPath, scope, blk.Trailing, expected)
	}
	blk.Type = trail
	return trail
}

// checkStmt types a statement. Statements do not themselves carry
// TypeIds (the Node interface doesn't require it); checking flows
// through to the expressions inside them.
func (c *checker) checkStmt(modPath string, scope *bodyScope, s hir.Stmt) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.Value == nil {
			if x.DeclaredType == c.tab.Infer() {
				c.diagnose(x.Span, "let binding without initializer must have a type annotation",
					"write `let x: T;`")
			}
			return
		}
		// Expected type: the explicit annotation if present, else infer.
		expected := x.DeclaredType
		got := c.checkExpr(modPath, scope, x.Value, expected)
		if expected == c.tab.Infer() || expected == typetable.NoType {
			x.DeclaredType = got
			expected = got
		}
		// Bind the pattern's names with the resolved type.
		c.bindPatternNames(scope, x.Pattern, expected)
	case *hir.VarStmt:
		expected := x.DeclaredType
		got := c.checkExpr(modPath, scope, x.Value, expected)
		if expected == c.tab.Infer() {
			x.DeclaredType = got
			expected = got
		}
		scope.bind(x.Name, expected)
	case *hir.ReturnStmt:
		ret := scope.returnType()
		if x.Value == nil {
			if ret != c.tab.Unit() {
				c.diagnose(x.Span,
					fmt.Sprintf("bare `return` in fn returning %s", c.typeName(ret)),
					"return a value or change the return type to Unit")
			}
			return
		}
		got := c.checkExpr(modPath, scope, x.Value, ret)
		if !c.isAssignable(got, ret) {
			c.diagnose(x.Value.NodeSpan(),
				fmt.Sprintf("return value type %s does not match fn return %s",
					c.typeName(got), c.typeName(ret)),
				"adjust the expression or the fn's return type")
		}
	case *hir.BreakStmt:
		if x.Value != nil {
			c.checkExpr(modPath, scope, x.Value, typetable.NoType)
		}
	case *hir.ContinueStmt:
		// nothing to type
	case *hir.ExprStmt:
		if x.Expr != nil {
			c.checkExpr(modPath, scope, x.Expr, typetable.NoType)
		}
	case *hir.ItemStmt:
		if x.Item != nil {
			c.checkBodyOfItem(modPath, x.Item)
		}
	}
}

// bindPatternNames extracts identifiers from structured patterns and
// binds them in scope with the scrutinee's type. Full pattern
// typing for enum variants and struct destructuring is W10 work;
// at W06 we support BindPat (a plain name), WildcardPat, and
// ConstructorPat fields at the shallow level.
func (c *checker) bindPatternNames(scope *bodyScope, p hir.Pat, tid typetable.TypeId) {
	if p == nil {
		return
	}
	switch x := p.(type) {
	case *hir.BindPat:
		scope.bind(x.Name, tid)
		x.Type = tid
	case *hir.WildcardPat:
		x.Type = tid
	case *hir.ConstructorPat:
		x.Type = tid
		for _, f := range x.Fields {
			if bp, ok := f.Pattern.(*hir.BindPat); ok {
				scope.bind(bp.Name, c.tab.Infer())
			}
		}
	}
}

// isAssignable returns true when a value of type `from` may be used
// where a value of type `to` is expected. At W06 the rule is:
// identical TypeIds, or either side is KindInfer (treated
// optimistically during inference), or numeric widening is legal
// (checked via isWidenable).
func (c *checker) isAssignable(from, to typetable.TypeId) bool {
	if from == typetable.NoType || to == typetable.NoType {
		return true
	}
	if from == to {
		return true
	}
	if from == c.tab.Infer() || to == c.tab.Infer() {
		return true
	}
	if c.isWidenable(from, to) {
		return true
	}
	// Never type is assignable to anything.
	if c.tab.Get(from) != nil && c.tab.Get(from).Kind == typetable.KindNever {
		return true
	}
	return false
}

// isWidenable reports whether `from` widens to `to` under the
// reference §5.8 numeric lattice. The lattice is:
//
//   I8 ⊂ I16 ⊂ I32 ⊂ I64 ⊂ ISize
//   U8 ⊂ U16 ⊂ U32 ⊂ U64 ⊂ USize
//   F32 ⊂ F64
//
// Cross-sign widening is not implicit (requires an explicit cast).
func (c *checker) isWidenable(from, to typetable.TypeId) bool {
	fr := c.tab.Get(from)
	tr := c.tab.Get(to)
	if fr == nil || tr == nil {
		return false
	}
	fRank, fFam := primitiveRank(fr.Kind)
	tRank, tFam := primitiveRank(tr.Kind)
	if fFam == "" || fFam != tFam {
		return false
	}
	return fRank <= tRank
}

// primitiveRank returns (rank, family) for a numeric primitive. The
// family groups signed-int, unsigned-int, and float so widening
// stays within the same signedness.
func primitiveRank(k typetable.Kind) (int, string) {
	switch k {
	case typetable.KindI8:
		return 1, "int-signed"
	case typetable.KindI16:
		return 2, "int-signed"
	case typetable.KindI32:
		return 3, "int-signed"
	case typetable.KindI64:
		return 4, "int-signed"
	case typetable.KindISize:
		return 5, "int-signed"
	case typetable.KindU8:
		return 1, "int-unsigned"
	case typetable.KindU16:
		return 2, "int-unsigned"
	case typetable.KindU32:
		return 3, "int-unsigned"
	case typetable.KindU64:
		return 4, "int-unsigned"
	case typetable.KindUSize:
		return 5, "int-unsigned"
	case typetable.KindF32:
		return 1, "float"
	case typetable.KindF64:
		return 2, "float"
	}
	return 0, ""
}

// Silence unused-import warnings; lex is kept in scope for later
// additions to the checker without re-import churn.
var _ = lex.Span{}
