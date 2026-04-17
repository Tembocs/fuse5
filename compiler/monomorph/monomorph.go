package monomorph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Diagnostic mirrors lex.Diagnostic so callers merge monomorph
// output into a single diagnostic stream.
type Diagnostic = lex.Diagnostic

// Specialize returns a Program containing one concrete fn per
// observed (generic fn, type-arg tuple) pair, with call sites
// rewritten to reference the specialized names. The input program
// is not mutated — the returned Program shares the TypeTable but
// owns its own module/item slices.
//
// Rejection rules:
//
//   - A call site whose callee is a generic fn but has no
//     turbofish type args produces a "partial instantiation"
//     diagnostic. Full inference of missing type args is W06-
//     scoped for contextual inference; explicit turbofish is
//     the only W08 entry point.
//   - A call site whose turbofish arity doesn't match the fn's
//     generic-param count is rejected.
func Specialize(prog *hir.Program) (*hir.Program, []Diagnostic) {
	m := newMono(prog)
	m.collectInstantiations()
	m.specializeAll()
	m.rewriteCallSites()
	m.buildOutput()
	return m.out, m.diags
}

// instKey identifies one specialization request — (generic fn
// symbol, concrete type args). The type-arg slice is stringified
// at specialization time to derive a deterministic mangled name.
type instKey struct {
	symbol int
	args   typeArgKey // canonical string form of args slice
}

// typeArgKey is a string-valued canonical form of a []TypeId so
// instKey is map-hashable without slice identity concerns.
type typeArgKey string

func makeArgKey(tids []typetable.TypeId) typeArgKey {
	parts := make([]string, len(tids))
	for i, t := range tids {
		parts[i] = fmt.Sprintf("%d", t)
	}
	return typeArgKey(strings.Join(parts, ","))
}

// instRecord holds per-specialization state.
type instRecord struct {
	key      instKey
	origFn   *hir.FnDecl       // the generic fn being specialized
	origMod  string            // module of the generic fn
	argTIDs  []typetable.TypeId // concrete type args
	subst    map[typetable.TypeId]typetable.TypeId // GenericParam TypeId → concrete TypeId
	spec     *hir.FnDecl       // the specialized fn (set during specializeAll)
	specName string            // mangled name (set during specializeAll)
	specSym  int               // synthesized symbol id for the specialization
}

// monomorph is the per-call state accumulator.
type monomorph struct {
	prog *hir.Program
	tab  *typetable.Table
	out  *hir.Program

	// genericFnBySym is the set of fns whose Generics slice is non-empty,
	// indexed by their resolve.SymbolID. Populated before instantiation
	// collection.
	genericFnBySym map[int]*hir.FnDecl
	genericFnMod   map[int]string

	// instByKey maps every observed (fn, args) pair to its record.
	// Key ordering in iteration uses instOrder for determinism.
	instByKey map[instKey]*instRecord
	instOrder []instKey

	// symSeq allocates fresh int-valued symbols for specializations.
	// Specialization symbols live in a distinct high range so they
	// never collide with resolve.SymbolID values from the resolver.
	symSeq int

	// callSitesToRewrite maps original PathExpr pointers observed at
	// call callees to their specialization record. Populated during
	// collectInstantiations; consumed in rewriteCallSites.
	callSitesToRewrite map[*hir.PathExpr]*instRecord

	diags []Diagnostic
}

func newMono(prog *hir.Program) *monomorph {
	return &monomorph{
		prog:               prog,
		tab:                prog.Types,
		genericFnBySym:     map[int]*hir.FnDecl{},
		genericFnMod:       map[int]string{},
		instByKey:          map[instKey]*instRecord{},
		callSitesToRewrite: map[*hir.PathExpr]*instRecord{},
		symSeq:             1 << 30, // high bit range for synthesized symbols
	}
}

