# core.debuggable

> Public module: `core.debuggable`
>
> Status: Stage 1 baseline draft.

`core.debuggable` documents the trait surface for debug-oriented formatting and
inspection.

## Example

```fuse
import core.debuggable.Debuggable;
import core.fmt.format;

fn debug_line[T: Debuggable](value: ref T) -> String {
    return format!("{:?}", value);
}
```

## Notes

The debug contract may reveal structural detail that the printable contract does
not.