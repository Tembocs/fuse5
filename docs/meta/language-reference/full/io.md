# full.io

> Public module: `full.io`
>
> Status: Stage 1 baseline draft.

`full.io` defines reader/writer traits, standard streams, and common text/byte
I/O operations.

## Example

```fuse
import full.io as io;

io.stdout().write_line("hello")?;
let line = io.stdin().read_line()?;
io.stderr().write_line(line.trim())?;
```

## Notes

I/O failures should use `Result`, not hidden panic paths.