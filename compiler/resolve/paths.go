package resolve

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// resolveAllPaths walks every surviving item in every module and
// resolves the path occurrences it contains (PathExpr in expressions,
// PathType in type positions, CtorPat in patterns). Resolution failures
// emit diagnostics; successful lookups record a binding keyed by
// (module, span) into Resolved.Bindings.
func (r *resolver) resolveAllPaths() {
	for _, modPath := range r.graph.Order {
		m := r.graph.Modules[modPath]
		for _, it := range m.Items {
			r.walkItem(m, it)
		}
	}
}

// walkItem dispatches on the AST item kind and descends into every
// position that may contain a path. Items like `impl` have nested item
// lists; each is walked in the module's scope.
func (r *resolver) walkItem(m *Module, it ast.Item) {
	switch x := it.(type) {
	case *ast.FnDecl:
		r.walkFn(m, x)
	case *ast.StructDecl:
		for _, f := range x.Fields {
			r.walkType(m, f.Type)
		}
		for _, t := range x.Tuple {
			r.walkType(m, t)
		}
	case *ast.EnumDecl:
		for _, v := range x.Variants {
			for _, t := range v.Tuple {
				r.walkType(m, t)
			}
			for _, f := range v.Fields {
				r.walkType(m, f.Type)
			}
			if v.Explicit != nil {
				r.walkExpr(m, v.Explicit)
			}
		}
	case *ast.TraitDecl:
		for _, s := range x.Supertrs {
			r.walkType(m, s)
		}
		for _, sub := range x.Items {
			r.walkItem(m, sub)
		}
	case *ast.ImplDecl:
		r.walkType(m, x.Target)
		if x.Trait != nil {
			r.walkType(m, x.Trait)
		}
		for _, sub := range x.Items {
			r.walkItem(m, sub)
		}
	case *ast.ConstDecl:
		r.walkType(m, x.Type)
		if x.Value != nil {
			r.walkExpr(m, x.Value)
		}
	case *ast.StaticDecl:
		r.walkType(m, x.Type)
		if x.Value != nil {
			r.walkExpr(m, x.Value)
		}
	case *ast.TypeDecl:
		r.walkType(m, x.Target)
	case *ast.UnionDecl:
		for _, f := range x.Fields {
			r.walkType(m, f.Type)
		}
	case *ast.ExternDecl:
		if x.Item != nil {
			r.walkItem(m, x.Item)
		}
	case *ast.TraitTypeItem:
		for _, b := range x.Bounds {
			r.walkType(m, b)
		}
	case *ast.TraitConstItem:
		r.walkType(m, x.Type)
		if x.Default != nil {
			r.walkExpr(m, x.Default)
		}
	case *ast.ImplTypeItem:
		r.walkType(m, x.Target)
	}
}

func (r *resolver) walkFn(m *Module, fn *ast.FnDecl) {
	for _, p := range fn.Params {
		r.walkType(m, p.Type)
	}
	if fn.Return != nil {
		r.walkType(m, fn.Return)
	}
	if fn.Body != nil {
		r.walkExpr(m, fn.Body)
	}
}

