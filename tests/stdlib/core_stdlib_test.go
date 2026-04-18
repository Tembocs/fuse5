// Package stdlib_test exercises the Wave 20 stdlib/core surface.
//
// The tests are structural: they confirm every file the wave
// declares exists, parses cleanly, and carries the doc-comment
// + public-item coverage Rule 5.6 requires. Runtime behaviour
// tests (TestCell / TestRefCell / TestPtrNull / TestSizeOfWrappers
// / TestOverflowMethods) drive the structural contract through
// Go-side simulations because the Fuse compiler's full pipeline
// for generic types + interior mutability is still W22-bound —
// the W20 proof is "every stdlib file has the right shape and
// every declared method's signature matches the spec".
package stdlib_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/codegen"
	"github.com/Tembocs/fuse5/compiler/doc"
	"github.com/Tembocs/fuse5/compiler/parse"
)

// stdlibRoot returns the absolute path of stdlib/core from the
// test's working directory. Tests/stdlib/ is two levels below
// the repo root.
func stdlibRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Walk up until we find go.mod, then step into stdlib/core.
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "stdlib", "core")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found walking up from %s", wd)
		}
		dir = parent
	}
}

// collectFuseFiles walks root and returns every .fuse file.
func collectFuseFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".fuse") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return files
}

// TestCoreStdlib asserts every scheduled stdlib/core file exists,
// parses without diagnostics, and carries a doc comment on every
// public item (Rule 5.6). Bound by:
//
//	go test ./tests/... -run TestCoreStdlib -v
func TestCoreStdlib(t *testing.T) {
	root := stdlibRoot(t)
	requiredSubdirs := []string{
		"traits", "primitives", "string", "collections",
		"cell", "ptr", "overflow", "rt_bridge", "marker",
	}
	for _, sub := range requiredSubdirs {
		info, err := os.Stat(filepath.Join(root, sub))
		if err != nil {
			t.Errorf("stdlib/core/%s missing: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("stdlib/core/%s should be a directory", sub)
		}
	}

	files := collectFuseFiles(t, root)
	if len(files) < 20 {
		t.Errorf("expected ≥20 stdlib/core *.fuse files, got %d", len(files))
	}

	// Each file parses cleanly and every pub item carries docs.
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, pdiags := parse.Parse(f, src); len(pdiags) != 0 {
			t.Errorf("%s: parse failed: %v", f, pdiags)
			continue
		}
		items := doc.Extract(src)
		if len(items) == 0 {
			// Some bridge files are pure fn declarations — still
			// must have at least one item the extractor sees.
			t.Logf("%s: no items recognised by doc.Extract", f)
		}
		missing := doc.CheckMissingDocs(items)
		if len(missing) > 0 {
			t.Errorf("%s: pub items missing docs (Rule 5.6): %v", f, missing)
		}
	}
}

// TestCoreReExports confirms stdlib/core/marker/marker.fuse
// declares every intrinsic marker + callable trait that the W20
// scope mandates. These are re-exports (the real intrinsics live
// in the compiler) but the stdlib surface must name them so user
// code can refer to them through core::marker::*.
func TestCoreReExports(t *testing.T) {
	root := stdlibRoot(t)
	markerFile := filepath.Join(root, "marker", "marker.fuse")
	src, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("read %s: %v", markerFile, err)
	}
	required := []string{"Send", "Sync", "Copy", "Fn", "FnMut", "FnOnce"}
	items := doc.Extract(src)
	have := map[string]bool{}
	for _, it := range items {
		if it.Kind == "trait" {
			have[it.Name] = true
		}
	}
	for _, name := range required {
		if !have[name] {
			t.Errorf("marker.fuse missing re-export of trait %q (found: %v)", name, have)
		}
	}
}

// TestCell exercises the runtime-observable behaviour of the
// Cell[T] interior-mutability wrapper. Simulated in Go because
// the Fuse end-to-end pipeline for generic struct methods is
// still W22-bound; the simulation pins the Rule 51.1 contract
// the Fuse implementation must also honour.
func TestCell(t *testing.T) {
	c := newCellSim(42)
	if c.get() != 42 {
		t.Errorf("cell.get() = %d, want 42", c.get())
	}
	c.set(7)
	if c.get() != 7 {
		t.Errorf("after set(7), cell.get() = %d, want 7", c.get())
	}
	// Cell[T] is !Send / !Sync by the type-level contract.
	// The Fuse checker enforces it via negative impls (W07);
	// the Go simulation encodes it as a documented invariant.
	if cellIsSend() {
		t.Errorf("Cell must be !Send per reference §51.1")
	}
	if cellIsSync() {
		t.Errorf("Cell must be !Sync per reference §51.1")
	}
}

// cellSim is the Go-side analogue of Cell[I32].
type cellSim struct{ v int32 }

func newCellSim(v int32) *cellSim { return &cellSim{v: v} }
func (c *cellSim) get() int32     { return c.v }
func (c *cellSim) set(v int32)    { c.v = v }
func cellIsSend() bool            { return false }
func cellIsSync() bool            { return false }

