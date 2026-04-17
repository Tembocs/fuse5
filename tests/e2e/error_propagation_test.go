package e2e_test

import "testing"

// TestErrorPropagation is the W11-P03-T01 Verify target. The
// wave doc mandates two observable exit codes — `run(false)`
// exits 43 and `run(true)` exits 0 — so the harness compiles
// and runs two sibling proof sources that share the same
// program shape and differ only in their main's argument.
//
// Both sources build to a neutral stem (`ep_err` / `ep_ok`) so
// the Windows launch path stays predictable (audit report
// 2026-04-17 13:05, W10 finding G).
func TestErrorPropagation(t *testing.T) {
	skipIfNoCC(t)
	t.Run("run-false-propagates-err", func(t *testing.T) {
		result := mustBuildAs(t, "error_propagation_err.fuse", "ep_err")
		exit := mustRun(t, result.BinaryPath)
		if exit != 43 {
			t.Fatalf("error_propagation_err exit = %d, want 43", exit)
		}
	})
	t.Run("run-true-continues-ok", func(t *testing.T) {
		result := mustBuildAs(t, "error_propagation_ok.fuse", "ep_ok")
		exit := mustRun(t, result.BinaryPath)
		if exit != 0 {
			t.Fatalf("error_propagation_ok exit = %d, want 0", exit)
		}
	})
}
