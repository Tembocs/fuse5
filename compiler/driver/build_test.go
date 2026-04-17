package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMinimalBuildInvocation drives the full pipeline end-to-end:
// write a Fuse source file, invoke Build, execute the produced
// binary, and verify the exit code. Skips when no C compiler is
// available on the host.
func TestMinimalBuildInvocation(t *testing.T) {
	skipIfNoCC(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "hello.fuse")
	writeFile(t, srcPath, `fn main() -> I32 { return 0; }`)

	res, diags, err := Build(BuildOptions{Source: srcPath, WorkDir: dir, KeepC: true})
	if err != nil {
		t.Fatalf("Build: %v (diags: %v)", err, diags)
	}
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if res == nil {
		t.Fatalf("Build returned nil result")
	}
	if _, err := os.Stat(res.BinaryPath); err != nil {
		t.Fatalf("binary not produced at %s: %v", res.BinaryPath, err)
	}
	// Execute the binary and check exit code.
	cmd := exec.Command(res.BinaryPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run binary: %v", err)
	}
	if exit := cmd.ProcessState.ExitCode(); exit != 0 {
		t.Fatalf("binary exited with %d, want 0", exit)
	}
	// KeepC should have left the C source next to the binary.
	if _, err := os.Stat(res.CSourcePath); err != nil {
		t.Fatalf("KeepC set but C source missing: %v", err)
	}
}

// TestMinimalBuildInvocation_ExitCode confirms a non-zero exit
// propagates through build-and-run.
func TestMinimalBuildInvocation_ExitCode(t *testing.T) {
	skipIfNoCC(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "exit.fuse")
	writeFile(t, srcPath, `fn main() -> I32 { return 7; }`)

	res, _, err := Build(BuildOptions{Source: srcPath, WorkDir: dir})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cmd := exec.Command(res.BinaryPath)
	runErr := cmd.Run()
	// A non-zero exit is surfaced by cmd.Run as *exec.ExitError; we
	// just read ProcessState.ExitCode() either way.
	_ = runErr
	if exit := cmd.ProcessState.ExitCode(); exit != 7 {
		t.Fatalf("exit = %d, want 7", exit)
	}
}

// TestMinimalBuildInvocation_RejectsInvalid confirms the driver
// returns diagnostics rather than panicking on un-lowerable input.
func TestMinimalBuildInvocation_RejectsInvalid(t *testing.T) {
	skipIfNoCC(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.fuse")
	// Bool return is not yet lowered by the W05 spine.
	writeFile(t, srcPath, `fn main() -> I32 { return true; }`)

	_, diags, err := Build(BuildOptions{Source: srcPath, WorkDir: dir})
	if err == nil {
		t.Fatalf("expected build error for unsupported body")
	}
	if len(diags) == 0 {
		t.Fatalf("expected lowerer diagnostic, got none")
	}
}

// writeFile is a tiny helper for the tests above.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// skipIfNoCC skips the test when the host has no C compiler. The
// W05 e2e proofs require one; when running under CI they're always
// available.
func skipIfNoCC(t *testing.T) {
	t.Helper()
	candidates := []string{"cc", "gcc", "clang"}
	if runtime.GOOS == "windows" {
		candidates = []string{"cc", "gcc", "clang", "cl"}
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return
		}
	}
	t.Skipf("no host C compiler; skipping driver end-to-end test")
}
