// Package goldens implements the byte-stable golden-file harness for the
// project (Rule 6.2, Rule 7.1). Tests use Assert to compare produced output
// against a tracked golden file; updates require an intentional workflow
// through the UPDATE_GOLDENS environment variable.
package goldens

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// TB is the subset of testing.TB used by Assert. Accepting an interface
// instead of *testing.T lets callers wrap the tester (for example, to
// capture expected failures in negative tests).
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
}

// Assert compares actual against the golden file at path. If they differ,
// the test fails. When the UPDATE_GOLDENS environment variable is set to a
// non-empty value, Assert writes actual to the golden file instead — this
// is the sanctioned update path.
//
// Relative paths resolve against the caller's package directory.
func Assert(t TB, path string, actual []byte) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDENS") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("goldens: mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, actual, 0o644); err != nil {
			t.Fatalf("goldens: write %s: %v", path, err)
		}
		t.Logf("goldens: updated %s (%d bytes)", path, len(actual))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("goldens: read %s: %v (set UPDATE_GOLDENS=1 to seed)", path, err)
	}
	if !bytes.Equal(want, actual) {
		t.Fatalf("goldens: output for %s does not match golden\n--- want (%d bytes)\n%s\n--- got (%d bytes)\n%s\n(set UPDATE_GOLDENS=1 to update)",
			path, len(want), truncate(want), len(actual), truncate(actual))
	}
}

func truncate(b []byte) string {
	const max = 256
	if len(b) <= max {
		return string(b)
	}
	return fmt.Sprintf("%s... [%d bytes truncated]", b[:max], len(b)-max)
}
