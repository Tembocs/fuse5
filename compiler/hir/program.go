package hir

import (
	"sort"

	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Program is the root of the HIR. It owns every Module produced by
// the bridge, the shared TypeTable, and the symbol-to-TypeId map
// every downstream pass consults.
//
// Iteration over Modules.* uses the sorted Order slice; passes that
// `range` the bare map violate Rule 7.1 (determinism).
type Program struct {
	Types *typetable.Table

	// Modules keyed by dotted module path (empty for crate root).
	Modules map[string]*Module

	// Order is the lexicographically sorted list of module paths.
	// Downstream iteration uses this; Rule 7.1 forbids map-range
	// iteration for anything whose output order is observable.
	Order []string

	// ItemTypes binds a resolve.SymbolID (stored as int to avoid
	// import cycles) to the nominal TypeId of the symbol it declares.
	// W06 and later consult this instead of re-deriving types from
	// the AST.
	ItemTypes map[int]typetable.TypeId
}

// NewProgram builds an empty Program backed by tab. The caller is
// expected to register modules via RegisterModule in the bridge.
func NewProgram(tab *typetable.Table) *Program {
	return &Program{
		Types:     tab,
		Modules:   map[string]*Module{},
		ItemTypes: map[int]typetable.TypeId{},
	}
}

// RegisterModule stores m under its Path. Order is re-sorted so the
// Program exposes a deterministic iteration order without callers
// having to call Finalize manually; a later Finalize call is a no-op.
func (p *Program) RegisterModule(m *Module) {
	p.Modules[m.Path] = m
	p.Order = append(p.Order, m.Path)
	sort.Strings(p.Order)
}

// BindItemType records that the symbol identified by symID declares
// a value with TypeId t. Used by the bridge when it lowers a nominal
// declaration (struct/enum/union/trait/type-alias) and needs to make
// the TypeId addressable through the symbol table.
func (p *Program) BindItemType(symID int, t typetable.TypeId) {
	p.ItemTypes[symID] = t
}

// ItemType returns the nominal TypeId for symID, or NoType when the
// bridge has not registered it. Callers that receive NoType must emit
// a diagnostic — silent defaulting is forbidden (L013).
func (p *Program) ItemType(symID int) typetable.TypeId {
	return p.ItemTypes[symID]
}

// SortedItemSymbols returns every registered item symbol ID in
// numeric order. Tests that compare ItemTypes across runs iterate
// this slice for stability.
func (p *Program) SortedItemSymbols() []int {
	out := make([]int, 0, len(p.ItemTypes))
	for k := range p.ItemTypes {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
