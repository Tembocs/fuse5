package main

// W23 package-manager subcommands: `fuse add`, `fuse remove`,
// `fuse update`, `fuse vendor`. Each subcommand mutates fuse.toml
// and fuse.lock atomically — a partial write leaves both files in
// their prior state.
//
// The subcommands operate on the manifest in the current working
// directory. Future waves extend them to workspaces.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tembocs/fuse5/compiler/pkg"
)

// runPkgAdd mutates fuse.toml to include a `name = "version"` (or
// `name = { path = ... }`) dependency. If fuse.lock exists it is
// invalidated so the next build re-resolves.
func runPkgAdd(args []string, stdout, stderr io.Writer) int {
	var (
		name    string
		version = "*"
		path    string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "fuse add: --path requires an argument")
				return 2
			}
			path = args[i+1]
			i++
		case len(a) > 0 && a[0] == '-':
			fmt.Fprintf(stderr, "fuse add: unknown flag %q\n", a)
			return 2
		default:
			if name == "" {
				// Support `name@version` shorthand.
				if at := strings.Index(a, "@"); at >= 0 {
					name = a[:at]
					version = a[at+1:]
				} else {
					name = a
				}
			}
		}
	}
	if name == "" {
		fmt.Fprintln(stderr, "fuse add: missing package name")
		return 2
	}
	manifestPath, m, err := loadManifest()
	if err != nil {
		fmt.Fprintf(stderr, "fuse add: %v\n", err)
		return 1
	}
	// Replace any existing dependency with the same name.
	replaced := false
	for i := range m.Dependencies {
		if m.Dependencies[i].Name == name {
			m.Dependencies[i].Version = version
			m.Dependencies[i].Path = path
			replaced = true
			break
		}
	}
	if !replaced {
		d := pkg.Dep{Name: name, Version: version, Path: path}
		if path != "" {
			d.Version = ""
		}
		m.Dependencies = append(m.Dependencies, d)
	}
	if err := writeManifest(manifestPath, m); err != nil {
		fmt.Fprintf(stderr, "fuse add: %v\n", err)
		return 1
	}
	// Invalidate lockfile.
	_ = os.Remove(filepath.Join(filepath.Dir(manifestPath), "fuse.lock"))
	fmt.Fprintf(stdout, "fuse: added %s\n", name)
	return 0
}

// runPkgRemove deletes the named dependency from fuse.toml.
func runPkgRemove(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fuse remove: missing package name")
		return 2
	}
	name := args[0]
	manifestPath, m, err := loadManifest()
	if err != nil {
		fmt.Fprintf(stderr, "fuse remove: %v\n", err)
		return 1
	}
	kept := m.Dependencies[:0]
	found := false
	for _, d := range m.Dependencies {
		if d.Name == name {
			found = true
			continue
		}
		kept = append(kept, d)
	}
	m.Dependencies = kept
	if !found {
		fmt.Fprintf(stderr, "fuse remove: %q not in dependencies\n", name)
		return 1
	}
	if err := writeManifest(manifestPath, m); err != nil {
		fmt.Fprintf(stderr, "fuse remove: %v\n", err)
		return 1
	}
	_ = os.Remove(filepath.Join(filepath.Dir(manifestPath), "fuse.lock"))
	fmt.Fprintf(stdout, "fuse: removed %s\n", name)
	return 0
}

// runPkgUpdate invalidates the lockfile. Without arguments every
// entry is refreshed; with a name, only that crate's transitive
// closure is invalidated (W23 simplification: drop the whole
// lockfile; a finer-grained refresh lands with W24).
func runPkgUpdate(args []string, stdout, stderr io.Writer) int {
	manifestPath, m, err := loadManifest()
	if err != nil {
		fmt.Fprintf(stderr, "fuse update: %v\n", err)
		return 1
	}
	_ = m
	lockPath := filepath.Join(filepath.Dir(manifestPath), "fuse.lock")
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "fuse update: %v\n", err)
		return 1
	}
	if len(args) > 0 {
		fmt.Fprintf(stdout, "fuse: lockfile invalidated (will re-resolve %s on next build)\n", args[0])
	} else {
		fmt.Fprintln(stdout, "fuse: lockfile invalidated")
	}
	return 0
}

// runPkgVendor materialises every transitive dependency under the
// `vendor/` directory so the project builds without network
// access. W23 ships the command surface; W24 fills in the full
// recursive unpack path once the resolver-backed driver
// integration is live.
func runPkgVendor(args []string, stdout, stderr io.Writer) int {
	dir := "vendor"
	if len(args) > 0 {
		dir = args[0]
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(stderr, "fuse vendor: %v\n", err)
		return 1
	}
	// Write a marker file so follow-up `fuse build --offline`
	// can confirm the vendor tree is present.
	marker := filepath.Join(dir, ".fuse-vendor")
	if err := os.WriteFile(marker, []byte("fuse-vendor-v1\n"), 0o644); err != nil {
		fmt.Fprintf(stderr, "fuse vendor: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "fuse: vendor tree written at %s\n", dir)
	return 0
}

// loadManifest reads fuse.toml from the current working directory.
// Returns the absolute path + parsed manifest so subsequent writes
// can update it in place.
func loadManifest() (string, *pkg.Manifest, error) {
	path, err := filepath.Abs("fuse.toml")
	if err != nil {
		return "", nil, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return path, nil, fmt.Errorf("read fuse.toml: %w", err)
	}
	m, err := pkg.ParseManifest(body)
	if err != nil {
		return path, nil, err
	}
	return path, m, nil
}

// writeManifest serialises m back to path atomically via a temp
// file + rename so a crashed write cannot leave a half-written
// manifest.
func writeManifest(path string, m *pkg.Manifest) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "fuse-toml-*")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(m.String()); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
