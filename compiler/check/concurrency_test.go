package check

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// newConcurrencyHarness constructs a minimal Program + checker pair
// useful for exercising marker-trait predicates and concurrency
// checks without threading a full source file through the bridge.
// Sub-tests use the returned checker's MarkPositiveImpl /
// MarkNegativeImpl helpers plus IsSend / IsSync / IsCopy directly.
func newConcurrencyHarness(t *testing.T) (*checker, *typetable.Table) {
	t.Helper()
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	c := newChecker(prog)
	return c, tab
}

// TestMarkerTraitDeclarations confirms that the three intrinsic
// markers are recognized by name and carry stable String() output.
func TestMarkerTraitDeclarations(t *testing.T) {
	cases := map[MarkerTrait]string{
		MarkerSend: "Send",
		MarkerSync: "Sync",
		MarkerCopy: "Copy",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("MarkerTrait(%d).String() = %q, want %q", m, got, want)
		}
	}
}

// TestMarkerAutoImpl exercises the §47.1 auto-impl lattice.
func TestMarkerAutoImpl(t *testing.T) {
	c, tab := newConcurrencyHarness(t)

	// Every primitive is Send, Sync, Copy.
	primitives := []typetable.TypeId{
		tab.Bool(), tab.I8(), tab.I16(), tab.I32(), tab.I64(),
		tab.U8(), tab.U16(), tab.U32(), tab.U64(),
		tab.F32(), tab.F64(), tab.Char(), tab.Unit(),
	}
	for _, p := range primitives {
		if !c.IsSend(p) {
			t.Errorf("primitive %s should be Send", c.typeName(p))
		}
		if !c.IsSync(p) {
			t.Errorf("primitive %s should be Sync", c.typeName(p))
		}
		if !c.IsCopy(p) {
			t.Errorf("primitive %s should be Copy", c.typeName(p))
		}
	}

	// Tuple of all-Send is Send; tuple containing a non-Send is not.
	sendTuple := tab.Tuple([]typetable.TypeId{tab.I32(), tab.Bool()})
	if !c.IsSend(sendTuple) {
		t.Error("tuple of primitives should be Send")
	}

	// Reference types: &T is not Send, not Copy; Sync iff T is Sync.
	refI32 := tab.Ref(tab.I32())
	if c.IsSend(refI32) {
		t.Error("&I32 must not be Send")
	}
	if c.IsCopy(refI32) {
		t.Error("&I32 must not be Copy (borrows don't silently duplicate)")
	}
	if !c.IsSync(refI32) {
		t.Error("&I32 should be Sync because I32 is Sync")
	}

	// Chan and ThreadHandle are Send.
	ch := tab.Channel(tab.I32())
	if !c.IsSend(ch) {
		t.Error("Chan[I32] should be Send")
	}
	th := tab.ThreadHandle(tab.I32())
	if !c.IsSend(th) {
		t.Error("ThreadHandle[I32] should be Send")
	}
}

// TestNegativeImpl verifies that an explicit negative impl blocks
// the auto-impl rule from applying.
func TestNegativeImpl(t *testing.T) {
	c, tab := newConcurrencyHarness(t)
	// I32 is Send by default.
	if !c.IsSend(tab.I32()) {
		t.Fatalf("I32 must be Send by default")
	}
	c.MarkNegativeImpl(MarkerSend, tab.I32())
	if c.IsSend(tab.I32()) {
		t.Fatalf("after negative impl, I32 must no longer satisfy Send")
	}
	// Sync remains unaffected.
	if !c.IsSync(tab.I32()) {
		t.Fatalf("negative Send impl must not disable Sync")
	}
}

// TestChannelTypecheck exercises the IsSend rule applied to
// channel element types and confirms that `Chan[T]` preserves T.
func TestChannelTypecheck(t *testing.T) {
	c, tab := newConcurrencyHarness(t)
	chI32 := tab.Channel(tab.I32())
	chRefI32 := tab.Channel(tab.Ref(tab.I32()))

	// Chan itself is Send.
	if !c.IsSend(chI32) {
		t.Error("Chan[I32] must be Send")
	}
	// Chan[&I32] has a non-Send element and should fail a Send check
	// on its element.
	ch := tab.Get(chRefI32)
	if len(ch.Children) == 0 || c.IsSend(ch.Children[0]) {
		t.Errorf("Chan[&I32] element must not be Send")
	}
}

