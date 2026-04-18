package doc

import (
	"strings"
	"testing"
)

// TestPubTraitMethodInheritsVisibility pins the 2026-04-18 audit
// fix: methods declared inside a `pub trait` body inherit pub
// visibility and therefore flow through the Rule 5.6 missing-
// docs check.
//
// Before the fix, `doc.CheckMissingDocs` flagged methods whose
// trait declaration was pub as priv (because the `pub` modifier
// sat on the preceding trait line, not the method). That made
// `fuse doc --check stdlib/core` silently miss every undocumented
// trait method.
//
// Impl-block methods carry their own `pub` modifier in Fuse
// syntax (`impl T { pub fn m(...) }`); the classifier handles
// those via the raw regex match without needing scope
// inheritance.
func TestPubTraitMethodInheritsVisibility(t *testing.T) {
	src := `
/// A documented public trait.
pub trait Eq {
    /// Documented method.
    fn eq(self, other: Self) -> Bool;

    // Not documented: should now show up in CheckMissingDocs.
    fn ne(self, other: Self) -> Bool;
}

// A private trait: its methods stay private.
trait Hidden {
    fn hide(self) -> I32;
}

/// A documented inherent impl with its own pub method.
impl Counter {
    /// Documented method.
    pub fn get(self) -> I32 {
        return 0;
    }

    // Private method (no pub); must NOT be flagged.
    fn tick(self) -> I32 {
        return 0;
    }
}
`
	items := Extract([]byte(src))

	byName := map[string]Item{}
	for _, it := range items {
		if it.Kind == "fn" {
			byName[it.Name] = it
		}
	}

	// eq is inside a pub trait → should be pub.
	if it, ok := byName["eq"]; !ok {
		t.Fatalf("eq not extracted")
	} else if !it.IsPub {
		t.Errorf("eq: IsPub = false, want true (pub-trait inheritance)")
	}
	// ne is inside a pub trait, no explicit modifier, no doc.
	if it, ok := byName["ne"]; !ok {
		t.Fatalf("ne not extracted")
	} else if !it.IsPub {
		t.Errorf("ne: IsPub = false, want true (pub-trait inheritance)")
	}
	// hide is inside a private trait: must stay private.
	if it, ok := byName["hide"]; !ok {
		t.Fatalf("hide not extracted")
	} else if it.IsPub {
		t.Errorf("hide: IsPub = true, want false (inside private trait)")
	}
	// get carries its own pub; must be pub.
	if it, ok := byName["get"]; !ok {
		t.Fatalf("get not extracted")
	} else if !it.IsPub {
		t.Errorf("get: IsPub = false, want true (explicit pub)")
	}
	// tick inside a non-pub impl with no explicit pub: stays private.
	if it, ok := byName["tick"]; !ok {
		t.Fatalf("tick not extracted")
	} else if it.IsPub {
		t.Errorf("tick: IsPub = true, want false (private impl method)")
	}

	// CheckMissingDocs now flags ne (undocumented pub-trait method).
	missing := CheckMissingDocs(items)
	sawNE := false
	for _, m := range missing {
		if strings.Contains(m, "ne") {
			sawNE = true
		}
	}
	if !sawNE {
		t.Errorf("CheckMissingDocs did not flag `ne` (undocumented pub-trait method): %v", missing)
	}
	// hide and tick are private; must NOT be flagged.
	for _, m := range missing {
		if strings.Contains(m, "hide") {
			t.Errorf("private trait method `hide` incorrectly flagged: %v", missing)
		}
		if strings.Contains(m, "tick") {
			t.Errorf("private impl method `tick` incorrectly flagged: %v", missing)
		}
	}
}
