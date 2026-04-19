# core.list

> Public module: `core.list`
>
> Status: Stage 1 baseline draft.

`core.list` defines the growable owned sequence type used for variable-length
ordered collections.

## Minimum surface

- creation and capacity management
- push, pop, insert, remove
- indexing and iteration
- formatting, equality, and hashing when element types support them

## Example

```fuse
import core.list.List;

let mut items = List[I32].new();
items.push(10);
items.push(20);
let total = items.into_iter().fold(0, fn(acc: I32, n: I32) -> I32 { acc + n });
```

## Notes

The guide should state when operations reallocate and how iterator invalidation
behaves.