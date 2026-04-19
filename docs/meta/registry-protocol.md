# Fuse Registry Protocol (v1)

> Status: **frozen**. This document pins the Wave 23 registry
> protocol wire format. Implementations MUST reject payloads
> whose `schema_version` field differs from `1`.
>
> Governance: `tools/checkdocs -registry-protocol-frozen` asserts
> this file carries the `frozen: true` marker.

---

## 1. Overview

A Fuse registry serves three artifacts:

1. **Index** — a line-oriented file listing every published
   `(name, version, url, sha256, dependencies)` tuple.
2. **Package metadata** — a per-`(name, version)` JSON payload
   consumable without downloading the tarball.
3. **Source tarball** — the package body, SHA-256-addressed.

The protocol is content-addressed and deterministic. No code
executes on the client during fetch; there are no pre-install
hooks; the registry never gets to run user code on a developer
machine.

---

## 2. Index format

The index is a UTF-8, line-oriented file. Each non-blank,
non-comment line is a complete JSON object. Order is not
significant; the client sorts entries lexicographically.

### 2.1 Header

The first non-blank, non-comment line MUST be:

```json
{"schema_version": 1}
```

Any other shape rejects the whole file.

### 2.2 Entry

Every subsequent line is a JSON object of this shape:

```json
{
  "name": "alpha",
  "version": "1.2.3",
  "url": "https://registry.example/alpha-1.2.3.tar.gz",
  "sha256": "<64-hex-char-digest>",
  "dependencies": [
    {"name": "beta", "range": "^2.0.0"}
  ]
}
```

Fields:

| Field          | Type    | Required | Notes                                        |
|----------------|---------|----------|----------------------------------------------|
| `name`         | string  | yes      | ASCII, `[A-Za-z0-9_-]+`, case-sensitive.     |
| `version`      | string  | yes      | `MAJOR.MINOR.PATCH` semver; no pre-release.  |
| `url`          | string  | yes      | `https://` only for public registries.       |
| `sha256`       | string  | yes      | Hex-encoded digest of the tarball bytes.     |
| `dependencies` | array   | no       | Each entry is `{name, range}`.               |

Duplicate `(name, version)` pairs in the same index are a hard
error — the parser rejects the file.

### 2.3 Blank lines and comments

Lines whose first non-whitespace character is `#` are comments
and are ignored. Blank lines are ignored.

---

## 3. Package metadata format

A registry serves per-`(name, version)` metadata at a URL the
index can reference. The metadata is a single JSON object:

```json
{
  "schema_version": 1,
  "name": "alpha",
  "version": "1.2.3",
  "description": "Human-readable description",
  "url": "https://registry.example/alpha-1.2.3.tar.gz",
  "sha256": "<64-hex-char-digest>",
  "dependencies": [{"name":"beta","range":"^2.0.0"}],
  "license": "Apache-2.0",
  "published_at": 1714500000
}
```

`schema_version` MUST equal `1`. Unknown top-level keys are
silently ignored (forward-compat for additive fields).

---

## 4. Source tarball

The source tarball is a gzipped tar archive whose root is the
crate directory. The archive MUST:

- contain a `fuse.toml` at the root;
- not contain `fuse.lock`, `target/`, `.fuse-cache/`, or any
  path outside the crate root;
- be reproducible (sorted entries, zero mtimes, zero uid/gid).

Integrity is verified by SHA-256 against the registry-declared
digest before the client unpacks.

---

## 5. Client contract

Clients (the `fuse` CLI, IDE integrations) MUST:

1. Download the index over HTTPS.
2. Parse it per §2.
3. For each resolved `(name, version)`:
   - Fetch the source tarball via the declared `url`.
   - Verify the SHA-256.
   - Unpack into the local cache only after verification.

Clients MUST NOT execute any build script, shell command, or
download hook during fetch. Fuse does not support pre-install
hooks; a crate that requires a build step at install time
cannot be published.

---

## 6. Publishing model (spec only)

A v1 publish flow:

1. The publisher assembles a reproducible source tarball per §4.
2. The publisher authenticates to the registry with a bearer
   token (`Authorization: Bearer <token>`).
3. The publisher `POST`s the tarball to
   `<registry>/api/v1/packages/<name>/<version>` with
   `Content-Type: application/x-tar+gzip`.
4. The registry validates the tarball shape (§4), computes the
   SHA-256, appends a new entry to the index, writes the
   metadata payload, stores the tarball at the advertised URL,
   and responds `201 Created` with the index delta.

A hosted reference registry is not in scope for Fuse v1; the
spec is frozen so an eventual implementation does not require
a protocol revision.

---

## 7. Versioning

The registry protocol follows its own `schema_version`
discipline separate from Fuse language versions. A breaking
change to the index or metadata shape bumps the version; the
protocol MUST NOT change in a backward-incompatible way without
a version increment.

---

frozen: true