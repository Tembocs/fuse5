package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// locateRuntimeArtifacts resolves the include directory, static-
// library path, and platform link libraries the driver must pass
// to the host C compiler when the current translation unit depends
// on the Wave 16 runtime ABI (see codegen.UsesRuntimeABI).
//
// The search strategy is:
//
//  1. Honour FUSE_RUNTIME_DIR when set — tests and the driver can
//     point at a pre-built artifacts directory.
//  2. Otherwise, locate `runtime/include/fuse_rt.h` by walking up
//     from the Go-module source root. This works in every checkout
//     without further configuration.
//
// On first call we build libfuse_rt.a via `make -C runtime all`
// (once per process) so subsequent compiles link against a ready
// archive. The build is a no-op when the archive is up to date.
//
// Return triple: (include dirs, extra objects, extra libs).
// On POSIX platforms `pthread` is appended to libs; on Windows the
// libs slice is empty (pthread is in libc).
func locateRuntimeArtifacts() ([]string, []string, []string, error) {
	runtimeDir, err := findRuntimeDir()
	if err != nil {
		return nil, nil, nil, err
	}
	includeDir := filepath.Join(runtimeDir, "include")
	libPath := filepath.Join(runtimeDir, "build", "libfuse_rt.a")
	if err := ensureRuntimeLib(runtimeDir, libPath); err != nil {
		return nil, nil, nil, err
	}
	libs := []string{}
	if runtime.GOOS != "windows" {
		libs = append(libs, "pthread")
	}
	return []string{includeDir}, []string{libPath}, libs, nil
}

// findRuntimeDir returns the absolute path to the runtime/ tree.
// Honours FUSE_RUNTIME_DIR; otherwise walks up from this file's
// location (resolved through runtime.Caller) to find a sibling
// runtime/include/fuse_rt.h.
func findRuntimeDir() (string, error) {
	if env := os.Getenv("FUSE_RUNTIME_DIR"); env != "" {
		if _, err := os.Stat(filepath.Join(env, "include", "fuse_rt.h")); err == nil {
			return env, nil
		}
	}
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("driver.findRuntimeDir: runtime.Caller failed")
	}
	dir := filepath.Dir(self)
	// Walk up looking for a sibling `runtime/include/fuse_rt.h`.
	for {
		candidate := filepath.Join(dir, "runtime", "include", "fuse_rt.h")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, "runtime"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("driver.findRuntimeDir: runtime/ not found walking up from %s", self)
		}
		dir = parent
	}
}

// runtimeBuildOnce ensures we attempt the `make` only once per
// process. Subsequent callers re-use the outcome and, by that point,
// either succeed (the archive exists) or fail (build failed).
var runtimeBuildOnce sync.Once
var runtimeBuildErr error

// ensureRuntimeLib guarantees that `libPath` exists. When absent it
// runs `make -C runtimeDir all` using the caller's PATH. If make is
// unavailable the error explains the recovery — typically "run
// `make -C runtime all` yourself".
func ensureRuntimeLib(runtimeDir, libPath string) error {
	if _, err := os.Stat(libPath); err == nil {
		return nil
	}
	runtimeBuildOnce.Do(func() {
		makeBin, err := exec.LookPath("make")
		if err != nil {
			runtimeBuildErr = fmt.Errorf("runtime library missing and `make` not on PATH: build %s first (set FUSE_RUNTIME_DIR to skip)", libPath)
			return
		}
		cmd := exec.Command(makeBin, "-C", runtimeDir, "all")
		out, err := cmd.CombinedOutput()
		if err != nil {
			runtimeBuildErr = fmt.Errorf("runtime build failed: %v\n%s", err, out)
		}
	})
	if runtimeBuildErr != nil {
		return runtimeBuildErr
	}
	if _, err := os.Stat(libPath); err != nil {
		return fmt.Errorf("runtime build produced no archive at %s", libPath)
	}
	return nil
}
