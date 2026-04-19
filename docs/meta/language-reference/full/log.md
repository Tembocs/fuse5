# full.log

> Public module: `full.log`
>
> Status: Stage 1 baseline draft.

`full.log` defines leveled or structured logging, formatter integration, and
pluggable sinks.

## Example

```fuse
import full.log.Log;

Log.info("server started");
Log.with("port", 8080).debug("binding listener");
```

## Notes

The guide should document default sinks and whether logging is synchronous or
buffered.