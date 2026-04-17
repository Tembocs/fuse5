package resolve

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/parse"
)

// SourceFile is one input to resolution: a parsed AST and the module
// path that file contributes to. The module path is the dotted form
// `foo.bar` (reference §18.1); the root module is the empty string.
type SourceFile struct {
	ModulePath string
	File       *ast.File
}

// Module is one resolved module — exactly one entry per discovered
// dotted module path. Items is the @cfg-filtered subset of the file's
// top-level items (the raw list is File.Items).
type Module struct {
	Path    string
	Source  *SourceFile
	Items   []ast.Item
	Scope   *Scope
	// Symbol is the SymModule entry in the SymbolTable. It is registered
	// during module graph construction, not during item indexing.
	Symbol SymbolID
}

// ModuleGraph is the discovered-and-indexed set of modules plus the
// directed edges introduced by their imports.
type ModuleGraph struct {
	// Modules keyed by dotted path. The root module has the empty key.
	Modules map[string]*Module
	// Order is the deterministic module list (sorted by Path). Passes
	// that iterate modules must use this slice, never `range Modules`.
	Order []string
	// Edges[p] is the sorted list of module paths that module p imports
	// (module-first resolution target only — when an import resolves to
	// an item, the edge still names the item's module). Filled by
	// resolveImports.
	Edges map[string][]string
}

// DiscoverFromDir walks root and returns one SourceFile per `.fuse`
// source, with module paths derived from each file's relative directory.
// Walk order is deterministic: directories are visited in
// lexicographic order of their basename. Files named `mod.fuse` collapse
// to the directory's own module path; other files contribute a child
// module named after the file stem.
//
// Parse diagnostics from every file are aggregated into diags; a parse
// error does not abort discovery (Rule 6.9 — surface as many diagnostics
// as possible).
func DiscoverFromDir(root string) (srcs []*SourceFile, diags []lex.Diagnostic, err error) {
	root = filepath.Clean(root)
	paths, err := collectFuseFiles(root)
	if err != nil {
		return nil, nil, err
	}
	for _, p := range paths {
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil, nil, rerr
		}
		modPath := modulePathFor(root, p)
		file, fd := parse.Parse(p, data)
		diags = append(diags, fd...)
		srcs = append(srcs, &SourceFile{ModulePath: modPath, File: file})
	}
	return srcs, diags, nil
}

// collectFuseFiles returns every `.fuse` file under root, walked in a
// stable order (directories lexicographic, files lexicographic).
func collectFuseFiles(root string) ([]string, error) {
	var out []string
	err := walkSorted(root, func(path string, info os.FileInfo) error {
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".fuse") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// walkSorted walks root in deterministic order: each directory's
// children are sorted lexicographically before recursion, and fn is
// invoked on every file and directory exactly once.
func walkSorted(root string, fn func(path string, info os.FileInfo) error) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if err := fn(root, info); err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		if err := walkSorted(filepath.Join(root, name), fn); err != nil {
			return err
		}
	}
	return nil
}

// modulePathFor maps an absolute file path under root to its dotted
// module path. `<root>/lib.fuse` is the root module (""). `<root>/a.fuse`
// is module "a". `<root>/a/b.fuse` is "a.b". `<root>/a/mod.fuse` is "a".
func modulePathFor(root, file string) string {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	// Strip the `.fuse` extension.
	if dot := strings.LastIndex(rel, "."); dot >= 0 {
		rel = rel[:dot]
	}
	parts := strings.Split(rel, "/")
	// `lib.fuse` at the root is the crate root module.
	if len(parts) == 1 && (parts[0] == "lib" || parts[0] == "main") {
		return ""
	}
	// `<dir>/mod.fuse` names module `<dir>`.
	if n := len(parts); n > 0 && parts[n-1] == "mod" {
		parts = parts[:n-1]
	}
	return strings.Join(parts, ".")
}

// newModuleGraph builds an empty graph with sorted Order ready to be
// filled from a SourceFile slice.
func newModuleGraph() *ModuleGraph {
	return &ModuleGraph{
		Modules: map[string]*Module{},
		Edges:   map[string][]string{},
	}
}

// register inserts m under its Path key and appends to Order. Callers
// must guard against duplicate paths before calling this.
func (g *ModuleGraph) register(m *Module) {
	g.Modules[m.Path] = m
	g.Order = append(g.Order, m.Path)
}

// finalize sorts Order and each edge list lexicographically. Called
// once all modules and edges have been added so downstream iteration is
// deterministic.
func (g *ModuleGraph) finalize() {
	sort.Strings(g.Order)
	for k := range g.Edges {
		edges := g.Edges[k]
		sort.Strings(edges)
		g.Edges[k] = uniqueSorted(edges)
	}
}

// addEdge records that module `from` imports module `to`. Duplicate
// edges are collapsed in finalize.
func (g *ModuleGraph) addEdge(from, to string) {
	g.Edges[from] = append(g.Edges[from], to)
}

// uniqueSorted returns a deduplicated view of an already-sorted slice.
func uniqueSorted(in []string) []string {
	if len(in) < 2 {
		return in
	}
	out := in[:1]
	for i := 1; i < len(in); i++ {
		if in[i] != in[i-1] {
			out = append(out, in[i])
		}
	}
	return out
}

// pathString turns an ast.Ident slice into its dotted spelling, used
// when reporting imports in diagnostics.
func pathString(segs []ast.Ident) string {
	parts := make([]string, len(segs))
	for i, s := range segs {
		parts[i] = s.Name
	}
	return strings.Join(parts, ".")
}

// moduleAndTail splits an import path into the prefix-as-module and the
// trailing segment. Used by module-first fallback: if the full path does
// not name a module, try (prefix, last) as (module, item).
func moduleAndTail(segs []ast.Ident) (mod string, tail string) {
	if len(segs) == 0 {
		return "", ""
	}
	parts := make([]string, len(segs)-1)
	for i := 0; i < len(segs)-1; i++ {
		parts[i] = segs[i].Name
	}
	return strings.Join(parts, "."), segs[len(segs)-1].Name
}
