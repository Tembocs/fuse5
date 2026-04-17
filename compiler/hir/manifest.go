package hir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// The pass manifest is the architectural foundation for:
//
//   - Deterministic pipeline ordering (W04-P04-T03).
//   - Invariant-walker placement between passes (W04-P04-T02).
//   - Incremental recompilation (W18) and the language server (W19).
//
// Every pass declares its inputs (named outputs of prior passes), a
// stable output key, and a Fingerprint function that hashes its
// inputs into a byte-identical digest across runs and platforms.
//
// At W04 the manifest is shaped — no real checker/monomorph/codegen
// passes are wired yet (those are W06+ work). What matters here is
// that later waves plug in without reshaping the graph.

// Pass is the minimal contract every compilation pass must satisfy.
//
//   - Name returns a globally unique identifier. Passes may not
//     share names; Manifest.Register panics on a duplicate.
//   - Inputs declares other pass Names whose outputs this pass
//     consumes. Used for topological ordering AND for the
//     fingerprint function's domain.
//   - OutputKey is a stable string naming what this pass produces.
//     The manifest wires it into the cache that W18 will read.
//   - Fingerprint returns a byte-deterministic digest of the pass's
//     logical inputs. It must be stable across runs and platforms
//     (Rule 7.1). The manifest hashes in the pass name so two
//     passes with the same inputs still produce distinct prints.
//   - Run executes the pass against a PassContext. A pass's Run is
//     free to mutate PassContext.Outputs but must not change its
//     Inputs.
type Pass interface {
	Name() string
	Inputs() []string
	OutputKey() string
	Fingerprint(inputs map[string][]byte) []byte
	Run(ctx *PassContext) error
}

// PassContext is the shared state handed to every pass's Run. It
// owns the running Program, the TypeTable, and a cache of each
// pass's output bytes keyed by that pass's OutputKey.
type PassContext struct {
	Program *Program
	Outputs map[string][]byte
}

// NewPassContext builds a PassContext for p. Outputs is always
// allocated so passes never have to nil-check.
func NewPassContext(p *Program) *PassContext {
	return &PassContext{
		Program: p,
		Outputs: map[string][]byte{},
	}
}

// Manifest is the ordered pass graph. Passes are registered, then
// Validate is called to check for cycles and missing inputs, and
// Run executes them in topological order.
type Manifest struct {
	byName map[string]Pass
	order  []string // topological order, filled by Validate
	built  bool
}

// NewManifest returns an empty Manifest.
func NewManifest() *Manifest {
	return &Manifest{byName: map[string]Pass{}}
}

// Register adds a pass. Duplicate names panic (a collision is a
// pipeline bug, not a user error).
func (m *Manifest) Register(p Pass) {
	name := p.Name()
	if _, ok := m.byName[name]; ok {
		panic(fmt.Sprintf("hir.Manifest: duplicate pass %q", name))
	}
	m.byName[name] = p
	m.built = false
}

// Passes returns the passes in registration order (pre-Validate) or
// topological order (post-Validate). Tests assert determinism
// against this slice.
func (m *Manifest) Passes() []Pass {
	if m.built {
		out := make([]Pass, len(m.order))
		for i, n := range m.order {
			out[i] = m.byName[n]
		}
		return out
	}
	// Not yet validated — return a lexicographic listing for
	// stability (tests that inspect pre-Validate state still see a
	// deterministic order).
	names := make([]string, 0, len(m.byName))
	for n := range m.byName {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Pass, len(names))
	for i, n := range names {
		out[i] = m.byName[n]
	}
	return out
}

// Validate confirms every declared Inputs name exists in the
// manifest and the resulting dependency graph is acyclic. On
// success it computes the topological order used by subsequent
// calls to Run. The order is deterministic: ties between
// equally-ready passes are broken by Name in lexicographic order
// (Rule 7.1).
func (m *Manifest) Validate() error {
	// First pass: check declared inputs exist.
	for name, p := range m.byName {
		for _, inp := range p.Inputs() {
			if _, ok := m.byName[inp]; !ok {
				return fmt.Errorf("pass %q declares unknown input %q", name, inp)
			}
		}
	}
	// Kahn's algorithm with lexicographic tie-breaking for
	// determinism.
	indeg := map[string]int{}
	adj := map[string][]string{}
	for name, p := range m.byName {
		if _, ok := indeg[name]; !ok {
			indeg[name] = 0
		}
		for _, inp := range p.Inputs() {
			adj[inp] = append(adj[inp], name)
			indeg[name]++
		}
	}
	// Sort adjacency lists for determinism.
	for k := range adj {
		sort.Strings(adj[k])
	}
	// Ready queue: every pass with indeg 0, sorted lexicographically.
	var ready []string
	for name, d := range indeg {
		if d == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(m.byName))
	for len(ready) > 0 {
		// Always drain the lexicographically smallest ready pass
		// first so the traversal is deterministic.
		next := ready[0]
		ready = ready[1:]
		order = append(order, next)
		for _, dep := range adj[next] {
			indeg[dep]--
			if indeg[dep] == 0 {
				ready = append(ready, dep)
			}
		}
		sort.Strings(ready)
	}
	if len(order) != len(m.byName) {
		return fmt.Errorf("pass graph has a cycle; resolved %d of %d passes", len(order), len(m.byName))
	}
	m.order = order
	m.built = true
	return nil
}

// Order returns the validated topological order. Callers must have
// already invoked Validate; a nil/empty return indicates Validate
// was never called (or failed).
func (m *Manifest) Order() []string {
	if !m.built {
		return nil
	}
	out := make([]string, len(m.order))
	copy(out, m.order)
	return out
}

// Run executes every pass in validated order against ctx. Returns
// on the first pass error. Callers must Validate first.
func (m *Manifest) Run(ctx *PassContext) error {
	if !m.built {
		return fmt.Errorf("hir.Manifest.Run: call Validate first")
	}
	for _, name := range m.order {
		if err := m.byName[name].Run(ctx); err != nil {
			return fmt.Errorf("pass %q: %w", name, err)
		}
	}
	return nil
}

// ComputeFingerprint returns the sha256 digest of a pass's logical
// inputs, folding the pass Name into the hash so two passes with
// identical inputs still produce distinct digests.
func ComputeFingerprint(name string, inputs map[string][]byte) []byte {
	h := sha256.New()
	// Pass name first; terminated with NUL to avoid any ambiguity
	// between name bytes and the first input key.
	h.Write([]byte(name))
	h.Write([]byte{0})
	// Input keys sorted for determinism.
	keys := make([]string, 0, len(inputs))
	for k := range inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write(inputs[k])
		h.Write([]byte{0})
	}
	sum := h.Sum(nil)
	return sum
}

// FingerprintHex is a small convenience for diagnostic output.
func FingerprintHex(fp []byte) string { return hex.EncodeToString(fp) }
