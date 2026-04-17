package resolve

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// TestModuleDiscovery exercises the filesystem discovery path. Running
// with `-count=3` (as the wave spec requires) proves the same root
// produces the same module list every time (Rule 7.1 determinism).
func TestModuleDiscovery(t *testing.T) {
	root := t.TempDir()
	writeFuse(t, root, "lib.fuse", "fn root_entry() {}")
	writeFuse(t, root, "io/mod.fuse", "pub fn stdout() {}")
	writeFuse(t, root, "io/fmt.fuse", "pub fn format() {}")
	writeFuse(t, root, "math/vec.fuse", "pub struct V2 { x: I32, y: I32 }")

	srcs, diags, err := DiscoverFromDir(root)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	assertNoDiags(t, diags)

	got := make([]string, len(srcs))
	for i, s := range srcs {
		got[i] = s.ModulePath
	}

	// Walk order is deterministic: directories are visited before sibling
	// files (both sorted lexicographically). With io/, lib.fuse, math/
	// at the root that produces: io/fmt.fuse, io/mod.fuse, lib.fuse,
	// math/vec.fuse. Mapping each filename to its dotted module path
	// gives: io.fmt, io (from mod.fuse), "" (from lib.fuse), math.vec.
	want := []string{"io.fmt", "io", "", "math.vec"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("module paths:\n got  %v\n want %v", got, want)
	}
}

// TestModuleDiscovery_EmptyRoot covers the no-sources case.
func TestModuleDiscovery_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	srcs, diags, err := DiscoverFromDir(root)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	assertNoDiags(t, diags)
	if len(srcs) != 0 {
		t.Fatalf("expected no sources, got %d", len(srcs))
	}
}

// TestModuleGraph asserts that Resolve builds a ModuleGraph whose
// Modules map, Order slice, and Edges map accurately reflect the
// input.
func TestModuleGraph(t *testing.T) {
	srcs := []*SourceFile{
		mkSource(t, "", "lib.fuse", "import io;\nimport math.vec;"),
		mkSource(t, "io", "io.fuse", "pub fn stdout() {}"),
		mkSource(t, "math", "math.fuse", "pub struct M {}"),
		mkSource(t, "math.vec", "vec.fuse", "pub struct V2 { x: I32, y: I32 }"),
	}
	out, msgs := resolveStrings(t, srcs, BuildConfig{})
	if len(msgs) != 0 {
		t.Fatalf("unexpected diagnostics: %v", msgs)
	}
	got := out.Graph.Order
	want := []string{"", "io", "math", "math.vec"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("module order:\n got  %v\n want %v", got, want)
	}
	if _, ok := out.Graph.Modules["math.vec"]; !ok {
		t.Fatalf("math.vec missing from Modules map")
	}
	// Root module imports `io` and `math.vec` (direct module targets).
	edges := out.Graph.Edges[""]
	wantEdges := []string{"io", "math.vec"}
	if !reflect.DeepEqual(edges, wantEdges) {
		t.Fatalf("root edges:\n got  %v\n want %v", edges, wantEdges)
	}
}

// TestModuleGraph_DuplicatePath ensures two files claiming the same
// module path emit exactly one duplicate-module diagnostic.
func TestModuleGraph_DuplicatePath(t *testing.T) {
	srcs := []*SourceFile{
		mkSource(t, "dup", "a.fuse", "fn a() {}"),
		mkSource(t, "dup", "b.fuse", "fn b() {}"),
	}
	_, msgs := resolveStrings(t, srcs, BuildConfig{})
	found := false
	for _, m := range msgs {
		if contains(m, "duplicate module") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate-module diagnostic, got %v", msgs)
	}
}

// --- test helpers for this file ---

func writeFuse(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func contains(haystack, needle string) bool {
	return indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
outer:
	for i := 0; i+len(needle) <= len(haystack); i++ {
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}

// Silence unused-import warning on non-windows builders.
var _ = runtime.GOOS
