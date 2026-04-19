# full.http

> Public module: `full.http`
>
> Status: Stage 1 baseline draft.

`full.http` defines requests, responses, headers, status codes, and the Stage 1
blocking client surface.

## Example

```fuse
import full.http.Client;

let client = Client.new();
let response = client.get("https://example.com")?;
let body = response.text()?;
```

## Notes

Stage 1 may be blocking and thread-based; async support is not required to make
this a real baseline module.