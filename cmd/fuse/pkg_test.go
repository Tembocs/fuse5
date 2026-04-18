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

	// TestVendorRecursiveUnpack pins the W24 "fuse vendor recursive
	// unpack" stub retirement. A two-crate project with a path
	// dependency must, after `fuse vendor`, contain the dep's
	// source tree under `vendor/<name>/` — not just the marker file
	// the W23-era implementation wrote.
	t.Run("vendor-unpacks-path-deps", func(t *testing.T) {
		// Build a temp layout:
		//   root/
		//     fuse.toml      (declares mathlib as a path dep)
		//     src/main.fuse
		//   mathlib/
		//     fuse.toml
		//     src/lib.fuse
		outer := t.TempDir()
		rootDir := filepath.Join(outer, "root")
		mathDir := filepath.Join(outer, "mathlib")
		for _, d := range []string{
			filepath.Join(rootDir, "src"),
			filepath.Join(mathDir, "src"),
		} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", d, err)
			}
		}
		rootManifest := `
[package]
name = "root"
version = "0.1.0"

[dependencies]
mathlib = { path = "../mathlib" }
`
		mathManifest := `
[package]
name = "mathlib"
version = "0.1.0"
`
		writes := map[string]string{
			filepath.Join(rootDir, "fuse.toml"):     rootManifest,
			filepath.Join(rootDir, "src", "main.fuse"): `fn main() -> I32 { return 42; }`,
			filepath.Join(mathDir, "fuse.toml"):     mathManifest,
			filepath.Join(mathDir, "src", "lib.fuse"): `pub fn answer() -> I32 { return 42; }`,
		}
		for path, body := range writes {
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
		}

		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		if err := os.Chdir(rootDir); err != nil {
			t.Fatalf("chdir root: %v", err)
		}

		var stdout, stderr bytes.Buffer
		if code := run([]string{"vendor"}, &stdout, &stderr); code != 0 {
			t.Fatalf("vendor exit = %d; stderr=%q", code, stderr.String())
		}

		// Marker file exists.
		if _, err := os.Stat(filepath.Join(rootDir, "vendor", ".fuse-vendor")); err != nil {
			t.Errorf("marker missing: %v", err)
		}
		// Dep's lib.fuse is unpacked.
		unpackedLib := filepath.Join(rootDir, "vendor", "mathlib", "src", "lib.fuse")
		contents, err := os.ReadFile(unpackedLib)
		if err != nil {
			t.Fatalf("expected vendored lib source at %s: %v", unpackedLib, err)
		}
		if !strings.Contains(string(contents), "pub fn answer()") {
			t.Errorf("vendored lib.fuse missing expected content; got:\n%s", contents)
		}
		// Dep's manifest is unpacked (confirms the recursive
		// descent found it for transitive traversal).
		if _, err := os.Stat(filepath.Join(rootDir, "vendor", "mathlib", "fuse.toml")); err != nil {
			t.Errorf("dep manifest not vendored: %v", err)
		}
		// stdout reports the crate count.
		if !strings.Contains(stdout.String(), "1 crate(s)") {
			t.Errorf("stdout should mention crate count; got %q", stdout.String())
		}
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
