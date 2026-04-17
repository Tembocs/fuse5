package check

import "testing"

// TestGenericParamScoping — W08-P01-T01. Inside a generic fn's
// body, each generic parameter `T` is in scope as a TypeId. The
// bridge registers it; the checker consults the registration when
// typing the parameter and return types. A fn that type-checks
// here means the scoping works end-to-end.
func TestGenericParamScoping(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}

// TestCallSiteTypeArgs — W08-P01-T02. Explicit turbofish at a
// call site substitutes the fn's generics and makes the call
// well-typed. The bridge reshapes `identity[I32](42)` into a
// PathExpr with TypeArgs=[I32]; the checker substitutes to get
// (I32) -> I32.
func TestCallSiteTypeArgs(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity[I32](42); }
`)
	wantClean(t, diags)
}
