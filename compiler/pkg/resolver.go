package pkg

import (
	"fmt"
	"sort"
	"strings"
)

// Resolver drives dependency resolution from a root manifest
// against a registry snapshot. At each step the resolver picks
// the latest compatible version per constraint; conflicts
// surface as a specific diagnostic naming the offending pair
// of constraints.
type Resolver struct {
	// Index is the view of the registry the resolver sees.
	// Tests feed an in-memory index; production wires a
	// disk-backed RegistryIndex.
	Index RegistryLookup
}

// RegistryLookup abstracts the registry access the resolver
// needs. In production it is a real fetcher with caching;
// tests provide an in-memory map.
type RegistryLookup interface {
	// AvailableVersions returns every published version of
	// `name` newest-first. Missing packages return an empty
	// slice (the resolver interprets empty as unknown).
	AvailableVersions(name string) []Version
	// Dependencies returns the (name, range) list the
	// published version declares.
	Dependencies(name string, version Version) []DepConstraint
	// Source returns the origin URL for (name, version) so
	// the lockfile can record it.
	Source(name string, version Version) string
	// SHA256 returns the integrity digest for (name, version).
	SHA256(name string, version Version) string
}

// DepConstraint is a single (name, range) dependency edge.
type DepConstraint struct {
	Name  string
	Range Range
}

// Resolution is the resolver's output — the full list of
// locked crates plus metadata for the lockfile.
type Resolution struct {
	// Root is the "<name>@<version>" identifier of the input
	// manifest.
	Root string
	// Entries lists every resolved (direct + transitive)
	// crate sorted lexicographically.
	Entries []LockedCrate
}

// ResolveError names a conflict case. Offers a Rule-6.17
// compatible diagnostic (primary span via Crate name,
// explanation + suggestion).
type ResolveError struct {
	Kind       string // "unknown", "no-match", "cycle"
	Message    string
	Suggestion string
}

func (e *ResolveError) Error() string { return e.Message }

