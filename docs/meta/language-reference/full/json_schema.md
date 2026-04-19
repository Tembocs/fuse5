# full.json_schema

> Public module: `full.json_schema`
>
> Status: Stage 1 baseline draft.

`full.json_schema` defines JSON Schema representation and validation over
`full.json` values.

## Example

```fuse
import full.json.Json;
import full.json_schema.JsonSchema;

let schema = JsonSchema.parse("{\"type\":\"object\"}")?;
let value = Json.parse("{\"ok\":true}")?;
let report = schema.validate(ref value);
```

## Notes

Validation should report diagnostics-rich results rather than a bare boolean.