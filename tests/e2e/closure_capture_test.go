package e2e_test

import "testing"

// TestClosureCaptureRuns is the W12-P03-T01 Verify target. The
// fixture `closure_capture.fuse` defines and immediately invokes
// a no-capture closure; the lowerer inlines the body. Exit 42
// proves the full pipeline (bridge → check → monomorph →
// liveness → lower → codegen → cc) agrees on closure shape.
//
// Builds to a neutral output stem per the audit-followup
// pattern (W10 finding G).
func TestClosureCaptureRuns(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuildAs(t, "closure_capture.fuse", "cproof")
	exit := mustRun(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("closure_capture exit = %d, want 42", exit)
	}
}
