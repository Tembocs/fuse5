# full.fs

> Public module: `full.fs`
>
> Status: Stage 1 baseline draft.

`full.fs` defines files, directories, metadata, traversal helpers, and file
system mutation operations.

## Example

```fuse
import full.fs as fs;
import full.path.Path;

let path = Path.from("config/app.toml");
let text = fs.read_to_string(ref path)?;
let meta = fs.metadata(ref path)?;
```

## Notes

Creation, deletion, and traversal behavior should be documented separately for
files, directories, and symlink-like entities if Fuse supports them.