// collectInstantiations walks every module's items, records which
// fns are generic, then walks every body for call expressions
// whose callee is a generic fn and generates an instRecord.
func (m *monomorph) collectInstantiations() {
	// First: index all generic fns by their resolve symbol.
	for _, modPath := range m.prog.Order {
		mod := m.prog.Modules[modPath]
		for _, it := range mod.Items {
			fn, ok := it.(*hir.FnDecl)
			if !ok || len(fn.Generics) == 0 {
				continue
			}
			// Find the symbol id that maps to this fn's TypeID.
			for symID, tid := range m.prog.ItemTypes {
				if tid == fn.TypeID {
					m.genericFnBySym[symID] = fn
					m.genericFnMod[symID] = modPath
					break
				}
			}
		}
	}
	// Second: walk every body for call sites referencing generics.
	for _, modPath := range m.prog.Order {
		mod := m.prog.Modules[modPath]
		for _, it := range mod.Items {
			m.collectInItem(modPath, it)
		}
	}
}

func (m *monomorph) collectInItem(modPath string, it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		if x.Body != nil {
			m.collectInBlock(modPath, x.Body)
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			m.collectInItem(modPath, sub)
		}
	case *hir.ConstDecl:
		if x.Value != nil {
			m.collectInExpr(modPath, x.Value)
		}
	case *hir.StaticDecl:
		if x.Value != nil {
			m.collectInExpr(modPath, x.Value)
		}
	}
}

func (m *monomorph) collectInBlock(modPath string, b *hir.Block) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		m.collectInStmt(modPath, s)
	}
	if b.Trailing != nil {
		m.collectInExpr(modPath, b.Trailing)
	}
}

func (m *monomorph) collectInStmt(modPath string, s hir.Stmt) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.Value != nil {
			m.collectInExpr(modPath, x.Value)
		}
	case *hir.VarStmt:
		if x.Value != nil {
			m.collectInExpr(modPath, x.Value)
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			m.collectInExpr(modPath, x.Value)
		}
	case *hir.ExprStmt:
		if x.Expr != nil {
			m.collectInExpr(modPath, x.Expr)
		}
	}
}

func (m *monomorph) collectInExpr(modPath string, e hir.Expr) {
	if e == nil {
		return
	}
	if call, ok := e.(*hir.CallExpr); ok {
		m.considerCall(modPath, call)
	}
	switch x := e.(type) {
	case *hir.BinaryExpr:
		m.collectInExpr(modPath, x.Lhs)
		m.collectInExpr(modPath, x.Rhs)
	case *hir.UnaryExpr:
		m.collectInExpr(modPath, x.Operand)
	case *hir.CallExpr:
		m.collectInExpr(modPath, x.Callee)
		for _, a := range x.Args {
			m.collectInExpr(modPath, a)
		}
	case *hir.Block:
		m.collectInBlock(modPath, x)
	case *hir.IfExpr:
		m.collectInExpr(modPath, x.Cond)
		m.collectInBlock(modPath, x.Then)
		if x.Else != nil {
			m.collectInExpr(modPath, x.Else)
		}
	case *hir.TupleExpr:
		for _, el := range x.Elements {
			m.collectInExpr(modPath, el)
		}
	}
}

// considerCall checks whether a CallExpr's callee targets a
// generic fn; when so, records an instantiation keyed by the
// turbofish type args.
func (m *monomorph) considerCall(modPath string, call *hir.CallExpr) {
	path, ok := call.Callee.(*hir.PathExpr)
	if !ok {
		return
	}
	fn, isGeneric := m.genericFnBySym[path.Symbol]
	if !isGeneric {
		return
	}
	if len(path.TypeArgs) == 0 {
		m.diags = append(m.diags, Diagnostic{
			Span: call.Span,
			Message: fmt.Sprintf(
				"generic fn %q called without explicit type args",
				fn.Name),
			Hint: "use the turbofish form at the call site: `" + fn.Name + "[I32](...)`",
		})
		return
	}
	if len(path.TypeArgs) != len(fn.Generics) {
		m.diags = append(m.diags, Diagnostic{
			Span: call.Span,
			Message: fmt.Sprintf(
				"generic fn %q takes %d type args, got %d",
				fn.Name, len(fn.Generics), len(path.TypeArgs)),
			Hint: "provide exactly one type per declared generic parameter",
		})
		return
	}
	// Build substitution map: GenericParam TypeId → concrete TypeId.
	subst := map[typetable.TypeId]typetable.TypeId{}
	for i, g := range fn.Generics {
		subst[g.TypeID] = path.TypeArgs[i]
	}
	key := instKey{symbol: path.Symbol, args: makeArgKey(path.TypeArgs)}
	rec, seen := m.instByKey[key]
	if !seen {
		argsCopy := make([]typetable.TypeId, len(path.TypeArgs))
		copy(argsCopy, path.TypeArgs)
		rec = &instRecord{
			key:     key,
			origFn:  fn,
			origMod: m.genericFnMod[path.Symbol],
			argTIDs: argsCopy,
			subst:   subst,
		}
		m.instByKey[key] = rec
		m.instOrder = append(m.instOrder, key)
	}
	m.callSitesToRewrite[path] = rec
}

