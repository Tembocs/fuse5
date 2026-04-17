package check

import (
	"reflect"
	"testing"
)

// TestCallableTraitDeclaration — W12-P02-T01. The three callable
// traits Fn, FnMut, FnOnce are recognized by name and stringify
// to their declared spellings.
func TestCallableTraitDeclaration(t *testing.T) {
	cases := map[CallableTrait]string{
		CallableFn:     "Fn",
		CallableFnMut:  "FnMut",
		CallableFnOnce: "FnOnce",
	}
	for tr, want := range cases {
		if got := tr.String(); got != want {
			t.Errorf("CallableTrait(%d).String() = %q, want %q", tr, got, want)
		}
	}
}

// TestCallableAutoImpl — W12-P02-T02. Each CallableShape
// auto-impls the expected set of traits, and the tightest
// trait is always at the head of the returned slice.
func TestCallableAutoImpl(t *testing.T) {
	cases := []struct {
		shape CallableShape
		want  []CallableTrait
	}{
		{ShapeFnPointer, []CallableTrait{CallableFn, CallableFnMut, CallableFnOnce}},
		{ShapeNoCaptureClosure, []CallableTrait{CallableFn, CallableFnMut, CallableFnOnce}},
		{ShapeReadOnlyClosure, []CallableTrait{CallableFn, CallableFnMut, CallableFnOnce}},
		{ShapeMutClosure, []CallableTrait{CallableFnMut, CallableFnOnce}},
		{ShapeOnceClosure, []CallableTrait{CallableFnOnce}},
	}
	for _, tc := range cases {
		got := CallableTraitFor(tc.shape)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("CallableTraitFor(%v) = %v, want %v", tc.shape, got, tc.want)
		}
		tightest, ok := TightestCallableTrait(tc.shape)
		if !ok {
			t.Errorf("TightestCallableTrait(%v) reported no traits", tc.shape)
			continue
		}
		if tightest != tc.want[0] {
			t.Errorf("TightestCallableTrait(%v) = %v, want %v", tc.shape, tightest, tc.want[0])
		}
	}
}
