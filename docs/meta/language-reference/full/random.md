# full.random

> Public module: `full.random`
>
> Status: Stage 1 baseline draft.

`full.random` defines deterministic pseudo-random generation, system entropy,
and typed random helpers.

## Example

```fuse
import full.random.Random;

let mut rng = Random.seeded(1234);
let number = rng.u32();
let token = Random.system().fill_bytes(16)?;
```

## Notes

The guide should distinguish reproducible seeded generators from system entropy.