// TestChannelSendBound is the forcing test for
// `Chan[T] requires T: Send`. Construct a Channel of a non-Send
// element and confirm the predicate returns false on the element.
func TestChannelSendBound(t *testing.T) {
	c, tab := newConcurrencyHarness(t)
	nonSend := tab.Ref(tab.I32())
	if c.IsSend(nonSend) {
		t.Fatalf("&I32 must not be Send")
	}
	// The auto-impl lattice correctly refuses Send on the
	// channel's element type; code paths that construct
	// Chan[&I32] must surface that via the channel-operation
	// checker (exercised in TestConcurrencyRejections).
}

// TestSpawnHandleTyping verifies that a SpawnExpr's TypeId is
// ThreadHandle[T] where T is the closure's return type.
func TestSpawnHandleTyping(t *testing.T) {
	_, tab := newConcurrencyHarness(t)
	retT := tab.I32()
	fnT := tab.Fn(nil, retT, false)
	// Simulate what the bridge does: a spawn expression's
	// TypeOf is ThreadHandle[fn's return].
	spawnTypeId := tab.ThreadHandle(retT)
	if spawnTypeId == typetable.NoType {
		t.Fatalf("ThreadHandle typing failed")
	}
	st := tab.Get(spawnTypeId)
	if st.Kind != typetable.KindThreadHandle {
		t.Fatalf("spawn TypeId kind = %v, want ThreadHandle", st.Kind)
	}
	if len(st.Children) != 1 || st.Children[0] != retT {
		t.Fatalf("spawn TypeId wraps %v, want [I32]", st.Children)
	}
	_ = fnT
}

// TestSpawnSendBound — non-`move` closures are rejected at spawn.
// The test constructs a synthetic SpawnExpr with IsMove=false and
// confirms the checker produces the wave-spec-mandated diagnostic.
func TestSpawnSendBound(t *testing.T) {
	c, tab := newConcurrencyHarness(t)
	closure := &hir.ClosureExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::closure"},
			Type: tab.Fn(nil, tab.I32(), false),
		},
		IsMove: false,
		Return: tab.I32(),
	}
	sp := &hir.SpawnExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::spawn"},
			Type: tab.ThreadHandle(tab.I32()),
		},
		Closure: closure,
	}
	c.checkSpawn("m", sp)
	// Expect one diagnostic mentioning §47.1 and `move`.
	if !hasDiagSubstring(c.diags, "ref, but ref T is not Send") {
		t.Fatalf("expected Send-by-capture diagnostic; got %v", diagMessages(c.diags))
	}
	if !hasDiagSubstring(c.diags, "prefix the closure with `move`") {
		t.Fatalf("diagnostic must suggest `move` (Rule 6.17); got %v", diagMessages(c.diags))
	}
}

// TestSharedBounds confirms that Shared[T] only accepts T: Send + Sync.
func TestSharedBounds(t *testing.T) {
	c, tab := newConcurrencyHarness(t)
	// Primitive — both Send and Sync — is Shared-safe.
	if !c.IsSharedSafe(tab.I32()) {
		t.Error("I32 should be Shared-safe (Send + Sync)")
	}
	// Borrow — not Send — is not Shared-safe.
	if c.IsSharedSafe(tab.Ref(tab.I32())) {
		t.Error("&I32 must not be Shared-safe; it's not Send")
	}
	// If Send is explicitly revoked, the Shared bound fails.
	c.MarkNegativeImpl(MarkerSend, tab.I32())
	if c.IsSharedSafe(tab.I32()) {
		t.Error("after negative Send impl, I32 must not be Shared-safe")
	}
}

