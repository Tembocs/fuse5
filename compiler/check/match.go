package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// W10 match-expression semantics: exhaustiveness and unreachable-arm
// detection on top of the W04 structured pattern nodes. The checker
// enforces these rules on every MatchExpr; match lowering is the
// lowerer's job (W10-P02) and the codegen's cascading-branch
// emission.
//
// Exhaustiveness at W10 covers two scrutinee shapes:
//
//   - KindBool: arms must cover both `true` and `false`, or one of
//     them plus a WildcardPat.
//   - KindEnum: arms must cover every declared variant, or include
//     a WildcardPat. Partial coverage is a diagnostic.
//
// Other scrutinee types (integer ranges, strings, tuples) default
// to "requires a wildcard arm" — a stricter stance than the full
// W10 letter, but it keeps the checker honest until W14 const
// evaluation lets us reason about integer ranges exactly.
//
// Unreachable-arm detection runs after exhaustiveness: an arm whose
// pattern is subsumed by a prior arm is reported with an
// "unreachable arm" diagnostic. W10 handles the common cases:
//
//   - A wildcard arm followed by any other arm: the later arm is
//     unreachable.
//   - A specific variant arm followed by the same variant arm: the
//     second is unreachable.

// CheckMatchExhaustiveness is the entry point called from the
// checker's expression walker when it encounters a MatchExpr. It
// runs both exhaustiveness and unreachable-arm detection against
// the scrutinee's TypeId and the arm list.
func (c *checker) CheckMatchExhaustiveness(m *hir.MatchExpr) {
	if m == nil || m.Scrutinee == nil {
		return
	}
	scrutType := m.Scrutinee.TypeOf()
	c.checkUnreachableArms(m)
	c.checkExhaustive(m, scrutType)
}

// checkExhaustive verifies that the arm list covers every value of
// the scrutinee type. On miss it reports the specific uncovered
// set (e.g. "missing variant `South`").
func (c *checker) checkExhaustive(m *hir.MatchExpr, scrutType typetable.TypeId) {
	if c.armsContainWildcard(m.Arms) {
		return
	}
	t := c.tab.Get(scrutType)
	if t == nil {
		return
	}
	switch t.Kind {
	case typetable.KindBool:
		c.checkExhaustiveBool(m)
	case typetable.KindEnum:
		c.checkExhaustiveEnum(m, scrutType)
	default:
		// Non-enum, non-bool scrutinees must carry a wildcard;
		// we already returned early if one was present. Without
		// a wildcard, the match is non-exhaustive.
		c.diagnose(m.Span,
			fmt.Sprintf("non-exhaustive match on %s: the scrutinee type needs a wildcard arm because the checker cannot enumerate it",
				c.typeName(scrutType)),
			"add a `_ => ...` arm as a catch-all")
	}
}

// checkExhaustiveBool asserts arms cover both `true` and `false`.
func (c *checker) checkExhaustiveBool(m *hir.MatchExpr) {
	coverTrue, coverFalse := false, false
	for _, arm := range m.Arms {
		if lp, ok := arm.Pattern.(*hir.LiteralPat); ok && lp.Kind == hir.LitBool {
			if lp.Bool {
				coverTrue = true
			} else {
				coverFalse = true
			}
		}
		if orp, ok := arm.Pattern.(*hir.OrPat); ok {
			for _, a := range orp.Alts {
				if lp, ok := a.(*hir.LiteralPat); ok && lp.Kind == hir.LitBool {
					if lp.Bool {
						coverTrue = true
					} else {
						coverFalse = true
					}
				}
			}
		}
	}
	if !coverTrue || !coverFalse {
		missing := []string{}
		if !coverTrue {
			missing = append(missing, "true")
		}
		if !coverFalse {
			missing = append(missing, "false")
		}
		c.diagnose(m.Span,
			fmt.Sprintf("non-exhaustive match on Bool: missing %s", strings.Join(missing, ", ")),
			"add the missing arm(s) or a `_ => ...` wildcard")
	}
}

