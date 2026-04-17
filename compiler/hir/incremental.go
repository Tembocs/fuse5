package hir

import (
	"sort"
)

// Incremental substitutability is the pass-graph property that lets a
// downstream consumer (W18 driver, W19 language server) recompute
// only the passes whose inputs changed, rather than the whole
// pipeline.
//
// The contract is:
//
//  1. Every pass has a deterministic Fingerprint over its declared
//     inputs. If no input fingerprint changed, the pass output can
//     be reused verbatim (see Manifest + ComputeFingerprint in
//     manifest.go).
//  2. Passes expose a stable OutputKey. A cache entry is keyed by
//     (pass-name, fingerprint) and the cache's value is the
//     serialized output stored under OutputKey.
//  3. When a user edits source, the driver re-runs early passes
//     (parse, resolve, bridge) to produce new inputs. For later
//     passes, the driver computes the new fingerprint and compares
//     to the cached fingerprint. If equal, the driver reuses the
//     cached output without re-running.
//
// IncrementalPlan below is the pure function that implements that
// algorithm. It takes a validated Manifest, the previous run's
// fingerprints, and the current set of "dirty" pass names, and
// returns the list of passes that must re-run.
//
// At W04 there is no I/O to a real cache. The plan is a data
// structure; tests feed it and verify the recompute set shrinks
// when only unrelated inputs change (W04-P05-T03).

// Plan describes which passes must re-run given a set of dirty
// inputs and the previous fingerprint set.
type Plan struct {
	// Rerun is the set of pass Names that must re-execute, in the
	// manifest's topological order.
	Rerun []string
	// Reuse is the set of pass Names that can be served from cache.
	Reuse []string
}

// IncrementalPlan computes a Plan for a Manifest given:
//
//   - m: a validated Manifest (panics otherwise).
//   - dirtyInputs: the set of input names whose contents changed
//     (typically produced by comparing file-level fingerprints
//     before and after the edit).
//
// A pass is marked Rerun when:
//   (a) any of its declared Inputs is in dirtyInputs, OR
//   (b) any Inputs-transitive pass is itself in Rerun.
//
// A pass whose Inputs are entirely clean and transitively clean is
// placed in Reuse.
//
// The algorithm is linear in the manifest size; it walks the
// topological order once and propagates dirty bits.
func IncrementalPlan(m *Manifest, dirtyInputs map[string]bool) Plan {
	order := m.Order()
	if order == nil {
		panic("hir.IncrementalPlan: Manifest not Validate()d")
	}
	rerunSet := map[string]bool{}
	for _, name := range order {
		p := m.byName[name]
		dirty := false
		for _, inp := range p.Inputs() {
			if dirtyInputs[inp] || rerunSet[inp] {
				dirty = true
				break
			}
		}
		// A pass named in dirtyInputs itself also re-runs. This
		// covers the "source file changed" case where the earliest
		// pass (parse) is the source of truth.
		if dirtyInputs[name] {
			dirty = true
		}
		if dirty {
			rerunSet[name] = true
		}
	}

	plan := Plan{}
	for _, name := range order {
		if rerunSet[name] {
			plan.Rerun = append(plan.Rerun, name)
		} else {
			plan.Reuse = append(plan.Reuse, name)
		}
	}
	// Sort for stable output (Rule 7.1) — Rerun and Reuse preserve
	// the manifest's topological order by construction, but the
	// caller's tests often assert against a sorted listing for
	// readability.
	sort.Strings(plan.Rerun)
	sort.Strings(plan.Reuse)
	return plan
}
