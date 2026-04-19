# core.int8

> Public module: `core.int8`
>
> Status: Stage 1 baseline draft.

`core.int8` defines the guide surface for `I8` constants, parsing, formatting,
and width-specific helpers.

## Example

```fuse
import core.int8 as int8;

let gain: I8 = int8.parse("12").unwrap();
let min_gain = int8.min_value();
let label = int8.to_string(gain);
```

## Notes

Overflow during parsing must return `Result[I8, ParseIntError]`.