# full.timer

> Public module: `full.timer`
>
> Status: Stage 1 baseline draft.

`full.timer` defines one-shot and repeating timer helpers built on top of the
lower-level time primitives.

## Example

```fuse
import full.time.Duration;
import full.timer.Timer;

let timer = Timer.after(Duration.from_millis(250));
timer.wait()?;
```

## Notes

If cancellation exists, the guide should define what happens to a timer that is
dropped before firing.