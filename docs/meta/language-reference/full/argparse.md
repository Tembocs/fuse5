# full.argparse

> Public module: `full.argparse`
>
> Status: Stage 1 baseline draft.

`full.argparse` defines command-line parsing, flags, positional arguments,
subcommands, and help/usage generation.

## Example

```fuse
import full.argparse.Command;

let cmd = Command.new("fuse")
    .flag("--json")
    .positional("path")
    .parse()?;

let path = cmd.get_string("path")?;
```

## Notes

Parse errors should be structured enough to drive user-facing help output.