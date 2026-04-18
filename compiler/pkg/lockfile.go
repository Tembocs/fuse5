package pkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Lockfile is the decoded form of `fuse.lock`. It records the
// resolved version, source URL, and SHA-256 of every transitive
// dependency. Byte-stable across machines and runs (Rule 7.1).
type Lockfile struct {
	// SchemaVersion identifies the lockfile format. Bumped when
	// the on-disk shape changes; parsers reject unknown values.
	SchemaVersion int
	// Root is the `<name>@<version>` identifier of the root
	// crate this lockfile was produced for. Downstream
	// consumers compare against Manifest.IDString to detect a
	// stale lockfile.
	Root string
	// Entries lists every resolved crate, sorted
	// lexicographically by `Name@Version`.
	Entries []LockedCrate
	// Digest is the SHA-256 of the canonical serialization of
	// this lockfile excluding the Digest field itself.
	// Downstream consumers can verify integrity without
	// re-resolving.
	Digest string
}

// LockedCrate is one resolved dependency entry.
type LockedCrate struct {
	Name         string
	Version      string
	Source       string   // path, URL, or registry reference
	SHA256       string   // hex-encoded digest of the source tarball
	Dependencies []string // names of transitive dependencies, sorted
}

// CurrentLockfileSchema is the format version this package
// emits. Readers invalidate lockfiles with any other version.
const CurrentLockfileSchema = 1

// NewLockfile returns an empty lockfile keyed to the given root.
func NewLockfile(root string) *Lockfile {
	return &Lockfile{SchemaVersion: CurrentLockfileSchema, Root: root}
}

// AddCrate inserts a locked crate. Callers must add every
// transitive dependency before calling Finalize; the add order
// is irrelevant because Finalize sorts by key.
func (l *Lockfile) AddCrate(c LockedCrate) {
	// Keep transitive dep lists sorted per-entry.
	sort.Strings(c.Dependencies)
	l.Entries = append(l.Entries, c)
}

// Finalize sorts entries deterministically and computes the
// top-level digest. Must be called before Serialize.
func (l *Lockfile) Finalize() {
	sort.Slice(l.Entries, func(i, j int) bool {
		if l.Entries[i].Name != l.Entries[j].Name {
			return l.Entries[i].Name < l.Entries[j].Name
		}
		return l.Entries[i].Version < l.Entries[j].Version
	})
	body := serializeWithoutDigest(l)
	sum := sha256.Sum256([]byte(body))
	l.Digest = hex.EncodeToString(sum[:])
}

// Serialize returns the canonical on-disk representation. Rule
// 7.1 — same input, same bytes. The digest line is appended
// under the [root] section so the parser picks it up.
func (l *Lockfile) Serialize() []byte {
	body := serializeWithoutDigest(l)
	// Insert `digest = "..."` as the last line under [root],
	// before the first [crate]. Simpler: append to the end of
	// the body; ParseLockfile's applyRootKey handles digest as
	// a root-level key regardless of position.
	return []byte(body + fmt.Sprintf("\n[digest]\nvalue = %q\n", l.Digest))
}

// serializeWithoutDigest produces the digest-excluded canonical
// body. The digest line is appended separately by Serialize.
// Top-level keys live under a `[root]` section so the TOML
// parser can route them through a named section (our parser
// requires every key-value to belong to one).
func serializeWithoutDigest(l *Lockfile) string {
	var sb strings.Builder
	sb.WriteString("[root]\n")
	fmt.Fprintf(&sb, "schema_version = %d\n", l.SchemaVersion)
	fmt.Fprintf(&sb, "root = %q\n", l.Root)
	for _, c := range l.Entries {
		sb.WriteString("\n[crate]\n")
		fmt.Fprintf(&sb, "name = %q\n", c.Name)
		fmt.Fprintf(&sb, "version = %q\n", c.Version)
		fmt.Fprintf(&sb, "source = %q\n", c.Source)
		fmt.Fprintf(&sb, "sha256 = %q\n", c.SHA256)
		if len(c.Dependencies) > 0 {
			sb.WriteString("dependencies = [")
			for i, d := range c.Dependencies {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(&sb, "%q", d)
			}
			sb.WriteString("]\n")
		}
	}
	return sb.String()
}