// walkExpr recursively resolves paths inside an expression. The
// resolver does not attempt to track local bindings at this wave —
// W04 owns that when HIR lowering introduces its own scope. Here we
// resolve every PathExpr against the module scope and record the
// binding; unresolved PathExprs that look like locals (single-segment,
// no type args, resolve to nothing) are left silent because they may
// refer to a local let-binding that this pass has no knowledge of.
// Multi-segment PathExprs MUST resolve because they are module- or
// enum-qualified.
func (r *resolver) walkExpr(m *Module, e ast.Expr) {
	switch x := e.(type) {
	case *ast.PathExpr:
		r.resolvePathExpr(m, x)
	case *ast.BinaryExpr:
		r.walkExpr(m, x.Lhs)
		r.walkExpr(m, x.Rhs)
	case *ast.AssignExpr:
		r.walkExpr(m, x.Lhs)
		r.walkExpr(m, x.Rhs)
	case *ast.UnaryExpr:
		r.walkExpr(m, x.Operand)
	case *ast.CastExpr:
		r.walkExpr(m, x.Expr)
		r.walkType(m, x.Type)
	case *ast.CallExpr:
		r.walkExpr(m, x.Callee)
		for _, a := range x.Args {
			r.walkExpr(m, a)
		}
	case *ast.FieldExpr:
		if r.tryResolveFieldChainAsPath(m, x) {
			// The chain was a static path rooted at a module, enum, or
			// import alias; the flattened-path binding is recorded.
			// We intentionally do not descend into the receiver again.
			return
		}
		r.walkExpr(m, x.Receiver)
	case *ast.OptFieldExpr:
		r.walkExpr(m, x.Receiver)
	case *ast.TryExpr:
		r.walkExpr(m, x.Receiver)
	case *ast.IndexExpr:
		r.walkExpr(m, x.Receiver)
		r.walkExpr(m, x.Index)
	case *ast.IndexRangeExpr:
		r.walkExpr(m, x.Receiver)
		if x.Low != nil {
			r.walkExpr(m, x.Low)
		}
		if x.High != nil {
			r.walkExpr(m, x.High)
		}
	case *ast.BlockExpr:
		for _, s := range x.Stmts {
			r.walkStmt(m, s)
		}
		if x.Trailing != nil {
			r.walkExpr(m, x.Trailing)
		}
	case *ast.IfExpr:
		r.walkExpr(m, x.Cond)
		r.walkExpr(m, x.Then)
		if x.Else != nil {
			r.walkExpr(m, x.Else)
		}
	case *ast.MatchExpr:
		r.walkExpr(m, x.Scrutinee)
		for _, arm := range x.Arms {
			r.walkPat(m, arm.Pattern)
			if arm.Guard != nil {
				r.walkExpr(m, arm.Guard)
			}
			r.walkExpr(m, arm.Body)
		}
	case *ast.LoopExpr:
		r.walkExpr(m, x.Body)
	case *ast.WhileExpr:
		r.walkExpr(m, x.Cond)
		r.walkExpr(m, x.Body)
	case *ast.ForExpr:
		r.walkPat(m, x.Pattern)
		r.walkExpr(m, x.Iter)
		r.walkExpr(m, x.Body)
	case *ast.TupleExpr:
		for _, el := range x.Elements {
			r.walkExpr(m, el)
		}
	case *ast.StructLitExpr:
		if x.Path != nil {
			r.resolvePathExpr(m, x.Path)
		}
		for _, f := range x.Fields {
			if f.Value != nil {
				r.walkExpr(m, f.Value)
			}
		}
		if x.Base != nil {
			r.walkExpr(m, x.Base)
		}
	case *ast.ClosureExpr:
		for _, p := range x.Params {
			r.walkType(m, p.Type)
		}
		if x.Return != nil {
			r.walkType(m, x.Return)
		}
		if x.Body != nil {
			r.walkExpr(m, x.Body)
		}
	case *ast.SpawnExpr:
		if x.Inner != nil {
			r.walkExpr(m, x.Inner)
		}
	case *ast.UnsafeExpr:
		if x.Body != nil {
			r.walkExpr(m, x.Body)
		}
	case *ast.ParenExpr:
		r.walkExpr(m, x.Inner)
	}
}

// walkStmt dispatches statement forms. We walk via the statement's
// concrete shape through a small shim so callers don't import every
// ast.Stmt implementation name.
func (r *resolver) walkStmt(m *Module, s ast.Stmt) {
	walkStmtExprs(s, func(e ast.Expr) { r.walkExpr(m, e) }, func(p ast.Pat) { r.walkPat(m, p) }, func(t ast.Type) { r.walkType(m, t) })
}