// specializeAll iterates instantiations in observation order and
// produces a concrete FnDecl for each. The specialized fn's body
// is a deep copy of the generic's body with every GenericParam
// TypeId remapped via the substitution map.
func (m *monomorph) specializeAll() {
	// Sort instantiation order canonically: by (symbol, argKey) so
	// subsequent codegen emits them deterministically (Rule 7.1).
	sort.Slice(m.instOrder, func(i, j int) bool {
		a, b := m.instOrder[i], m.instOrder[j]
		if a.symbol != b.symbol {
			return a.symbol < b.symbol
		}
		return a.args < b.args
	})
	for _, key := range m.instOrder {
		rec := m.instByKey[key]
		rec.specName = mangleName(rec.origFn.Name, rec.argTIDs, m.tab)
		rec.specSym = m.freshSym()
		rec.spec = m.specializeFn(rec)
	}
}

// freshSym allocates a synthetic symbol id for a specialization.
func (m *monomorph) freshSym() int {
	m.symSeq++
	return m.symSeq
}

// specializeFn builds a concrete FnDecl from a generic original
// plus a substitution map. The returned fn has no generics, its
// param and return TypeIds are concrete, and its body is a deep
// copy with GenericParam TypeIds remapped.
func (m *monomorph) specializeFn(rec *instRecord) *hir.FnDecl {
	orig := rec.origFn
	// Substituted Fn TypeId.
	newParamTypes := make([]typetable.TypeId, 0, len(orig.Params))
	for _, p := range orig.Params {
		newParamTypes = append(newParamTypes, m.substType(p.TypeOf(), rec.subst))
	}
	newRet := m.substType(orig.Return, rec.subst)
	newFnType := m.tab.Fn(newParamTypes, newRet, orig.Variadic)

	// Specialized params.
	newParams := make([]*hir.Param, len(orig.Params))
	for i, p := range orig.Params {
		newParams[i] = &hir.Param{
			TypedBase: hir.TypedBase{
				Base: hir.Base{
					ID:   hir.NodeID(string(p.ID) + "$spec:" + string(rec.key.args)),
					Span: p.Span,
				},
				Type: m.substType(p.TypeOf(), rec.subst),
			},
			Name:      p.Name,
			Ownership: p.Ownership,
		}
	}

	// Deep-copy the body with type substitution.
	var newBody *hir.Block
	if orig.Body != nil {
		newBody = m.copyBlock(orig.Body, rec, rec.specName)
	}

	return &hir.FnDecl{
		Base: hir.Base{
			ID:   hir.ItemID(rec.origMod, rec.specName),
			Span: orig.Span,
		},
		Name:   rec.specName,
		Params: newParams,
		Return: newRet,
		TypeID: newFnType,
		Body:   newBody,
		// Generics intentionally empty — this is the concrete copy.
		IsExtern: orig.IsExtern,
		IsConst:  orig.IsConst,
		Variadic: orig.Variadic,
	}
}

