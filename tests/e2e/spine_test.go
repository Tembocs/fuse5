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

// TestIdentityGeneric is the W08-P06-T01 Verify target.
// `identity_generic.fuse` exercises a generic `fn identity[T]`
// called with an explicit turbofish; monomorphization produces
// `identity__I32` which is what main calls.
func TestIdentityGeneric(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuild(t, "identity_generic.fuse")
	exit := mustRun(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("identity_generic exit = %d, want 42", exit)
	}
}

// TestMultipleInstantiations is the W08-P06-T02 Verify target.
// `multiple_instantiations.fuse` exercises two specializations of
// the same generic fn with different type args; both must reach
// codegen as distinct C functions and both calls must link.
func TestMultipleInstantiations(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuild(t, "multiple_instantiations.fuse")
	exit := mustRun(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("multiple_instantiations exit = %d, want 42", exit)
	}
}

// TestConstFnProof is the W14-P05-T01 Verify target.
// `const_fn.fuse` computes `factorial(5)` at compile time (120)
// via a recursive `const fn` body, stores it in the const FACT_5,
// and returns `FACT_5 as I32` from main. The expected exit code
// is 120, which proves the W14 evaluator ran, the substitution
// pass propagated the value, and the cast lowering narrowed U64
// to I32 correctly.
func TestConstFnProof(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuild(t, "const_fn.fuse")
	exit := mustRun(t, result.BinaryPath)
	if exit != 120 {
		t.Fatalf("const_fn exit = %d, want 120", exit)
	}
}

// mustBuild invokes the Stage 1 driver on the named proof
// program. The produced binary's stem defaults to the source
// stem — use mustBuildAs when a test needs a specific output
// name (for instance, to dodge host-level heuristics that react
// to the default name; see audit report 2026-04-17 13:05 for
// the W10 finding that motivated mustBuildAs).
// The binary lives under the test's temporary directory so
// parallel test runs don't collide.
func mustBuild(t *testing.T, sourceName string) *driver.BuildResult {
	t.Helper()
	return mustBuildAs(t, sourceName, trimExt(sourceName))
}

// mustBuildAs is mustBuild with an explicit output-binary stem.
// Windows hosts occasionally trip Defender / SmartScreen / UAC
// heuristics on certain executable names; tests that want to
// avoid that pass a short, neutral stem like "mproof".
func mustBuildAs(t *testing.T, sourceName, outputStem string) *driver.BuildResult {
	t.Helper()
	src, err := filepath.Abs(sourceName)
	if err != nil {
		t.Fatalf("abs %s: %v", sourceName, err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, binaryName(outputStem))
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

// mustRun executes the binary and returns its exit code.
//
// Launch failures are surfaced explicitly via t.Fatalf. A process
// that never starts (file-not-found, permission denied, UAC
// elevation refusal, etc.) is a real test failure distinct from
// a non-zero exit. Earlier versions of this helper discarded the
// error and read ProcessState.ExitCode() unconditionally, which
// let Windows launch failures masquerade as ambiguous exit codes
// (audit report 2026-04-17 13:05, W10 finding G).
//
// A non-zero exit is NOT a helper-level failure — callers assert
// on the returned value.
func mustRun(t *testing.T, binPath string) int {
	t.Helper()
	cmd := exec.Command(binPath)
	err := cmd.Run()
	if err != nil {
		// *exec.ExitError means the process ran to completion
		// and returned non-zero — that's a legal result for
		// rejection-style proofs.
		if _, isExit := err.(*exec.ExitError); !isExit {
			t.Fatalf("launch %s: %v", binPath, err)
		}
	}
	if cmd.ProcessState == nil {
		t.Fatalf("launch %s: no ProcessState (the process never started)", binPath)
	}
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
