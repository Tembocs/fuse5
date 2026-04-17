package resolve

import "testing"

// TestScopeLookup verifies that Scope.Lookup walks the parent chain and
// that LookupLocal does not.
func TestScopeLookup(t *testing.T) {
	tab := newSymbolTable()
	outer := newScope(nil, "")
	inner := newScope(outer, "child")

	foo := tab.Add(Symbol{Kind: SymFn, Name: "foo", Module: ""})
	if !outer.Insert("foo", foo) {
		t.Fatalf("outer.Insert(foo) reported duplicate on empty scope")
	}

	bar := tab.Add(Symbol{Kind: SymFn, Name: "bar", Module: "child"})
	if !inner.Insert("bar", bar) {
		t.Fatalf("inner.Insert(bar) reported duplicate on empty scope")
	}

	if got := inner.Lookup("foo"); got != foo {
		t.Fatalf("inner.Lookup(foo) = %d, want %d (chained)", got, foo)
	}
	if got := inner.Lookup("bar"); got != bar {
		t.Fatalf("inner.Lookup(bar) = %d, want %d (local)", got, bar)
	}
	if got := inner.LookupLocal("foo"); got != NoSymbol {
		t.Fatalf("inner.LookupLocal(foo) = %d, want NoSymbol (must not walk parents)", got)
	}
	if got := outer.Lookup("bar"); got != NoSymbol {
		t.Fatalf("outer.Lookup(bar) = %d, want NoSymbol (must not walk children)", got)
	}
	if got := outer.Lookup("missing"); got != NoSymbol {
		t.Fatalf("outer.Lookup(missing) = %d, want NoSymbol", got)
	}
}

// TestScopeLookup_DuplicateInsert confirms Insert refuses duplicates.
func TestScopeLookup_DuplicateInsert(t *testing.T) {
	tab := newSymbolTable()
	s := newScope(nil, "")
	a := tab.Add(Symbol{Name: "x"})
	b := tab.Add(Symbol{Name: "x"})
	if !s.Insert("x", a) {
		t.Fatalf("first Insert rejected")
	}
	if s.Insert("x", b) {
		t.Fatalf("second Insert accepted; must reject duplicate")
	}
	if s.Lookup("x") != a {
		t.Fatalf("duplicate insert replaced the prior binding")
	}
}

// TestTopLevelIndex exercises indexModule across every item kind that
// contributes a top-level symbol plus enum variant hoisting.
func TestTopLevelIndex(t *testing.T) {
	src := `
fn f() {}
struct S {}
enum E { A, B, C }
trait T { fn hi(self); }
const K: I32 = 1;
static G: I32 = 2;
type Alias = I32;
union U { a: I32, b: F32 }
`
	srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
	out, msgs := resolveStrings(t, srcs, BuildConfig{})
	if len(msgs) != 0 {
		t.Fatalf("unexpected diagnostics: %v", msgs)
	}
	m := out.Graph.Modules["m"]
	if m == nil {
		t.Fatalf("module m not registered")
	}
	cases := []struct {
		name string
		kind SymKind
	}{
		{"f", SymFn},
		{"S", SymStruct},
		{"E", SymEnum},
		{"A", SymEnumVariant},
		{"B", SymEnumVariant},
		{"C", SymEnumVariant},
		{"T", SymTrait},
		{"K", SymConst},
		{"G", SymStatic},
		{"Alias", SymTypeAlias},
		{"U", SymUnion},
	}
	for _, tc := range cases {
		id := m.Scope.LookupLocal(tc.name)
		if id == NoSymbol {
			t.Errorf("%q not registered", tc.name)
			continue
		}
		sym := out.Symbols.Get(id)
		if sym.Kind != tc.kind {
			t.Errorf("%q kind = %v, want %v", tc.name, sym.Kind, tc.kind)
		}
	}
}

// TestTopLevelIndex_DuplicateDefinition confirms that two items
// sharing a name produce a duplicate-item diagnostic.
func TestTopLevelIndex_DuplicateDefinition(t *testing.T) {
	src := `
fn dup() {}
fn dup() {}
`
	srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
	_, msgs := resolveStrings(t, srcs, BuildConfig{})
	if len(msgs) == 0 {
		t.Fatalf("expected duplicate-item diagnostic, got none")
	}
	if !hasSubstring(msgs, "duplicate item") {
		t.Fatalf("expected 'duplicate item' in %v", msgs)
	}
}

// TestTopLevelIndex_VariantHoistConflict — an enum variant that shadows
// a prior item must emit a diagnostic.
func TestTopLevelIndex_VariantHoistConflict(t *testing.T) {
	src := `
fn North() {}
enum Dir { North, South }
`
	srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
	_, msgs := resolveStrings(t, srcs, BuildConfig{})
	if !hasSubstring(msgs, "duplicate item") {
		t.Fatalf("expected variant-vs-fn conflict diagnostic, got %v", msgs)
	}
}

// --- helpers ---

func hasSubstring(msgs []string, substr string) bool {
	for _, m := range msgs {
		if indexOf(m, substr) >= 0 {
			return true
		}
	}
	return false
}
