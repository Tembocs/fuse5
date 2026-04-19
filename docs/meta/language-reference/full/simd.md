# full.simd

> Public module: `full.simd`
>
> Status: Stage 1 baseline draft.

`full.simd` defines target-gated vector types and operations where the project
chooses to expose them as part of the hosted baseline.

## Example

```fuse
import full.simd.F32x4;

let a = F32x4.splat(1.0f32);
let b = F32x4.from_array([1.0f32, 2.0f32, 3.0f32, 4.0f32]);
let c = a + b;
```

## Notes

The guide should define capability gating and fallback expectations explicitly.