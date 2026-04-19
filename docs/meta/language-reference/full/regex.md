# full.regex

> Public module: `full.regex`
>
> Status: Stage 1 baseline draft.

`full.regex` defines regex compilation, matching, captures, and replacement
helpers.

## Example

```fuse
import full.regex.Regex;

let re = Regex.compile("[a-z]+")?;
let matched = re.is_match("fuse-5");
let replaced = re.replace_all("fuse-5", "lang");
```

## Notes

Compilation errors must be returned as `Result`, not deferred until first use.