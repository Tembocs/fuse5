# full.sync

> Public module: `full.sync`
>
> Status: Stage 1 baseline draft.

`full.sync` defines mutexes, reader/writer locks, condition variables, and
once-only initialization.

## Example

```fuse
import full.sync.Mutex;

let counter = Mutex[I32].new(0);
{
    let mut guard = counter.lock()?;
    *guard += 1;
}
```

## Notes

The guide should say whether poisoning exists and how lock acquisition failures
are reported.