// Package runtime_test validates that the C runtime's ABI surface is
// declared correctly. At W05 the actual C implementation is
// exercised by the e2e tests; this Go test inspects the header file
// to confirm every function the wave spec declares is present.
package runtime_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStubRuntime asserts that runtime/include/fuse_rt.h declares the
// ABI surface W05 requires and that the accompanying abort.c source
// file exists.
func TestStubRuntime(t *testing.T) {
	repoRoot := findRepoRoot(t)
	headerPath := filepath.Join(repoRoot, "runtime", "include", "fuse_rt.h")
	header := readFile(t, headerPath)

	required := []string{
		"fuse_rt_abort",
		"fuse_rt_panic",
		"fuse_rt_write_stdout",
		"fuse_rt_write_stderr",
		"fuse_rt_thread_spawn",
		"fuse_rt_thread_join",
		"fuse_rt_chan_new",
		"fuse_rt_chan_send",
		"fuse_rt_chan_recv",
	}
	for _, fn := range required {
		if !strings.Contains(header, fn) {
			t.Errorf("fuse_rt.h missing declaration for %s", fn)
		}
	}

	// W05 also requires the abort.c implementation. The later waves
	// add more .c files; here we just check the one we shipped.
	abortPath := filepath.Join(repoRoot, "runtime", "src", "abort.c")
	abortSrc := readFile(t, abortPath)
	if !strings.Contains(abortSrc, "fuse_rt_abort") {
		t.Errorf("abort.c missing fuse_rt_abort implementation")
	}
	if !strings.Contains(abortSrc, "abort()") {
		t.Errorf("abort.c does not call the host abort() from fuse_rt_abort")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// findRepoRoot walks upward from the current test directory until it
// finds the go.mod that anchors the Fuse repo. The test file lives
// at runtime/tests/, so go.mod is two levels up.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found walking up from %s", wd)
		}
		dir = parent
	}
}
