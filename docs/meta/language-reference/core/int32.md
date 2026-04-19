# core.int32

> Public module: `core.int32`
>
> Status: Stage 1 baseline draft.

`core.int32` is the width-specific guide for `I32` parsing, formatting, bounds,
and arithmetic helpers that are specific to 32-bit signed integers.

## Example

```fuse
import core.int32 as int32;

let port_delta: I32 = int32.parse("32").unwrap();
let safe = int32.clamp(port_delta, -1024, 1024);
let text = int32.to_string(safe);
```

## Notes

This module should expose bit-width-aware constants and conversion helpers.