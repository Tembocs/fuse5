package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a semver (MAJOR.MINOR.PATCH) identifier. Pre-release
// and build metadata are not supported at W23 — they land with
// the reference-registry launch.
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a version token like "1.2.3". Short
// forms ("1" / "1.2") are accepted for range specs and are
// padded with zeros — ParseRange relies on this so `^1` is
// `^1.0.0`.
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Version{}, fmt.Errorf("version %q: expected MAJOR[.MINOR[.PATCH]]", s)
	}
	out := Version{}
	var err error
	if out.Major, err = strconv.Atoi(parts[0]); err != nil {
		return out, fmt.Errorf("version %q: invalid major: %v", s, err)
	}
	if len(parts) >= 2 {
		if out.Minor, err = strconv.Atoi(parts[1]); err != nil {
			return out, fmt.Errorf("version %q: invalid minor: %v", s, err)
		}
	}
	if len(parts) == 3 {
		if out.Patch, err = strconv.Atoi(parts[2]); err != nil {
			return out, fmt.Errorf("version %q: invalid patch: %v", s, err)
		}
	}
	if out.Major < 0 || out.Minor < 0 || out.Patch < 0 {
		return out, fmt.Errorf("version %q: components must be non-negative", s)
	}
	return out, nil
}

// String renders the version as MAJOR.MINOR.PATCH.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1 / 0 / +1 in the usual semver ordering.
func (v Version) Compare(o Version) int {
	switch {
	case v.Major != o.Major:
		if v.Major < o.Major {
			return -1
		}
		return 1
	case v.Minor != o.Minor:
		if v.Minor < o.Minor {
			return -1
		}
		return 1
	case v.Patch != o.Patch:
		if v.Patch < o.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// LessThan reports whether v < o.
func (v Version) LessThan(o Version) bool { return v.Compare(o) < 0 }

// Range expresses a semver range. Supported forms:
//
//	"1.2.3"   — exact version match
//	"^1.2.3"  — compatible-release: >=1.2.3, <2.0.0
//	"~1.2.3"  — tilde-patch: >=1.2.3, <1.3.0
//	">=1.0.0" / "<2.0.0" / ">=1.0.0, <2.0.0" — explicit bounds
//	"*"       — any version
//
// The range semantics is precisely specified so tie-breaks and
// conflict cases are never emergent (W23-P02 exit criterion).
type Range struct {
	// LowerInclusive and UpperExclusive bound the allowed
	// version set. A zero Version for Upper means "unbounded
	// upward"; see Unbounded.
	Lower     Version
	Upper     Version
	Unbounded bool // true when there is no upper bound
	// Raw preserves the input form for diagnostics.
	Raw string
}

// ParseRange decodes a range spec per the grammar above.
func ParseRange(spec string) (Range, error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return Range{}, fmt.Errorf("empty version range")
	}
	if s == "*" {
		return Range{Unbounded: true, Raw: spec}, nil
	}
	if strings.HasPrefix(s, "^") {
		base, err := ParseVersion(strings.TrimPrefix(s, "^"))
		if err != nil {
			return Range{}, err
		}
		upper := Version{Major: base.Major + 1}
		if base.Major == 0 {
			// ^0.x.y → >=0.x.y, <0.(x+1).0 per semver cargo
			// tradition; ^0.0.y → >=0.0.y, <0.0.(y+1).
			if base.Minor == 0 {
				upper = Version{Major: 0, Minor: 0, Patch: base.Patch + 1}
			} else {
				upper = Version{Major: 0, Minor: base.Minor + 1}
			}
		}
		return Range{Lower: base, Upper: upper, Raw: spec}, nil
	}
	if strings.HasPrefix(s, "~") {
		base, err := ParseVersion(strings.TrimPrefix(s, "~"))
		if err != nil {
			return Range{}, err
		}
		upper := Version{Major: base.Major, Minor: base.Minor + 1}
		return Range{Lower: base, Upper: upper, Raw: spec}, nil
	}
	if strings.Contains(s, ",") || strings.HasPrefix(s, ">") || strings.HasPrefix(s, "<") || strings.HasPrefix(s, "=") {
		return parseCompoundRange(s)
	}
	// Exact version.
	v, err := ParseVersion(s)
	if err != nil {
		return Range{}, err
	}
	upper := Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	return Range{Lower: v, Upper: upper, Raw: spec}, nil
}

// parseCompoundRange handles ">=X.Y.Z, <A.B.C" and single-
// sided inequality forms.
func parseCompoundRange(s string) (Range, error) {
	clauses := strings.Split(s, ",")
	r := Range{Unbounded: true, Raw: s}
	haveLower := false
	for _, c := range clauses {
		c = strings.TrimSpace(c)
		switch {
		case strings.HasPrefix(c, ">="):
			v, err := ParseVersion(strings.TrimPrefix(c, ">="))
			if err != nil {
				return Range{}, err
			}
			r.Lower = v
			haveLower = true
		case strings.HasPrefix(c, "<"):
			v, err := ParseVersion(strings.TrimPrefix(c, "<"))
			if err != nil {
				return Range{}, err
			}
			r.Upper = v
			r.Unbounded = false
		case strings.HasPrefix(c, "="):
			v, err := ParseVersion(strings.TrimPrefix(c, "="))
			if err != nil {
				return Range{}, err
			}
			r.Lower = v
			r.Upper = Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
			r.Unbounded = false
			haveLower = true
		default:
			return Range{}, fmt.Errorf("unknown range clause %q", c)
		}
	}
	if !haveLower {
		// Single upper-only clause; set lower to 0.0.0.
		r.Lower = Version{}
	}
	return r, nil
}

// Contains reports whether v satisfies the range.
func (r Range) Contains(v Version) bool {
	if v.LessThan(r.Lower) {
		return false
	}
	if r.Unbounded {
		return true
	}
	return v.LessThan(r.Upper)
}

// Intersect returns the range that is the conjunction of r and
// other, or ok=false when the two are disjoint.
func (r Range) Intersect(other Range) (Range, bool) {
	lower := r.Lower
	if other.Lower.Compare(lower) > 0 {
		lower = other.Lower
	}
	out := Range{Lower: lower, Raw: r.Raw + " & " + other.Raw}
	if r.Unbounded && other.Unbounded {
		out.Unbounded = true
		return out, true
	}
	upper := r.Upper
	if r.Unbounded {
		upper = other.Upper
	} else if !other.Unbounded && other.Upper.Compare(upper) < 0 {
		upper = other.Upper
	}
	out.Upper = upper
	out.Unbounded = false
	// The intersected range is empty (disjoint) when the
	// tightened lower bound is NOT strictly less than the
	// tightened upper bound. An equal pair is also disjoint
	// because Upper is exclusive.
	if !out.Lower.LessThan(out.Upper) {
		return Range{}, false
	}
	return out, true
}

// SelectLatest picks the largest version from `candidates` that
// satisfies `r`. Returns ok=false when no candidate satisfies.
// Deterministic: ties break by preferring the first occurrence
// in input order, but since candidates are expected to be
// sorted newest-first the result is stable.
func (r Range) SelectLatest(candidates []Version) (Version, bool) {
	var best Version
	have := false
	for _, c := range candidates {
		if !r.Contains(c) {
			continue
		}
		if !have || best.LessThan(c) {
			best = c
			have = true
		}
	}
	return best, have
}