// walkType resolves path-typed nodes. Composed types descend recursively.
func (r *resolver) walkType(m *Module, t ast.Type) {
	switch x := t.(type) {
	case *ast.PathType:
		r.resolvePathType(m, x)
	case *ast.TupleType:
		for _, el := range x.Elements {
			r.walkType(m, el)
		}
	case *ast.ArrayType:
		r.walkType(m, x.Element)
		if x.Length != nil {
			r.walkExpr(m, x.Length)
		}
	case *ast.SliceType:
		r.walkType(m, x.Element)
	case *ast.PtrType:
		r.walkType(m, x.Pointee)
	case *ast.FnType:
		for _, p := range x.Params {
			r.walkType(m, p)
		}
		if x.Return != nil {
			r.walkType(m, x.Return)
		}
	case *ast.DynType:
		for _, tr := range x.Traits {
			r.walkType(m, tr)
		}
	case *ast.ImplType:
		if x.Trait != nil {
			r.walkType(m, x.Trait)
		}
	}
}

// walkPat resolves path-carrying patterns (CtorPat). Inner patterns are
// walked recursively.
func (r *resolver) walkPat(m *Module, p ast.Pat) {
	switch x := p.(type) {
	case *ast.CtorPat:
		r.resolveCtorPat(m, x)
		for _, sub := range x.Tuple {
			r.walkPat(m, sub)
		}
		for _, f := range x.Struct {
			if f.Pattern != nil {
				r.walkPat(m, f.Pattern)
			}
		}
	case *ast.TuplePat:
		for _, el := range x.Elements {
			r.walkPat(m, el)
		}
	case *ast.OrPat:
		for _, a := range x.Alts {
			r.walkPat(m, a)
		}
	case *ast.AtPat:
		if x.Pattern != nil {
			r.walkPat(m, x.Pattern)
		}
	}
}

// resolvePathExpr resolves an expression path (PathExpr). The rules are:
//
//   - single-segment path: try the module scope; if found, bind.
//     Otherwise leave silent (may be a local binding — W04 settles this).
//   - multi-segment path: the prefix must name a module (possibly via
//     an import alias); the final segment must be an item in that
//     module. Unresolvable multi-segment paths emit a diagnostic.
//   - `Enum.Variant`: when the first segment resolves to a local enum,
//     the second segment is looked up as a variant of that enum
//     (reference §11.6).
func (r *resolver) resolvePathExpr(m *Module, p *ast.PathExpr) {
	if len(p.Segments) == 0 {
		return
	}
	id := r.resolvePath(m, p.Segments, p.NodeSpan(), true)
	if id != NoSymbol {
		r.recordBinding(m.Path, p.NodeSpan(), id)
	}
}

// resolvePathType resolves a type path. Primitive names and
// single-segment non-primitive names (which may be generic params
// introduced by an enclosing fn/impl) are tolerated silently; the
// type checker in W06 is responsible for the final "is this really
// a type" judgment. Multi-segment type paths (`mod.Type`) are
// strict because they can only refer to module-qualified items.
func (r *resolver) resolvePathType(m *Module, t *ast.PathType) {
	if len(t.Segments) == 0 {
		return
	}
	// Known primitive type names are skipped (their resolution lives in
	// W04's TypeTable). The resolver only binds user-defined type paths.
	if len(t.Segments) == 1 && isPrimitiveTypeName(t.Segments[0].Name) {
		return
	}
	// Single-segment user type names may be generic parameters that
	// the checker tracks; stay silent on a miss so generics work.
	id := r.resolvePath(m, t.Segments, t.NodeSpan(), true)
	if id != NoSymbol {
		r.recordBinding(m.Path, t.NodeSpan(), id)
	}
}

// resolveCtorPat resolves the path portion of a constructor pattern.
// Enum variants are required to resolve (reference §11.6); plain single
// identifiers that do not match an in-scope constructor are re-treated
// by the parser as BindPat at earlier stages, so by the time we see a
// CtorPat with a single segment, it *does* refer to a constructor.
func (r *resolver) resolveCtorPat(m *Module, c *ast.CtorPat) {
	if len(c.Path) == 0 {
		return
	}
	id := r.resolvePath(m, c.Path, c.NodeSpan(), false)
	if id != NoSymbol {
		r.recordBinding(m.Path, c.NodeSpan(), id)
	}
}

