package stdlib_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/doc"
	"github.com/Tembocs/fuse5/compiler/parse"
)

// hostedRoot returns the absolute path of stdlib/full.
func hostedRoot(t *testing.T) string {
	t.Helper()
	core := stdlibRoot(t)
	// stdlib/core/ → stdlib/full/
	return filepath.Join(filepath.Dir(core), "full")
}

// TestHostedStdlib is the W22 structural proof. Every scheduled
// subdirectory of `stdlib/full/` exists, every .fuse file parses
// cleanly, and every pub item has a /// doc comment (Rule 5.6).
//
// Also asserts the core/hosted boundary: no stdlib/core/ file
// references stdlib/full/. A reference would be a boundary
// violation — hosted modules depend on core, never the reverse.
//
// Bound by:
//
//	go test ./tests/stdlib/... -run TestHostedStdlib -v
func TestHostedStdlib(t *testing.T) {
	root := hostedRoot(t)
	requiredSubdirs := []string{
		"io", "fs", "os", "time", "thread", "sync", "chan",
	}
	for _, sub := range requiredSubdirs {
		info, err := os.Stat(filepath.Join(root, sub))
		if err != nil {
			t.Errorf("stdlib/full/%s missing: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("stdlib/full/%s should be a directory", sub)
		}
	}

	files := collectFuseFiles(t, root)
	if len(files) < 10 {
		t.Errorf("expected ≥10 stdlib/full *.fuse files, got %d", len(files))
	}

	// Parse + Rule 5.6 coverage for every file.
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
		missing := doc.CheckMissingDocs(items)
		if len(missing) > 0 {
			t.Errorf("%s: pub items missing docs (Rule 5.6): %v", f, missing)
		}
	}

	// Boundary invariant: no core file imports from full.
	// Walk stdlib/core; strip /// doc lines (which may mention
	// stdlib/full as a cross-reference) and search the
	// remaining code for import patterns pointing at
	// stdlib/full. A true leak would be a `use stdlib::full`
	// or similar qualified-path reference in a non-comment
	// position.
	coreFiles := collectFuseFiles(t, stdlibRoot(t))
	for _, f := range coreFiles {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		code := stripDocLines(string(src))
		for _, bad := range []string{
			"use stdlib::full",
			"use stdlib.full",
			"import stdlib/full",
			"stdlib::full::",
		} {
			if strings.Contains(code, bad) {
				t.Errorf("%s leaks hosted reference %q (core must not depend on full)", f, bad)
			}
		}
	}
}

// stripDocLines removes lines whose first non-whitespace
// characters are `///`. Doc comments may mention stdlib/full
// as a conceptual cross-reference without implying a real
// dependency; the leak check should not flag them.
func stripDocLines(text string) string {
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trim, "///") {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// TestConcurrency covers the W22 threading + sync + channel
// surface. The tests are structural — every concurrency-
// primitive file declares the required types and methods so
// downstream user code written against the spec shapes
// continues to compile.
//
// The runtime behavioural contract was proven end-to-end in
// W16 (TestSpawnObservable + TestChannelRoundTrip) using the
// underlying runtime entry points. The W22 hosted surface
// layers typed Fuse-level wrappers on top without changing
// the runtime ABI.
//
// Bound by:
//
//	go test ./tests/stdlib/... -run TestConcurrency -v
func TestConcurrency(t *testing.T) {
	root := hostedRoot(t)

	t.Run("thread-module-declares-ThreadHandle", func(t *testing.T) {
		threadPath := filepath.Join(root, "thread", "thread.fuse")
		src, err := os.ReadFile(threadPath)
		if err != nil {
			t.Fatalf("read %s: %v", threadPath, err)
		}
		text := string(src)
		if !strings.Contains(text, "pub struct ThreadHandle[T]") {
			t.Errorf("thread.fuse missing `pub struct ThreadHandle[T]`")
		}
		for _, method := range []string{"fn join(", "fn detach(", "fn current_id(", "fn yield_now("} {
			if !strings.Contains(text, method) {
				t.Errorf("thread.fuse missing %q", method)
			}
		}
	})

	t.Run("sync-primitives", func(t *testing.T) {
		mutexPath := filepath.Join(root, "sync", "mutex.fuse")
		mutexSrc, err := os.ReadFile(mutexPath)
		if err != nil {
			t.Fatalf("read %s: %v", mutexPath, err)
		}
		mutexText := string(mutexSrc)
		for _, decl := range []string{
			"pub struct Mutex[T]",
			"pub struct RwLock[T]",
			"pub struct Cond",
			"fn lock(",
			"fn unlock(",
			"fn read(",
			"fn write(",
			"fn wait(",
			"fn notify_one(",
			"fn notify_all(",
		} {
			if !strings.Contains(mutexText, decl) {
				t.Errorf("sync/mutex.fuse missing %q", decl)
			}
		}

		oncePath := filepath.Join(root, "sync", "once.fuse")
		onceSrc, err := os.ReadFile(oncePath)
		if err != nil {
			t.Fatalf("read %s: %v", oncePath, err)
		}
		if !strings.Contains(string(onceSrc), "pub struct Once") {
			t.Errorf("sync/once.fuse missing pub struct Once")
		}
		if !strings.Contains(string(onceSrc), "fn call_once(") {
			t.Errorf("sync/once.fuse missing fn call_once")
		}

		sharedPath := filepath.Join(root, "sync", "shared.fuse")
		sharedSrc, err := os.ReadFile(sharedPath)
		if err != nil {
			t.Fatalf("read %s: %v", sharedPath, err)
		}
		if !strings.Contains(string(sharedSrc), "pub struct Shared[T]") {
			t.Errorf("sync/shared.fuse missing pub struct Shared[T]")
		}
	})

	t.Run("channel-surface", func(t *testing.T) {
		chanPath := filepath.Join(root, "chan", "chan.fuse")
		src, err := os.ReadFile(chanPath)
		if err != nil {
			t.Fatalf("read %s: %v", chanPath, err)
		}
		text := string(src)
		if !strings.Contains(text, "pub struct Chan[T]") {
			t.Errorf("chan/chan.fuse missing pub struct Chan[T]")
		}
		if !strings.Contains(text, "pub struct ChanResult") {
			t.Errorf("chan/chan.fuse missing pub struct ChanResult")
		}
		for _, method := range []string{
			"fn send(", "fn recv(", "fn try_send(", "fn try_recv(",
			"fn close(", "fn is_ok(", "fn is_closed(", "fn would_block(",
		} {
			if !strings.Contains(text, method) {
				t.Errorf("chan/chan.fuse missing %q", method)
			}
		}
	})
}
