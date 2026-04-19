# core.string

> Public module: `core.string`
>
> Status: Stage 1 baseline draft.

`core.string` owns the `String` type, UTF-8 construction, byte views, text
manipulation, and string parsing helpers.

## Minimum surface

- owned UTF-8 string construction
- byte and character views
- concatenation and formatting integration
- parsing helpers for numeric and structured text

## Example

```fuse
import core.string.String;

let name = String.from("Fuse");
let greeting = name + String.from(" language");
let bytes = greeting.as_bytes();
let rebuilt = String.from_utf8(ref bytes).unwrap();
```

## Notes

The UTF-8 invariant must hold at all times outside explicit unsafe escape
surfaces.