// substType applies a TypeId substitution recursively. Structural
// TypeIds whose children contain substituted TypeIds are re-interned
// so identity is preserved across the substitution.
func (m *monomorph) substType(tid typetable.TypeId, subst map[typetable.TypeId]typetable.TypeId) typetable.TypeId {
	if sub, ok := subst[tid]; ok {
		return sub
	}
	t := m.tab.Get(tid)
	if t == nil {
		return tid
	}
	switch t.Kind {
	case typetable.KindTuple:
		newCh := make([]typetable.TypeId, len(t.Children))
		changed := false
		for i, c := range t.Children {
			newCh[i] = m.substType(c, subst)
			if newCh[i] != c {
				changed = true
			}
		}
		if !changed {
			return tid
		}
		return m.tab.Tuple(newCh)
	case typetable.KindSlice:
		if len(t.Children) == 0 {
			return tid
		}
		newElem := m.substType(t.Children[0], subst)
		if newElem == t.Children[0] {
			return tid
		}
		return m.tab.Slice(newElem)
	case typetable.KindPtr:
		if len(t.Children) == 0 {
			return tid
		}
		return m.tab.Ptr(m.substType(t.Children[0], subst))
	case typetable.KindRef:
		if len(t.Children) == 0 {
			return tid
		}
		return m.tab.Ref(m.substType(t.Children[0], subst))
	case typetable.KindMutref:
		if len(t.Children) == 0 {
			return tid
		}
		return m.tab.Mutref(m.substType(t.Children[0], subst))
	case typetable.KindFn:
		newParams := make([]typetable.TypeId, len(t.Children))
		for i, c := range t.Children {
			newParams[i] = m.substType(c, subst)
		}
		newRet := m.substType(t.Return, subst)
		return m.tab.Fn(newParams, newRet, t.IsVariadic)
	case typetable.KindChannel:
		if len(t.Children) == 0 {
			return tid
		}
		return m.tab.Channel(m.substType(t.Children[0], subst))
	case typetable.KindThreadHandle:
		if len(t.Children) == 0 {
			return tid
		}
		return m.tab.ThreadHandle(m.substType(t.Children[0], subst))
	}
	return tid
}

// copyBlock deep-copies a block with TypeId substitution. The
// anchor string prefixes newly-assigned NodeIDs so the specialized
// body has stable identity distinct from the generic original.
func (m *monomorph) copyBlock(b *hir.Block, rec *instRecord, anchor string) *hir.Block {
	if b == nil {
		return nil
	}
	nb := &hir.Block{
		TypedBase: hir.TypedBase{
			Base: hir.Base{
				ID:   rewriteID(b.ID, rec),
				Span: b.Span,
			},
			Type: m.substType(b.Type, rec.subst),
		},
	}
	for _, s := range b.Stmts {
		nb.Stmts = append(nb.Stmts, m.copyStmt(s, rec))
	}
	if b.Trailing != nil {
		nb.Trailing = m.copyExpr(b.Trailing, rec)
	}
	return nb
}

func (m *monomorph) copyStmt(s hir.Stmt, rec *instRecord) hir.Stmt {
	switch x := s.(type) {
	case *hir.LetStmt:
		return &hir.LetStmt{
			Base: hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span},
			Pattern:      m.copyPat(x.Pattern, rec),
			DeclaredType: m.substType(x.DeclaredType, rec.subst),
			Value:        m.copyExpr(x.Value, rec),
		}
	case *hir.VarStmt:
		return &hir.VarStmt{
			Base: hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span},
			Name:         x.Name,
			DeclaredType: m.substType(x.DeclaredType, rec.subst),
			Value:        m.copyExpr(x.Value, rec),
		}
	case *hir.ReturnStmt:
		return &hir.ReturnStmt{
			Base:  hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span},
			Value: m.copyExpr(x.Value, rec),
		}
	case *hir.BreakStmt:
		return &hir.BreakStmt{
			Base:  hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span},
			Value: m.copyExpr(x.Value, rec),
		}
	case *hir.ContinueStmt:
		return &hir.ContinueStmt{Base: hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span}}
	case *hir.ExprStmt:
		return &hir.ExprStmt{
			Base: hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span},
			Expr: m.copyExpr(x.Expr, rec),
		}
	}
	return s
}

