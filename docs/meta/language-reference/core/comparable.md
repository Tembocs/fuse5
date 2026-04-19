# core.comparable

> Public module: `core.comparable`
>
> Status: Stage 1 baseline draft.

`core.comparable` documents ordering-oriented traits and helpers such as total
ordering contracts, `compare`, and sorting compatibility.

## Example

```fuse
import core.comparable.Comparable;

fn is_sorted[T: Comparable](left: ref T, right: ref T) -> Bool {
    return left.compare(ref right).is_le();
}
```

## Notes

Where total ordering is required, the guide should say so explicitly.