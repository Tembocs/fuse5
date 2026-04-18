package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestSubcommandParser is the W18-P01-T01 Verify target. Every
// subcommand the plan declares (build / run / check / test / fmt
// / doc / repl / version / help) must be dispatched by the
// top-level router; unknown subcommands must exit 2 with a
// useful error; missing-argument cases must exit 2 (CLI misuse),
// not 1 (runtime error).
//
// The tests drive each subcommand with an argument list that
// makes the dispatch observable without requiring a compiler
// toolchain.
func TestSubcommandParser(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if got := run([]string{"version"}, &stdout, &stderr); got != 0 {
			t.Fatalf("version exit = %d (stderr=%q)", got, stderr.String())
		}
	})

	t.Run("help", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if got := run([]string{"help"}, &stdout, &stderr); got != 0 {
			t.Fatalf("help exit = %d", got)
		}
		// Every subcommand must appear in help output so a user
		// can discover them without reading the source.
		for _, name := range []string{"build", "run", "check", "test", "fmt", "doc", "repl", "lsp", "version", "help"} {
			if !strings.Contains(stdout.String(), name) {
				t.Errorf("help missing subcommand listing for %q: %s", name, stdout.String())
			}
		}
	})

	t.Run("unknown-subcommand-exits-2", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if got := run([]string{"xyzzy"}, &stdout, &stderr); got != 2 {
			t.Fatalf("unknown subcommand exit = %d, want 2", got)
		}
	})

	t.Run("no-args-exits-2", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if got := run(nil, &stdout, &stderr); got != 2 {
			t.Fatalf("empty args exit = %d, want 2", got)
		}
	})

	// Missing-source cases for each source-taking subcommand.
	// Exit code 2 (CLI misuse), not 1 (runtime).
	for _, sub := range []string{"build", "run", "check", "test", "fmt", "doc"} {
		sub := sub
		t.Run("missing-source-"+sub, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if got := run([]string{sub}, &stdout, &stderr); got != 2 {
				t.Fatalf("%s with no args exit = %d, want 2 (stderr=%q)", sub, got, stderr.String())
			}
		})
	}

	// Unknown flag on a source-taking subcommand.
	t.Run("unknown-flag-exits-2", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if got := run([]string{"build", "--no-such-flag", "a.fuse"}, &stdout, &stderr); got != 2 {
			t.Fatalf("unknown flag exit = %d, want 2", got)
		}
	})
}
