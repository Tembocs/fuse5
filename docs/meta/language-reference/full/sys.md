# full.sys

> Public module: `full.sys`
>
> Status: Stage 1 baseline draft.

`full.sys` covers low-level platform queries that belong in the safe public
surface but sit below `full.os`.

## Example

```fuse
import full.sys as sys;

let page = sys.page_size();
let cpu = sys.cpu_count();
let target = sys.target_triple();
```

## Notes

This module should be narrow and explicit; it is not a shadow unsafe runtime.