// resolvePath is the shared workhorse for PathExpr, PathType, and
// CtorPat. silentOnSingle controls whether a failed single-segment
// lookup emits a diagnostic: expression paths are silent (may be a
// local), but type/constructor paths are not.
func (r *resolver) resolvePath(m *Module, segs []ast.Ident, span lex.Span, silentOnSingle bool) SymbolID {
	if len(segs) == 1 {
		id := m.Scope.Lookup(segs[0].Name)
		if id != NoSymbol {
			return r.followAlias(id)
		}
		if !silentOnSingle {
			r.diagnose(segs[0].Span,
				fmt.Sprintf("unresolved name %q", segs[0].Name),
				fmt.Sprintf("no item or import named %q in module %q", segs[0].Name, m.Path))
		}
		return NoSymbol
	}
	// Multi-segment: first segment must be a module or an enum.
	firstID := m.Scope.Lookup(segs[0].Name)
	if firstID == NoSymbol {
		// It may be a directly-spelled module path not aliased locally:
		// `std.io.stdout` where `std` is not imported. Try the full
		// prefix match against the module graph.
		if id := r.resolveViaModuleGraph(segs, span); id != NoSymbol {
			return id
		}
		r.diagnose(segs[0].Span,
			fmt.Sprintf("unresolved path %q", pathString(segs)),
			fmt.Sprintf("no item, import, or module named %q in scope", segs[0].Name))
		return NoSymbol
	}
	firstID = r.followAlias(firstID)
	first := r.symbols.Get(firstID)

	// Enum.Variant (reference §11.6).
	if first.Kind == SymEnum && len(segs) == 2 {
		targetMod := r.graph.Modules[first.Module]
		if targetMod != nil {
			if vid := targetMod.Scope.LookupLocal(segs[1].Name); vid != NoSymbol {
				v := r.symbols.Get(vid)
				if v.Kind == SymEnumVariant && v.Parent == first.ID {
					return vid
				}
			}
		}
		r.diagnose(segs[1].Span,
			fmt.Sprintf("enum %q has no variant %q", first.Name, segs[1].Name),
			"variants are spelled exactly as in the enum definition")
		return NoSymbol
	}

	// Module prefix (possibly traversed through an alias).
	if first.Kind == SymModule || (first.Kind == SymImportAlias && first.ModulePath != "") {
		modPath := first.Module
		if first.Kind == SymImportAlias {
			modPath = first.ModulePath
		}
		return r.walkModulePath(modPath, segs[1:], span)
	}

	r.diagnose(span,
		fmt.Sprintf("path %q does not start with a module or enum", pathString(segs)),
		"module-qualified paths need a module or an enum as the first segment")
	return NoSymbol
}

// followAlias collapses an import alias to its target symbol. If the
// alias points at a module (rather than an item), the alias itself is
// returned because the caller needs the alias record (the ModulePath
// field) to continue traversal.
func (r *resolver) followAlias(id SymbolID) SymbolID {
	for i := 0; i < 8; i++ {
		s := r.symbols.Get(id)
		if s == nil || s.Kind != SymImportAlias {
			return id
		}
		if s.Target == NoSymbol {
			return id
		}
		id = s.Target
	}
	return id
}

