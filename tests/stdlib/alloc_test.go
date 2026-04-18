package stdlib_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/doc"
	"github.com/Tembocs/fuse5/compiler/parse"
)

// TestAllocatorTrait asserts stdlib/core/alloc/allocator.fuse
// declares a pub `Allocator` trait with the alloc / realloc /
// dealloc method surface reference §52.1 mandates.
//
// Bound by:
//
//	go test ./tests/stdlib/... -run TestAllocatorTrait -v
func TestAllocatorTrait(t *testing.T) {
	path := filepath.Join(stdlibRoot(t), "alloc", "allocator.fuse")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	// File parses.
	if _, diags := parse.Parse(path, src); len(diags) != 0 {
		t.Fatalf("allocator.fuse parse failed: %v", diags)
	}
	// Trait Allocator is declared + is pub.
	items := doc.Extract(src)
	var trait doc.Item
	for _, it := range items {
		if it.Name == "Allocator" && it.Kind == "trait" {
			trait = it
		}
	}
	if trait.Name == "" {
		t.Fatalf("stdlib/core/alloc/allocator.fuse missing `pub trait Allocator`: %+v", items)
	}
	if !trait.IsPub {
		t.Errorf("Allocator trait must be pub")
	}
	// Required methods appear as `fn alloc(...)` / `fn realloc(...)` /
	// `fn dealloc(...)` inside the trait body.
	text := string(src)
	for _, method := range []string{"fn alloc(", "fn realloc(", "fn dealloc("} {
		if !strings.Contains(text, method) {
			t.Errorf("Allocator trait missing method signature: %q", method)
		}
	}
}

// TestGlobalAllocator confirms SystemAllocator exists and the
// global-allocator hook file declares the expected override
// points. Reference §52.1 — a global allocator is declared
// once per binary and defaults to wrapping fuse_rt_alloc_*.
func TestGlobalAllocator(t *testing.T) {
	// SystemAllocator struct in stdlib/core/alloc/system.fuse.
	sysPath := filepath.Join(stdlibRoot(t), "alloc", "system.fuse")
	sysSrc, err := os.ReadFile(sysPath)
	if err != nil {
		t.Fatalf("read %s: %v", sysPath, err)
	}
	sysText := string(sysSrc)
	if !strings.Contains(sysText, "pub struct SystemAllocator") {
		t.Errorf("system.fuse missing pub struct SystemAllocator")
	}
	for _, method := range []string{"fn alloc(", "fn realloc(", "fn dealloc("} {
		if !strings.Contains(sysText, method) {
			t.Errorf("SystemAllocator missing method: %q", method)
		}
	}

	// Global hooks in stdlib/core/alloc/global.fuse.
	globalPath := filepath.Join(stdlibRoot(t), "alloc", "global.fuse")
	globalSrc, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("read %s: %v", globalPath, err)
	}
	globalText := string(globalSrc)
	for _, fn := range []string{
		"fn global_allocator(",
		"fn global_alloc(",
		"fn global_dealloc(",
	} {
		if !strings.Contains(globalText, fn) {
			t.Errorf("global.fuse missing %q", fn)
		}
	}

	// Every file in stdlib/core/alloc/ parses and has Rule 5.6
	// doc coverage.
	allocDir := filepath.Join(stdlibRoot(t), "alloc")
	entries, err := os.ReadDir(allocDir)
	if err != nil {
		t.Fatalf("read %s: %v", allocDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".fuse") {
			continue
		}
		p := filepath.Join(allocDir, e.Name())
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if _, diags := parse.Parse(p, body); len(diags) != 0 {
			t.Errorf("%s parse failed: %v", p, diags)
		}
		items := doc.Extract(body)
		missing := doc.CheckMissingDocs(items)
		if len(missing) > 0 {
			t.Errorf("%s: missing docs on pub items: %v", p, missing)
		}
	}
}

// TestCollectionsInAllocator confirms the W21 allocator-
// parameterised collection shapes exist — `Vec[T, A]`,
// `Box[T, A]`, `HashMap[K, V, A]` — and that every allocation-
// routing concern is visible in the struct definition (the
// `alloc: A` field is stored alongside ptr/len/cap, so dealloc
// always routes to the owning allocator).
func TestCollectionsInAllocator(t *testing.T) {
	cases := []struct {
		file   string
		decl   string
		hasAlloc bool
	}{
		{"vec.fuse", "pub struct Vec[T, A]", true},
		{"boxed.fuse", "pub struct Box[T, A]", true},
		{"hashmap.fuse", "pub struct HashMap[K, V, A]", true},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			path := filepath.Join(stdlibRoot(t), "alloc", c.file)
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(body)
			if !strings.Contains(text, c.decl) {
				t.Errorf("%s missing declaration %q", c.file, c.decl)
			}
			if c.hasAlloc && !strings.Contains(text, "alloc: A") {
				t.Errorf("%s missing `alloc: A` field — collection must store its allocator to avoid silent global fallback", c.file)
			}
		})
	}

	// BumpAllocator exists as a reference implementation.
	bumpPath := filepath.Join(stdlibRoot(t), "alloc", "bump.fuse")
	bumpSrc, err := os.ReadFile(bumpPath)
	if err != nil {
		t.Fatalf("read %s: %v", bumpPath, err)
	}
	bumpText := string(bumpSrc)
	if !strings.Contains(bumpText, "pub struct BumpAllocator") {
		t.Errorf("bump.fuse missing pub struct BumpAllocator")
	}
	for _, method := range []string{"fn alloc(", "fn dealloc(", "fn realloc(", "fn reset(", "fn cursor("} {
		if !strings.Contains(bumpText, method) {
			t.Errorf("BumpAllocator missing method: %q", method)
		}
	}
}
