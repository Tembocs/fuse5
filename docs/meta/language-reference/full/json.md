# full.json

> Public module: `full.json`
>
> Status: Stage 1 baseline draft.

`full.json` defines the JSON value model, parse/stringify helpers, and
Result-based error handling for invalid input.

## Example

```fuse
import full.json.Json;

let value = Json.parse("{\"name\":\"Fuse\"}")?;
let name = value.get("name").and_then(fn(v) { v.as_string() }).unwrap();
let text = Json.stringify(ref value)?;
```

## Notes

Parsing invalid data must report structured errors rather than panicking.