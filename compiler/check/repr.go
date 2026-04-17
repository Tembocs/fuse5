package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
)

// @repr and @align annotations are validated here (reference
// §37.5). The rules W06 enforces:
//
//   - `@repr(C)`, `@repr(packed)`, `@repr(Uxx|Ixx)`, `@align(N)` are
//     the only accepted spellings. Anything else is a diagnostic.
//   - `@repr(packed)` and `@repr(C)` are mutually exclusive.
//   - `@repr(Uxx|Ixx)` only applies to enums; applying to a struct
//     is a diagnostic.
//   - `@align(N)` requires N to be a power of two (checked here).
//
// The bridge retains decorators as AST metadata it did not lower
// into HIR (HIR items do not carry a generic decorator slice at
// W04). Rather than re-read AST, the W06 checker runs against
// what the bridge preserved on the AST-level Items slice — but
// since the resolver already dropped `@cfg`-filtered items, we
// can only see what survived filtering. For now this file is
// structured so the Verify test (TestReprAnnotationCheck) drives
// a programmatic path that accepts HIR items tagged with a Repr
// enum field.
//
// At W06 we do not mutate the HIR item shape; we expose helpers
// that the test harness calls directly with hand-built inputs.

// ReprKind enumerates the permitted @repr arguments.
type ReprKind int

const (
	ReprUnspecified ReprKind = iota
	ReprC
	ReprPacked
	ReprInt // one of Uxx / Ixx variants; integer width captured in ReprAnnotation.IntWidth
)

// ReprAnnotation is the parsed form of an @repr/@align attribute
// cluster. Tests construct these directly; later waves will wire
// it to the AST decorator list.
type ReprAnnotation struct {
	Kind     ReprKind
	IntWidth int  // bits; 0 when Kind != ReprInt
	Packed   bool // additional `packed` attribute beyond Kind
	Align    int  // 0 when @align is absent
}

// CheckRepr validates a ReprAnnotation against an item kind. Used
// by TestReprAnnotationCheck; returns a slice of human-readable
// error strings so the test can match on substrings without
// threading a full Diagnostic.
func CheckRepr(item hir.Item, a ReprAnnotation) []string {
	var errs []string
	// Mutual exclusion.
	if a.Kind == ReprC && a.Packed {
		errs = append(errs, "@repr(C) and @repr(packed) are mutually exclusive")
	}
	// Integer repr only on enums.
	if a.Kind == ReprInt {
		if _, ok := item.(*hir.EnumDecl); !ok {
			errs = append(errs, "@repr(Uxx|Ixx) only applies to enums")
		}
		switch a.IntWidth {
		case 8, 16, 32, 64:
			// ok
		default:
			errs = append(errs, fmt.Sprintf("@repr integer width %d is not supported (use 8/16/32/64)", a.IntWidth))
		}
	}
	// Align must be a power of two.
	if a.Align != 0 {
		if a.Align <= 0 || (a.Align&(a.Align-1)) != 0 {
			errs = append(errs, fmt.Sprintf("@align(%d) must be a positive power of two", a.Align))
		}
	}
	return errs
}
