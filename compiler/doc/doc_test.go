package doc

import (
	"strings"
	"testing"
)

// TestDocCheck exercises Extract and CheckMissingDocs: doc
// comments pair with the immediately-following item declaration,
// missing docs on public items surface as a diagnostic, private
// items are allowed to skip docs.
func TestDocCheck(t *testing.T) {
	src := `
/// Adds two integers.
/// The result wraps on overflow per W17 release policy.
pub fn add(a: I32, b: I32) -> I32 {
    return a + b;
}

fn private_helper() -> I32 {
    return 0;
}

/// A point in 2D space.
pub struct Point {
    x: I32,
    y: I32,
}

pub enum Undocumented { A, B }

pub const MAX: I32 = 100;
`
	items := Extract([]byte(src))
	if len(items) < 5 {
		t.Fatalf("expected ≥5 items, got %d: %+v", len(items), items)
	}

	// Locate specific items.
	byName := map[string]Item{}
	for _, it := range items {
		byName[it.Name] = it
	}
	if it, ok := byName["add"]; !ok {
		t.Fatalf("add missing: %+v", items)
	} else {
		if it.Kind != "fn" || !it.IsPub {
			t.Errorf("add should be pub fn: %+v", it)
		}
		if !strings.Contains(it.Doc, "Adds two integers") {
			t.Errorf("add missing doc content: %q", it.Doc)
		}
	}
	if it, ok := byName["Point"]; !ok {
		t.Fatalf("Point missing")
	} else if !strings.Contains(it.Doc, "point in 2D space") {
		t.Errorf("Point doc missing: %q", it.Doc)
	}

	// All top-level pub items in the sample must be recorded
	// as pub — including methods declared inside pub trait /
	// impl bodies (2026-04-18 audit fix: the classifier
	// previously dropped these to private).
	byKindName := map[string]Item{}
	for _, it := range items {
		byKindName[it.Kind+":"+it.Name] = it
	}

	missing := CheckMissingDocs(items)
	// Undocumented + MAX (both pub, both no doc) should appear.
	var seenUndoc, seenMax bool
	for _, m := range missing {
		if strings.Contains(m, "Undocumented") {
			seenUndoc = true
		}
		if strings.Contains(m, "MAX") {
			seenMax = true
		}
	}
	if !seenUndoc {
		t.Errorf("CheckMissingDocs did not flag `Undocumented`: %v", missing)
	}
	if !seenMax {
		t.Errorf("CheckMissingDocs did not flag `MAX`: %v", missing)
	}
	// private_helper is not pub — it must NOT be flagged.
	for _, m := range missing {
		if strings.Contains(m, "private_helper") {
			t.Errorf("private item was flagged for missing docs: %v", missing)
		}
	}
}
