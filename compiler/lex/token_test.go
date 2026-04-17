package lex

import "testing"

// TestTokenKindCoverage confirms every TokenKind value between TokInvalid and
// tokKindCount has a stable String() name in the kindNames table. A new kind
// that lacks an entry falls through to the numeric fallback — which breaks
// golden stability (Rule 6.2) and diagnostic rendering.
func TestTokenKindCoverage(t *testing.T) {
	seen := make(map[string]TokenKind)
	for k := TokInvalid; k < tokKindCount; k++ {
		name, ok := kindNames[k]
		if !ok {
			t.Errorf("TokenKind(%d) has no entry in kindNames", int(k))
			continue
		}
		if name == "" {
			t.Errorf("TokenKind(%d) has empty name", int(k))
		}
		if prior, dup := seen[name]; dup {
			t.Errorf("TokenKind(%d) name %q duplicates TokenKind(%d)", int(k), name, int(prior))
		}
		seen[name] = k
	}

	// Every reserved word in reference §1.9 must map to an in-range token kind.
	for _, kw := range keywordList {
		k, ok := keywords[kw]
		if !ok {
			t.Errorf("keyword %q missing from keywords map", kw)
			continue
		}
		if k <= TokInvalid || k >= tokKindCount {
			t.Errorf("keyword %q maps to out-of-range TokenKind(%d)", kw, int(k))
		}
	}

	// keywords and keywordList must be the same set — tests and diagnostics
	// rely on the list being authoritative.
	if len(keywords) != len(keywordList) {
		t.Errorf("keywords map has %d entries, keywordList has %d", len(keywords), len(keywordList))
	}
	listed := make(map[string]bool, len(keywordList))
	for _, kw := range keywordList {
		listed[kw] = true
	}
	for kw := range keywords {
		if !listed[kw] {
			t.Errorf("keyword %q in map but not in keywordList", kw)
		}
	}
}
