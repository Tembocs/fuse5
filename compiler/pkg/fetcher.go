package pkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Fetcher retrieves source-only packages over HTTPS (or file://
// for local path deps) and verifies their SHA-256 integrity
// against the lockfile's recorded digest. Build scripts do NOT
// execute during fetch — Fuse has no pre-install hooks.
type Fetcher struct {
	// CacheDir is the root of the on-disk cache. Fetched
	// artifacts live under CacheDir/<sha256>.
	CacheDir string
	// Client is the HTTP client used for HTTPS fetches. Nil
	// defaults to a timeout-bounded http.Client.
	Client *http.Client
	// Offline, when true, refuses any network access. A miss
	// in the local cache produces a specific diagnostic
	// ("run `fuse build` online once to populate the cache").
	Offline bool
}

// NewFetcher constructs a Fetcher with a default cache path
// under the current user's home (~/.cache/fuse/). Tests pass
// an explicit CacheDir to keep runs hermetic.
func NewFetcher(cacheDir string) *Fetcher {
	return &Fetcher{
		CacheDir: cacheDir,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Fetch retrieves the artifact identified by (URL, expectedSHA)
// and returns a path to the local cache entry. If the cache
// already holds a file with a matching digest, no network
// access is performed.
//
// Integrity is verified before the file reaches the caller:
// a mismatched hash aborts with a specific diagnostic and
// never writes partial content (Rule 6.9 — no silent bad
// state). The caller is safe to unpack the returned path.
func (f *Fetcher) Fetch(url, expectedSHA string) (string, error) {
	if expectedSHA == "" {
		return "", errors.New("fetcher: refusing to fetch without an expected SHA-256")
	}
	if err := os.MkdirAll(f.CacheDir, 0o755); err != nil {
		return "", fmt.Errorf("fetcher: mkdir cache: %w", err)
	}
	cachePath := filepath.Join(f.CacheDir, expectedSHA)
	if existing, err := os.ReadFile(cachePath); err == nil {
		if sha256Hex(existing) == expectedSHA {
			return cachePath, nil
		}
		// Cache-corruption self-heal: remove the mismatched
		// file so the refetch has a clean slot.
		_ = os.Remove(cachePath)
	}

	if f.Offline {
		return "", &OfflineError{URL: url}
	}

	// Source the bytes. Local file:// URLs and HTTP(S) URLs
	// both go through the same digest gate.
	bytes, err := f.read(url)
	if err != nil {
		return "", err
	}
	got := sha256Hex(bytes)
	if got != expectedSHA {
		return "", &IntegrityError{URL: url, Expected: expectedSHA, Got: got}
	}
	// Write atomically via a temp file so a crashed fetch
	// cannot leave a half-written artifact behind.
	tmp, err := os.CreateTemp(f.CacheDir, "fetch-*")
	if err != nil {
		return "", fmt.Errorf("fetcher: tmpfile: %w", err)
	}
	if _, err := tmp.Write(bytes); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("fetcher: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("fetcher: close: %w", err)
	}
	if err := os.Rename(tmp.Name(), cachePath); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("fetcher: rename: %w", err)
	}
	return cachePath, nil
}

// read reads bytes from url — either local file:// or network
// http(s)://. No other schemes are accepted.
func (f *Fetcher) read(url string) ([]byte, error) {
	if strings.HasPrefix(url, "file://") {
		return os.ReadFile(strings.TrimPrefix(url, "file://"))
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		client := f.Client
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("fetcher: GET %s: %w", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetcher: %s returned %d", url, resp.StatusCode)
		}
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, resp.Body); err != nil {
			return nil, fmt.Errorf("fetcher: read body: %w", err)
		}
		return buf.Bytes(), nil
	}
	return nil, fmt.Errorf("fetcher: unsupported URL scheme %q (supported: file, http, https)", url)
}

// IntegrityError reports a SHA-256 mismatch. Satisfies Rule
// 6.17 with a primary URL span, a clear explanation, and a
// suggestion to re-run `fuse update` or inspect the lockfile.
type IntegrityError struct {
	URL      string
	Expected string
	Got      string
}

func (e *IntegrityError) Error() string {
	return fmt.Sprintf("fetcher: integrity check failed for %s\n  expected: sha256=%s\n  got:      sha256=%s\n  hint: the upstream source changed or was tampered with; run `fuse update` to regenerate the lockfile after vetting the new content",
		e.URL, e.Expected, e.Got)
}

// OfflineError reports that offline mode was requested but the
// artifact is not in the local cache.
type OfflineError struct {
	URL string
}

func (e *OfflineError) Error() string {
	return fmt.Sprintf("fetcher: offline mode and %s not cached\n  hint: run `fuse build` once online to populate the local cache, or ship a vendored tree via `fuse vendor`",
		e.URL)
}

// sha256Hex returns the hex-encoded SHA-256 of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
