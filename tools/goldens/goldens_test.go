package goldens

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestGoldenStability is the W00-P03-T03 Verify target. It confirms the
// golden harness is byte-stable: the same input produces the same golden
// bytes across multiple runs (Rule 6.2, Rule 7.1). The test is run with
// -count=2 to exercise the stability property across repeated invocations.
func TestGoldenStability(t *testing.T) {
	tmp := t.TempDir()
	golden := filepath.Join(tmp, "sample.golden")

	// Produce a deterministic payload: no wall-clock, no randomness, no
	// environment-dependent content (Rule 7.3, Rule 7.4).
	payload := []byte("fuse-golden-stability\nline2\nline3\n")

	// Seed the golden via the sanctioned update path.
	t.Setenv("UPDATE_GOLDENS", "1")
	Assert(t, golden, payload)

	// Switch off update mode and assert the same payload round-trips.
	t.Setenv("UPDATE_GOLDENS", "")
	Assert(t, golden, payload)

	// Corrupt the golden and confirm Assert fails via a subtest.
	if err := os.WriteFile(golden, []byte("different"), 0o644); err != nil {
		t.Fatalf("prep: %v", err)
	}
	t.Run("detects mismatch", func(t *testing.T) {
		// Use a sub-testing.T that captures failure rather than propagating.
		st := &subT{T: t}
		Assert(st, golden, payload)
		if !st.failed {
			t.Errorf("Assert did not fail on mismatched golden")
		}
	})
}

// subT wraps testing.T to capture whether Fatalf fired, so we can assert
// that Assert correctly reports a mismatch.
type subT struct {
	*testing.T
	failed bool
}

func (s *subT) Fatalf(format string, args ...any) {
	s.failed = true
	// Log and return rather than terminate the subtest; the outer test
	// checks s.failed.
	s.T.Logf("expected failure captured: "+format, args...)
}

func (s *subT) Helper() { s.T.Helper() }

// Ensure the type satisfies what Assert needs. Assert uses t.Helper() and
// t.Fatalf(); both are here. Sanity-check at package init so a refactor to
// Assert's surface trips immediately.
func init() {
	var _ interface {
		Helper()
		Fatalf(string, ...any)
	} = (*subT)(nil)
	_ = fmt.Sprintf
}
