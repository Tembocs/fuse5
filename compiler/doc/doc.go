// Package doc extracts Fuse source-level documentation comments
// and renders a documentation index. W18 scope: every `fn` /
// `struct` / `enum` / `trait` / `impl` / `const` / `static` item
// in a Fuse source is paired with the doc comment immediately
// preceding it. Missing docs on public items surface as a
// diagnostic.
//
// The Rule 5.6 contract (public stdlib APIs require docs) is the
// load-bearing gate. W18 delivers the enforcement path; W20
// stdlib-core is the first consumer.
package doc

import (
	"regexp"
	"strings"
)

// Item is one documented item. Line is the 1-based source line of
// the item declaration. Doc is the extracted doc comment body —
// the `///` prefix is stripped; blank lines become paragraph
// breaks.
type Item struct {
	Kind  string // "fn", "struct", "enum", "trait", "impl", "const", "static"
	Name  string
	Line  int
	Doc   string
	IsPub bool
}

// Extract walks src and returns the Item list in source order.
// The walker is deliberately line-based — the W18 scope is "pair
// doc comments with the immediately-following item declaration";
// a full AST walk lands with W19 when LSP-level doc needs it.
func Extract(src []byte) []Item {
	lines := strings.Split(string(src), "\n")
	var out []Item
	var pendingDoc []string
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(trimmed, "///"):
			pendingDoc = append(pendingDoc, strings.TrimSpace(strings.TrimPrefix(trimmed, "///")))
			continue
		case trimmed == "":
			// Preserve a blank line within a doc run as a
			// paragraph break; drop it between doc runs.
			if len(pendingDoc) > 0 {
				pendingDoc = append(pendingDoc, "")
			}
			continue
		}
		if kind, name, isPub, ok := classifyItem(trimmed); ok {
			out = append(out, Item{
				Kind:  kind,
				Name:  name,
				Line:  i + 1,
				Doc:   strings.TrimSpace(strings.Join(pendingDoc, "\n")),
				IsPub: isPub,
			})
			pendingDoc = nil
			continue
		}
		// Non-item, non-doc line: reset any pending doc.
		pendingDoc = nil
	}
	return out
}

// fnPat / structPat / etc. classify an item-declaration line and
// return (kind, name, isPub). Rule 5.6 needs the pub modifier to
// decide whether docs are mandatory.
var (
	fnPat     = regexp.MustCompile(`^(pub(?:\([^)]*\))?\s+)?fn\s+([A-Za-z_][A-Za-z_0-9]*)`)
	structPat = regexp.MustCompile(`^(pub(?:\([^)]*\))?\s+)?struct\s+([A-Za-z_][A-Za-z_0-9]*)`)
	enumPat   = regexp.MustCompile(`^(pub(?:\([^)]*\))?\s+)?enum\s+([A-Za-z_][A-Za-z_0-9]*)`)
	traitPat  = regexp.MustCompile(`^(pub(?:\([^)]*\))?\s+)?trait\s+([A-Za-z_][A-Za-z_0-9]*)`)
	implPat   = regexp.MustCompile(`^impl\s+(?:[A-Za-z_][A-Za-z_0-9]*\s+for\s+)?([A-Za-z_][A-Za-z_0-9]*)`)
	constPat  = regexp.MustCompile(`^(pub(?:\([^)]*\))?\s+)?const\s+([A-Za-z_][A-Za-z_0-9]*)`)
	staticPat = regexp.MustCompile(`^(pub(?:\([^)]*\))?\s+)?static\s+([A-Za-z_][A-Za-z_0-9]*)`)
)

func classifyItem(line string) (kind, name string, isPub, ok bool) {
	type patMatch struct {
		kind string
		re   *regexp.Regexp
	}
	pats := []patMatch{
		{"fn", fnPat},
		{"struct", structPat},
		{"enum", enumPat},
		{"trait", traitPat},
		{"const", constPat},
		{"static", staticPat},
	}
	for _, p := range pats {
		if m := p.re.FindStringSubmatch(line); m != nil {
			return p.kind, m[2], m[1] != "", true
		}
	}
	if m := implPat.FindStringSubmatch(line); m != nil {
		return "impl", m[1], false, true
	}
	return "", "", false, false
}

// CheckMissingDocs returns a list of names for every public item
// that has an empty Doc field. Rule 5.6 — missing docs on public
// items are a correctness issue.
func CheckMissingDocs(items []Item) []string {
	var missing []string
	for _, it := range items {
		if it.IsPub && it.Doc == "" {
			missing = append(missing, it.Kind+" "+it.Name)
		}
	}
	return missing
}