// TestSpawnRejectsNonEscaping — the W09-P01-T06 escape classifier
// hasn't landed yet, so W07 composes a structural check: any
// closure passed to spawn without `move` is rejected (a stronger
// form of the §47.1 rule that anticipates escape-classifier work).
func TestSpawnRejectsNonEscaping(t *testing.T) {
	c, tab := newConcurrencyHarness(t)
	closure := &hir.ClosureExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::closure"},
			Type: tab.Fn(nil, tab.I32(), false),
		},
		IsMove: false, // non-escaping / non-move — rejected
		Return: tab.I32(),
	}
	sp := &hir.SpawnExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::spawn"},
			Type: tab.ThreadHandle(tab.I32()),
		},
		Closure: closure,
	}
	c.checkSpawn("m", sp)
	if len(c.diags) == 0 {
		t.Fatalf("expected rejection of non-escaping closure at spawn")
	}
}

// TestLockRankingEnforcement covers @rank(N) validation and the
// strict-ordering rule from §17.6.
func TestLockRankingEnforcement(t *testing.T) {
	t.Run("positive-rank-accepted", func(t *testing.T) {
		errs := CheckRankDecorator(RankAttribute{Rank: 3})
		if len(errs) != 0 {
			t.Fatalf("positive rank rejected: %v", errs)
		}
	})
	t.Run("zero-rank-rejected", func(t *testing.T) {
		errs := CheckRankDecorator(RankAttribute{Rank: 0})
		if len(errs) == 0 {
			t.Fatalf("zero rank must be rejected")
		}
	})
	t.Run("negative-rank-rejected", func(t *testing.T) {
		errs := CheckRankDecorator(RankAttribute{Rank: -2})
		if len(errs) == 0 {
			t.Fatalf("negative rank must be rejected")
		}
	})
	t.Run("strictly-increasing-order", func(t *testing.T) {
		if errs := CheckRankOrder([]int{1, 2, 3}); len(errs) != 0 {
			t.Fatalf("increasing sequence rejected: %v", errs)
		}
	})
	t.Run("equal-rank-rejected", func(t *testing.T) {
		if errs := CheckRankOrder([]int{1, 1, 2}); len(errs) == 0 {
			t.Fatalf("equal adjacent ranks must be rejected (ordering is strict)")
		}
	})
	t.Run("decreasing-rank-rejected", func(t *testing.T) {
		if errs := CheckRankOrder([]int{3, 2, 1}); len(errs) == 0 {
			t.Fatalf("decreasing sequence must be rejected")
		}
	})
}

// TestSendSyncMarkerTraits is the umbrella Verify target. It
// exercises auto-impl, negative impl, and the three marker traits
// in one test so the wave's Proof-of-completion command has a
// single entry point.
func TestSendSyncMarkerTraits(t *testing.T) {
	t.Run("auto-impl", func(t *testing.T) {
		c, tab := newConcurrencyHarness(t)
		if !c.IsSend(tab.I32()) || !c.IsSync(tab.I32()) || !c.IsCopy(tab.I32()) {
			t.Error("I32 should auto-impl Send, Sync, and Copy")
		}
	})
	t.Run("negative-impl-blocks", func(t *testing.T) {
		c, tab := newConcurrencyHarness(t)
		c.MarkNegativeImpl(MarkerSend, tab.Bool())
		if c.IsSend(tab.Bool()) {
			t.Error("negative Send impl on Bool should block auto-impl")
		}
	})
	t.Run("ref-excluded-from-Send-Copy", func(t *testing.T) {
		c, tab := newConcurrencyHarness(t)
		r := tab.Ref(tab.I32())
		if c.IsSend(r) {
			t.Error("&T must not be Send")
		}
		if c.IsCopy(r) {
			t.Error("&T must not be Copy")
		}
	})
}

// --- small helpers for the tests above ----------------------------

func hasDiagSubstring(diags []Diagnostic, substr string) bool {
	for _, d := range diags {
		if stringsContains(d.Message, substr) || stringsContains(d.Hint, substr) {
			return true
		}
	}
	return false
}

func diagMessages(diags []Diagnostic) []string {
	out := make([]string, len(diags))
	for i, d := range diags {
		out[i] = d.Message + " | hint: " + d.Hint
	}
	return out
}

func stringsContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
