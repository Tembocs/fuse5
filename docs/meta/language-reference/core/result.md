# core.result

> Public module: `core.result`
>
> Status: Stage 1 baseline draft.

`core.result` defines `Result[T, E]`, its constructors, combinators, and the
expected error-carrying surface for library APIs.

## Example

```fuse
import core.result.Result;
import core.string.String;

fn parse_port(text: ref String) -> Result[U16, String] {
    return text.parse_u16().map_err(fn() -> String { String.from("bad port") });
}

let port = parse_port(ref String.from("8080"))?;
```

## Notes

Modules that consume external input should prefer `Result` over panic-first
failure handling.