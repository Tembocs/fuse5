package lex

import "fmt"

// Position is a single location in a source file: a 1-based (line, column) pair
// with a 0-based byte offset. Columns count UTF-8 bytes past the start of the
// logical line, not user-visible characters — this keeps spans trivially mappable
// back to a byte range (Rule 7.4).
//
// Line counting treats LF and CRLF as a single newline (reference §1.1). The
// byte offset always points at the original bytes, before any normalization.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Span is the inclusive-start, exclusive-end byte range a token occupies in the
// source file. File is the logical filename the scanner was given; spans are
// not cross-file — a later wave composes spans across files via SourceMap.
type Span struct {
	File  string
	Start Position
	End   Position
}

// IsZero reports whether the span has no location information.
func (s Span) IsZero() bool {
	return s == Span{}
}

// Len returns the span's byte length. Always non-negative for spans produced by
// the scanner.
func (s Span) Len() int {
	return s.End.Offset - s.Start.Offset
}

// String renders the span as file:line:col..line:col for diagnostics and
// goldens. The format is stable (Rule 7.4).
func (s Span) String() string {
	if s.File == "" {
		return fmt.Sprintf("%d:%d..%d:%d",
			s.Start.Line, s.Start.Column, s.End.Line, s.End.Column)
	}
	return fmt.Sprintf("%s:%d:%d..%d:%d",
		s.File, s.Start.Line, s.Start.Column, s.End.Line, s.End.Column)
}