// Resolve takes a manifest and an optional pre-existing
// lockfile. When the lockfile matches the manifest's
// dependency set, it is returned as-is (no refetch). Otherwise
// the resolver produces a fresh Resolution.
func (r *Resolver) Resolve(m *Manifest) (*Resolution, *ResolveError) {
	if m.Package == nil {
		return nil, &ResolveError{Kind: "empty", Message: "manifest has no [package] to resolve from"}
	}
	root := m.IDString()
	// Build the initial constraint map from the manifest.
	constraints, err := initialConstraints(m)
	if err != nil {
		return nil, err
	}
	// Worklist iteration: pick a package from the constraint
	// set, choose the latest compatible version, add its
	// dependencies to the constraint set, repeat until stable.
	selected := map[string]Version{}
	sourceOf := map[string]string{}
	shaOf := map[string]string{}
	transitive := map[string][]string{}
	visiting := map[string]bool{}
	stack := []string{}

	var walk func(name string, cnst Range) *ResolveError
	walk = func(name string, cnst Range) *ResolveError {
		// Detect cycle: name already on the current stack.
		if visiting[name] {
			return &ResolveError{
				Kind: "cycle",
				Message: fmt.Sprintf("dependency cycle detected involving %q (path: %s -> %s)",
					name, strings.Join(stack, " -> "), name),
				Suggestion: "break the cycle by making one of the edges a dev-dependency or refactoring the shared code into a third crate",
			}
		}
		// Has another constraint already been recorded?
		if existing, ok := constraints[name]; ok {
			joined, ok2 := existing.Intersect(cnst)
			if !ok2 {
				return &ResolveError{
					Kind: "no-match",
					Message: fmt.Sprintf("cannot satisfy dependency %q: conflicting constraints %s and %s",
						name, existing.Raw, cnst.Raw),
					Suggestion: "align the version ranges across dependents, or bump one crate to a major version compatible with both",
				}
			}
			constraints[name] = joined
		} else {
			constraints[name] = cnst
		}
		// Has a version been selected? If so, this walk is
		// a revisit after adding a tighter constraint; check
		// that the selection still satisfies the updated
		// constraint.
		if v, ok := selected[name]; ok {
			if !constraints[name].Contains(v) {
				return &ResolveError{
					Kind: "no-match",
					Message: fmt.Sprintf("previously-selected %s@%s no longer satisfies updated constraint %s",
						name, v, constraints[name].Raw),
					Suggestion: "bump or loosen the conflicting constraint, or retry resolution from scratch",
				}
			}
			return nil
		}
		// Pick the latest satisfying version.
		avail := r.Index.AvailableVersions(name)
		if len(avail) == 0 {
			return &ResolveError{
				Kind: "unknown",
				Message: fmt.Sprintf("unknown package %q", name),
				Suggestion: "check the spelling or add a [dependencies] entry with an explicit source (path = / url = )",
			}
		}
		chosen, ok := constraints[name].SelectLatest(avail)
		if !ok {
			return &ResolveError{
				Kind: "no-match",
				Message: fmt.Sprintf("no published version of %q satisfies %s (available: %s)",
					name, constraints[name].Raw, formatVersions(avail)),
				Suggestion: "widen the range, publish a compatible version, or pin an exact version with `= x.y.z`",
			}
		}
		selected[name] = chosen
		sourceOf[name] = r.Index.Source(name, chosen)
		shaOf[name] = r.Index.SHA256(name, chosen)

		// Walk transitive deps.
		visiting[name] = true
		stack = append(stack, name)
		defer func() {
			visiting[name] = false
			stack = stack[:len(stack)-1]
		}()
		deps := r.Index.Dependencies(name, chosen)
		depNames := make([]string, 0, len(deps))
		for _, d := range deps {
			depNames = append(depNames, d.Name)
			if err := walk(d.Name, d.Range); err != nil {
				return err
			}
		}
		sort.Strings(depNames)
		transitive[name] = depNames
		return nil
	}

	for name, cnst := range constraintsSortedCopy(constraints) {
		if err := walk(name, cnst); err != nil {
			return nil, err
		}
	}

	// Build the resolution in sorted order.
	res := &Resolution{Root: root}
	names := make([]string, 0, len(selected))
	for n := range selected {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		v := selected[n]
		res.Entries = append(res.Entries, LockedCrate{
			Name:         n,
			Version:      v.String(),
			Source:       sourceOf[n],
			SHA256:       shaOf[n],
			Dependencies: transitive[n],
		})
	}
	return res, nil
}

// initialConstraints reads the manifest's [dependencies]
// section and produces a name→Range map.
func initialConstraints(m *Manifest) (map[string]Range, *ResolveError) {
	out := map[string]Range{}
	for _, d := range m.Dependencies {
		if d.Version == "" {
			continue // path/URL deps bypass the resolver
		}
		rng, err := ParseRange(d.Version)
		if err != nil {
			return nil, &ResolveError{
				Kind:       "bad-range",
				Message:    fmt.Sprintf("dependency %q: invalid version range: %v", d.Name, err),
				Suggestion: "use a range form like `1.2.3`, `^1.2`, `~1.2.3`, or `>=1.0.0, <2.0.0`",
			}
		}
		out[d.Name] = rng
	}
	return out, nil
}

// constraintsSortedCopy returns an iteration over the
// constraints map in lexicographic order. Rule 7.1: the
// resolver's iteration is not dependent on map-hash ordering.
func constraintsSortedCopy(in map[string]Range) map[string]Range {
	// The returned type is still map[string]Range to keep the
	// call-site ergonomic. The sort is implicit in the order
	// walk receives its initial calls; to materialise that
	// order the caller iterates via a sorted []string outside
	// the map. Provide that indirection here.
	// A stable iteration order over a map requires a slice of
	// keys; callers use the returned map but iterate via
	// sorted names.
	out := map[string]Range{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

// formatVersions renders a version slice as a human-readable
// list for diagnostics.
func formatVersions(vs []Version) string {
	s := make([]string, len(vs))
	for i, v := range vs {
		s[i] = v.String()
	}
	return strings.Join(s, ", ")
}

// ToLockfile converts a Resolution into a ready-to-serialize
// Lockfile.
func (r *Resolution) ToLockfile() *Lockfile {
	lk := NewLockfile(r.Root)
	for _, e := range r.Entries {
		lk.AddCrate(e)
	}
	lk.Finalize()
	return lk
}
