# core.fmt

> Public module: `core.fmt`
>
> Status: Stage 1 baseline draft.

`core.fmt` owns formatter types, formatting traits, and the format-string
surface for `format!`, `print!`, and `eprint!`.

## Example

```fuse
import core.fmt.format;

let name = String.from("Fuse");
let text = format!("hello, {}", name);
print!("{}\n", text);
eprint!("debug={:?}\n", text);
```

## Notes

Format-string and argument mismatches must be compile errors, not runtime
formatting failures.