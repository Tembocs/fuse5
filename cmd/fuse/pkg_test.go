package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPkgSubcommands exercises the W23 package-manager CLI
// surface: `fuse add`, `fuse remove`, `fuse update`, `fuse
// vendor`. Each subcommand mutates fuse.toml / fuse.lock
// atomically; partial writes leave both in consistent prior
// state.
func TestPkgSubcommands(t *testing.T) {
	// Each sub-test runs in a fresh temp directory with a
	// minimal fuse.toml. chdir into it so the subcommands
	// find the manifest.
	workIn := func(t *testing.T, manifest string, fn func(dir string)) {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "fuse.toml"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("seed manifest: %v", err)
		}
		orig, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer os.Chdir(orig)
		fn(dir)
	}

	baseManifest := `
[package]
name = "root"
version = "0.1.0"

[dependencies]
alpha = "^1.0.0"
`

	t.Run("add-new-dep", func(t *testing.T) {
		workIn(t, baseManifest, func(dir string) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"add", "beta@2.0.0"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("add exit = %d, stderr=%s", code, stderr.String())
			}
			body, _ := os.ReadFile(filepath.Join(dir, "fuse.toml"))
			if !strings.Contains(string(body), "beta = \"2.0.0\"") {
				t.Errorf("manifest after add missing beta: %s", body)
			}
		})
	})

	t.Run("add-invalidates-lockfile", func(t *testing.T) {
		workIn(t, baseManifest, func(dir string) {
			lockPath := filepath.Join(dir, "fuse.lock")
			if err := os.WriteFile(lockPath, []byte("placeholder lock"), 0o644); err != nil {
				t.Fatalf("seed lockfile: %v", err)
			}
			var stdout, stderr bytes.Buffer
			if code := run([]string{"add", "gamma@1.0.0"}, &stdout, &stderr); code != 0 {
				t.Fatalf("add exit = %d", code)
			}
			if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
				t.Errorf("lockfile not invalidated after add")
			}
		})
	})

	t.Run("add-path-dependency", func(t *testing.T) {
		workIn(t, baseManifest, func(dir string) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"add", "--path", "../local", "local"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("add --path exit = %d, stderr=%s", code, stderr.String())
			}
			body, _ := os.ReadFile(filepath.Join(dir, "fuse.toml"))
			if !strings.Contains(string(body), `path = "../local"`) {
				t.Errorf("path dep not recorded: %s", body)
			}
		})
	})

	t.Run("remove-existing", func(t *testing.T) {
		workIn(t, baseManifest, func(dir string) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"remove", "alpha"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("remove exit = %d", code)
			}
			body, _ := os.ReadFile(filepath.Join(dir, "fuse.toml"))
			if strings.Contains(string(body), "alpha") {
				t.Errorf("manifest still mentions alpha: %s", body)
			}
		})
	})

	t.Run("remove-missing-is-user-error", func(t *testing.T) {
		workIn(t, baseManifest, func(dir string) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"remove", "nonexistent"}, &stdout, &stderr)
			if code != 1 {
				t.Errorf("remove of missing dep exit = %d, want 1", code)
			}
		})
	})

	t.Run("update-drops-lockfile", func(t *testing.T) {
		workIn(t, baseManifest, func(dir string) {
			lockPath := filepath.Join(dir, "fuse.lock")
			os.WriteFile(lockPath, []byte("stale"), 0o644)
			var stdout, stderr bytes.Buffer
			code := run([]string{"update"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("update exit = %d", code)
			}
			if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
				t.Errorf("update did not drop fuse.lock")
			}
		})
	})

	t.Run("vendor-creates-dir", func(t *testing.T) {
		workIn(t, baseManifest, func(dir string) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"vendor"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("vendor exit = %d", code)
			}
			if _, err := os.Stat(filepath.Join(dir, "vendor", ".fuse-vendor")); err != nil {
				t.Errorf("vendor marker missing: %v", err)
			}
		})
	})

	t.Run("missing-manifest-is-error", func(t *testing.T) {
		// In a directory with no fuse.toml, subcommands fail.
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		dir := t.TempDir()
		os.Chdir(dir)
		var stdout, stderr bytes.Buffer
		if code := run([]string{"add", "x@1.0.0"}, &stdout, &stderr); code != 1 {
			t.Errorf("add with no manifest exit = %d, want 1", code)
		}
	})
}
