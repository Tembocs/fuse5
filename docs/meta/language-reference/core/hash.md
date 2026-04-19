# core.hash

> Public module: `core.hash`
>
> Status: Stage 1 baseline draft.

`core.hash` covers hasher state, hash combination, and helper APIs used by
`core.map`, `core.set`, and user-defined keyed types.

## Example

```fuse
import core.hash.Hash;

let mut hasher = Hash.default_hasher();
hasher.write_u32(42);
hasher.write_bytes(ref [1u8, 2u8, 3u8]);
let digest = hasher.finish();
```

## Notes

The guide should define whether hashing is stable only within a process or also
across runs.