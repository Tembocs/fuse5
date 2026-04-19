# full.crypto

> Public module: `full.crypto`
>
> Status: Stage 1 baseline draft.

`full.crypto` defines the baseline cryptographic surface that Fuse chooses to
ship as standard library rather than as an external package.

## Example

```fuse
import full.crypto.Sha256;

let mut hasher = Sha256.new();
hasher.update_bytes(ref [1u8, 2u8, 3u8]);
let digest = hasher.finish();
```

## Notes

This guide should name the v1-safe subset explicitly instead of implying a full
cryptography framework.