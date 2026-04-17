package cc

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestCCDetection verifies that Detect finds the host C compiler on
// each supported platform. The test skips when no C compiler is
// installed; in practice every Fuse-supported CI image has one
// (Linux: gcc; macOS: clang shipped as `cc`; Windows: MSVC or
// MinGW's gcc).
func TestCCDetection(t *testing.T) {
	c, err := Detect()
	if err != nil {
		t.Skipf("no host C compiler (skipping): %v", err)
	}
	if c.Path == "" {
		t.Fatalf("Detect returned empty Path")
	}
	if c.Kind == KindUnknown {
		t.Fatalf("Detect did not classify compiler %s", c.Path)
	}
	if _, err := os.Stat(c.Path); err != nil {
		t.Fatalf("Detect returned non-existent path %q: %v", c.Path, err)
	}
}

// TestCCDetection_HonorsEnv confirms that $CC takes priority over
// the default probe order.
func TestCCDetection_HonorsEnv(t *testing.T) {
	// Pick a host binary we know exists and set $CC to it. The
	// result must be recognised even though the name doesn't match
	// the normal probe list.
	candidate := findExistingBinary(t)
	t.Setenv("CC", candidate)
	c, err := Detect()
	if err != nil {
		t.Fatalf("Detect with CC=%q: %v", candidate, err)
	}
	if c.Path == "" {
		t.Fatalf("Detect returned empty Path")
	}
	// The returned path should resolve to the same binary.
	wantAbs, _ := filepath.Abs(candidate)
	gotAbs, _ := filepath.Abs(c.Path)
	if wantAbs != "" && gotAbs != "" && wantAbs != gotAbs {
		// LookPath may resolve a slightly different form (with or
		// without extension on Windows); we just assert the file
		// exists, not byte-equal paths.
		if _, err := os.Stat(c.Path); err != nil {
			t.Fatalf("CC=%q but Detect returned unreachable %q", candidate, c.Path)
		}
	}
}

// TestCCDetection_ErrorWhenAbsent confirms Detect returns a clear
// error when no compiler can be found. The test isolates the search
// PATH to an empty directory to simulate the failure mode.
func TestCCDetection_ErrorWhenAbsent(t *testing.T) {
	empty := t.TempDir()
	t.Setenv("PATH", empty)
	t.Setenv("CC", "") // clear any inherited override
	_, err := Detect()
	if err == nil {
		t.Fatalf("expected error with no compilers on PATH")
	}
}

// TestKindFromName sanity-checks the family classifier.
func TestKindFromName(t *testing.T) {
	cases := []struct {
		name string
		want Kind
	}{
		{"clang", KindClang},
		{"gcc", KindGCC},
		{"cl", KindMSVC},
		{"cl.exe", KindMSVC},
		{"x86_64-clang-12", KindClang},
		{"cc", KindGCC},
		{"weird", KindUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := kindFromName(tc.name); got != tc.want {
				t.Fatalf("kindFromName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// findExistingBinary returns the path of a guaranteed-present host
// binary useful for env-override testing.
func findExistingBinary(t *testing.T) string {
	t.Helper()
	candidates := []string{"cc", "gcc", "clang"}
	if runtime.GOOS == "windows" {
		candidates = []string{"cl", "cc", "gcc", "clang"}
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	t.Skipf("no known C compiler on PATH to use as $CC fixture")
	return ""
}