// ParseLockfile reads a `fuse.lock` file. Unknown schema
// versions are invalidated (the caller re-resolves) rather than
// silently accepted.
func ParseLockfile(src []byte) (*Lockfile, error) {
	doc, err := parseToml(src)
	if err != nil {
		return nil, err
	}
	lk := &Lockfile{}
	// Implicit "root" section comes first — schema_version and
	// root live there. We treat each section by name; the
	// crate array is a list of sections named `crate`.
	for _, section := range doc.sections {
		switch section.name {
		case "root":
			// Not emitted by Serialize (we put root at top-level
			// without a [section] header) but tolerated if an
			// alternate toolchain produces it.
			for _, kv := range section.entries {
				if err := applyRootKey(lk, kv); err != nil {
					return nil, err
				}
			}
		case "crate":
			c, err := decodeLockedCrate(section)
			if err != nil {
				return nil, err
			}
			lk.Entries = append(lk.Entries, c)
		case "digest":
			// Single [digest] section with a `value = "..."`
			// key. Parse the top-level key.
			for _, kv := range section.entries {
				if kv.key == "value" {
					s, err := decodeString(kv.value)
					if err != nil {
						return nil, fmt.Errorf("fuse.lock: [digest].value: %w", err)
					}
					lk.Digest = s
				}
			}
		default:
			return nil, fmt.Errorf("fuse.lock: unknown section [%s]", section.name)
		}
	}
	// If the parser didn't handle top-level keys (our Serialize
	// puts schema_version / root / digest at the top level),
	// we need a second pass that handles them. Our parser
	// requires keys to belong to a section; we emit lockfiles
	// with a leading [root] section so readers pick it up
	// uniformly.
	if lk.SchemaVersion != 0 && lk.SchemaVersion != CurrentLockfileSchema {
		return nil, fmt.Errorf("fuse.lock: unsupported schema version %d (expected %d)", lk.SchemaVersion, CurrentLockfileSchema)
	}
	return lk, nil
}

func applyRootKey(lk *Lockfile, kv tomlKV) error {
	switch kv.key {
	case "schema_version":
		if kv.value.kind != tomlKindInteger {
			return fmt.Errorf("fuse.lock: schema_version must be an integer")
		}
		lk.SchemaVersion = int(kv.value.integer)
	case "root":
		s, err := decodeString(kv.value)
		if err != nil {
			return fmt.Errorf("fuse.lock: root: %w", err)
		}
		lk.Root = s
	case "digest":
		s, err := decodeString(kv.value)
		if err != nil {
			return fmt.Errorf("fuse.lock: digest: %w", err)
		}
		lk.Digest = s
	default:
		return fmt.Errorf("fuse.lock: unknown root-level key %q", kv.key)
	}
	return nil
}

func decodeLockedCrate(s *tomlSection) (LockedCrate, error) {
	c := LockedCrate{}
	for _, kv := range s.entries {
		switch kv.key {
		case "name":
			v, err := decodeString(kv.value)
			if err != nil {
				return c, err
			}
			c.Name = v
		case "version":
			v, err := decodeString(kv.value)
			if err != nil {
				return c, err
			}
			c.Version = v
		case "source":
			v, err := decodeString(kv.value)
			if err != nil {
				return c, err
			}
			c.Source = v
		case "sha256":
			v, err := decodeString(kv.value)
			if err != nil {
				return c, err
			}
			c.SHA256 = v
		case "dependencies":
			list, err := parseStringList(kv.value)
			if err != nil {
				return c, err
			}
			c.Dependencies = list
		default:
			return c, fmt.Errorf("fuse.lock: [[crate]] unknown key %q", kv.key)
		}
	}
	return c, nil
}

// Equal reports whether two lockfiles are byte-identical after
// Finalize. Used by tests for round-trip stability.
func (l *Lockfile) Equal(other *Lockfile) bool {
	return bytes.Equal(l.Serialize(), other.Serialize())
}
