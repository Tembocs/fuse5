# full.net

> Public module: `full.net`
>
> Status: Stage 1 baseline draft.

`full.net` defines socket addresses, streams, listeners, and baseline name
resolution.

## Example

```fuse
import full.net.{TcpListener, TcpStream};

let listener = TcpListener.bind("127.0.0.1:8080")?;
let stream = TcpStream.connect("127.0.0.1:8080")?;
```

## Notes

The guide should cover both client and server lifecycle basics and their error
reporting behavior.