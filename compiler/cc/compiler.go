package cc

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Kind identifies the C compiler family so invocation can pick the
// right flag spelling (GCC/Clang use `-std=c11`; MSVC uses `/std:c11`).
type Kind int

const (
	KindUnknown Kind = iota
	KindGCC
	KindClang
	KindMSVC
)

// String returns a stable identifier used in diagnostics and logs.
func (k Kind) String() string {
	switch k {
	case KindGCC:
		return "gcc"
	case KindClang:
		return "clang"
	case KindMSVC:
		return "msvc"
	}
	return "unknown"
}

// Compiler names a detected host C compiler. Path is the absolute
// path to the executable; Kind is the family (for flag spelling).
type Compiler struct {
	Path string
	Kind Kind
}

// Detect picks a C compiler from the host environment. The policy:
//
//  1. If `$CC` is set and points to an existing executable, use it.
//     The Kind is guessed from the basename.
//  2. Otherwise, probe `cc`, `gcc`, `clang`, `cl` in that order on
//     Windows; `cc`, `clang`, `gcc` on Unix (macOS ships clang as
//     `cc`; Linux varies).
//  3. If none are found, return an error. The error names every
//     probed candidate so the user can tell what Fuse looked for.
func Detect() (*Compiler, error) {
	if envCC := strings.TrimSpace(os.Getenv("CC")); envCC != "" {
		if full, err := exec.LookPath(envCC); err == nil {
			return &Compiler{Path: full, Kind: kindFromName(envCC)}, nil
		}
	}
	var probes []string
	if runtime.GOOS == "windows" {
		probes = []string{"cc", "gcc", "clang", "cl"}
	} else {
		probes = []string{"cc", "clang", "gcc"}
	}
	for _, name := range probes {
		if full, err := exec.LookPath(name); err == nil {
			return &Compiler{Path: full, Kind: kindFromName(name)}, nil
		}
	}
	return nil, fmt.Errorf("no C compiler found: tried %s (set $CC to override)", strings.Join(probes, ", "))
}

// kindFromName guesses the compiler family from the executable
// basename. The mapping is intentionally permissive — a binary named
// `cc` might be clang or gcc; we pick a reasonable default and let
// Compile's flag translation stay compatible with both.
func kindFromName(name string) Kind {
	// Trim path and extension for matching.
	n := strings.ToLower(baseName(name))
	switch {
	case strings.Contains(n, "clang"):
		return KindClang
	case strings.Contains(n, "gcc"):
		return KindGCC
	case strings.HasPrefix(n, "cl"):
		return KindMSVC
	case n == "cc":
		// `cc` on macOS is clang; on Linux it's usually gcc. Either
		// way the flag set we use is common to both.
		return KindGCC
	}
	return KindUnknown
}

// baseName returns the final path component of p without its
// extension. Used by kindFromName.
func baseName(p string) string {
	// Strip directories.
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			p = p[i+1:]
			break
		}
	}
	// Strip extension.
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '.' {
			return p[:i]
		}
	}
	return p
}

// Compile invokes c on cInputPath and produces a native binary at
// outBinaryPath. The returned error, when non-nil, includes the
// compiler's stderr so the caller can surface the native diagnostic
// directly.
//
// C11 is requested explicitly. Warnings are *not* treated as errors
// at W05 — later waves (W17) tighten the warning policy.
func (c *Compiler) Compile(cInputPath, outBinaryPath string) error {
	var args []string
	switch c.Kind {
	case KindMSVC:
		// `cl` spelling: `/Fe:out.exe in.c /std:c11`. MSVC emits
		// object files alongside the source by default; we route
		// them into the target directory to avoid cluttering cwd.
		args = []string{"/std:c11", "/nologo", "/TC", cInputPath, "/Fe:" + outBinaryPath}
	default:
		// GCC/Clang spelling: `cc -std=c11 -o out in.c`.
		args = []string{"-std=c11", "-o", outBinaryPath, cInputPath}
	}
	cmd := exec.Command(c.Path, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %v\n%s", c.Kind, err, out)
	}
	return nil
}
