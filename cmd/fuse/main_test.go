package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestCliStub is the W00-P02-T03 Verify target. It confirms the Stage 1 CLI
// binary exists, builds, and exposes the minimal subcommand surface expected
// at Wave 00: version and help. The subcommand surface expands in W18.
func TestCliStub(t *testing.T) {
	t.Run("version prints and exits zero", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"version"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("version exit = %d, want 0; stderr=%q", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "W00") {
			t.Errorf("version stdout = %q, want to contain W00", stdout.String())
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

	t.Run("unknown subcommand exits non-zero", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"build"}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("unknown subcommand 'build' exit = 0, want non-zero (W18 wires build)")
		}
	})
}
