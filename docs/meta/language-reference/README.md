# Language Reference Tree

This directory is the staging form of the future `docs/language-reference/`
tree for Fuse.

It exists so that `docs/meta/` can later be copied into a new repository and
renamed to `docs/` without changing the intended documentation layout.

## Entry points

- [fuse-language-reference.md](fuse-language-reference.md) — the canonical
  monolithic reference entry point for the staging tree. It incorporates the
  staged stdlib baseline directly and links to the detailed `core/` and
  `full/` module guides.

## Core guides

- [core/bool.md](core/bool.md)
- [core/int.md](core/int.md)
- [core/int8.md](core/int8.md)
- [core/int32.md](core/int32.md)
- [core/uint8.md](core/uint8.md)
- [core/uint32.md](core/uint32.md)
- [core/uint64.md](core/uint64.md)
- [core/float.md](core/float.md)
- [core/float32.md](core/float32.md)
- [core/math.md](core/math.md)
- [core/traits.md](core/traits.md)
- [core/comparable.md](core/comparable.md)
- [core/equatable.md](core/equatable.md)
- [core/hashable.md](core/hashable.md)
- [core/hash.md](core/hash.md)
- [core/printable.md](core/printable.md)
- [core/debuggable.md](core/debuggable.md)
- [core/fmt.md](core/fmt.md)
- [core/string.md](core/string.md)
- [core/option.md](core/option.md)
- [core/result.md](core/result.md)
- [core/list.md](core/list.md)
- [core/map.md](core/map.md)
- [core/set.md](core/set.md)

## Hosted guides

- [full/io.md](full/io.md)
- [full/fs.md](full/fs.md)
- [full/os.md](full/os.md)
- [full/env.md](full/env.md)
- [full/path.md](full/path.md)
- [full/process.md](full/process.md)
- [full/sys.md](full/sys.md)
- [full/time.md](full/time.md)
- [full/timer.md](full/timer.md)
- [full/thread.md](full/thread.md)
- [full/sync.md](full/sync.md)
- [full/shared.md](full/shared.md)
- [full/chan.md](full/chan.md)
- [full/random.md](full/random.md)
- [full/net.md](full/net.md)
- [full/http.md](full/http.md)
- [full/http_server.md](full/http_server.md)
- [full/simd.md](full/simd.md)
- [full/json.md](full/json.md)
- [full/yaml.md](full/yaml.md)
- [full/toml.md](full/toml.md)
- [full/json_schema.md](full/json_schema.md)
- [full/uri.md](full/uri.md)
- [full/regex.md](full/regex.md)
- [full/argparse.md](full/argparse.md)
- [full/crypto.md](full/crypto.md)
- [full/jsonrpc.md](full/jsonrpc.md)
- [full/log.md](full/log.md)
- [full/test.md](full/test.md)

## Intended end state

This tree should become the canonical normative reference system for Fuse.
It may remain one large document or be split into several tightly owned
normative documents, but it must remain one source of truth. The corresponding
implementation schedule belongs in `../implementation/`, not in this tree.

The module guides under `core/` and `full/` are detailed library guides
referenced by the monolith, not competing reference roots.