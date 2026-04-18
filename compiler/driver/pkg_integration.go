package driver

// W23 package-manager integration. On `fuse build` and
// `fuse check`, the driver looks for a `fuse.toml` in the
// manifest's directory (or the source's directory) and resolves
// its dependencies on first invocation, writing `fuse.lock`.
// Subsequent invocations load from the lockfile without
// re-resolving.
//
// `fuse check` does not fetch — it only reads the already-
// populated lockfile. A check without a lockfile is a user
// warning, not an error; the check path can still complete
// without dependency info at the W23 surface.

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Tembocs/fuse5/compiler/pkg"
)

// ResolveForSource resolves the dependencies declared in the
// `fuse.toml` adjacent to `sourcePath`. Returns the resulting
// lockfile, or nil when no manifest is present.
//
// Policy:
//   - If fuse.lock exists and its Root matches the manifest's
//     package identity, load it as-is (no re-resolve).
//   - Otherwise, run the resolver against an empty registry
//     (W23 scope: path-dependencies only — registry integration
//     lands when the reference registry comes online).
//   - Write the lockfile back to disk.
func ResolveForSource(sourcePath string, offline bool) (*pkg.Lockfile, error) {
	manifestPath, err := findManifest(sourcePath)
	if err != nil {
		return nil, nil // no manifest is not an error
	}
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", manifestPath, err)
	}
	m, err := pkg.ParseManifest(body)
	if err != nil {
		return nil, err
	}
	if m.Package == nil {
		// Workspace root only; no crate-level resolution at
		// this path. W24 workspace wave extends.
		return nil, nil
	}
	lockPath := filepath.Join(filepath.Dir(manifestPath), "fuse.lock")
	if body, err := os.ReadFile(lockPath); err == nil {
		lk, err := pkg.ParseLockfile(body)
		if err == nil && lk.Root == m.IDString() {
			return lk, nil
		}
		// Stale lockfile — re-resolve.
	}

	if offline {
		return nil, fmt.Errorf("fuse check/build --offline: no cached lockfile at %s", lockPath)
	}

	// Resolve against a path-only registry (empty index;
	// path deps are already satisfied by the manifest's
	// local tree). Missing manifest for a path dep is a
	// resolver error per the Rule 6.17 unknown-name
	// diagnostic.
	emptyIdx := &emptyRegistry{}
	r := &pkg.Resolver{Index: emptyIdx}
	res, rerr := r.Resolve(m)
	manifestDir := filepath.Dir(manifestPath)
	if rerr != nil {
		// At W23 the resolver errors only when the manifest
		// declares a registry dep that the empty index cannot
		// satisfy. Path-only projects resolve successfully to
		// an empty entry list.
		if rerr.Kind == "unknown" && hasOnlyPathDeps(m) {
			lk := pkg.NewLockfile(m.IDString())
			appendPathDeps(lk, m, manifestDir)
			lk.Finalize()
			_ = writeLockfile(lockPath, lk)
			return lk, nil
		}
		return nil, fmt.Errorf("resolve: %s", rerr.Error())
	}
	// The resolver ignores path-dependencies (they bypass the
	// version index). Augment its Resolution with one LockedCrate
	// per declared path dep so the lockfile records every crate the
	// build will read — not just the registry ones.
	lk := res.ToLockfile()
	appendPathDeps(lk, m, manifestDir)
	lk.Finalize()
	if err := writeLockfile(lockPath, lk); err != nil {
		return nil, err
	}
	return lk, nil
}

// appendPathDeps adds one LockedCrate per declared path-dependency
// to `lk`. The version is taken from the dep crate's own fuse.toml
// when that file is readable; otherwise a "0.0.0-path" sentinel is
// recorded so downstream consumers can still distinguish a path dep
// from a registry one. The Source column stores the **absolute**
// path so the lockfile stays usable from any working directory.
func appendPathDeps(lk *pkg.Lockfile, m *pkg.Manifest, manifestDir string) {
	for _, d := range m.Dependencies {
		if d.Path == "" {
			continue
		}
		absPath := d.Path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Clean(filepath.Join(manifestDir, d.Path))
		}
		version := "0.0.0-path"
		if body, err := os.ReadFile(filepath.Join(absPath, "fuse.toml")); err == nil {
			if depM, err := pkg.ParseManifest(body); err == nil && depM.Package != nil && depM.Package.Version != "" {
				version = depM.Package.Version
			}
		}
		lk.AddCrate(pkg.LockedCrate{
			Name:    d.Name,
			Version: version,
			Source:  absPath,
		})
	}
}

// findManifest walks up from sourcePath until it finds a
// directory containing fuse.toml. Returns the absolute path to
// the manifest or an error when none is found.
func findManifest(sourcePath string) (string, error) {
	dir, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		candidate := filepath.Join(dir, "fuse.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no fuse.toml found walking up from %s", sourcePath)
		}
		dir = parent
	}
}

// writeLockfile atomically writes lk to path.
func writeLockfile(path string, lk *pkg.Lockfile) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "fuse-lock-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(lk.Serialize()); err != nil {
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

// hasOnlyPathDeps reports whether every declared dep is a
// path-dependency.
func hasOnlyPathDeps(m *pkg.Manifest) bool {
	for _, d := range m.Dependencies {
		if d.Path == "" {
			return false
		}
	}
	return true
}

// emptyRegistry is a RegistryLookup that knows about nothing —
// the W23 driver's fallback when a project has no registry
// dependencies (only path deps) or the index hasn't been
// fetched.
type emptyRegistry struct{}

func (e *emptyRegistry) AvailableVersions(name string) []pkg.Version { return nil }
func (e *emptyRegistry) Dependencies(name string, v pkg.Version) []pkg.DepConstraint {
	return nil
}
func (e *emptyRegistry) Source(name string, v pkg.Version) string { return "" }
func (e *emptyRegistry) SHA256(name string, v pkg.Version) string { return "" }
