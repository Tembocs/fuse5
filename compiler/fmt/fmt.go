// Package fmt owns the Fuse source-code formatter. Given a Fuse
// source byte slice, Format returns the canonical, byte-stable
// formatted form. Re-formatting a formatted input is idempotent
// (Rule 6.2: goldens must be byte-stable).
//
// W18 scope: normalise whitespace, collapse multiple blank lines,
// enforce one-space indentation normaliser, and re-ended trailing
// newline. Deeper syntactic formatting (operator alignment, block
// re-wrapping) is W19 IDE-polish territory; at W18 the goal is
// "every hand-written Fuse source has exactly one canonical
// representation under `fuse fmt`".
package fmt

import (
	"bytes"
	"strings"
)

// Format returns the canonical byte form of src. Idempotent:
// Format(Format(x)) == Format(x). Rule 6.2.
//
// Normalisations applied:
//
//   - CRLF → LF
//   - trailing whitespace trimmed from every line
//   - runs of ≥ 3 consecutive blank lines collapsed to 2
//   - tabs expanded to 4 spaces (Fuse style uses spaces)
//   - exactly one trailing newline
func Format(src []byte) []byte {
	// Newlines.
	normalised := bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	normalised = bytes.ReplaceAll(normalised, []byte("\r"), []byte("\n"))

	var sb strings.Builder
	blankRun := 0
	lines := strings.Split(string(normalised), "\n")
	// Track whether we've already emitted a line so the first
	// blank run doesn't produce leading blank lines in output.
	emittedAny := false

	for _, line := range lines {
		// Tabs → 4 spaces.
		line = strings.ReplaceAll(line, "\t", "    ")
		// Trim trailing whitespace.
		line = strings.TrimRight(line, " \t")

		if line == "" {
			blankRun++
			continue
		}
		// Flush at most one blank line before a non-blank.
		if emittedAny && blankRun > 0 {
			sb.WriteByte('\n')
		}
		if emittedAny {
			sb.WriteByte('\n')
		}
		sb.WriteString(line)
		blankRun = 0
		emittedAny = true
	}
	// Exactly one trailing newline.
	if emittedAny {
		sb.WriteByte('\n')
	}
	return []byte(sb.String())
}