// walkModulePath traverses module-qualified path segments starting
// from a known module. The final segment names an item; intermediate
// segments must be submodules — except for the `Enum.Variant` tail,
// where an intermediate segment may name an enum whose final segment
// is one of its variants (reference §11.6).
func (r *resolver) walkModulePath(startMod string, rest []ast.Ident, span lex.Span) SymbolID {
	cur := startMod
	for i, seg := range rest {
		if i == len(rest)-1 {
			return r.lookupItemInModule(cur, seg)
		}
		// Last-two special case: `module.Enum.Variant`.
		if i == len(rest)-2 {
			if enumID := r.lookupEnumInModule(cur, seg.Name); enumID != NoSymbol {
				tail := rest[len(rest)-1]
				return r.lookupVariantOfEnum(enumID, tail)
			}
		}
		next := joinModulePath(cur, seg.Name)
		if _, ok := r.graph.Modules[next]; !ok {
			r.diagnose(seg.Span,
				fmt.Sprintf("unknown submodule %q of %q", seg.Name, cur),
				"no module with this dotted path exists in the build")
			return NoSymbol
		}
		cur = next
	}
	return NoSymbol
}

// lookupItemInModule resolves a single-segment name as an item inside
// the module at modPath. Emits a diagnostic and returns NoSymbol when
// the lookup fails.
func (r *resolver) lookupItemInModule(modPath string, seg ast.Ident) SymbolID {
	m := r.graph.Modules[modPath]
	if m == nil {
		r.diagnose(seg.Span,
			fmt.Sprintf("unknown module %q", modPath),
			"the module was not discovered by the resolver")
		return NoSymbol
	}
	id := m.Scope.LookupLocal(seg.Name)
	if id == NoSymbol {
		r.diagnose(seg.Span,
			fmt.Sprintf("no item %q in module %q", seg.Name, modPath),
			fmt.Sprintf("module %q exists but does not declare %q", modPath, seg.Name))
		return NoSymbol
	}
	return r.followAlias(id)
}

// lookupEnumInModule returns the SymEnum bound to name in module
// modPath, or NoSymbol if no such enum exists. Unlike
// lookupItemInModule, this helper never emits a diagnostic: the caller
// uses it as a probe to decide between "module.Enum.Variant" and
// "module.Submodule.Item" interpretations.
func (r *resolver) lookupEnumInModule(modPath, name string) SymbolID {
	m := r.graph.Modules[modPath]
	if m == nil {
		return NoSymbol
	}
	id := m.Scope.LookupLocal(name)
	if id == NoSymbol {
		return NoSymbol
	}
	id = r.followAlias(id)
	s := r.symbols.Get(id)
	if s == nil || s.Kind != SymEnum {
		return NoSymbol
	}
	return id
}

// lookupVariantOfEnum resolves a single variant name against an enum
// symbol. Emits the variant-not-found diagnostic when the tail does
// not match.
func (r *resolver) lookupVariantOfEnum(enumID SymbolID, tail ast.Ident) SymbolID {
	enum := r.symbols.Get(enumID)
	if enum == nil {
		return NoSymbol
	}
	m := r.graph.Modules[enum.Module]
	if m == nil {
		return NoSymbol
	}
	vid := m.Scope.LookupLocal(tail.Name)
	if vid != NoSymbol {
		v := r.symbols.Get(vid)
		if v != nil && v.Kind == SymEnumVariant && v.Parent == enumID {
			return vid
		}
	}
	r.diagnose(tail.Span,
		fmt.Sprintf("enum %q has no variant %q", enum.Name, tail.Name),
		"variants are spelled exactly as in the enum definition")
	return NoSymbol
}

// resolveViaModuleGraph is the "this might be a fully-qualified path
// without an alias in scope" fallback. It tries successively longer
// prefixes of segs against the module graph until it finds the
// longest matching module, then delegates to walkModulePath for the
// tail. Returns NoSymbol when nothing matches.
func (r *resolver) resolveViaModuleGraph(segs []ast.Ident, span lex.Span) SymbolID {
	longest := -1
	for i := 1; i <= len(segs)-1; i++ {
		candidate := joinSegments(segs[:i])
		if _, ok := r.graph.Modules[candidate]; ok {
			longest = i
		}
	}
	if longest < 0 {
		return NoSymbol
	}
	modPath := joinSegments(segs[:longest])
	return r.walkModulePath(modPath, segs[longest:], span)
}

