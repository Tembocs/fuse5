package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Diagnostic is one type-check diagnostic. The shape mirrors
// lex.Diagnostic so callers merge them into a single diagnostic
// stream without translation.
type Diagnostic = lex.Diagnostic

// Check runs semantic analysis on prog. The returned diagnostics
// enumerate every type error found. On successful completion, every
// Typed HIR node has a concrete TypeId (no KindInfer survives) —
// callers can verify this with `RunInvariantWalker` from the hir
// package or by consulting the `Stats` returned here.
//
// The checker mutates prog in place, replacing KindInfer TypeIds
// with concrete TypeIds as inference succeeds. This is the W04
// contract: HIR is the typed IR, and later passes consult TypeOf()
// directly without a side-car type table.
func Check(prog *hir.Program) []Diagnostic {
	c := newChecker(prog)
	c.collectItems()
	c.registerImplBlocks()
	c.checkItemShapes()
	c.checkAssociatedTypesCoverage()
	c.checkBodies()
	c.checkConcurrency()
	return c.diags
}

// Stats reports high-level counters from a check run. Used by tests
// that want to confirm the checker did meaningful work and to
// monitor pass health over time.
type Stats struct {
	FunctionsChecked int
	ExprsTyped       int
	InferResolved    int
	TraitImpls       int
}

// checker is the mutable state accumulator. One checker per Check
// call; scoped so tests can inspect intermediate state.
type checker struct {
	prog  *hir.Program
	tab   *typetable.Table
	diags []Diagnostic

	// Item-level tables populated in pass 1.
	fnSigs    map[string]*fnSig               // fn NodeID → signature
	traits    map[string]*traitInfo           // trait NodeID → trait body metadata
	impls     []*implBlock                    // all impl blocks in registration order
	implByPair map[coherenceKey]*implBlock    // (trait-TypeId, target-TypeId) → impl; NoType trait = inherent

	// W07 concurrency state (lazy).
	concur *concurrencyContext

	stats Stats
}

// fnSig is the signature-only view of a fn used during body
// checking. Full HIR is preserved via the *hir.FnDecl pointer.
type fnSig struct {
	Decl   *hir.FnDecl
	Module string
}

// traitInfo is the signature-only view of a trait.
type traitInfo struct {
	Decl   *hir.TraitDecl
	Module string
	// Methods is the list of trait method signatures, keyed by the
	// method's name. Associated types and const items are recorded
	// separately for W06-P05 projection.
	Methods    map[string]*hir.FnDecl
	AssocTypes map[string]bool // name → present (no defaults at W06)
}

// implBlock records one impl site. Trait is NoType for inherent
// impls (`impl T { ... }`); otherwise it's the trait's nominal
// TypeId. The methods are registered under (TypeId,method-name) so
// method-call resolution is O(1).
type implBlock struct {
	Decl    *hir.ImplDecl
	Module  string
	Target  typetable.TypeId
	Trait   typetable.TypeId // NoType for inherent
	Methods map[string]*hir.FnDecl
}

// coherenceKey is a (Trait, Type) pair used to enforce the single-
// impl invariant required by reference §12.7.
type coherenceKey struct {
	Trait  typetable.TypeId
	Target typetable.TypeId
}

// newChecker is a tiny helper for tests that want to run setup
// phases in isolation.
func newChecker(prog *hir.Program) *checker {
	return &checker{
		prog:       prog,
		tab:        prog.Types,
		fnSigs:     map[string]*fnSig{},
		traits:     map[string]*traitInfo{},
		implByPair: map[coherenceKey]*implBlock{},
	}
}

// collectItems is pass 1. It walks every module and registers
// fn signatures, trait definitions, and impl blocks. No body
// checking happens here; bodies are handled in pass 2 so that
// items can reference each other regardless of declaration order
// (W06-P01-T02).
func (c *checker) collectItems() {
	if c.fnSigs == nil {
		c.fnSigs = map[string]*fnSig{}
		c.traits = map[string]*traitInfo{}
		c.implByPair = map[coherenceKey]*implBlock{}
	}
	for _, modPath := range c.prog.Order {
		m := c.prog.Modules[modPath]
		for _, it := range m.Items {
			c.collectItem(modPath, it)
		}
	}
}

