# core.option

> Public module: `core.option`
>
> Status: Stage 1 baseline draft.

`core.option` defines `Option[T]`, its constructors, combinators, and the
library-facing optional-value contract.

## Example

```fuse
import core.option.Option;

fn first(items: ref [I32]) -> Option[I32] {
    if items.len() == 0 { return None; }
    return Some(items[0]);
}

let value = first(ref [10, 20, 30]).map(fn(n: I32) -> I32 { n + 1 });
```

## Notes

This guide inherits the propagation and combinator semantics already described
in the main language reference.