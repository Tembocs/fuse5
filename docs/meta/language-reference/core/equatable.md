# core.equatable

> Public module: `core.equatable`
>
> Status: Stage 1 baseline draft.

`core.equatable` defines the equality trait surface and the contracts consumed
by maps, sets, and user-defined value types.

## Example

```fuse
import core.equatable.Equatable;

fn same_user[T: Equatable](left: ref T, right: ref T) -> Bool {
    return left.eq(ref right);
}
```

## Notes

Equality semantics must remain consistent with hashing when a type is hashable.