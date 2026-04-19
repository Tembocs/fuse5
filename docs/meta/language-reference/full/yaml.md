# full.yaml

> Public module: `full.yaml`
>
> Status: Stage 1 baseline draft.

`full.yaml` defines YAML parse/emit support and a document model suitable for
configuration and data interchange.

## Example

```fuse
import full.yaml.Yaml;

let doc = Yaml.parse("name: Fuse\nfeatures:\n  - compiler\n")?;
let text = Yaml.stringify(ref doc)?;
```

## Notes

The guide should describe which YAML subset is guaranteed if full YAML is not in
scope for v1.