package consteval

import "github.com/Tembocs/fuse5/compiler/lex"

// DiagsToLex converts a slice of consteval.Diagnostic values into a
// slice of lex.Diagnostic. Because Diagnostic is a type alias for
// lex.Diagnostic today the conversion is a cheap copy; exposing it
// through a helper isolates callers from any future divergence
// (e.g. a wave that adds a consteval-specific extension field).
func DiagsToLex(in []Diagnostic) []lex.Diagnostic {
	if len(in) == 0 {
		return nil
	}
	out := make([]lex.Diagnostic, len(in))
	for i, d := range in {
		out[i] = d
	}
	return out
}
