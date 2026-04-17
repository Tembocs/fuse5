package resolve

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// resolveImports performs module-first import resolution and cycle
// detection (reference §18.7). For each import it decides whether the
// path names a module or an item inside a module, records the decision
// as a binding on the import's span, and if an alias is present binds
// the alias as a SymImportAlias symbol in the importing module's scope.
//
// Once every edge has been added the function runs cycle detection and
// emits one diagnostic per cycle found. A cycle does not prevent
// downstream path resolution — the resolver reports the cycle and
// continues so later passes can still produce diagnostics for unrelated
// problems (Rule 6.9).
func (r *resolver) resolveImports() {
	for _, modPath := range r.graph.Order {
		m := r.graph.Modules[modPath]
		for _, imp := range m.Source.File.Imports {
			r.resolveOneImport(m, imp)
		}
	}
	r.graph.finalize()
	r.detectImportCycles()
}

// resolveOneImport applies module-first resolution to a single import.
// After the call, the import has either a binding (module or item)
// recorded on its span, or a diagnostic has been emitted.
func (r *resolver) resolveOneImport(m *Module, imp *ast.Import) {
	if len(imp.Path) == 0 {
		return
	}
	full := pathString(imp.Path)

	// Step 1: full path as module.
	if tgt, ok := r.graph.Modules[full]; ok {
		r.recordBinding(m.Path, imp.NodeSpan(), tgt.Symbol)
		r.graph.addEdge(m.Path, full)
		r.bindImportAlias(m, imp, tgt.Symbol, full, ast.VisPub)
		return
	}

	// Step 2: (prefix, last) as (module, item). For single-segment
	// imports the "preceding module" is the root (""), which may or
	// may not be registered; either way the diagnostic wording below
	// collapses to "unresolved import" rather than the more specific
	// "no item X in module Y" form, since the user wrote only a top-
	// level name.
	prefix, tail := moduleAndTail(imp.Path)
	if len(imp.Path) >= 2 {
		if parent, ok := r.graph.Modules[prefix]; ok {
			if itemID := parent.Scope.LookupLocal(tail); itemID != NoSymbol {
				sym := r.symbols.Get(itemID)
				r.recordBinding(m.Path, imp.NodeSpan(), itemID)
				r.graph.addEdge(m.Path, prefix)
				r.bindImportAlias(m, imp, itemID, "", sym.Vis)
				return
			}
			r.diagnose(imp.NodeSpan(),
				fmt.Sprintf("no item %q in module %q", tail, prefix),
				fmt.Sprintf("module %q exists but does not declare an item named %q", prefix, tail))
			return
		}
	}

	// Neither a module nor an item-in-module match.
	r.diagnose(imp.NodeSpan(),
		fmt.Sprintf("unresolved import %q", full),
		"no module or item with this path exists in the build")
}

// bindImportAlias binds an import's alias identifier (or the last path
// segment when no alias is given) into the importing module's scope.
// Aliases let later path lookups find the imported name with a single
// identifier. The alias carries the target's visibility for re-export
// rules once they are implemented; at W03 the stored visibility is
// consulted by enforceVisibility only if another module reaches in.
func (r *resolver) bindImportAlias(m *Module, imp *ast.Import, target SymbolID, targetModule string, vis ast.Visibility) {
	name := importLocalName(imp)
	if name == "" {
		return
	}
	if prior := m.Scope.LookupLocal(name); prior != NoSymbol {
		r.diagnose(importAliasSpan(imp),
			fmt.Sprintf("import name %q conflicts with existing item", name),
			"rename the import with `as` to avoid the clash")
		return
	}
	id := r.symbols.Add(Symbol{
		Kind:       SymImportAlias,
		Name:       name,
		Module:     m.Path,
		Vis:        vis,
		Span:       importAliasSpan(imp),
		Target:     target,
		ModulePath: targetModule,
	})
	m.Scope.Insert(name, id)
}

// importLocalName returns the name this import binds into the current
// module's scope — either the explicit alias or the last path segment.
func importLocalName(imp *ast.Import) string {
	if imp.Alias != nil {
		return imp.Alias.Name
	}
	if n := len(imp.Path); n > 0 {
		return imp.Path[n-1].Name
	}
	return ""
}

// importAliasSpan returns the span that names the import's local
// binding — either the alias span or the last path segment.
func importAliasSpan(imp *ast.Import) lex.Span {
	if imp.Alias != nil {
		return imp.Alias.Span
	}
	if n := len(imp.Path); n > 0 {
		return imp.Path[n-1].Span
	}
	return imp.NodeSpan()
}

// detectImportCycles walks the module graph's import edges and emits a
// diagnostic for every strongly-connected component of size ≥ 2 and
// for every self-loop. Stability: modules are visited in Order
// (sorted) and cycles report the lexicographically smallest member
// first. Detection uses Tarjan's algorithm for linear-time grouping
// without recursion depth explosions on long import chains.
func (r *resolver) detectImportCycles() {
	idx := 0
	indices := map[string]int{}
	lowlink := map[string]int{}
	onStack := map[string]bool{}
	var stack []string

	var strong func(v string)
	strong = func(v string) {
		indices[v] = idx
		lowlink[v] = idx
		idx++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range r.graph.Edges[v] {
			if _, seen := indices[w]; !seen {
				strong(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}

		if lowlink[v] == indices[v] {
			var scc []string
			for {
				n := len(stack) - 1
				w := stack[n]
				stack = stack[:n]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			// Report cycles: SCCs of size > 1, or size 1 with a self-edge.
			if len(scc) > 1 || (len(scc) == 1 && hasSelfEdge(r.graph.Edges, scc[0])) {
				r.reportCycle(scc)
			}
		}
	}

	for _, v := range r.graph.Order {
		if _, seen := indices[v]; !seen {
			strong(v)
		}
	}
}

// hasSelfEdge reports whether module v imports itself.
func hasSelfEdge(edges map[string][]string, v string) bool {
	for _, e := range edges[v] {
		if e == v {
			return true
		}
	}
	return false
}

// reportCycle emits one diagnostic naming every module in a cycle.
// Members are sorted so the emitted path is stable across runs
// (Rule 7.1). The diagnostic span points at the first module's File
// (the lexicographically smallest) because there is no single "cycle
// span" in the source.
func (r *resolver) reportCycle(members []string) {
	sort.Strings(members)
	anchor := r.graph.Modules[members[0]]
	span := lex.Span{}
	if anchor != nil && anchor.Source != nil && anchor.Source.File != nil {
		span = lex.Span{File: anchor.Source.File.Filename}
	}
	r.diagnose(span,
		fmt.Sprintf("import cycle detected: %s", strings.Join(members, " -> ")),
		"break the cycle by removing one of the imports on the path")
}
