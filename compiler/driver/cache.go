// W18 incremental driver cache. The W04 HIR work established pass
// fingerprints; W18 persists the pass outputs to disk keyed by
// fingerprint so `fuse check` / `fuse build` reuse a pass's
// output when its input fingerprint has not changed.
//
// Layout:
//
//   .fuse-cache/
//     manifest.json              // version + list of cached entries
//     entries/<hash>              // opaque payload per cached pass
//
// Version mismatch invalidates the whole cache on read, not
// silently — the manifest's Version field is compared on every
// open and an incompatible file triggers a clean slate.

package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CacheVersion identifies the on-disk cache format. Bump this
// whenever the payload shape changes so older caches are
// invalidated rather than silently misread.
const CacheVersion = "fuse-cache-v1"

// Cache is the on-disk pass-output store. Open returns a Cache
// pointed at `.fuse-cache/` under the given root; creating the
// directory when missing and invalidating any version-mismatched
// manifest.
type Cache struct {
	root        string    // absolute path to .fuse-cache directory
	manifest    *Manifest // loaded / freshly-seeded manifest
	dirty       bool      // true when manifest needs flushing
	forceReuse  bool      // tests set this to force cache behaviour
}

// Manifest is the top-level cache index. Stored as JSON under
// manifest.json so it's human-readable at debug time.
type Manifest struct {
	Version string                 `json:"version"`
	Entries map[string]EntryHeader `json:"entries"`
}

// EntryHeader is per-entry metadata. The payload itself lives in
// a separate file under entries/<hash> so the manifest stays
// compact even when payloads are large.
type EntryHeader struct {
	// PassName is the pass that produced this entry
	// ("parse", "resolve", "bridge", "check", etc.).
	PassName string `json:"pass"`
	// InputFingerprint is the content-hash of the pass's input
	// (source text + config). Different inputs → different keys.
	InputFingerprint string `json:"fingerprint"`
	// ByteSize records the payload size for diagnostics.
	ByteSize int64 `json:"bytes"`
	// SchemaVersion lets per-pass payloads version independently.
	SchemaVersion string `json:"schema"`
}

// Open returns a Cache rooted at <rootDir>/.fuse-cache. The
// directory is created on first use. A version mismatch on the
// on-disk manifest produces a fresh cache — corrupt or stale
// caches MUST NOT be trusted silently.
func Open(rootDir string) (*Cache, error) {
	if rootDir == "" {
		return nil, errors.New("driver.Open: empty rootDir")
	}
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve rootDir: %w", err)
	}
	root := filepath.Join(abs, ".fuse-cache")
	if err := os.MkdirAll(filepath.Join(root, "entries"), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir cache: %w", err)
	}
	c := &Cache{root: root}
	manifest, err := loadManifest(root)
	if err != nil || manifest.Version != CacheVersion {
		// Invalidate the entire cache on version mismatch or
		// malformed manifest. Do not silently trust it.
		manifest = &Manifest{Version: CacheVersion, Entries: map[string]EntryHeader{}}
		c.dirty = true
	}
	c.manifest = manifest
	return c, nil
}

// Key computes the fingerprint for a (passName, inputBytes) pair.
// Stable across runs (SHA-256), stable across hosts, dependent
// only on the inputs. Callers compose fingerprints: a downstream
// pass's input fingerprint combines the pass name with its
// upstream fingerprints so a change in lex ripples into parse ∧
// resolve ∧ bridge ∧ ...
func Key(passName string, input []byte) string {
	h := sha256.New()
	h.Write([]byte(passName))
	h.Write([]byte{0})
	h.Write(input)
	return hex.EncodeToString(h.Sum(nil))
}

