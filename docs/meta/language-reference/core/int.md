# core.int

> Public module: `core.int`
>
> Status: Stage 1 baseline draft.

`core.int` documents the default signed-integer helper surface for the platform
native signed size.

## Minimum surface

- parsing and formatting
- min/max/limit constants
- clamping and sign helpers
- explicit conversion helpers for common integer families

## Example

```fuse
import core.int as ints;

let value: ISize = ints.parse("42").unwrap();
let bounded: ISize = ints.clamp(value, 0, 100);
let text = ints.to_string(bounded);
```

## Notes

This module does not introduce a new primitive type; it documents the default
signed integer surface promised by the standard library.