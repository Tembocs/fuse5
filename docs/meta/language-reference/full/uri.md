# full.uri

> Public module: `full.uri`
>
> Status: Stage 1 baseline draft.

`full.uri` defines URI parsing, normalization, component access, and
percent-encoding helpers.

## Example

```fuse
import full.uri.Uri;

let uri = Uri.parse("https://example.com/api?lang=fuse")?;
let host = uri.host().unwrap();
let normalized = uri.normalize();
```

## Notes

The guide should distinguish full URIs from relative references.