func (c *checker) collectItem(modPath string, it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		c.fnSigs[string(x.ID)] = &fnSig{Decl: x, Module: modPath}
	case *hir.TraitDecl:
		info := &traitInfo{
			Decl:       x,
			Module:     modPath,
			Methods:    map[string]*hir.FnDecl{},
			AssocTypes: map[string]bool{},
		}
		for _, sub := range x.Items {
			if fn, ok := sub.(*hir.FnDecl); ok {
				info.Methods[fn.Name] = fn
			}
		}
		c.traits[string(x.ID)] = info
	case *hir.ImplDecl:
		// Collected separately in registerImplBlocks to sequence
		// coherence checks after trait info is fully populated.
	}
}

// registerImplBlocks is a second half of pass 1 that runs after
// every trait is known. It registers impl methods, enforces
// coherence (§12.7), and applies the orphan rule.
func (c *checker) registerImplBlocks() {
	for _, modPath := range c.prog.Order {
		m := c.prog.Modules[modPath]
		for _, it := range m.Items {
			if impl, ok := it.(*hir.ImplDecl); ok {
				c.registerImpl(modPath, impl)
			}
		}
	}
}

func (c *checker) registerImpl(modPath string, impl *hir.ImplDecl) {
	blk := &implBlock{
		Decl:    impl,
		Module:  modPath,
		Target:  impl.Target,
		Trait:   impl.Trait, // NoType for inherent impls
		Methods: map[string]*hir.FnDecl{},
	}
	for _, sub := range impl.Items {
		if fn, ok := sub.(*hir.FnDecl); ok {
			blk.Methods[fn.Name] = fn
		}
	}
	key := coherenceKey{Trait: impl.Trait, Target: impl.Target}
	if prior, exists := c.implByPair[key]; exists {
		// Only report a coherence conflict for trait impls; two
		// inherent-impl blocks per type are fine and are how users
		// split methods across files.
		if impl.Trait != typetable.NoType {
			c.diagnose(impl.Span,
				fmt.Sprintf("conflicting impls of trait %s for type %s",
					c.typeName(impl.Trait), c.typeName(impl.Target)),
				fmt.Sprintf("prior impl is in module %q", prior.Module))
			return
		}
	}
	c.implByPair[key] = blk
	c.impls = append(c.impls, blk)
	c.stats.TraitImpls++

	if impl.Trait != typetable.NoType {
		c.checkOrphan(modPath, impl)
	}
}

// checkOrphan enforces the reference §12.7 orphan rule: an impl
// must live in the module that defines the trait OR the target
// type. Primitive targets are a special case — only the module
// that defines the trait can implement for them.
func (c *checker) checkOrphan(modPath string, impl *hir.ImplDecl) {
	traitMod := c.moduleOf(impl.Trait)
	targetMod := c.moduleOf(impl.Target)
	if modPath == traitMod {
		return
	}
	if targetMod == modPath && !c.isPrimitive(impl.Target) {
		return
	}
	c.diagnose(impl.Span,
		fmt.Sprintf("orphan impl: %s for %s must live in the module that defines the trait or the type",
			c.typeName(impl.Trait), c.typeName(impl.Target)),
		fmt.Sprintf("move this impl into %q or %q, or declare a local newtype wrapper",
			traitMod, targetMod))
}

// moduleOf returns the declaring module of a nominal TypeId, or
// the empty string for structural / primitive TypeIds (which belong
// to no single module).
func (c *checker) moduleOf(tid typetable.TypeId) string {
	t := c.tab.Get(tid)
	if t == nil {
		return ""
	}
	return t.Module
}

// isPrimitive returns true when tid names one of the built-in
// primitive kinds. Primitives have no declaring module (orphan
// rule above treats them specially).
func (c *checker) isPrimitive(tid typetable.TypeId) bool {
	t := c.tab.Get(tid)
	return t != nil && t.Kind.IsPrimitive()
}

// typeName returns a human-readable spelling of tid for diagnostics.
// Nominal types use their declared name; structural types fall
// back to the kind string.
func (c *checker) typeName(tid typetable.TypeId) string {
	if tid == typetable.NoType {
		return "<no-type>"
	}
	t := c.tab.Get(tid)
	if t == nil {
		return "<invalid>"
	}
	if t.Kind.IsNominal() && t.Name != "" {
		return t.Name
	}
	return t.Kind.String()
}

// diagnose appends a diagnostic with a primary span and a
// one-line message (Rule 6.17).
func (c *checker) diagnose(span lex.Span, msg, hint string) {
	c.diags = append(c.diags, Diagnostic{Span: span, Message: msg, Hint: hint})
}
