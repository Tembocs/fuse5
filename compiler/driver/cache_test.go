package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPassCache exercises the on-disk cache: open / put / get /
// flush / reload / clear, plus the version-mismatch
// invalidation path.
//
// Bound by:
//
//	go test ./compiler/driver/... -run TestPassCache -v
func TestPassCache(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if c.Size() != 0 {
		t.Fatalf("fresh cache should be empty, got %d", c.Size())
	}

	// Store + retrieve one entry.
	key := Key("parse", []byte("source bytes"))
	if err := c.Put(key, "parse", "v1", []byte("parsed payload")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if payload, ok := c.Get(key); !ok || string(payload) != "parsed payload" {
		t.Fatalf("Get hit mismatch: ok=%v payload=%q", ok, payload)
	}
	if err := c.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// Reopen: the entry must persist.
	c2, err := Open(dir)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if payload, ok := c2.Get(key); !ok || string(payload) != "parsed payload" {
		t.Fatalf("post-flush reopen Get mismatch: ok=%v payload=%q", ok, payload)
	}

	// Miss on a key that was never put.
	missKey := Key("parse", []byte("other bytes"))
	if _, ok := c2.Get(missKey); ok {
		t.Fatalf("Get should miss for unknown key")
	}

	// Version-mismatch invalidation: hand-edit the manifest with
	// a bogus version and confirm Open re-seeds.
	manifestPath := filepath.Join(dir, ".fuse-cache", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":"older","entries":{}}`), 0o644); err != nil {
		t.Fatalf("overwrite manifest: %v", err)
	}
	c3, err := Open(dir)
	if err != nil {
		t.Fatalf("Reopen after bad version: %v", err)
	}
	if c3.Size() != 0 {
		t.Fatalf("version-mismatched manifest should invalidate entries, got %d", c3.Size())
	}

	// Clear empties the cache.
	if err := c3.Put(key, "parse", "v1", []byte("payload")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := c3.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if c3.Size() != 0 {
		t.Fatalf("post-Clear size = %d, want 0", c3.Size())
	}
}

// TestPassCache_Determinism confirms Key / Combine are
// deterministic. Rule 7.1 — same input, same bytes.
func TestPassCache_Determinism(t *testing.T) {
	a := Key("parse", []byte("hello"))
	b := Key("parse", []byte("hello"))
	if a != b {
		t.Errorf("Key not deterministic: %q vs %q", a, b)
	}
	// Different pass names → different keys even with same input.
	if Key("parse", []byte("hello")) == Key("check", []byte("hello")) {
		t.Errorf("Key should differ by pass name")
	}
	// Combine is order-independent.
	first := Combine("x", "y", "z")
	second := Combine("z", "y", "x")
	if first != second {
		t.Errorf("Combine should be order-independent: %q vs %q", first, second)
	}
}

// TestPassCache_Corruption confirms a header whose fingerprint
// disagrees with its key is treated as a miss (self-heal).
func TestPassCache_Corruption(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	key := Key("parse", []byte("x"))
	if err := c.Put(key, "parse", "v1", []byte("payload")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	// Corrupt the in-memory header's fingerprint.
	hdr := c.manifest.Entries[key]
	hdr.InputFingerprint = "nonsense"
	c.manifest.Entries[key] = hdr
	if _, ok := c.Get(key); ok {
		t.Fatalf("Get should miss for header with fingerprint mismatch")
	}
	if _, still := c.manifest.Entries[key]; still {
		t.Errorf("corrupted entry should have been evicted from manifest")
	}
}

// TestIncrementalRebuild exercises PlanIncremental across a
// ten-function program where one function's fingerprint changed.
//
// Bound by:
//
//	go test ./compiler/driver/... -run TestIncrementalRebuild -v
func TestIncrementalRebuild(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Seed the cache with fingerprints for 10 functions.
	fns := map[string]string{}
	for i := 0; i < 10; i++ {
		name := "fn" + string(rune('A'+i))
		fp := "fingerprint-" + name + "-v1"
		fns[name] = fp
		key := Key("check/"+name, []byte(fp))
		if err := c.Put(key, "check/"+name, "v1", []byte("cached output")); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	// Edit one function: its fingerprint changes; all others stay.
	edited := fns
	edited["fnA"] = "fingerprint-fnA-v2"

	plan := c.PlanIncremental("check", edited)
	if len(plan.Hits) != 9 {
		t.Errorf("expected 9 cache hits after editing one fn, got %d (hits=%v, misses=%v)",
			len(plan.Hits), plan.Hits, plan.Misses)
	}
	if len(plan.Misses) != 1 {
		t.Errorf("expected 1 cache miss, got %d (misses=%v)", len(plan.Misses), plan.Misses)
	}
	if len(plan.Misses) == 1 && !strings.Contains(plan.Misses[0], "fnA") {
		t.Errorf("miss should be fnA, got %q", plan.Misses[0])
	}
}
