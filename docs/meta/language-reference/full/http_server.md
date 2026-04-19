# full.http_server

> Public module: `full.http_server`
>
> Status: Stage 1 baseline draft.

`full.http_server` defines the baseline blocking server surface on top of
`full.net` and `full.http`.

## Example

```fuse
import full.http_server.Server;

let server = Server.bind("127.0.0.1:8080")?;
server.serve(fn(req) {
    return Server.ok("hello from Fuse");
})?;
```

## Notes

The guide should define request handler shape, response helpers, and shutdown
behavior.