func (m *monomorph) copyExpr(e hir.Expr, rec *instRecord) hir.Expr {
	if e == nil {
		return nil
	}
	newType := m.substType(e.TypeOf(), rec.subst)
	baseID := rewriteID(e.NodeHirID(), rec)
	switch x := e.(type) {
	case *hir.LiteralExpr:
		return &hir.LiteralExpr{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: baseID, Span: x.Span}, Type: newType},
			Kind:      x.Kind,
			Text:      x.Text,
			Bool:      x.Bool,
		}
	case *hir.PathExpr:
		segs := make([]string, len(x.Segments))
		copy(segs, x.Segments)
		return &hir.PathExpr{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: baseID, Span: x.Span}, Type: newType},
			Symbol:    x.Symbol,
			Segments:  segs,
			// TypeArgs: not copied — specialization erases them.
		}
	case *hir.BinaryExpr:
		return &hir.BinaryExpr{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: baseID, Span: x.Span}, Type: newType},
			Op:        x.Op,
			Lhs:       m.copyExpr(x.Lhs, rec),
			Rhs:       m.copyExpr(x.Rhs, rec),
		}
	case *hir.UnaryExpr:
		return &hir.UnaryExpr{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: baseID, Span: x.Span}, Type: newType},
			Op:        x.Op,
			Operand:   m.copyExpr(x.Operand, rec),
		}
	case *hir.CallExpr:
		args := make([]hir.Expr, len(x.Args))
		for i, a := range x.Args {
			args[i] = m.copyExpr(a, rec)
		}
		return &hir.CallExpr{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: baseID, Span: x.Span}, Type: newType},
			Callee:    m.copyExpr(x.Callee, rec),
			Args:      args,
		}
	case *hir.Block:
		return m.copyBlock(x, rec, "")
	case *hir.IfExpr:
		return &hir.IfExpr{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: baseID, Span: x.Span}, Type: newType},
			Cond:      m.copyExpr(x.Cond, rec),
			Then:      m.copyBlock(x.Then, rec, ""),
			Else:      m.copyExpr(x.Else, rec),
		}
	case *hir.TupleExpr:
		els := make([]hir.Expr, len(x.Elements))
		for i, el := range x.Elements {
			els[i] = m.copyExpr(el, rec)
		}
		return &hir.TupleExpr{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: baseID, Span: x.Span}, Type: newType},
			Elements:  els,
		}
	}
	return e
}

func (m *monomorph) copyPat(p hir.Pat, rec *instRecord) hir.Pat {
	if p == nil {
		return nil
	}
	switch x := p.(type) {
	case *hir.BindPat:
		return &hir.BindPat{
			TypedBase: hir.TypedBase{
				Base: hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span},
				Type: m.substType(x.Type, rec.subst),
			},
			Name: x.Name,
		}
	case *hir.WildcardPat:
		return &hir.WildcardPat{
			TypedBase: hir.TypedBase{
				Base: hir.Base{ID: rewriteID(x.ID, rec), Span: x.Span},
				Type: m.substType(x.Type, rec.subst),
			},
		}
	}
	return p
}

// rewriteID appends the specialization key to a NodeID so the
// specialized body has a distinct NodeID from the generic original.
// This preserves the W04-P05 stability property: unrelated edits
// don't shift identities.
func rewriteID(id hir.NodeID, rec *instRecord) hir.NodeID {
	if id == "" {
		return id
	}
	return hir.NodeID(string(id) + "$" + string(rec.key.args))
}

// rewriteCallSites walks every surviving (non-generic-original) fn
// body and retargets calls to specialized symbols / names.
func (m *monomorph) rewriteCallSites() {
	for _, modPath := range m.prog.Order {
		mod := m.prog.Modules[modPath]
		for _, it := range mod.Items {
			m.rewriteItem(it)
		}
	}
}

func (m *monomorph) rewriteItem(it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		if x.Body != nil {
			m.rewriteBlock(x.Body)
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			m.rewriteItem(sub)
		}
	}
}

func (m *monomorph) rewriteBlock(b *hir.Block) {
	if b == nil {
		return
	}
	for _, s := range b.Stmts {
		m.rewriteStmt(s)
	}
	if b.Trailing != nil {
		m.rewriteExpr(&b.Trailing)
	}
}

func (m *monomorph) rewriteStmt(s hir.Stmt) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.Value != nil {
			m.rewriteExpr(&x.Value)
		}
	case *hir.VarStmt:
		if x.Value != nil {
			m.rewriteExpr(&x.Value)
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			m.rewriteExpr(&x.Value)
		}
	case *hir.ExprStmt:
		if x.Expr != nil {
			m.rewriteExpr(&x.Expr)
		}
	}
}

