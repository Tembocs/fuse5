# core.float

> Public module: `core.float`
>
> Status: Stage 1 baseline draft.

`core.float` documents the default floating-point helper surface, including
parsing, formatting, rounding, and classification.

## Minimum surface

- parsing from text
- formatting to text
- `is_nan`, `is_infinite`, `round`, `floor`, `ceil`

## Example

```fuse
import core.float as floats;

let ratio: F64 = floats.parse("3.14159").unwrap();
let rounded = floats.round(ratio);
let valid = !floats.is_nan(ratio);
```

## Notes

Text parsing must report `Result[F64, ParseFloatError]` on invalid input.