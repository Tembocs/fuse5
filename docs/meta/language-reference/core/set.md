# core.set

> Public module: `core.set`
>
> Status: Stage 1 baseline draft.

`core.set` defines the standard unique-value collection for Fuse.

## Minimum surface

- insertion and removal
- membership queries
- iteration and set-style combinators
- explicit value requirements around equality and hashing

## Example

```fuse
import core.set.Set;

let mut seen = Set[I32].new();
seen.insert(10);
seen.insert(10);
seen.insert(20);
let present = seen.contains(ref 20);
```

## Notes

Duplicate insertion should be a defined no-op rather than an implicit error.