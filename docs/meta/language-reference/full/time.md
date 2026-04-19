# full.time

> Public module: `full.time`
>
> Status: Stage 1 baseline draft.

`full.time` defines wall-clock, monotonic time, durations, and sleep helpers.

## Example

```fuse
import full.time.{Duration, Instant, sleep};

let start = Instant.now();
sleep(Duration.from_millis(10));
let elapsed = start.elapsed();
```

## Notes

Wall-clock and monotonic time should be separated clearly in the API surface.