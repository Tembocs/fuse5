// Package pkg owns the Wave 23 Fuse package manager: manifest
// parser (`fuse.toml`), lockfile serializer (`fuse.lock`), semver
// range algebra, deterministic resolver, HTTPS fetcher with
// SHA-256 integrity, and the registry protocol wire format.
//
// The package is self-contained — it depends only on the Go
// standard library and the existing compiler packages. No
// external TOML library is pulled in; a narrow TOML subset is
// parsed inline because manifest grammar is stable.
package pkg

import (
	"fmt"
	"sort"
	"strings"
)

// Manifest is the decoded form of a `fuse.toml` file. A manifest
// belongs to exactly one crate (workspace roots are manifests
// with a `[workspace]` table but no `[package]`).
type Manifest struct {
	// Package identity. Empty when this manifest is a pure
	// workspace root.
	Package *Package
	// Dependencies lists every (name → spec) declared under
	// the `[dependencies]` table. Key order is deterministic:
	// sorted by name.
	Dependencies []Dep
	// DevDependencies are used by `fuse test` only. Not
	// published; not required for downstream consumers.
	DevDependencies []Dep
	// Features maps a feature name to the list of
	// dependency-or-feature names it enables.
	Features map[string][]string
	// Workspace, when non-nil, declares this manifest as a
	// workspace root. Members are relative paths.
	Workspace *Workspace
}

// Package identifies the crate produced by this manifest.
type Package struct {
	Name        string
	Version     string
	Edition     string
	Description string
}

// Dep is one declared dependency. Exactly one of Version /
// Path / URL is set; the resolver treats each source type
// differently.
type Dep struct {
	Name     string
	Version  string // semver range; empty for path/URL deps
	Path     string // local relative path; empty for registry/URL deps
	URL      string // registry URL; empty for path deps
	Features []string
	Optional bool
	Target   string // cfg-target spec; empty means "all targets"
}

// Workspace is the `[workspace]` table of a root manifest.
type Workspace struct {
	Members []string
}

// ParseManifest parses `src` as a `fuse.toml` manifest. Unknown
// top-level keys produce a diagnostic; unknown keys inside
// known tables also produce diagnostics. Rule 6.9 — we do not
// silently drop content.
func ParseManifest(src []byte) (*Manifest, error) {
	doc, err := parseToml(src)
	if err != nil {
		return nil, err
	}
	m := &Manifest{Features: map[string][]string{}}
	for _, section := range doc.sections {
		switch section.name {
		case "package":
			pkg, err := decodePackage(section)
			if err != nil {
				return nil, err
			}
			m.Package = pkg
		case "dependencies":
			deps, err := decodeDeps(section)
			if err != nil {
				return nil, err
			}
			m.Dependencies = deps
		case "dev-dependencies":
			deps, err := decodeDeps(section)
			if err != nil {
				return nil, err
			}
			m.DevDependencies = deps
		case "features":
			for _, kv := range section.entries {
				list, err := parseStringList(kv.value)
				if err != nil {
					return nil, fmt.Errorf("features.%s: %w", kv.key, err)
				}
				m.Features[kv.key] = list
			}
		case "workspace":
			ws, err := decodeWorkspace(section)
			if err != nil {
				return nil, err
			}
			m.Workspace = ws
		default:
			return nil, fmt.Errorf("unknown section [%s] — Fuse.toml recognises [package], [dependencies], [dev-dependencies], [features], [workspace] only", section.name)
		}
	}
	if m.Package == nil && m.Workspace == nil {
		return nil, fmt.Errorf("manifest must declare either [package] or [workspace]")
	}
	if m.Package != nil {
		if m.Package.Name == "" {
			return nil, fmt.Errorf("[package] missing required key `name`")
		}
		if m.Package.Version == "" {
			return nil, fmt.Errorf("[package] missing required key `version`")
		}
	}
	// Deterministic ordering for downstream consumers.
	sort.Slice(m.Dependencies, func(i, j int) bool { return m.Dependencies[i].Name < m.Dependencies[j].Name })
	sort.Slice(m.DevDependencies, func(i, j int) bool { return m.DevDependencies[i].Name < m.DevDependencies[j].Name })
	return m, nil
}

// decodePackage extracts the [package] table.
func decodePackage(s *tomlSection) (*Package, error) {
	p := &Package{}
	for _, kv := range s.entries {
		val, err := decodeString(kv.value)
		if err != nil {
			return nil, fmt.Errorf("[package].%s: %w", kv.key, err)
		}
		switch kv.key {
		case "name":
			p.Name = val
		case "version":
			p.Version = val
		case "edition":
			p.Edition = val
		case "description":
			p.Description = val
		default:
			return nil, fmt.Errorf("[package] unknown key %q", kv.key)
		}
	}
	return p, nil
}

