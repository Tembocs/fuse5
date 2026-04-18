package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestCliStub is the original W00 Verify target. It now covers the
// W00-surface invariants that still hold at W05: version reports the
// active wave, help prints a subcommand listing, and missing
// arguments are a usage error.
func TestCliStub(t *testing.T) {
	t.Run("version prints and exits zero", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"version"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("version exit = %d, want 0; stderr=%q", code, stderr.String())
		}
		// The version carries the active-wave tag; pre-1.0 waves
		// bump it forward. At W18 we just assert the string looks
		// like a version (has a dash+letter) so future wave bumps
		// don't require editing this test every time.
		if !strings.Contains(stdout.String(), "-W") {
			t.Errorf("version stdout = %q, want to include a `-W<wave>` tag", stdout.String())
		}
	})

	t.Run("help prints and exits zero", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"help"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("help exit = %d, want 0", code)
		}
		if !strings.Contains(stdout.String(), "subcommands") {
			t.Errorf("help output missing subcommand listing: %q", stdout.String())
		}
	})

	t.Run("no arguments is a usage error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run(nil, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("empty args exit = 0, want non-zero")
		}
		if stderr.Len() == 0 {
			t.Errorf("empty args produced no stderr message")
		}
	})

	t.Run("truly unknown subcommand exits non-zero", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"xyzzy"}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("unknown subcommand 'xyzzy' exit = 0, want non-zero")
		}
	})
}
