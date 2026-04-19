# full.jsonrpc

> Public module: `full.jsonrpc`
>
> Status: Stage 1 baseline draft.

`full.jsonrpc` defines transport-independent JSON-RPC envelopes and helpers on
top of `full.json`.

## Example

```fuse
import full.jsonrpc.JsonRpc;

let request = JsonRpc.request("sum", [1, 2, 3], 1u64);
let response = JsonRpc.success(1u64, 6);
```

## Notes

The guide should explicitly separate protocol envelopes from HTTP or socket
transport concerns.