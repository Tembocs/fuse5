# full.thread

> Public module: `full.thread`
>
> Status: Stage 1 baseline draft.

`full.thread` owns thread creation, join handles, detach behavior, and thread
result propagation.

## Example

```fuse
import full.thread.ThreadHandle;

let handle: ThreadHandle[I32] = spawn move fn() -> I32 { return 42; };
let result = handle.join()?;
```

## Notes

This guide inherits the existing `spawn` and `ThreadHandle` semantics described
in the main reference and makes the public module contract explicit.