// decodeDeps extracts a [dependencies] or [dev-dependencies]
// table. Each entry is either a bare version string or a
// table-of-properties.
func decodeDeps(s *tomlSection) ([]Dep, error) {
	out := make([]Dep, 0, len(s.entries))
	for _, kv := range s.entries {
		d, err := decodeDep(kv.key, kv.value)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func decodeDep(name string, value *tomlValue) (Dep, error) {
	d := Dep{Name: name}
	if value.kind == tomlKindString {
		d.Version = value.str
		return d, nil
	}
	if value.kind != tomlKindInlineTable {
		return d, fmt.Errorf("dependency %q: expected version string or inline table", name)
	}
	for _, kv := range value.table {
		val, err := decodeString(kv.value)
		hasListValue := false
		var listVal []string
		if value := kv.value; value.kind == tomlKindArray {
			listVal, err = parseStringList(value)
			hasListValue = err == nil
		}
		switch kv.key {
		case "version":
			if err != nil {
				return d, fmt.Errorf("%s.version: %w", name, err)
			}
			d.Version = val
		case "path":
			if err != nil {
				return d, fmt.Errorf("%s.path: %w", name, err)
			}
			d.Path = val
		case "url":
			if err != nil {
				return d, fmt.Errorf("%s.url: %w", name, err)
			}
			d.URL = val
		case "target":
			if err != nil {
				return d, fmt.Errorf("%s.target: %w", name, err)
			}
			d.Target = val
		case "features":
			if !hasListValue {
				return d, fmt.Errorf("%s.features: expected a list of strings", name)
			}
			d.Features = listVal
		case "optional":
			if kv.value.kind != tomlKindBool {
				return d, fmt.Errorf("%s.optional: expected a boolean", name)
			}
			d.Optional = kv.value.boolean
		default:
			return d, fmt.Errorf("dependency %q unknown key %q", name, kv.key)
		}
	}
	return d, nil
}

func decodeWorkspace(s *tomlSection) (*Workspace, error) {
	w := &Workspace{}
	for _, kv := range s.entries {
		switch kv.key {
		case "members":
			list, err := parseStringList(kv.value)
			if err != nil {
				return nil, fmt.Errorf("workspace.members: %w", err)
			}
			w.Members = list
		default:
			return nil, fmt.Errorf("[workspace] unknown key %q", kv.key)
		}
	}
	return w, nil
}

// parseStringList accepts a tomlValue and returns the
// string-array form, or an error when the value is not a
// homogeneous list of strings.
func parseStringList(v *tomlValue) ([]string, error) {
	if v.kind != tomlKindArray {
		return nil, fmt.Errorf("expected an array")
	}
	out := make([]string, 0, len(v.array))
	for i, elem := range v.array {
		s, err := decodeString(elem)
		if err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		out = append(out, s)
	}
	return out, nil
}

func decodeString(v *tomlValue) (string, error) {
	if v == nil || v.kind != tomlKindString {
		return "", fmt.Errorf("expected string")
	}
	return v.str, nil
}

// IDString returns a stable "<name>@<version>" identifier for
// the package, or an empty string when the manifest is a
// workspace root.
func (m *Manifest) IDString() string {
	if m.Package == nil {
		return ""
	}
	return m.Package.Name + "@" + m.Package.Version
}

// String renders the manifest in a deterministic canonical form
// suitable for golden comparisons.
func (m *Manifest) String() string {
	var sb strings.Builder
	if m.Package != nil {
		sb.WriteString("[package]\n")
		fmt.Fprintf(&sb, "name = %q\n", m.Package.Name)
		fmt.Fprintf(&sb, "version = %q\n", m.Package.Version)
		if m.Package.Edition != "" {
			fmt.Fprintf(&sb, "edition = %q\n", m.Package.Edition)
		}
		if m.Package.Description != "" {
			fmt.Fprintf(&sb, "description = %q\n", m.Package.Description)
		}
	}
	if len(m.Dependencies) > 0 {
		sb.WriteString("\n[dependencies]\n")
		for _, d := range m.Dependencies {
			sb.WriteString(renderDep(d))
		}
	}
	if len(m.DevDependencies) > 0 {
		sb.WriteString("\n[dev-dependencies]\n")
		for _, d := range m.DevDependencies {
			sb.WriteString(renderDep(d))
		}
	}
	return sb.String()
}

func renderDep(d Dep) string {
	var sb strings.Builder
	// When only version is set, render as bare string form.
	if d.Path == "" && d.URL == "" && d.Target == "" && len(d.Features) == 0 && !d.Optional {
		fmt.Fprintf(&sb, "%s = %q\n", d.Name, d.Version)
		return sb.String()
	}
	fmt.Fprintf(&sb, "%s = { ", d.Name)
	comma := false
	add := func(k, v string) {
		if v == "" {
			return
		}
		if comma {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%s = %q", k, v)
		comma = true
	}
	add("version", d.Version)
	add("path", d.Path)
	add("url", d.URL)
	add("target", d.Target)
	if len(d.Features) > 0 {
		if comma {
			sb.WriteString(", ")
		}
		sb.WriteString("features = [")
		for i, f := range d.Features {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%q", f)
		}
		sb.WriteByte(']')
		comma = true
	}
	if d.Optional {
		if comma {
			sb.WriteString(", ")
		}
		sb.WriteString("optional = true")
	}
	sb.WriteString(" }\n")
	return sb.String()
}
