# full.path

> Public module: `full.path`
>
> Status: Stage 1 baseline draft.

`full.path` documents path construction, splitting, normalization, and
platform-aware separator behavior.

## Example

```fuse
import full.path.Path;

let root = Path.from("config");
let file = root.join("app.toml");
let ext = file.extension();
let normalized = file.normalize();
```

## Notes

Path manipulation should not silently assume POSIX-only rules.