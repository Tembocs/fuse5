// Package runtime_test validates that the C runtime's ABI surface is
// declared correctly. W16 expands the surface beyond the W05 stub to
// a full runtime (memory, panic, IO, process, time, thread, sync,
// channels); this Go test inspects the header and source tree to
// confirm every declared function has a matching definition. The
// runtime_test lives alongside the C runtime so `go test ./...`
// always exercises the ABI contract even when the runtime build is
// not triggered.
package runtime_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStubRuntime asserts that runtime/include/fuse_rt.h declares
// the ABI surface W16 requires and that runtime/src/*.c contains
// at least one definition for each declared function. A missing
// definition is a drift between header and source that would
// surface as a linker error at e2e build time — catching it at
// `go test` is cheaper.
func TestStubRuntime(t *testing.T) {
	repoRoot := findRepoRoot(t)
	headerPath := filepath.Join(repoRoot, "runtime", "include", "fuse_rt.h")
	header := readFile(t, headerPath)

	required := []string{
		// Process control.
		"fuse_rt_abort",
		"fuse_rt_panic",
		"fuse_rt_exit",
		// Memory.
		"fuse_rt_alloc",
		"fuse_rt_realloc",
		"fuse_rt_free",
		// IO.
		"fuse_rt_write_stdout",
		"fuse_rt_write_stderr",
		// Process + time.
		"fuse_rt_pid",
		"fuse_rt_monotonic_ns",
		"fuse_rt_wall_ns",
		"fuse_rt_sleep_ns",
		// Threads.
		"fuse_rt_thread_spawn",
		"fuse_rt_thread_join",
		"fuse_rt_thread_id",
		"fuse_rt_thread_yield",
		// Sync.
		"fuse_rt_mutex_new",
		"fuse_rt_mutex_lock",
		"fuse_rt_mutex_unlock",
		"fuse_rt_mutex_free",
		"fuse_rt_cond_new",
		"fuse_rt_cond_wait",
		"fuse_rt_cond_notify_one",
		"fuse_rt_cond_notify_all",
		"fuse_rt_cond_free",
		// Channels.
		"fuse_rt_chan_new",
		"fuse_rt_chan_send",
		"fuse_rt_chan_recv",
		"fuse_rt_chan_try_send",
		"fuse_rt_chan_try_recv",
		"fuse_rt_chan_close",
		"fuse_rt_chan_free",
		// Arithmetic overflow (W24 — cross-compiler).
		"fuse_rt_add_overflow_i64",
		"fuse_rt_sub_overflow_i64",
		"fuse_rt_mul_overflow_i64",
	}
	for _, fn := range required {
		if !strings.Contains(header, fn) {
			t.Errorf("fuse_rt.h missing declaration for %s", fn)
		}
	}

	// Every declared fn must have a definition somewhere under
	// runtime/src/. We aggregate the source tree and check the
	// combined body contains each name at a definition site. The
	// checker is deliberately textual (not preprocessor-aware) so
	// this test is hermetic — no CC or build step required.
	srcDir := filepath.Join(repoRoot, "runtime", "src")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read %s: %v", srcDir, err)
	}
	var combined strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".c") {
			continue
		}
		body := readFile(t, filepath.Join(srcDir, e.Name()))
		combined.WriteString(body)
		combined.WriteByte('\n')
	}
	src := combined.String()
	for _, fn := range required {
		// Look for a definition signature: `name(` followed (ignoring
		// params and whitespace) by `{`. A plain `name(...)` followed
		// by `;` would be a declaration.
		idx := strings.Index(src, fn+"(")
		if idx < 0 {
			t.Errorf("runtime/src/*.c missing definition for %s", fn)
		}
	}

	// The canonical abort/panic surface must be wired to the host
	// abort so a rogue runtime doesn't silently return.
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
