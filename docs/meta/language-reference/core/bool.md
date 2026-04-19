# core.bool

> Public module: `core.bool`
>
> Status: Stage 1 baseline draft.

`core.bool` is the guide for the primitive `Bool` type, its textual forms, and
its module-level helper surface.

## Minimum surface

- parsing from textual values
- rendering to textual values
- boolean helper functions such as `all`, `any`, and negation-friendly helpers

## Example

```fuse
import core.bool as bools;

let enabled: Bool = bools.parse("true").unwrap();
let disabled = enabled.not();
let summary = bools.to_string(disabled);
```

## Notes

Invalid text must return `Result[Bool, ParseBoolError]`, not panic.