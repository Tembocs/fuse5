# Implementation Draft Tree

This directory is the staging form of the future `docs/implementation/` tree
for Attempt 6.

It exists so that `docs/meta/` can later be copied into a new repository and
renamed to `docs/` without changing the intended documentation layout.

## Purpose

This tree is where the delivery plan belongs. It defines when and how features
land, which wave owns them, what proof closes them, and what residual work is
forbidden from leaking forward.

The implementation tree answers:

- when a feature lands
- which wave owns it
- what proof retires it
- how planning and closure are verified

The language reference tree answers what Fuse is. That separation is the point.

## Current migration status

The authoritative implementation material in this repository still lives in the
existing `docs/implementation-plan.md` file and the existing `docs/implementation/`
wave documents. Those documents have not yet been migrated under this staging
tree.

This directory is therefore the structural placeholder for the future tree, not
yet the full migrated plan.

## Intended end state

When this staging tree is promoted into the new repository, this directory
should hold at least:

- an implementation plan entry point
- per-wave planning documents
- closure and proof requirements bound to those waves
- any supporting planning indexes needed to keep reference features tied to
  concrete implementation ownership