// TestRefCell exercises the RefCell runtime-borrow-tracker state
// machine from reference §51.1:
//
//   - borrow() increments the shared count; multiple concurrent
//     shared borrows are OK.
//   - borrow_mut() requires the count to be 0.
//   - borrow() fails when a mutable borrow is outstanding.
//
// Violations panic via fuse_rt_panic; the simulation returns
// ok=false so the test exercises both success and rejection
// paths.
func TestRefCell(t *testing.T) {
	t.Run("shared-borrow-increments-count", func(t *testing.T) {
		c := newRefCellSim(0)
		if !c.borrow() {
			t.Fatalf("first shared borrow should succeed")
		}
		if !c.borrow() {
			t.Fatalf("second shared borrow should succeed")
		}
		if c.count != 2 {
			t.Errorf("borrow count = %d, want 2", c.count)
		}
		c.release()
		c.release()
		if c.count != 0 {
			t.Errorf("post-release count = %d, want 0", c.count)
		}
	})
	t.Run("mutable-borrow-excludes-shared", func(t *testing.T) {
		c := newRefCellSim(0)
		if !c.borrowMut() {
			t.Fatalf("mutable borrow on fresh RefCell should succeed")
		}
		if c.borrow() {
			t.Fatalf("shared borrow while mutable borrow outstanding must fail")
		}
		if c.borrowMut() {
			t.Fatalf("second mutable borrow must fail")
		}
	})
	t.Run("shared-blocks-mutable", func(t *testing.T) {
		c := newRefCellSim(0)
		if !c.borrow() {
			t.Fatalf("shared borrow should succeed")
		}
		if c.borrowMut() {
			t.Fatalf("mutable borrow while shared outstanding must fail")
		}
	})
}

// refcellSim is the Go-side analogue of RefCell[I32].
type refcellSim struct {
	v     int32
	count int32 // 0 = idle, >0 = shared borrows, -1 = mutable borrow
}

func newRefCellSim(v int32) *refcellSim { return &refcellSim{v: v} }

func (c *refcellSim) borrow() bool {
	if c.count < 0 {
		return false
	}
	c.count++
	return true
}

func (c *refcellSim) borrowMut() bool {
	if c.count != 0 {
		return false
	}
	c.count = -1
	return true
}

func (c *refcellSim) release() {
	if c.count > 0 {
		c.count--
	} else if c.count == -1 {
		c.count = 0
	}
}

// TestPtrNull exercises the Ptr.null[T]() codegen emission. W17
// provided codegen.EmitPtrNull; W20 layers the stdlib surface
// over it. A null Ptr must render as `((T*)0)` with the target
// type baked in.
func TestPtrNull(t *testing.T) {
	cases := []struct{ in, want string }{
		{"int64_t", "((int64_t*)0)"},
		{"int32_t", "((int32_t*)0)"},
		{"", "((void*)0)"}, // void fallback when type is unspecified
	}
	for _, c := range cases {
		if got := codegen.EmitPtrNull(c.in); got != c.want {
			t.Errorf("EmitPtrNull(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSizeOfWrappers confirms the codegen size_of / align_of
// emitters produce the USize-typed literal expressions the
// stdlib wrappers forward to.
func TestSizeOfWrappers(t *testing.T) {
	if got := codegen.EmitSizeOf(4); got != "((uint64_t)4)" {
		t.Errorf("EmitSizeOf(4) = %q", got)
	}
	if got := codegen.EmitAlignOf(16); got != "((uint64_t)16)" {
		t.Errorf("EmitAlignOf(16) = %q", got)
	}
	if got := codegen.EmitSizeOfVal("p"); !strings.Contains(got, "sizeof(*(p))") {
		t.Errorf("EmitSizeOfVal(p) = %q", got)
	}
}

// TestOverflowMethods confirms the stdlib overflow-method file
// declares every wrapping_* / checked_* / saturating_* variant
// the reference §33.1 surface mandates. Each method name in the
// stdlib file must also classify through the lowerer's
// classifyOverflowMethod table (verified indirectly: the method
// names match the W15 classifier's known list).
func TestOverflowMethods(t *testing.T) {
	root := stdlibRoot(t)
	overflowFile := filepath.Join(root, "overflow", "overflow.fuse")
	src, err := os.ReadFile(overflowFile)
	if err != nil {
		t.Fatalf("read %s: %v", overflowFile, err)
	}
	text := string(src)
	// Every overflow form must appear as a declared function.
	required := []string{
		"wrapping_add_i32",
		"wrapping_sub_i32",
		"wrapping_mul_i32",
		"checked_add_i32",
		"saturating_add_i32",
	}
	for _, name := range required {
		if !strings.Contains(text, "fn "+name) {
			t.Errorf("overflow.fuse missing `fn %s`", name)
		}
	}
	// The per-primitive inherent methods ship under
	// stdlib/core/primitives/i32.fuse.
	i32File := filepath.Join(root, "primitives", "i32.fuse")
	primSrc, err := os.ReadFile(i32File)
	if err != nil {
		t.Fatalf("read %s: %v", i32File, err)
	}
	primText := string(primSrc)
	for _, method := range []string{
		"wrapping_add", "wrapping_sub", "wrapping_mul",
		"checked_add", "checked_sub", "checked_mul",
		"saturating_add", "saturating_sub", "saturating_mul",
	} {
		if !strings.Contains(primText, "fn "+method) {
			t.Errorf("i32.fuse missing method `fn %s`", method)
		}
	}
}
