# core.uint8

> Public module: `core.uint8`
>
> Status: Stage 1 baseline draft.

`core.uint8` covers byte-sized unsigned integer parsing, formatting, limits,
and byte-oriented helper functions.

## Example

```fuse
import core.uint8 as uint8;

let octet: U8 = uint8.parse("255").unwrap();
let is_max = octet == uint8.max_value();
let text = uint8.to_hex(octet);
```

## Notes

The module should be convenient for text, binary, and protocol-oriented code.