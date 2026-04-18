package e2e_test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCliBasicWorkflow is the W18-P01-T02 Verify target. It drives
// the freshly-built `fuse` binary through the canonical developer
// workflow — version, help, check, build, run — and asserts each
// subcommand exits with the declared code. The test compiles the
// Fuse CLI once (into a temp binary) so the shelled-out
// invocations do not depend on a pre-installed `fuse` executable.
func TestCliBasicWorkflow(t *testing.T) {
	skipIfNoCC(t)
	preferGCCForRuntimeTests(t)

	fuseBin := buildFuseBinary(t)

	// Canonical proof program already in the repo.
	src, err := filepath.Abs("hello_exit.fuse")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	t.Run("version-includes-wave-tag", func(t *testing.T) {
		out, _, code := runCLI(t, fuseBin, "version")
		if code != 0 {
			t.Fatalf("version exit = %d", code)
		}
		if !strings.Contains(out, "-W") {
			t.Errorf("version %q missing wave tag", out)
		}
	})

	t.Run("help-lists-every-subcommand", func(t *testing.T) {
		out, _, code := runCLI(t, fuseBin, "help")
		if code != 0 {
			t.Fatalf("help exit = %d", code)
		}
		for _, sub := range []string{"build", "run", "check", "test", "fmt", "doc", "repl"} {
			if !strings.Contains(out, sub) {
				t.Errorf("help missing %q: %s", sub, out)
			}
		}
	})

	t.Run("check-exits-zero-for-good-source", func(t *testing.T) {
		_, _, code := runCLI(t, fuseBin, "check", src)
		if code != 0 {
			t.Errorf("check exit = %d for a known-good source", code)
		}
	})

	t.Run("build-wires-driver", func(t *testing.T) {
		out := t.TempDir()
		outBin := filepath.Join(out, "hello")
		_, _, code := runCLI(t, fuseBin, "build", "-o", outBin, src)
		if code != 0 {
			t.Errorf("build exit = %d", code)
		}
	})

	t.Run("run-executes-binary", func(t *testing.T) {
		_, _, code := runCLI(t, fuseBin, "run", src)
		if code != 0 {
			t.Errorf("run exit = %d for hello_exit", code)
		}
	})

	t.Run("unknown-subcommand-exits-two", func(t *testing.T) {
		_, _, code := runCLI(t, fuseBin, "bogus")
		if code != 2 {
			t.Errorf("unknown subcommand exit = %d, want 2", code)
		}
	})
}

// buildFuseBinary compiles cmd/fuse into a temp binary the test
// shells out against. Using `go build` keeps the test hermetic
// and close to how a user installs the CLI.
func buildFuseBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "fuse")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "../../cmd/fuse")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build cmd/fuse: %v\n%s", err, out.String())
	}
	return bin
}

// runCLI invokes the fuse binary with args, returning stdout,
// stderr, and the exit code.
func runCLI(t *testing.T, bin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatalf("launch %s %v: %v", bin, args, err)
		}
	}
	return stdout.String(), stderr.String(), code
}
