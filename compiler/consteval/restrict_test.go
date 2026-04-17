package consteval

import (
	"strings"
	"testing"
)

// TestConstFnRestrictions asserts the §46.1 restriction set —
// const fns may not call FFI, allocate, spawn, use unsafe blocks,
// take &mut references, or call non-const fns. Each subcase feeds
// a targeted source and asserts the diagnostic text names the
// violation; the checker runs independently of the evaluator so
// uncalled const fns are still subject to the rules.
func TestConstFnRestrictions(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		wantFrag string
	}{
		{
			name: "ffi-call",
			src: `
extern fn sys_time() -> U64;
const fn clock() -> U64 { return sys_time(); }
`,
			wantFrag: "FFI",
		},
		{
			name: "non-const-call",
			src: `
fn regular() -> I32 { return 1; }
const fn wrapper() -> I32 { return regular(); }
`,
			wantFrag: "non-const",
		},
		{
			name: "unsafe-block",
			src: `
const fn bad() -> I32 {
    return unsafe { 7 };
}
`,
			wantFrag: "unsafe",
		},
		{
			name: "closure-construction",
			src: `
const fn bad() -> I32 {
    let f = fn() -> I32 { return 1; };
    return 2;
}
`,
			wantFrag: "closures",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog := buildProgram(t, tc.src)
			diags := CheckRestrictions(prog)
			if len(diags) == 0 {
				t.Fatalf("expected restriction diagnostic, got none")
			}
			found := false
			for _, d := range diags {
				if strings.Contains(d.Message, tc.wantFrag) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("diagnostics did not mention %q; got %v", tc.wantFrag, diags)
			}
		})
	}
}
