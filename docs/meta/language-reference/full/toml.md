# full.toml

> Public module: `full.toml`
>
> Status: Stage 1 baseline draft.

`full.toml` defines TOML parse/emit support for configuration-oriented data.

## Example

```fuse
import full.toml.Toml;

let config = Toml.parse("port = 8080\n[server]\nhost = \"127.0.0.1\"\n")?;
let host = config.get_table("server")?.get_string("host")?;
```

## Notes

The guide should make table/array/scalar behavior explicit rather than implicit.