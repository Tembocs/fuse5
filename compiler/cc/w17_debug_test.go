package cc

import (
	"testing"
)

// TestDebugFlagPassthrough confirms that cc.Options{Debug: true}
// injects the host-specific debug flag into the compile invocation
// on every supported compiler family. The test does not invoke the
// real compiler — it inspects the arg vector the compiler would
// produce for a canonical input.
//
// Bound by the wave-doc Verify command:
//
//	go test ./compiler/cc/... -run TestDebugFlagPassthrough -v
func TestDebugFlagPassthrough(t *testing.T) {
	cases := []struct {
		name  string
		kind  Kind
		debug bool
		wants []string // every string must appear in the arg list
	}{
		{"gcc-debug", KindGCC, true, []string{"-g", "-std=c11"}},
		{"gcc-release", KindGCC, false, []string{"-std=c11"}},
		{"clang-debug", KindClang, true, []string{"-g", "-std=c11"}},
		{"msvc-debug", KindMSVC, true, []string{"/Zi", "/std:c11"}},
		{"msvc-release", KindMSVC, false, []string{"/std:c11"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := Options{Debug: tc.debug}
			args := BuildCompileArgs(tc.kind, "in.c", "out", opts)
			for _, want := range tc.wants {
				found := false
				for _, a := range args {
					if a == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: expected %q in args %v", tc.name, want, args)
				}
			}
			// Release must not carry the debug flag.
			if !tc.debug {
				for _, a := range args {
					if a == "-g" || a == "/Zi" {
						t.Errorf("%s: release build should not carry %q", tc.name, a)
					}
				}
			}
		})
	}
}
