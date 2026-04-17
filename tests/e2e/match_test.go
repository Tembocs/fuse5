package e2e_test

import "testing"

// TestMatchEnumDispatch is the W10-P04-T01 Verify target.
// `match_enum_dispatch.fuse` builds an enum Dir, a fn `pick` that
// matches on it returning 42 for North and 7 for South, and main
// invokes `pick(Dir.North)`. The produced binary must exit 42,
// proving match dispatch + enum-variant construction work end to
// end.
//
// Builds with a neutral output stem ("mproof") to avoid the
// Windows UAC / SmartScreen heuristic that the 2026-04-17 13:05
// audit reproduced against the default `match_enum_dispatch.exe`
// name on one host. The underlying source name and pipeline are
// unchanged.
func TestMatchEnumDispatch(t *testing.T) {
	skipIfNoCC(t)
	result := mustBuildAs(t, "match_enum_dispatch.fuse", "mproof")
	exit := mustRun(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("match_enum_dispatch exit = %d, want 42", exit)
	}
}