// checkExhaustiveEnum verifies that every declared variant of the
// enum TypeId is covered by at least one arm.
func (c *checker) checkExhaustiveEnum(m *hir.MatchExpr, enumType typetable.TypeId) {
	enumDecl := c.enumDeclForType(enumType)
	if enumDecl == nil {
		return // can't check exhaustiveness without the declaration
	}
	covered := map[string]bool{}
	for _, arm := range m.Arms {
		c.collectCoveredVariants(arm.Pattern, covered)
	}
	missing := []string{}
	for _, v := range enumDecl.Variants {
		if !covered[v.Name] {
			missing = append(missing, v.Name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		c.diagnose(m.Span,
			fmt.Sprintf("non-exhaustive match on enum %s: missing variant(s) %s",
				enumDecl.Name, strings.Join(missing, ", ")),
			"add arms for each missing variant, or add a `_ => ...` wildcard")
	}
}

// collectCoveredVariants flattens or-patterns to gather every
// variant name a single arm covers.
func (c *checker) collectCoveredVariants(p hir.Pat, out map[string]bool) {
	switch x := p.(type) {
	case *hir.ConstructorPat:
		if x.VariantName != "" {
			out[x.VariantName] = true
		} else if len(x.Path) > 0 {
			// Path like ["Dir", "North"] — last segment is variant.
			out[x.Path[len(x.Path)-1]] = true
		}
	case *hir.OrPat:
		for _, a := range x.Alts {
			c.collectCoveredVariants(a, out)
		}
	}
}

// armsContainWildcard returns true when at least one arm has a
// pattern that covers everything: a WildcardPat, a BindPat, or an
// OrPat containing one of those.
func (c *checker) armsContainWildcard(arms []*hir.MatchArm) bool {
	for _, arm := range arms {
		if c.patternIsTotal(arm.Pattern) {
			return true
		}
	}
	return false
}

// patternIsTotal returns true when p matches every possible value.
func (c *checker) patternIsTotal(p hir.Pat) bool {
	switch x := p.(type) {
	case *hir.WildcardPat:
		return true
	case *hir.BindPat:
		return true
	case *hir.OrPat:
		for _, a := range x.Alts {
			if c.patternIsTotal(a) {
				return true
			}
		}
	}
	return false
}

// checkUnreachableArms reports arms that follow a prior total arm
// or re-cover a previously seen variant. The first total arm
// observed wins; every subsequent arm is unreachable.
func (c *checker) checkUnreachableArms(m *hir.MatchExpr) {
	seenTotal := false
	seenVariants := map[string]bool{}
	for i, arm := range m.Arms {
		if seenTotal {
			c.diagnose(arm.Span,
				fmt.Sprintf("unreachable match arm #%d (a prior arm is a catch-all)", i+1),
				"remove the arm or move the catch-all to the end")
			continue
		}
		// Total arms make everything after unreachable.
		if c.patternIsTotal(arm.Pattern) {
			seenTotal = true
			continue
		}
		// Per-variant duplicates.
		thisArm := map[string]bool{}
		c.collectCoveredVariants(arm.Pattern, thisArm)
		for name := range thisArm {
			if seenVariants[name] {
				c.diagnose(arm.Span,
					fmt.Sprintf("unreachable match arm #%d: variant %q was already covered by a prior arm", i+1, name),
					"remove the duplicate arm")
			}
			seenVariants[name] = true
		}
	}
}

// enumDeclForType scans the program for a hir.EnumDecl whose
// TypeID matches tid. Returns nil if the type isn't a user-
// declared enum.
func (c *checker) enumDeclForType(tid typetable.TypeId) *hir.EnumDecl {
	for _, modPath := range c.prog.Order {
		mod := c.prog.Modules[modPath]
		for _, it := range mod.Items {
			if ed, ok := it.(*hir.EnumDecl); ok && ed.TypeID == tid {
				return ed
			}
		}
	}
	return nil
}
