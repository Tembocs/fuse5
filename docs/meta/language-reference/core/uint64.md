# core.uint64

> Public module: `core.uint64`
>
> Status: Stage 1 baseline draft.

`core.uint64` covers `U64` parsing, formatting, limits, and helper APIs for
large counters, identifiers, and sizes.

## Example

```fuse
import core.uint64 as uint64;

let request_id: U64 = uint64.parse("9007199254740991").unwrap();
let next_id = request_id.wrapping_add(1);
let text = uint64.to_string(next_id);
```

## Notes

This guide should support both decimal and explicit radix rendering.