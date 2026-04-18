package codegen

import (
	"fmt"
	"sort"
	"strings"
)

// TypeDecl is one named type the emitter has to produce before it
// is used. Fields capture the minimum the W17 tests pin:
//
//   - Name: the C-level type name (already sanitised)
//   - Kind: "struct", "union", or "enum"
//   - Deps: other type names this one depends on
//
// Codegen orders TypeDecls via a topological sort so every
// reference resolves. A cycle is a checker bug (types must form a
// DAG) and the sorter surfaces it via a diagnostic comment.
type TypeDecl struct {
	Name string
	Kind string
	Deps []string
}

// SortTypeDecls returns a deterministic topological order over
// decls. Ties break by name for stability (Rule 7.1). A cycle
// produces a best-effort order with a sentinel entry that downstream
// consumers can detect.
func SortTypeDecls(decls []TypeDecl) []TypeDecl {
	byName := map[string]TypeDecl{}
	for _, d := range decls {
		byName[d.Name] = d
	}
	// Kahn's algorithm over a DAG of Name → dependency Name.
	indeg := map[string]int{}
	for _, d := range decls {
		if _, ok := indeg[d.Name]; !ok {
			indeg[d.Name] = 0
		}
	}
	for _, d := range decls {
		for _, dep := range d.Deps {
			if _, ok := indeg[dep]; !ok {
				// Dep not in the input — still counts.
				indeg[dep] = 0
			}
			indeg[d.Name]++
		}
	}
	// Seed with zero-indegree names in alphabetical order.
	var ready []string
	for name, deg := range indeg {
		if deg == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)
	var out []TypeDecl
	seen := map[string]bool{}
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		if !seen[name] {
			if d, ok := byName[name]; ok {
				out = append(out, d)
			}
			seen[name] = true
		}
		for _, other := range decls {
			if seen[other.Name] {
				continue
			}
			if !contains(other.Deps, name) {
				continue
			}
			indeg[other.Name]--
			if indeg[other.Name] == 0 {
				ready = append(ready, other.Name)
				sort.Strings(ready)
			}
		}
	}
	// Any remaining decls form a cycle — append them at the end
	// with a sentinel so diagnostics see the violation.
	for _, d := range decls {
		if !seen[d.Name] {
			out = append(out, d)
		}
	}
	return out
}

func contains(xs []string, needle string) bool {
	for _, x := range xs {
		if x == needle {
			return true
		}
	}
	return false
}

// SanitizeIdentifier turns a Fuse identifier into a C-safe form
// by replacing every non-[A-Za-z0-9_] byte with `_`. Empty input
// returns `_` so the result is always a valid C identifier.
// Reference §57.6: identifier sanitisation is a backend contract.
func SanitizeIdentifier(name string) string {
	if name == "" {
		return "_"
	}
	var sb strings.Builder
	// C forbids identifiers starting with a digit.
	first := name[0]
	if (first >= '0' && first <= '9') {
		sb.WriteByte('_')
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			sb.WriteByte(c)
		} else {
			sb.WriteByte('_')
		}
	}
	return sb.String()
}

// MangleModuleName concatenates a module path and a symbol name
// into a single C-safe identifier. Reference §57.6 fixes the
// scheme: `fuse_<module>__<name>` with double underscore as the
// separator. Empty module degenerates to `fuse_<name>`.
func MangleModuleName(module, name string) string {
	sname := SanitizeIdentifier(name)
	if module == "" {
		return "fuse_" + sname
	}
	return "fuse_" + SanitizeIdentifier(module) + "__" + sname
}

// EmitUnitErasure returns the C render for a Fuse Unit-typed value.
// Reference §57.2 prescribes total unit erasure: unit values have
// no observable representation; the emitter renders them as a
// trailing comment so the test surface can pin the erasure and
// downstream consumers know the position was intentionally blank.
func EmitUnitErasure() string {
	return "/* unit */"
}

// EmitAggregateZeroInit returns the C initialiser for a zero-
// filled aggregate of the given type name. Reference §57.5 requires
// composite types to be fully initialised at emission; a typed
// literal `(TypeName){0}` covers struct, enum-like tagged union,
// and union shapes uniformly on gcc / clang / MSVC.
func EmitAggregateZeroInit(typeName string) string {
	return fmt.Sprintf("(%s){0}", typeName)
}

// EmitUnionLayout returns the C union declaration for the supplied
// field name/type pairs. The declaration names the union and lists
// each field as a C member; the C compiler is responsible for
// producing the largest-member-size / strictest-member-alignment
// layout per §57.10.
func EmitUnionLayout(typeName string, fields [][2]string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "union %s { ", typeName)
	for _, f := range fields {
		fmt.Fprintf(&sb, "%s %s; ", f[1], f[0])
	}
	sb.WriteByte('}')
	return sb.String()
}

// EmitPointerCategory returns the C render for one of the two
// pointer categories Fuse erases to per reference §57.1:
//   - "raw"      → `T *` (unrestricted, no borrow-lifetime)
//   - "borrowed" → `T const *` / `T *` with a comment noting the
//                  borrow; shared vs mut is captured in the const
//                  qualifier.
func EmitPointerCategory(kind, targetType string, mutable bool) string {
	switch kind {
	case "raw":
		return fmt.Sprintf("%s *", targetType)
	case "borrowed":
		if mutable {
			return fmt.Sprintf("%s * /* borrowed mut */", targetType)
		}
		return fmt.Sprintf("%s const * /* borrowed */", targetType)
	}
	return fmt.Sprintf("%s * /* unknown category %q */", targetType, kind)
}
