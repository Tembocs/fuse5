# full.env

> Public module: `full.env`
>
> Status: Stage 1 baseline draft.

`full.env` defines environment-variable and process-environment helpers.

## Example

```fuse
import full.env as env;

let home = env.var("HOME").unwrap_or_default();
let cwd = env.current_dir()?;
let args = env.args();
```

## Notes

Missing environment variables should be expressed through `Option` or `Result`,
not empty-string sentinel values.