# core.printable

> Public module: `core.printable`
>
> Status: Stage 1 baseline draft.

`core.printable` documents the trait or re-export surface used by the simple
printing APIs for user-facing output.

## Example

```fuse
import core.printable.Printable;

fn render[T: Printable](value: ref T) -> String {
    return value.to_string();
}
```

## Notes

This module is the ergonomic surface for ordinary user output, distinct from
debug-oriented rendering.