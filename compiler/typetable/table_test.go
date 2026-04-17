package typetable

import "testing"

// TestTypeInternEquality proves the core interning contract
// (Rule 7.2): equal Type descriptions share a TypeId, and TypeId
// equality is integer comparison.
func TestTypeInternEquality(t *testing.T) {
	t.Run("primitives-are-pre-interned", func(t *testing.T) {
		tab := New()
		if tab.I32() == NoType {
			t.Fatalf("I32 TypeId must be non-zero")
		}
		if tab.I32() != tab.I32() {
			t.Fatalf("I32 must be stable within a Table")
		}
		if tab.I32() == tab.I64() {
			t.Fatalf("I32 and I64 must be distinct")
		}
	})

	t.Run("structural-interning", func(t *testing.T) {
		tab := New()
		a := tab.Slice(tab.I32())
		b := tab.Slice(tab.I32())
		if a != b {
			t.Fatalf("two `[I32]` slice calls must share a TypeId (got %d vs %d)", a, b)
		}
		c := tab.Slice(tab.I64())
		if a == c {
			t.Fatalf("[I32] and [I64] must be distinct TypeIds")
		}
	})

	t.Run("tuple-element-order-matters", func(t *testing.T) {
		tab := New()
		ab := tab.Tuple([]TypeId{tab.I32(), tab.Bool()})
		ba := tab.Tuple([]TypeId{tab.Bool(), tab.I32()})
		if ab == ba {
			t.Fatalf("tuple element order must contribute to identity")
		}
	})

	t.Run("fn-variadic-contributes-to-identity", func(t *testing.T) {
		tab := New()
		ptr := tab.Ptr(tab.U8())
		a := tab.Fn([]TypeId{ptr}, tab.I32(), false)
		b := tab.Fn([]TypeId{ptr}, tab.I32(), true)
		if a == b {
			t.Fatalf("variadic flag must contribute to fn identity")
		}
	})

	t.Run("trait-object-bounds-are-a-set", func(t *testing.T) {
		tab := New()
		// Use generic param TypeIds as stand-in traits — they are
		// just TypeIds from the interner's perspective.
		a := tab.GenericParam(1, "m", "A")
		b := tab.GenericParam(2, "m", "B")
		x := tab.TraitObject([]TypeId{a, b})
		y := tab.TraitObject([]TypeId{b, a})
		if x != y {
			t.Fatalf("`dyn A + B` and `dyn B + A` must share a TypeId")
		}
	})

	t.Run("array-length-contributes-to-identity", func(t *testing.T) {
		tab := New()
		n3 := tab.Array(tab.I32(), 3)
		n4 := tab.Array(tab.I32(), 4)
		if n3 == n4 {
			t.Fatalf("array length must contribute to identity")
		}
	})
}

// TestNominalIdentity exercises the §2.8 rule: nominal identity is by
// defining symbol, not by name alone.
func TestNominalIdentity(t *testing.T) {
	t.Run("same-symbol-same-type", func(t *testing.T) {
		tab := New()
		a := tab.Nominal(KindStruct, 42, "m", "Point", nil)
		b := tab.Nominal(KindStruct, 42, "m", "Point", nil)
		if a != b {
			t.Fatalf("equal nominal descriptions must share a TypeId")
		}
	})

	t.Run("same-name-different-module-distinct", func(t *testing.T) {
		tab := New()
		a := tab.Nominal(KindStruct, 1, "a", "Expr", nil)
		b := tab.Nominal(KindStruct, 2, "b", "Expr", nil)
		if a == b {
			t.Fatalf("two `Expr` structs from different modules must be distinct (reference §2.8)")
		}
	})

	t.Run("generic-and-specialization-distinct", func(t *testing.T) {
		tab := New()
		unspec := tab.Nominal(KindStruct, 7, "m", "Vec", nil)
		spec := tab.Nominal(KindStruct, 7, "m", "Vec", []TypeId{tab.I32()})
		if unspec == spec {
			t.Fatalf("`Vec` and `Vec[I32]` must be distinct TypeIds")
		}
		// Two `Vec[I32]` calls still share.
		spec2 := tab.Nominal(KindStruct, 7, "m", "Vec", []TypeId{tab.I32()})
		if spec != spec2 {
			t.Fatalf("two `Vec[I32]` intern calls must share a TypeId")
		}
	})

	t.Run("nominal-requires-symbol", func(t *testing.T) {
		tab := New()
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic on nominal type with Symbol=0")
			}
		}()
		tab.Nominal(KindStruct, 0, "m", "Oops", nil)
	})

	t.Run("nominal-rejects-non-nominal-kinds", func(t *testing.T) {
		tab := New()
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic on Nominal with KindI32")
			}
		}()
		tab.Nominal(KindI32, 1, "m", "Oops", nil)
	})
}

// TestChannelTypeKindExists is the W04-P01-T03 verification: the
// KindChannel TypeId shape is reachable through the public API and
// round-trips via Get. W07 will add the checker integration.
func TestChannelTypeKindExists(t *testing.T) {
	tab := New()
	elem := tab.I32()
	ch := tab.Channel(elem)
	if ch == NoType {
		t.Fatalf("Channel(I32) returned NoType")
	}
	got := tab.Get(ch)
	if got == nil {
		t.Fatalf("Get(Channel) returned nil")
	}
	if got.Kind != KindChannel {
		t.Fatalf("Channel kind = %v, want KindChannel", got.Kind)
	}
	if len(got.Children) != 1 || got.Children[0] != elem {
		t.Fatalf("Channel element = %v, want [%d]", got.Children, elem)
	}
	// Interning: two Channel(I32) calls share.
	ch2 := tab.Channel(elem)
	if ch != ch2 {
		t.Fatalf("Channel(I32) must be interned across calls")
	}
	// Different element types produce distinct channels.
	if tab.Channel(tab.Bool()) == ch {
		t.Fatalf("Channel(Bool) and Channel(I32) must be distinct")
	}
}

// TestThreadHandleKindExists is the W04-P01-T04 verification.
func TestThreadHandleKindExists(t *testing.T) {
	tab := New()
	ret := tab.I64()
	th := tab.ThreadHandle(ret)
	if th == NoType {
		t.Fatalf("ThreadHandle(I64) returned NoType")
	}
	got := tab.Get(th)
	if got == nil {
		t.Fatalf("Get(ThreadHandle) returned nil")
	}
	if got.Kind != KindThreadHandle {
		t.Fatalf("ThreadHandle kind = %v, want KindThreadHandle", got.Kind)
	}
	if len(got.Children) != 1 || got.Children[0] != ret {
		t.Fatalf("ThreadHandle element = %v, want [%d]", got.Children, ret)
	}
	th2 := tab.ThreadHandle(ret)
	if th != th2 {
		t.Fatalf("ThreadHandle must be interned across calls")
	}
}

// TestInferIsExplicit confirms that the Infer TypeId is distinct,
// pre-interned, and carries KindInfer — so callers can see a
// pending-inference marker rather than defaulting to Unknown (L013).
func TestInferIsExplicit(t *testing.T) {
	tab := New()
	i := tab.Infer()
	if i == NoType {
		t.Fatalf("Infer must be a valid TypeId, not NoType")
	}
	if tab.Get(i).Kind != KindInfer {
		t.Fatalf("Infer TypeId must resolve to KindInfer")
	}
	// Infer != any primitive.
	for _, k := range primitiveOrder {
		if i == tab.Lookup(k) {
			t.Fatalf("Infer must be distinct from primitive %s", k)
		}
	}
}
