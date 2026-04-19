# core.hashable

> Public module: `core.hashable`
>
> Status: Stage 1 baseline draft.

`core.hashable` documents the trait contract for values that may be hashed and
stored in hash-based containers.

## Example

```fuse
import core.hash.Hash;
import core.hashable.Hashable;

fn fingerprint[T: Hashable](value: ref T) -> U64 {
    let mut hasher = Hash.default_hasher();
    value.hash(mutref hasher);
    return hasher.finish();
}
```

## Notes

Types that implement this trait should also describe their equality contract.