# core.map

> Public module: `core.map`
>
> Status: Stage 1 baseline draft.

`core.map` defines the standard associative key/value container for Fuse.

## Minimum surface

- creation and insertion
- lookup and update
- removal and iteration
- explicit key requirements around equality and hashing

## Example

```fuse
import core.map.Map;
import core.string.String;

let mut counts = Map[String, I32].new();
counts.insert(String.from("ok"), 1);
counts.insert(String.from("retry"), 2);
let current = counts.get(ref String.from("ok")).unwrap_or(0);
```

## Notes

The guide should define whether iteration order is stable or intentionally
unspecified.