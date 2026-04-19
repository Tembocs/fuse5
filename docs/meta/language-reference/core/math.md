# core.math

> Public module: `core.math`
>
> Status: Stage 1 baseline draft.

`core.math` collects common numeric helpers that are not tied to a single width
module, such as `clamp`, powers, roots, and trigonometric helpers where v1
promises them.

## Example

```fuse
import core.math.Math;

let length = Math.sqrt(25.0);
let limited = Math.clamp(length, 0.0, 10.0);
let angle = Math.atan2(1.0, 1.0);
```

## Notes

Domain errors should be documented explicitly rather than left implicit.