// joinModulePath appends a single segment to a dotted module path.
func joinModulePath(base, seg string) string {
	if base == "" {
		return seg
	}
	return base + "." + seg
}

// joinSegments turns an ast.Ident prefix into a dotted module path.
func joinSegments(segs []ast.Ident) string {
	parts := make([]string, len(segs))
	for i, s := range segs {
		parts[i] = s.Name
	}
	return joinStrings(parts, ".")
}

// joinStrings is a small join helper to keep this file independent of
// strings.Join (avoids introducing an import cycle if this package later
// needs to support an embedded strings shim).
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	n := len(sep) * (len(parts) - 1)
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 0, n)
	for i, p := range parts {
		if i > 0 {
			out = append(out, sep...)
		}
		out = append(out, p...)
	}
	return string(out)
}

// flattenFieldChain walks a tree composed of a PathExpr leaf with zero
// or more FieldExpr wrappers and returns the dotted Ident list plus
// the overall span. Returns ok=false if the chain contains anything
// other than plain field accesses rooted at a PathExpr (for example,
// a call, literal, or index).
func flattenFieldChain(e ast.Expr) (segs []ast.Ident, span lex.Span, ok bool) {
	switch x := e.(type) {
	case *ast.PathExpr:
		if len(x.TypeArgs) != 0 {
			return nil, lex.Span{}, false
		}
		segs := make([]ast.Ident, len(x.Segments))
		copy(segs, x.Segments)
		return segs, x.NodeSpan(), true
	case *ast.FieldExpr:
		inner, _, ok := flattenFieldChain(x.Receiver)
		if !ok {
			return nil, lex.Span{}, false
		}
		out := make([]ast.Ident, 0, len(inner)+1)
		out = append(out, inner...)
		out = append(out, x.Name)
		return out, x.NodeSpan(), true
	}
	return nil, lex.Span{}, false
}

// tryResolveFieldChainAsPath attempts to resolve a FieldExpr's
// receiver-chain as a module- or enum-rooted path. Returns true when
// the chain was taken over by the resolver (either resolved or
// diagnosed); false when the chain is not a static path and should be
// walked as ordinary field access.
//
// This handles the common spellings that the grammar produces for
// module-qualified names: the parser emits `util.secret` as
// `FieldExpr{Receiver: PathExpr{util}, Name: secret}` rather than a
// 2-segment PathExpr, so the resolver flattens the chain at use-site
// to decide whether the leading name names a module, enum, or import
// alias.
func (r *resolver) tryResolveFieldChainAsPath(m *Module, fe *ast.FieldExpr) bool {
	segs, span, ok := flattenFieldChain(fe)
	if !ok || len(segs) < 2 {
		return false
	}
	rootID := m.Scope.Lookup(segs[0].Name)
	if rootID == NoSymbol {
		// The root is not in scope; could be a local variable's field
		// access. Let the normal walker handle it.
		return false
	}
	rootID = r.followAlias(rootID)
	root := r.symbols.Get(rootID)
	if root == nil {
		return false
	}
	isPathRoot := root.Kind == SymModule || root.Kind == SymEnum ||
		(root.Kind == SymImportAlias && root.ModulePath != "")
	if !isPathRoot {
		return false
	}
	id := r.resolvePath(m, segs, span, false)
	if id != NoSymbol {
		r.recordBinding(m.Path, span, id)
	}
	return true
}

// isPrimitiveTypeName returns true for the fixed set of built-in type
// spellings recognized by the Fuse reference. The resolver skips these
// so that W04's TypeTable owns their identity without ambiguity.
func isPrimitiveTypeName(name string) bool {
	switch name {
	case "Bool",
		"I8", "I16", "I32", "I64", "ISize",
		"U8", "U16", "U32", "U64", "USize",
		"F32", "F64",
		"Char", "String", "CStr":
		return true
	}
	return false
}
