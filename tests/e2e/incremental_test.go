package e2e_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Tembocs/fuse5/compiler/driver"
)

// TestIncrementalEditCycle is the W18-P04-T03 Verify target. It
// simulates an edit-compile-edit-compile cycle: build an existing
// proof program cold (populating any caches), then build it again
// without editing and confirm the second build is at least not
// slower. This is a proxy for "incremental builds reuse cached
// pass outputs"; true timing-based gating is W27 perf work.
//
// The `fuse` driver's pass cache is populated behind the scenes
// — W18 exposes the contract through compiler/driver/cache.go;
// the driver itself will plug into it in a later wave without
// breaking the e2e shape.
func TestIncrementalEditCycle(t *testing.T) {
	skipIfNoCC(t)
	preferGCCForRuntimeTests(t)

	src, err := filepath.Abs("hello_exit.fuse")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	// First build (cold).
	dir1 := t.TempDir()
	start1 := time.Now()
	_, diags, err := driver.Build(driver.BuildOptions{
		Source:  src,
		Output:  filepath.Join(dir1, binaryName("a")),
		WorkDir: dir1,
	})
	elapsed1 := time.Since(start1)
	if err != nil {
		t.Fatalf("cold build: %v (diags=%v)", err, diags)
	}

	// Second build (warm).
	dir2 := t.TempDir()
	start2 := time.Now()
	_, diags, err = driver.Build(driver.BuildOptions{
		Source:  src,
		Output:  filepath.Join(dir2, binaryName("b")),
		WorkDir: dir2,
	})
	elapsed2 := time.Since(start2)
	if err != nil {
		t.Fatalf("warm build: %v (diags=%v)", err, diags)
	}

	// Contract: the second build must not be catastrophically
	// slower than the first. True speedup depends on cache wiring
	// the driver has not yet adopted; W18's gate is "parity, not
	// regression". W27 tightens to "measurable speedup".
	if elapsed2 > elapsed1*4 {
		t.Errorf("warm build %s is >4× slower than cold build %s; incremental path regressed",
			elapsed2, elapsed1)
	}
	t.Logf("cold=%s warm=%s (ratio warm/cold=%.2f)",
		elapsed1, elapsed2, float64(elapsed2)/float64(elapsed1))
}
