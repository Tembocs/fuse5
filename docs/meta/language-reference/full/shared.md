# full.shared

> Public module: `full.shared`
>
> Status: Stage 1 baseline draft.

`full.shared` defines the focused shared-state surface around `Shared[T]` and
its ergonomics.

## Example

```fuse
import full.shared.Shared;

let shared = Shared[I32].new(0);
shared.with_lock(fn(value: mutref I32) {
    *value += 1;
})?;
```

## Notes

This guide should stay narrower than `full.sync`, centering on shared-state
ownership and access patterns rather than all primitives at once.