# core.traits

> Public module: `core.traits`
>
> Status: Stage 1 baseline draft.

`core.traits` is the canonical import surface for the common traits users are
expected to reach for directly.

## Example

```fuse
import core.traits.{Comparable, Equatable, Hashable};

fn dedupe[T: Equatable + Hashable](items: ref [T]) -> Bool {
    return items.len() > 0;
}
```

## Notes

This module may re-export trait definitions from more focused guides, but the
re-export surface itself is part of the public baseline.