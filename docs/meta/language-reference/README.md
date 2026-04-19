# Language Reference Draft Tree

This directory is the staging form of the future `docs/language-reference/`
tree for Attempt 6.

It exists so that `docs/meta/` can later be copied into a new repository and
renamed to `docs/` without changing the intended documentation layout.

## Current contents

- `fuse-language-reference-draft.md` — the working replacement draft for the
  current monolithic language reference. It inherits unchanged language
  sections from the existing reference and adds the missing Stage 1 standard
  library baseline explicitly.

## Intended end state

This tree should become the canonical normative reference system for Fuse.
It may remain one large document or be split into several tightly owned
normative documents, but it must remain one source of truth.

At minimum, this tree is expected to cover:

- language core
- stdlib core
- stdlib hosted baseline
- FFI and runtime boundary

The corresponding implementation schedule belongs in `../implementation/`, not
in this tree.