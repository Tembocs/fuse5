# core.float32

> Public module: `core.float32`
>
> Status: Stage 1 baseline draft.

`core.float32` is the width-specific guide for `F32` parsing, classification,
rounding, and constants.

## Example

```fuse
import core.float32 as float32;

let sample: F32 = float32.parse("0.25").unwrap();
let clipped = float32.clamp(sample, 0.0f32, 1.0f32);
let text = float32.to_string(clipped);
```

## Notes

The guide should make the `F32`-specific precision trade-offs explicit.