package pkg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// RegistryIndex is the W23 on-disk registry index: a line-
// oriented file where each line is a JSON object describing
// one (name, version, url, sha256, dependencies) tuple.
//
// This format is deterministic and line-oriented so a registry
// can append to it incrementally, and a consumer can grep /
// awk it without a TOML dependency.
//
// See docs/registry-protocol.md for the full specification.
type RegistryIndex struct {
	// SchemaVersion identifies the on-the-wire format version.
	SchemaVersion int
	// Entries maps "name@version" → IndexEntry for O(1)
	// lookup. Deterministic iteration uses the sorted names
	// slice in callers.
	Entries map[string]*IndexEntry
}

// IndexEntry is one published (name, version) tuple's
// registry record.
type IndexEntry struct {
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	URL          string          `json:"url"`
	SHA256       string          `json:"sha256"`
	Dependencies []IndexDep      `json:"dependencies,omitempty"`
}

// IndexDep is a single dependency edge in an IndexEntry.
type IndexDep struct {
	Name  string `json:"name"`
	Range string `json:"range"`
}

// RegistrySchemaVersion pins the on-wire format.
const RegistrySchemaVersion = 1

// ParseRegistryIndex reads a line-oriented index file. The
// first line must be `{"schema_version":N}`; subsequent lines
// are IndexEntry JSON objects. Blank lines and `#` comments
// are tolerated.
func ParseRegistryIndex(src []byte) (*RegistryIndex, error) {
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	idx := &RegistryIndex{Entries: map[string]*IndexEntry{}}
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx.SchemaVersion == 0 {
			var header struct {
				SchemaVersion int `json:"schema_version"`
			}
			if err := json.Unmarshal([]byte(line), &header); err != nil {
				return nil, fmt.Errorf("registry-index line %d: %w", lineNum, err)
			}
			if header.SchemaVersion == 0 {
				return nil, fmt.Errorf("registry-index line %d: first non-empty line must be `{\"schema_version\":N}`", lineNum)
			}
			idx.SchemaVersion = header.SchemaVersion
			if idx.SchemaVersion != RegistrySchemaVersion {
				return nil, fmt.Errorf("registry-index unsupported schema_version %d (expected %d)", idx.SchemaVersion, RegistrySchemaVersion)
			}
			continue
		}
		var entry IndexEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("registry-index line %d: %w", lineNum, err)
		}
		if entry.Name == "" || entry.Version == "" {
			return nil, fmt.Errorf("registry-index line %d: name and version required", lineNum)
		}
		key := entry.Name + "@" + entry.Version
		if _, dup := idx.Entries[key]; dup {
			return nil, fmt.Errorf("registry-index line %d: duplicate entry %q", lineNum, key)
		}
		idx.Entries[key] = &entry
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("registry-index scan: %w", err)
	}
	if idx.SchemaVersion == 0 {
		return nil, errors.New("registry-index: empty or missing schema_version header")
	}
	return idx, nil
}

// AvailableVersions returns every published version of `name`
// newest-first (for the resolver). Implements RegistryLookup.
func (r *RegistryIndex) AvailableVersions(name string) []Version {
	var out []Version
	for _, entry := range r.Entries {
		if entry.Name != name {
			continue
		}
		v, err := ParseVersion(entry.Version)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	// Sort newest-first so SelectLatest iterates in the right
	// order. We sort descending by Compare.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Compare(out[j-1]) > 0; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Dependencies returns the dep edges for (name, version).
func (r *RegistryIndex) Dependencies(name string, version Version) []DepConstraint {
	entry, ok := r.Entries[name+"@"+version.String()]
	if !ok {
		return nil
	}
	out := make([]DepConstraint, 0, len(entry.Dependencies))
	for _, d := range entry.Dependencies {
		rng, err := ParseRange(d.Range)
		if err != nil {
			continue
		}
		out = append(out, DepConstraint{Name: d.Name, Range: rng})
	}
	return out
}

// Source returns the URL for (name, version).
func (r *RegistryIndex) Source(name string, version Version) string {
	entry, ok := r.Entries[name+"@"+version.String()]
	if !ok {
		return ""
	}
	return entry.URL
}

// SHA256 returns the digest for (name, version).
func (r *RegistryIndex) SHA256(name string, version Version) string {
	entry, ok := r.Entries[name+"@"+version.String()]
	if !ok {
		return ""
	}
	return entry.SHA256
}

// PackageMetadata is the JSON payload returned by a registry
// for a (name, version) GET. Schema version is explicit in
// the payload for wire-format stability.
type PackageMetadata struct {
	SchemaVersion int        `json:"schema_version"`
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	Description   string     `json:"description,omitempty"`
	URL           string     `json:"url"`
	SHA256        string     `json:"sha256"`
	Dependencies  []IndexDep `json:"dependencies,omitempty"`
	License       string     `json:"license,omitempty"`
	PublishedAt   int64      `json:"published_at,omitempty"`
}

// ParsePackageMetadata decodes a metadata JSON payload.
// Unknown schema versions are rejected.
func ParsePackageMetadata(src []byte) (*PackageMetadata, error) {
	var out PackageMetadata
	if err := json.Unmarshal(src, &out); err != nil {
		return nil, fmt.Errorf("package-metadata: %w", err)
	}
	if out.SchemaVersion != RegistrySchemaVersion {
		return nil, fmt.Errorf("package-metadata: unsupported schema_version %d", out.SchemaVersion)
	}
	if out.Name == "" || out.Version == "" {
		return nil, errors.New("package-metadata: name and version are required")
	}
	return &out, nil
}
