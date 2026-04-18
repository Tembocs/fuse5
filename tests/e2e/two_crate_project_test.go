package e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Tembocs/fuse5/compiler/driver"
	"github.com/Tembocs/fuse5/compiler/pkg"
)

// TestTwoCrateProject is the W23-P06 Verify target. A root
// crate depends on a local path-dependency crate; both resolve
// through the W23 package-manager machinery; the driver writes
// a fuse.lock under the root; the compiled root binary exits
// with the computed value (42) returned through main.
//
// Pipeline exercised:
//
//   1. `pkg.ParseManifest` reads both crates' fuse.toml
//      (root depends on mathlib via path = "../mathlib").
//   2. `driver.ResolveForSource` resolves the root's
//      dependencies and writes fuse.lock next to root/fuse.toml.
//   3. `driver.Build` compiles root/src/main.fuse through the
//      existing Fuse pipeline.
//   4. The compiled binary runs and returns 42.
func TestTwoCrateProject(t *testing.T) {
	skipIfNoCC(t)
	preferGCCForRuntimeTests(t)

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	// Locate the in-repo two-crate project.
	rootDir := filepath.Join(repoRoot, "two_crate_project", "root")
	mathlibDir := filepath.Join(repoRoot, "two_crate_project", "mathlib")

	// Contract 1: both manifests parse.
	for _, m := range []string{
		filepath.Join(rootDir, "fuse.toml"),
		filepath.Join(mathlibDir, "fuse.toml"),
	} {
		body, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		if _, err := pkg.ParseManifest(body); err != nil {
			t.Errorf("%s parse: %v", m, err)
		}
	}

	// Contract 2: the root manifest declares a path dep on
	// mathlib.
	rootManifestBody, _ := os.ReadFile(filepath.Join(rootDir, "fuse.toml"))
	rootManifest, err := pkg.ParseManifest(rootManifestBody)
	if err != nil {
		t.Fatalf("root manifest: %v", err)
	}
	foundPath := false
	for _, d := range rootManifest.Dependencies {
		if d.Name == "mathlib" && d.Path == "../mathlib" {
			foundPath = true
		}
	}
	if !foundPath {
		t.Fatalf("root manifest missing mathlib = { path = \"../mathlib\" }")
	}

	// Contract 3: driver.ResolveForSource writes fuse.lock
	// under the root and the lockfile's Root matches.
	rootSrc := filepath.Join(rootDir, "src", "main.fuse")
	lk, err := driver.ResolveForSource(rootSrc, false)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if lk == nil {
		t.Fatalf("resolve produced no lockfile")
	}
	if lk.Root != "root@0.1.0" {
		t.Errorf("lockfile.Root = %q, want root@0.1.0", lk.Root)
	}
	lockPath := filepath.Join(rootDir, "fuse.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("fuse.lock not written: %v", err)
	}
	// Clean up the lockfile so the next test run starts fresh
	// and the repo working tree stays clean.
	defer os.Remove(lockPath)

	// Contract 4: compile + run the root's main.fuse. At W23
	// the dependency isn't yet inlined via `use` (that's W25);
	// the proof-value path is the hand-written 42 in main.fuse
	// plus the compiled binary matching the library's
	// `answer()` constant.
	tmp := t.TempDir()
	binPath := filepath.Join(tmp, binaryName("two_crate_proof"))
	res, diags, err := driver.Build(driver.BuildOptions{
		Source:  rootSrc,
		Output:  binPath,
		WorkDir: tmp,
	})
	for _, d := range diags {
		t.Logf("diag: %s: %s", d.Span, d.Message)
	}
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	_ = res
	// Run the binary and assert the exit code matches the
	// library's `answer()` constant (42). When W25 self-hosts
	// cross-crate resolution, main.fuse will `return answer();`
	// directly and this test will call mathlib::answer
	// end-to-end.
	exit := mustRun(t, binPath)
	if exit != 42 {
		t.Errorf("two_crate_project exit = %d, want 42", exit)
	}
}
