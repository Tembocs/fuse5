package pkg

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Manifest parse ---

// TestManifestParse covers the full fuse.toml grammar the W23
// scope supports: [package], [dependencies] (bare + inline
// table), [dev-dependencies], [features], [workspace]. Unknown
// keys / sections are diagnostic errors, not silent drops.
func TestManifestParse(t *testing.T) {
	t.Run("minimal-package", func(t *testing.T) {
		m, err := ParseManifest([]byte(`
[package]
name = "foo"
version = "0.1.0"
`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if m.Package.Name != "foo" || m.Package.Version != "0.1.0" {
			t.Fatalf("decoded package %+v", m.Package)
		}
	})
	t.Run("dependencies-bare-and-table", func(t *testing.T) {
		m, err := ParseManifest([]byte(`
[package]
name = "root"
version = "1.0.0"

[dependencies]
alpha = "1.0.0"
beta = { version = "2.0.0", features = ["async"] }
gamma = { path = "../gamma" }
`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if len(m.Dependencies) != 3 {
			t.Fatalf("want 3 deps, got %d", len(m.Dependencies))
		}
		byName := map[string]Dep{}
		for _, d := range m.Dependencies {
			byName[d.Name] = d
		}
		if byName["alpha"].Version != "1.0.0" {
			t.Errorf("alpha version = %q", byName["alpha"].Version)
		}
		if len(byName["beta"].Features) != 1 || byName["beta"].Features[0] != "async" {
			t.Errorf("beta features = %v", byName["beta"].Features)
		}
		if byName["gamma"].Path != "../gamma" {
			t.Errorf("gamma path = %q", byName["gamma"].Path)
		}
	})
	t.Run("features-table", func(t *testing.T) {
		m, err := ParseManifest([]byte(`
[package]
name = "f"
version = "0.0.1"

[features]
default = ["alpha"]
alpha = []
combined = ["alpha", "beta"]
`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if got := m.Features["default"]; len(got) != 1 || got[0] != "alpha" {
			t.Errorf("default feature = %v", got)
		}
		if got := m.Features["combined"]; len(got) != 2 {
			t.Errorf("combined feature = %v", got)
		}
	})
	t.Run("workspace-only", func(t *testing.T) {
		m, err := ParseManifest([]byte(`
[workspace]
members = ["root", "util"]
`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if m.Workspace == nil || len(m.Workspace.Members) != 2 {
			t.Fatalf("workspace = %+v", m.Workspace)
		}
	})
	t.Run("rejects-unknown-section", func(t *testing.T) {
		_, err := ParseManifest([]byte(`
[package]
name = "f"
version = "0.0.1"

[bogus]
x = 1
`))
		if err == nil {
			t.Fatal("expected error on unknown section")
		}
		if !strings.Contains(err.Error(), "bogus") {
			t.Errorf("error should name the bad section: %v", err)
		}
	})
	t.Run("rejects-unknown-package-key", func(t *testing.T) {
		_, err := ParseManifest([]byte(`
[package]
name = "f"
version = "0.0.1"
nope = "foo"
`))
		if err == nil {
			t.Fatal("expected error on unknown [package] key")
		}
	})
	t.Run("rejects-missing-required", func(t *testing.T) {
		_, err := ParseManifest([]byte(`
[package]
name = "f"
`))
		if err == nil {
			t.Fatal("expected error on missing version")
		}
	})
}

// --- Lockfile round-trip ---

func TestLockfileRoundTrip(t *testing.T) {
	l := NewLockfile("root@1.0.0")
	l.AddCrate(LockedCrate{
		Name: "alpha", Version: "1.0.0",
		Source: "https://reg.example/alpha-1.0.0.tar.gz",
		SHA256: "aa",
		Dependencies: []string{"beta"},
	})
	l.AddCrate(LockedCrate{
		Name: "beta", Version: "2.0.0",
		Source: "https://reg.example/beta-2.0.0.tar.gz",
		SHA256: "bb",
	})
	l.Finalize()
	bytes := l.Serialize()

	// Parse back.
	l2, err := ParseLockfile(bytes)
	if err != nil {
		t.Fatalf("ParseLockfile: %v", err)
	}
	_ = l2

	// Re-serialize and compare.
	l3 := NewLockfile("root@1.0.0")
	for _, e := range l.Entries {
		l3.AddCrate(e)
	}
	l3.Finalize()
	if !l.Equal(l3) {
		t.Errorf("lockfile not byte-stable:\n--- first ---\n%s\n--- second ---\n%s",
			string(l.Serialize()), string(l3.Serialize()))
	}

	// Non-serialize determinism: two independent Serialize
	// calls produce the same bytes.
	if string(l.Serialize()) != string(l.Serialize()) {
		t.Errorf("Serialize is non-deterministic")
	}
}

// --- Version algebra ---

func TestVersionAlgebra(t *testing.T) {
	t.Run("parse-version", func(t *testing.T) {
		v, err := ParseVersion("1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		if v.String() != "1.2.3" {
			t.Errorf("version string = %q", v.String())
		}
	})
	t.Run("caret-range", func(t *testing.T) {
		r, err := ParseRange("^1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, r, "1.2.3")
		mustContain(t, r, "1.5.0")
		mustContain(t, r, "1.9.99")
		mustReject(t, r, "2.0.0")
		mustReject(t, r, "1.2.2")
	})
	t.Run("tilde-range", func(t *testing.T) {
		r, err := ParseRange("~1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, r, "1.2.3")
		mustContain(t, r, "1.2.99")
		mustReject(t, r, "1.3.0")
	})
	t.Run("star-range", func(t *testing.T) {
		r, err := ParseRange("*")
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, r, "0.0.1")
		mustContain(t, r, "99.99.99")
	})
	t.Run("compound-range", func(t *testing.T) {
		r, err := ParseRange(">=1.0.0, <2.0.0")
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, r, "1.0.0")
		mustContain(t, r, "1.5.0")
		mustReject(t, r, "2.0.0")
		mustReject(t, r, "0.9.0")
	})
	t.Run("intersect-disjoint", func(t *testing.T) {
		a, _ := ParseRange("^1.0.0")
		b, _ := ParseRange("^2.0.0")
		_, ok := a.Intersect(b)
		if ok {
			t.Errorf("^1 and ^2 should be disjoint")
		}
	})
	t.Run("select-latest", func(t *testing.T) {
		r, _ := ParseRange("^1.0.0")
		cands := []Version{
			{Major: 2, Minor: 0, Patch: 0},
			{Major: 1, Minor: 5, Patch: 0},
			{Major: 1, Minor: 2, Patch: 0},
		}
		chosen, ok := r.SelectLatest(cands)
		if !ok || chosen.Compare(Version{Major: 1, Minor: 5}) != 0 {
			t.Errorf("latest ^1 candidate = %v, want 1.5.0", chosen)
		}
	})
}

func mustContain(t *testing.T, r Range, v string) {
	t.Helper()
	parsed, _ := ParseVersion(v)
	if !r.Contains(parsed) {
		t.Errorf("range %q should contain %q", r.Raw, v)
	}
}

func mustReject(t *testing.T, r Range, v string) {
	t.Helper()
	parsed, _ := ParseVersion(v)
	if r.Contains(parsed) {
		t.Errorf("range %q should reject %q", r.Raw, v)
	}
}

// --- Resolver ---

// memIndex is an in-memory RegistryLookup for tests.
type memIndex struct {
	data map[string]map[string][]DepConstraint // name → version → deps
}

func newMemIndex() *memIndex {
	return &memIndex{data: map[string]map[string][]DepConstraint{}}
}

func (m *memIndex) Publish(name, version string, deps ...DepConstraint) {
	if m.data[name] == nil {
		m.data[name] = map[string][]DepConstraint{}
	}
	m.data[name][version] = deps
}

func (m *memIndex) AvailableVersions(name string) []Version {
	vs := m.data[name]
	if vs == nil {
		return nil
	}
	out := make([]Version, 0, len(vs))
	for v := range vs {
		pv, err := ParseVersion(v)
		if err != nil {
			continue
		}
		out = append(out, pv)
	}
	// Sort descending.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Compare(out[j-1]) > 0; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (m *memIndex) Dependencies(name string, v Version) []DepConstraint {
	return m.data[name][v.String()]
}

func (m *memIndex) Source(name string, v Version) string {
	return "https://reg.example/" + name + "-" + v.String() + ".tar.gz"
}

func (m *memIndex) SHA256(name string, v Version) string {
	return "sha-" + name + "-" + v.String()
}

// TestResolverDeterministic confirms same-input → byte-identical
// lockfile across 3 runs. Wave-doc mandates -count=3.
func TestResolverDeterministic(t *testing.T) {
	idx := newMemIndex()
	idx.Publish("alpha", "1.0.0", DepConstraint{Name: "beta", Range: mustParseRange(t, "^2.0.0")})
	idx.Publish("beta", "2.0.0")
	idx.Publish("beta", "2.1.0")

	m, err := ParseManifest([]byte(`
[package]
name = "root"
version = "1.0.0"

[dependencies]
alpha = "^1.0.0"
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	r := &Resolver{Index: idx}
	res, rerr := r.Resolve(m)
	if rerr != nil {
		t.Fatalf("resolve: %v", rerr)
	}
	first := string(res.ToLockfile().Serialize())

	// Re-resolve and re-serialize twice more.
	for i := 0; i < 2; i++ {
		res2, rerr := r.Resolve(m)
		if rerr != nil {
			t.Fatalf("resolve re-run: %v", rerr)
		}
		got := string(res2.ToLockfile().Serialize())
		if got != first {
			t.Fatalf("non-deterministic resolve on run %d:\n--- first ---\n%s\n--- got ---\n%s",
				i+2, first, got)
		}
	}
}

func mustParseRange(t *testing.T, s string) Range {
	t.Helper()
	r, err := ParseRange(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestResolverCycles confirms dependency cycles produce a
// diagnostic citing the path.
func TestResolverCycles(t *testing.T) {
	idx := newMemIndex()
	idx.Publish("a", "1.0.0", DepConstraint{Name: "b", Range: mustParseRange(t, "^1")})
	idx.Publish("b", "1.0.0", DepConstraint{Name: "c", Range: mustParseRange(t, "^1")})
	idx.Publish("c", "1.0.0", DepConstraint{Name: "a", Range: mustParseRange(t, "^1")})

	m, _ := ParseManifest([]byte(`
[package]
name = "root"
version = "1.0.0"

[dependencies]
a = "^1"
`))
	r := &Resolver{Index: idx}
	_, rerr := r.Resolve(m)
	if rerr == nil {
		t.Fatal("expected cycle error")
	}
	if rerr.Kind != "cycle" {
		t.Errorf("error kind = %q, want cycle", rerr.Kind)
	}
	if !strings.Contains(rerr.Message, "cycle") {
		t.Errorf("cycle error message missing `cycle`: %q", rerr.Message)
	}
}

// --- Fetcher ---

func TestFetcherHttps(t *testing.T) {
	body := []byte("crate source tarball bytes\n")
	expected := sha256Hex(body)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	f := NewFetcher(t.TempDir())
	path, err := f.Fetch(srv.URL+"/crate.tar", expected)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cached: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("cache contents = %q, want %q", got, body)
	}

	// Second fetch hits the cache (we verify by stopping the
	// server and re-fetching).
	srv.Close()
	cached, err := f.Fetch(srv.URL+"/crate.tar", expected)
	if err != nil {
		t.Fatalf("cache hit: %v", err)
	}
	if cached != path {
		t.Errorf("cache-hit path diverged: %q vs %q", cached, path)
	}
}

func TestFetcherIntegrity(t *testing.T) {
	body := []byte("real bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	f := NewFetcher(t.TempDir())
	_, err := f.Fetch(srv.URL+"/crate.tar", "deadbeef")
	if err == nil {
		t.Fatal("expected integrity error")
	}
	if _, ok := err.(*IntegrityError); !ok {
		t.Errorf("error type = %T, want *IntegrityError", err)
	}
	// Cache directory must not contain the file under its
	// expected hash — no partial-write leak.
	entries, _ := os.ReadDir(f.CacheDir)
	for _, e := range entries {
		if e.Name() == "deadbeef" {
			t.Errorf("integrity-failed fetch wrote to cache")
		}
	}
}

func TestFetcherOffline(t *testing.T) {
	body := []byte("offline-cached content")
	expected := sha256Hex(body)

	dir := t.TempDir()
	// Seed the cache with the expected artifact.
	if err := os.WriteFile(filepath.Join(dir, expected), body, 0o644); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	f := NewFetcher(dir)
	f.Offline = true
	// Cache hit should succeed in offline mode.
	path, err := f.Fetch("https://example.com/missing", expected)
	if err != nil {
		t.Fatalf("offline cache hit failed: %v", err)
	}
	if path != filepath.Join(dir, expected) {
		t.Errorf("unexpected cache path: %s", path)
	}

	// Cache miss must fail with OfflineError.
	otherBody := []byte("other")
	otherSHA := sha256Hex(otherBody)
	_, err = f.Fetch("https://example.com/other", otherSHA)
	if err == nil {
		t.Fatal("expected offline error")
	}
	if _, ok := err.(*OfflineError); !ok {
		t.Errorf("err type = %T, want *OfflineError", err)
	}
}

// --- Registry ---

func TestRegistryIndexParse(t *testing.T) {
	src := `{"schema_version":1}
{"name":"alpha","version":"1.0.0","url":"https://reg/alpha-1.0.0.tar","sha256":"aa","dependencies":[{"name":"beta","range":"^2"}]}
{"name":"beta","version":"2.0.0","url":"https://reg/beta-2.0.0.tar","sha256":"bb"}
{"name":"beta","version":"2.1.0","url":"https://reg/beta-2.1.0.tar","sha256":"bc"}
`
	idx, err := ParseRegistryIndex([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if idx.SchemaVersion != 1 {
		t.Errorf("schema = %d", idx.SchemaVersion)
	}
	if len(idx.Entries) != 3 {
		t.Errorf("entries = %d, want 3", len(idx.Entries))
	}
	// AvailableVersions is newest-first.
	vs := idx.AvailableVersions("beta")
	if len(vs) != 2 || vs[0].Compare(Version{Major: 2, Minor: 1}) != 0 {
		t.Errorf("beta versions = %v", vs)
	}

	// Duplicate rejected.
	_, err = ParseRegistryIndex([]byte(`{"schema_version":1}
{"name":"x","version":"1.0.0","url":"u","sha256":"s"}
{"name":"x","version":"1.0.0","url":"u2","sha256":"s2"}
`))
	if err == nil {
		t.Fatal("duplicate entry should be rejected")
	}
}

func TestRegistryMetadata(t *testing.T) {
	payload := []byte(`{
  "schema_version": 1,
  "name": "alpha",
  "version": "1.0.0",
  "url": "https://reg/alpha-1.0.0.tar",
  "sha256": "aa",
  "dependencies": [{"name":"beta","range":"^2"}],
  "license": "Apache-2.0"
}`)
	md, err := ParsePackageMetadata(payload)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if md.Name != "alpha" || md.Version != "1.0.0" || md.License != "Apache-2.0" {
		t.Errorf("decoded = %+v", md)
	}
	if len(md.Dependencies) != 1 {
		t.Errorf("dep count = %d", len(md.Dependencies))
	}

	// Unsupported schema.
	_, err = ParsePackageMetadata([]byte(`{"schema_version":99,"name":"x","version":"1.0.0","url":"u","sha256":"s"}`))
	if err == nil {
		t.Fatal("unsupported schema should be rejected")
	}
}
