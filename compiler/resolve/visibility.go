package resolve

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/lex"
)

// enforceVisibility walks every recorded binding and emits a diagnostic
// when the using module is not allowed to see the target symbol under
// reference §53.1. Four levels are recognized:
//
//   - private (default): only the declaring module itself.
//   - `pub(mod)`: the declaring module and descendants of the declaring
//     module (dotted-prefix match).
//   - `pub(pkg)`: every module in the build (the whole package boundary
//     at W03 — the multi-package story lands with W23).
//   - `pub`: visible everywhere an import can reach.
//
// Imports are treated as visibility-checked at the import site and
// re-use via the alias within a module does not re-check. Likewise,
// enum variants inherit the enclosing enum's visibility (reference
// §11.6, §53.1).
func (r *resolver) enforceVisibility() {
	for key, id := range r.bindings {
		s := r.symbols.Get(id)
		if s == nil {
			continue
		}
		if !isVisibleFrom(s, key.Module) {
			r.diagnose(key.Span,
				fmt.Sprintf("%s %q is not visible from module %q",
					s.Kind.String(), s.Name, key.Module),
				visibilityHint(s))
		}
	}
}

// isVisibleFrom returns true when the site in `usingMod` is allowed to
// name `s`. The rules follow the four-level hierarchy from §53.1.
func isVisibleFrom(s *Symbol, usingMod string) bool {
	if s.Module == usingMod {
		return true
	}
	switch s.Vis {
	case 0: // VisPrivate
		return false
	case 1: // VisPub
		return true
	case 2: // VisPubMod
		return isDescendant(s.Module, usingMod)
	case 3: // VisPubPkg
		return true
	}
	return false
}

// isDescendant returns true when usingMod equals ancestorMod or is a
// dotted descendant of it. Reference §53.1: `pub(mod)` is visible in
// the declaring module and its descendants.
func isDescendant(ancestorMod, usingMod string) bool {
	if ancestorMod == usingMod {
		return true
	}
	if ancestorMod == "" {
		return true
	}
	if len(usingMod) <= len(ancestorMod) {
		return false
	}
	return usingMod[:len(ancestorMod)] == ancestorMod && usingMod[len(ancestorMod)] == '.'
}

// visibilityHint returns a suggestion consistent with Rule 6.17: name
// the visibility level that would let the use site compile.
func visibilityHint(s *Symbol) string {
	switch s.Vis {
	case 0:
		return fmt.Sprintf("mark it `pub(mod)`, `pub(pkg)`, or `pub` in module %q", s.Module)
	case 2:
		return "declare the use site inside the declaring module or one of its descendants"
	}
	return "widen the item's visibility if the use site needs it"
}

// discardUnusedLex keeps the import referenced even when no callable
// function in this file uses it directly — the file references lex via
// Diagnostic creation elsewhere.
var _ = lex.Span{}
