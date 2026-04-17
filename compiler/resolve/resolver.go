package resolve

import (
	"fmt"
	"sort"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// Resolved is the output of name resolution. AST nodes are not mutated
// by the resolver (Rule 3.2); the tables below are the sole carriers of
// resolved information that later passes (HIR lowering in W04, type
// checking in W06) consume.
//
// Bindings maps every syntactic path occurrence the resolver visited to
// the SymbolID it resolved to. Failing resolutions are not recorded
// here — they produce diagnostics instead (Rule 6.9).
type Resolved struct {
	Graph    *ModuleGraph
	Symbols  *SymbolTable
	Bindings map[SiteKey]SymbolID
}

// SiteKey identifies a path occurrence by its module and source span.
// The (module, span) pair is unique because spans are byte-addressed
// (Rule 7.4) and never cross module boundaries.
type SiteKey struct {
	Module string
	Span   lex.Span
}

// resolver is the mutable state accumulator. It is built once per
// Resolve call and discarded; callers see only the Resolved output.
type resolver struct {
	cfg      BuildConfig
	graph    *ModuleGraph
	symbols  *SymbolTable
	bindings map[SiteKey]SymbolID
	diags    []lex.Diagnostic
}

// newResolver returns a fresh resolver configured with cfg.
func newResolver(cfg BuildConfig) *resolver {
	return &resolver{
		cfg:      cfg,
		graph:    newModuleGraph(),
		symbols:  newSymbolTable(),
		bindings: map[SiteKey]SymbolID{},
	}
}

// Resolve is the package entry point. It takes a list of SourceFiles,
// builds the module graph, filters `@cfg`-gated items, indexes every
// top-level name, resolves imports, resolves path occurrences, and
// enforces visibility. Non-fatal diagnostics accumulate; the final
// Resolved is always non-nil (callers never need nil guards).
func Resolve(srcs []*SourceFile, cfg BuildConfig) (*Resolved, []lex.Diagnostic) {
	r := newResolver(cfg)
	r.buildGraph(srcs)
	r.filterCfg()
	r.index()
	r.resolveImports()
	r.resolveAllPaths()
	r.enforceVisibility()
	out := &Resolved{
		Graph:    r.graph,
		Symbols:  r.symbols,
		Bindings: r.bindings,
	}
	return out, r.diags
}

// buildGraph registers one Module per SourceFile and creates the
// SymModule entry in the SymbolTable. Two files that claim the same
// module path are a duplicate-module error; the first file wins and
// subsequent files are skipped entirely (so later passes do not
// double-index their contents).
func (r *resolver) buildGraph(srcs []*SourceFile) {
	// Pre-sort inputs by module path for determinism even when callers
	// pass them in an arbitrary order.
	sorted := make([]*SourceFile, len(srcs))
	copy(sorted, srcs)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ModulePath != sorted[j].ModulePath {
			return sorted[i].ModulePath < sorted[j].ModulePath
		}
		return sorted[i].File.Filename < sorted[j].File.Filename
	})
	for _, src := range sorted {
		if _, exists := r.graph.Modules[src.ModulePath]; exists {
			span := lex.Span{File: src.File.Filename}
			if !src.File.NodeSpan().IsZero() {
				span = src.File.NodeSpan()
			}
			r.diags = append(r.diags, lex.Diagnostic{
				Span:    span,
				Message: fmt.Sprintf("duplicate module %q", src.ModulePath),
				Hint:    "each module path must map to exactly one source file",
			})
			continue
		}
		m := &Module{Path: src.ModulePath, Source: src}
		m.Scope = newScope(nil, src.ModulePath)
		id := r.symbols.Add(Symbol{
			Kind:       SymModule,
			Name:       moduleLeaf(src.ModulePath),
			Module:     src.ModulePath,
			Vis:        ast.VisPub, // modules are always reachable through their path
			Span:       lex.Span{File: src.File.Filename},
			ModulePath: src.ModulePath,
		})
		m.Symbol = id
		r.graph.register(m)
	}
}

// moduleLeaf returns the last dotted segment of a module path — the
// name by which the module is bound in a parent scope. For the root
// module ("") it returns "".
func moduleLeaf(path string) string {
	if path == "" {
		return ""
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}
	return path
}

// filterCfg walks every module's top-level items, evaluates `@cfg`
// decorators, and assigns the surviving items to Module.Items. Items
// whose predicate is false are discarded. Duplicate-item detection
// across `@cfg` runs after filtering (reference §50.1): if two items
// share a name and both survive, a diagnostic is emitted.
func (r *resolver) filterCfg() {
	for _, path := range r.graph.Order {
		m := r.graph.Modules[path]
		filtered := make([]ast.Item, 0, len(m.Source.File.Items))
		for _, it := range m.Source.File.Items {
			keep := true
			for _, d := range cfgDecorators(itemDecorators(it)) {
				ok, ds := evalCfgDecorator(d, r.cfg, m.Source.File.Filename)
				r.diags = append(r.diags, ds...)
				if !ok {
					keep = false
					break
				}
			}
			if keep {
				filtered = append(filtered, it)
			}
		}
		m.Items = filtered
	}
}

// index registers every surviving item in every module into that
// module's scope. Duplicate-item diagnostics (both pre-cfg and
// post-cfg-overlap) surface here.
func (r *resolver) index() {
	for _, path := range r.graph.Order {
		m := r.graph.Modules[path]
		r.diags = append(r.diags, r.indexModule(m, m.Items)...)
	}
}

// recordBinding attaches a resolved symbol to the (module, span)
// identity of a path occurrence. Callers check len(bindings) before
// and after to confirm uniqueness; a later pass reading the map uses
// the SiteKey of the PathExpr/PathType/CtorPat span.
func (r *resolver) recordBinding(module string, span lex.Span, id SymbolID) {
	r.bindings[SiteKey{Module: module, Span: span}] = id
}

// lookupModule returns the Module at the given dotted path or nil.
func (r *resolver) lookupModule(path string) *Module {
	return r.graph.Modules[path]
}

// diagnose appends a diagnostic. Convenience wrapper used across passes
// so every site threads through one call.
func (r *resolver) diagnose(span lex.Span, msg, hint string) {
	r.diags = append(r.diags, lex.Diagnostic{Span: span, Message: msg, Hint: hint})
}
