// Package e2e_test owns the Fuse end-to-end proof program suite. At
// W05 it contains the two spine proofs declared in this directory's
// README: hello_exit (exit 0) and exit_with_value (exit 42). Each
// test runs the Stage 1 driver against the committed `.fuse` source,
// runs the produced binary, and asserts on the exit code.
package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Tembocs/fuse5/compiler/driver"
)

// TestHelloExit is the W05-P05-T01 Verify target. `hello_exit.fuse`
// compiles, links, runs, and returns 0.
func TestHelloExit(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuild(t, "hello_exit.fuse")
	exit := mustRun(t, result.BinaryPath)
	if exit != 0 {
		t.Fatalf("hello_exit exit = %d, want 0", exit)
	}
}

// TestExitWithValue is the W05-P05-T02 Verify target.
// `exit_with_value.fuse` computes 6*7 and exits with that value.
func TestExitWithValue(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuild(t, "exit_with_value.fuse")
	exit := mustRun(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("exit_with_value exit = %d, want 42", exit)
	}
}

// TestCheckerBasicProof is the W06-P09-T01 Verify target.
// `checker_basic.fuse` exercises a multi-fn program with typed
// parameters, integer arithmetic in a callee, and a direct call
// from main — the whole pipeline must agree on types for the
// binary to exit 42.
func TestCheckerBasicProof(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuild(t, "checker_basic.fuse")
	exit := mustRun(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("checker_basic exit = %d, want 42", exit)
	}
}

// mustBuild invokes the Stage 1 driver on the named proof program.
// The binary lives under the test's temporary directory so parallel
// test runs don't collide.
func mustBuild(t *testing.T, sourceName string) *driver.BuildResult {
	t.Helper()
	src, err := filepath.Abs(sourceName)
	if err != nil {
		t.Fatalf("abs %s: %v", sourceName, err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, binaryName(trimExt(sourceName)))
	res, diags, err := driver.Build(driver.BuildOptions{
		Source:  src,
		Output:  out,
		WorkDir: dir,
	})
	for _, d := range diags {
		t.Logf("diag: %s: %s", d.Span, d.Message)
	}
	if err != nil {
		t.Fatalf("build %s: %v", sourceName, err)
	}
	return res
}

// mustRun executes the binary and returns its exit code. A zero
// exit is reported as 0; a non-zero exit (including signal
// termination on Unix) is returned numerically via ProcessState.
func mustRun(t *testing.T, binPath string) int {
	t.Helper()
	cmd := exec.Command(binPath)
	_ = cmd.Run() // non-zero exit is not a test failure by itself
	return cmd.ProcessState.ExitCode()
}

// binaryName appends the host executable extension when needed.
func binaryName(stem string) string {
	if runtime.GOOS == "windows" {
		return stem + ".exe"
	}
	return stem
}

// trimExt strips the final `.xxx` extension from a filename.
func trimExt(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[:i]
		}
	}
	return name
}

// skipIfNoCC skips the test when no host C compiler is available.
// The Verify command in the wave spec is run on CI images that
// always ship a C compiler (ubuntu/macos: cc; windows: gcc or cl).
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
	t.Skipf("no host C compiler; skipping e2e test")
}

// osStub keeps the `os` import referenced if later helpers drop it.
var _ = os.Stdout
