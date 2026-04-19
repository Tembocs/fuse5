# full.process

> Public module: `full.process`
>
> Status: Stage 1 baseline draft.

`full.process` defines child-process spawning, waiting, exit-status inspection,
and standard-stream redirection.

## Example

```fuse
import full.process.Process;

let child = Process.command("fuse")
    .arg("check")
    .arg("src/main.fuse")
    .spawn()?;

let status = child.wait()?;
```

## Notes

The guide should define how stderr/stdout capture and exit code handling work.