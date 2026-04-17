package resolve

import (
	"sort"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// SymKind tags what a Symbol represents. Concrete kinds mirror the AST
// item surface (reference Appendix C).
type SymKind int

const (
	SymFn SymKind = iota
	SymStruct
	SymEnum
	SymEnumVariant
	SymTrait
	SymConst
	SymStatic
	SymTypeAlias
	SymUnion
	SymExternFn
	SymExternStatic
	SymModule
	SymImportAlias // binding introduced by an `import ... as` clause
)

// kindName returns a human-readable kind label for diagnostics.
func (k SymKind) String() string {
	switch k {
	case SymFn:
		return "function"
	case SymStruct:
		return "struct"
	case SymEnum:
		return "enum"
	case SymEnumVariant:
		return "enum variant"
	case SymTrait:
		return "trait"
	case SymConst:
		return "const"
	case SymStatic:
		return "static"
	case SymTypeAlias:
		return "type alias"
	case SymUnion:
		return "union"
	case SymExternFn:
		return "extern fn"
	case SymExternStatic:
		return "extern static"
	case SymModule:
		return "module"
	case SymImportAlias:
		return "import alias"
	}
	return "unknown"
}

// SymbolID is an opaque handle into the SymbolTable.
type SymbolID int

// NoSymbol is the zero SymbolID returned when no symbol matches a lookup.
const NoSymbol SymbolID = 0

// Symbol names something at module scope (item, variant, module, or
// import alias). Symbols are immutable after the indexer registers them.
type Symbol struct {
	ID     SymbolID
	Kind   SymKind
	Name   string
	Module string         // dotted module path this symbol belongs to
	Vis    ast.Visibility // VisPrivate if not applicable (e.g. enum variant inherits from enum)
	Span   lex.Span       // defining occurrence span
	Parent SymbolID       // for enum variants, the enum symbol; NoSymbol otherwise
	// Target is set for SymImportAlias symbols; it names what the alias
	// binds to after import resolution (NoSymbol until imports resolve).
	Target SymbolID
	// ModulePath is set for SymImportAlias when the alias binds to a
	// module rather than an item.
	ModulePath string
	// Node is the defining AST node (or nil for synthetic module symbols).
	Node ast.Node
}

// SymbolTable is the flat owner of every symbol. Indexes into the slice
// are SymbolIDs. The zero-th slot is reserved so that a SymbolID of 0
// unambiguously means "not found" (NoSymbol).
type SymbolTable struct {
	syms []Symbol
}

// newSymbolTable returns a fresh table with the reserved zero slot.
func newSymbolTable() *SymbolTable {
	return &SymbolTable{syms: []Symbol{{}}}
}

// Add registers s and returns its assigned ID.
func (t *SymbolTable) Add(s Symbol) SymbolID {
	id := SymbolID(len(t.syms))
	s.ID = id
	t.syms = append(t.syms, s)
	return id
}

// Get returns the Symbol behind id. Panics if id is out of range; callers
// never pass in an ID they did not receive from Add.
func (t *SymbolTable) Get(id SymbolID) *Symbol {
	if id <= 0 || int(id) >= len(t.syms) {
		return nil
	}
	return &t.syms[id]
}

// setTarget updates an import alias's resolved target. Called by the
// import resolver once module-first fallback has decided what the alias
// refers to.
func (t *SymbolTable) setTarget(id SymbolID, target SymbolID, modulePath string) {
	s := &t.syms[id]
	s.Target = target
	s.ModulePath = modulePath
}

// Len returns the number of registered symbols (excluding the reserved
// zero slot). Useful in tests that want to assert table size.
func (t *SymbolTable) Len() int { return len(t.syms) - 1 }

// Scope maps unqualified names to SymbolIDs. Scopes chain to a parent;
// lookup walks the chain. A Scope is owned by exactly one module; the
// Module field is used to report diagnostic locations for duplicate
// definitions.
type Scope struct {
	Parent  *Scope
	Module  string
	Entries map[string]SymbolID
}

// newScope allocates an empty scope linked to parent (which may be nil).
func newScope(parent *Scope, module string) *Scope {
	return &Scope{Parent: parent, Module: module, Entries: map[string]SymbolID{}}
}

// Insert registers name → id in this scope. It returns false and leaves
// the existing binding in place when the name is already present;
// callers that want duplicate detection should check the return value.
func (s *Scope) Insert(name string, id SymbolID) bool {
	if _, ok := s.Entries[name]; ok {
		return false
	}
	s.Entries[name] = id
	return true
}

// Lookup returns the symbol bound to name in this scope or any ancestor,
// or NoSymbol if none matches. The resolver uses this for unqualified
// identifier lookups.
func (s *Scope) Lookup(name string) SymbolID {
	for cur := s; cur != nil; cur = cur.Parent {
		if id, ok := cur.Entries[name]; ok {
			return id
		}
	}
	return NoSymbol
}

// LookupLocal returns the symbol bound in exactly this scope. It does
// not walk ancestors. Used when checking for duplicates at a single
// level.
func (s *Scope) LookupLocal(name string) SymbolID {
	if id, ok := s.Entries[name]; ok {
		return id
	}
	return NoSymbol
}

// sortedNames returns the names registered in this scope in
// lexicographic order. Used by diagnostics to keep output deterministic.
func (s *Scope) sortedNames() []string {
	out := make([]string, 0, len(s.Entries))
	for k := range s.Entries {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