func (m *monomorph) rewriteExpr(ep *hir.Expr) {
	if *ep == nil {
		return
	}
	switch x := (*ep).(type) {
	case *hir.CallExpr:
		if path, ok := x.Callee.(*hir.PathExpr); ok {
			if rec, found := m.callSitesToRewrite[path]; found {
				// Rewrite the path's symbol and segment; clear type args.
				path.Symbol = rec.specSym
				path.Segments = []string{rec.specName}
				path.TypeArgs = nil
				path.Type = rec.spec.TypeID
			}
		}
		m.rewriteExpr(&x.Callee)
		for i := range x.Args {
			m.rewriteExpr(&x.Args[i])
		}
	case *hir.BinaryExpr:
		m.rewriteExpr(&x.Lhs)
		m.rewriteExpr(&x.Rhs)
	case *hir.UnaryExpr:
		m.rewriteExpr(&x.Operand)
	case *hir.Block:
		m.rewriteBlock(x)
	case *hir.IfExpr:
		m.rewriteExpr(&x.Cond)
		m.rewriteBlock(x.Then)
		if x.Else != nil {
			m.rewriteExpr(&x.Else)
		}
	case *hir.TupleExpr:
		for i := range x.Elements {
			m.rewriteExpr(&x.Elements[i])
		}
	}
}

// buildOutput constructs a new Program whose modules contain every
// non-generic original item plus the specialization records.
// Generic originals are excluded so downstream lowering and
// codegen never see them — "only concrete instantiations reach
// codegen" (W08 exit criterion).
func (m *monomorph) buildOutput() {
	out := hir.NewProgram(m.tab)
	// Copy ItemTypes for non-generic originals.
	for symID, tid := range m.prog.ItemTypes {
		if _, isGeneric := m.genericFnBySym[symID]; !isGeneric {
			out.BindItemType(symID, tid)
		}
	}
	// Register specializations' symbol→TypeId entries.
	for _, key := range m.instOrder {
		rec := m.instByKey[key]
		out.BindItemType(rec.specSym, rec.spec.TypeID)
	}
	// Per module: keep non-generic items; append specializations
	// declared in that module's origin.
	for _, modPath := range m.prog.Order {
		mod := m.prog.Modules[modPath]
		nm := &hir.Module{
			Base: hir.Base{ID: mod.ID, Span: mod.Span},
			Path: mod.Path,
		}
		for _, it := range mod.Items {
			if fn, ok := it.(*hir.FnDecl); ok && len(fn.Generics) != 0 {
				continue // drop the generic original
			}
			nm.Items = append(nm.Items, it)
		}
		for _, key := range m.instOrder {
			rec := m.instByKey[key]
			if rec.origMod == modPath {
				nm.Items = append(nm.Items, rec.spec)
			}
		}
		out.RegisterModule(nm)
	}
	m.out = out
}

// mangleName builds the deterministic specialization name.
//
// Format: `<fn_name>__<TypeName1>_<TypeName2>...` where TypeName is
// the canonical `typetable.Type.Name` for nominals, or the Kind
// string for primitives. Examples:
//
//   identity[I32]           -> identity__I32
//   pair[I32, Bool]         -> identity__I32_Bool
//   container[Vec]          -> container__Vec
//
// The mangled name is C-safe (only alphanumerics and underscore).
func mangleName(base string, args []typetable.TypeId, tab *typetable.Table) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, base)
	for _, a := range args {
		parts = append(parts, typeNameForMangle(tab, a))
	}
	return strings.Join(parts, "__")
}

// typeNameForMangle returns the C-safe spelling for a TypeId.
func typeNameForMangle(tab *typetable.Table, tid typetable.TypeId) string {
	t := tab.Get(tid)
	if t == nil {
		return "void"
	}
	switch t.Kind {
	case typetable.KindStruct, typetable.KindEnum, typetable.KindUnion,
		typetable.KindTrait, typetable.KindTypeAlias:
		if t.Name != "" {
			return safeName(t.Name)
		}
	case typetable.KindTuple:
		return "tuple" + fmt.Sprintf("%d", len(t.Children))
	case typetable.KindPtr:
		return "Ptr"
	case typetable.KindSlice:
		return "Slice"
	case typetable.KindRef:
		return "Ref"
	case typetable.KindMutref:
		return "Mutref"
	case typetable.KindFn:
		return "Fn"
	case typetable.KindChannel:
		return "Chan"
	case typetable.KindThreadHandle:
		return "ThreadHandle"
	}
	return t.Kind.String()
}

// safeName strips characters that aren't safe in a C identifier,
// replacing them with `_`.
func safeName(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			b = append(b, c)
		} else {
			b = append(b, '_')
		}
	}
	return string(b)
}
