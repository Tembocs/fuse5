# core.uint32

> Public module: `core.uint32`
>
> Status: Stage 1 baseline draft.

`core.uint32` defines the width-specific guide for `U32` constants, parsing,
formatting, and conversion helpers.

## Example

```fuse
import core.uint32 as uint32;

let flags: U32 = uint32.parse("1024").unwrap();
let shifted = flags.wrapping_shl(2);
let text = uint32.to_string(shifted);
```

## Notes

The module should expose stable textual conversions and width-specific limits.