// Combine produces a deterministic fingerprint from multiple
// upstream fingerprints. Keys compose associatively (sorted
// before hashing) so order doesn't affect the result.
func Combine(keys ...string) string {
	s := append([]string(nil), keys...)
	sort.Strings(s)
	h := sha256.New()
	for _, k := range s {
		h.Write([]byte(k))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Get looks up the payload for key. Returns (payload, true) on a
// cache hit; (nil, false) on a miss. A payload whose header's
// stored fingerprint disagrees with its key is treated as a miss
// so callers never receive a silently-corrupt entry.
func (c *Cache) Get(key string) ([]byte, bool) {
	hdr, ok := c.manifest.Entries[key]
	if !ok {
		return nil, false
	}
	if hdr.InputFingerprint != key {
		// Header-key mismatch: treat as miss + self-heal.
		delete(c.manifest.Entries, key)
		c.dirty = true
		return nil, false
	}
	path := filepath.Join(c.root, "entries", key)
	payload, err := os.ReadFile(path)
	if err != nil {
		delete(c.manifest.Entries, key)
		c.dirty = true
		return nil, false
	}
	return payload, true
}

// Put stores payload under key, recording the pass name and
// schema. The manifest is marked dirty; callers must call Flush
// to persist.
func (c *Cache) Put(key, passName, schemaVersion string, payload []byte) error {
	if key == "" {
		return errors.New("Cache.Put: empty key")
	}
	path := filepath.Join(c.root, "entries", key)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write cache payload: %w", err)
	}
	c.manifest.Entries[key] = EntryHeader{
		PassName:         passName,
		InputFingerprint: key,
		ByteSize:         int64(len(payload)),
		SchemaVersion:    schemaVersion,
	}
	c.dirty = true
	return nil
}

// Flush writes the manifest to disk if it has changed since Open.
func (c *Cache) Flush() error {
	if !c.dirty {
		return nil
	}
	path := filepath.Join(c.root, "manifest.json")
	buf, err := json.MarshalIndent(c.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	c.dirty = false
	return nil
}

// Clear wipes every cache entry. Useful for tests and for the
// `fuse clean` subcommand (future W20).
func (c *Cache) Clear() error {
	dir := filepath.Join(c.root, "entries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
	c.manifest.Entries = map[string]EntryHeader{}
	c.dirty = true
	return c.Flush()
}

// Size returns the count of cache entries. Stable even when the
// manifest hasn't been flushed.
func (c *Cache) Size() int {
	return len(c.manifest.Entries)
}

// loadManifest reads the on-disk manifest if present; returns a
// fresh one if the file is missing. Returns an error only for
// disk failures other than "file not found".
func loadManifest(root string) (*Manifest, error) {
	path := filepath.Join(root, "manifest.json")
	buf, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Manifest{Version: CacheVersion, Entries: map[string]EntryHeader{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(buf, &m); err != nil {
		return &Manifest{Version: "", Entries: map[string]EntryHeader{}}, nil
	}
	if m.Entries == nil {
		m.Entries = map[string]EntryHeader{}
	}
	return &m, nil
}

// IncrementalResult records which passes re-ran for an incremental
// build. Surfaced to callers so tests can assert "nine out of ten
// passes came from cache".
type IncrementalResult struct {
	Hits   []string // keys served from cache
	Misses []string // keys computed fresh
}

// PlanIncremental computes a simple rebuild plan given the input
// fingerprints. Keys whose fingerprints match the cache are
// reported as hits; the rest are misses the caller must recompute.
// A plan is not a recomputation — it's the predicate the driver
// uses to decide which passes to invoke.
func (c *Cache) PlanIncremental(passName string, perItemFingerprints map[string]string) IncrementalResult {
	var res IncrementalResult
	names := make([]string, 0, len(perItemFingerprints))
	for n := range perItemFingerprints {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fp := perItemFingerprints[n]
		key := Key(passName+"/"+n, []byte(fp))
		if _, hit := c.Get(key); hit {
			res.Hits = append(res.Hits, n)
		} else {
			res.Misses = append(res.Misses, n)
		}
	}
	return res
}

// ensure `strings` import used.
var _ = strings.HasPrefix
