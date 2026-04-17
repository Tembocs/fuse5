package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMinimalCli is the W05 CLI Verify target. It drives the `build`
// subcommand end-to-end: write a Fuse source, run `fuse build`, run
// the produced binary, check exit code. Skips when no C compiler is
// on the host.
func TestMinimalCli(t *testing.T) {
	skipIfNoCC(t)

	t.Run("build zero", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "zero.fuse")
		writeFile(t, src, `fn main() -> I32 { return 0; }`)
		out := filepath.Join(dir, binaryName("zero"))

		var stdout, stderr bytes.Buffer
		code := run([]string{"build", "-o", out, src}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("fuse build exit = %d; stderr=%q", code, stderr.String())
		}
		if _, err := os.Stat(out); err != nil {
			t.Fatalf("binary not produced at %s: %v", out, err)
		}
		if !strings.Contains(stdout.String(), "wrote") {
			t.Errorf("stdout missing 'wrote' confirmation: %q", stdout.String())
		}

		cmd := exec.Command(out)
		_ = cmd.Run()
		if exit := cmd.ProcessState.ExitCode(); exit != 0 {
			t.Fatalf("binary exit = %d, want 0", exit)
		}
	})

	t.Run("build reports diagnostics", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "bad.fuse")
		writeFile(t, src, `fn main() -> I32 { return true; }`)

		var stdout, stderr bytes.Buffer
		code := run([]string{"build", src}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("fuse build on invalid source should fail")
		}
		if !strings.Contains(stderr.String(), "only lowers integer literals") &&
			!strings.Contains(stderr.String(), "integer") {
			t.Errorf("stderr missing expected diagnostic: %q", stderr.String())
		}
	})

	t.Run("build without source is a usage error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"build"}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("fuse build without source should fail")
		}
		if !strings.Contains(stderr.String(), "missing source") {
			t.Errorf("stderr missing usage message: %q", stderr.String())
		}
	})

	t.Run("build unknown flag is a usage error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"build", "--bogus", "x.fuse"}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("unknown flag should fail")
		}
		if !strings.Contains(stderr.String(), "unknown flag") {
			t.Errorf("stderr missing 'unknown flag': %q", stderr.String())
		}
	})
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func skipIfNoCC(t *testing.T) {
	t.Helper()
	candidates := []string{"cc", "gcc", "clang"}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, "cl")
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return
		}
	}
	t.Skipf("no host C compiler; skipping CLI end-to-end test")
}

func binaryName(stem string) string {
	if runtime.GOOS == "windows" {
		return stem + ".exe"
	}
	return stem
}
