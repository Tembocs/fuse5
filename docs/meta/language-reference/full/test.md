# full.test

> Public module: `full.test`
>
> Status: Stage 1 baseline draft.

`full.test` defines the standard testing helpers exposed to Fuse code, such as
assertions, expected-failure helpers, and fixture or golden utilities.

## Example

```fuse
import full.test as test;

test.assert_eq(2 + 2, 4);
test.assert_true("fuse".len() == 4);
test.assert_err(parse_port("bad"));
```

## Notes

The guide should describe how these helpers integrate with the project test
runner and fixture corpus.