package check

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// mkTrait constructs a synthetic TraitDecl with the given
// methods. Each method is a FnDecl with a first param named
// "self" (with the given ownership) and the given non-self
// param TypeIds. Return type defaults to Unit unless overridden.
func mkTrait(tab *typetable.Table, name string, methods []*hir.FnDecl) *hir.TraitDecl {
	traitID := tab.Nominal(typetable.KindTrait, 500, "m", name, nil)
	items := make([]hir.Item, len(methods))
	for i, m := range methods {
		items[i] = m
	}
	return &hir.TraitDecl{
		Base:   hir.Base{ID: hir.ItemID("m", name)},
		Name:   name,
		TypeID: traitID,
		Items:  items,
	}
}

// mkMethod is a small constructor for trait methods.
func mkMethod(tab *typetable.Table, name string, recvOwn hir.Ownership, extraParams []typetable.TypeId, retType typetable.TypeId) *hir.FnDecl {
	self := &hir.Param{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: hir.ItemID("m", name+"::self")},
			Type: tab.Unit(), // placeholder; real type is the trait's Self
		},
		Name:      "self",
		Ownership: recvOwn,
	}
	params := []*hir.Param{self}
	for i, t := range extraParams {
		params = append(params, &hir.Param{
			TypedBase: hir.TypedBase{
				Base: hir.Base{ID: hir.ItemID("m", name+"::p"+string(rune('0'+i)))},
				Type: t,
			},
			Name: "p" + string(rune('0'+i)),
		})
	}
	if retType == typetable.NoType {
		retType = tab.Unit()
	}
	return &hir.FnDecl{
		Base:   hir.Base{ID: hir.ItemID("m", name)},
		Name:   name,
		TypeID: tab.Fn([]typetable.TypeId{tab.Unit()}, retType, false),
		Params: params,
		Return: retType,
	}
}

// TestObjectSafety — W13-P01-T01. A minimal trait with `fn
// hello(self) -> I32` is object-safe; adding a generic method,
// replacing the receiver with a non-self parameter, adding an
// associated const, or mentioning Self in a non-receiver
// position rejects the trait.
func TestObjectSafety(t *testing.T) {
	tab := typetable.New()

	t.Run("plain-receiver-is-safe", func(t *testing.T) {
		tr := mkTrait(tab, "Greet", []*hir.FnDecl{
			mkMethod(tab, "hello", hir.OwnNone, nil, tab.I32()),
		})
		reason, method := IsObjectSafeWithTab(tr, tab)
		if reason != ObjectSafetyOK {
			t.Fatalf("plain `fn hello(self) -> I32` should be object-safe; got %q on method %q", reason, method)
		}
	})

	t.Run("ref-receiver-is-safe", func(t *testing.T) {
		tr := mkTrait(tab, "Show", []*hir.FnDecl{
			mkMethod(tab, "show", hir.OwnRef, nil, tab.I32()),
		})
		reason, _ := IsObjectSafeWithTab(tr, tab)
		if reason != ObjectSafetyOK {
			t.Fatalf("ref-self receiver rejected: %q", reason)
		}
	})

	t.Run("generic-method-rejected", func(t *testing.T) {
		m := mkMethod(tab, "apply", hir.OwnNone, nil, tab.Unit())
		m.Generics = []*hir.GenericParam{
			{Base: hir.Base{ID: "m::apply::T"}, Name: "T"},
		}
		tr := mkTrait(tab, "Applier", []*hir.FnDecl{m})
		reason, method := IsObjectSafeWithTab(tr, tab)
		if reason != ObjectSafetyGenericMethod {
			t.Fatalf("generic method should be rejected; got %q on %q", reason, method)
		}
	})

	t.Run("non-self-receiver-rejected", func(t *testing.T) {
		m := mkMethod(tab, "weird", hir.OwnNone, nil, tab.Unit())
		m.Params[0].Name = "x" // not `self`
		tr := mkTrait(tab, "Weird", []*hir.FnDecl{m})
		reason, _ := IsObjectSafeWithTab(tr, tab)
		if reason != ObjectSafetyBadReceiver {
			t.Fatalf("non-self receiver should be rejected; got %q", reason)
		}
	})

	t.Run("self-in-param-rejected", func(t *testing.T) {
		traitID := tab.Nominal(typetable.KindTrait, 501, "m", "Eq", nil)
		m := &hir.FnDecl{
			Base:   hir.Base{ID: hir.ItemID("m", "cmp")},
			Name:   "cmp",
			TypeID: tab.Fn([]typetable.TypeId{traitID, traitID}, tab.I32(), false),
			Params: []*hir.Param{
				{TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::cmp::self"}, Type: tab.Unit()}, Name: "self"},
				{TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::cmp::p0"}, Type: traitID}, Name: "other"},
			},
			Return: tab.I32(),
		}
		tr := &hir.TraitDecl{
			Base:   hir.Base{ID: hir.ItemID("m", "Eq")},
			Name:   "Eq",
			TypeID: traitID,
			Items:  []hir.Item{m},
		}
		reason, _ := IsObjectSafeWithTab(tr, tab)
		if reason != ObjectSafetySelfInNonRecv {
			t.Fatalf("Self in param slot should be rejected; got %q", reason)
		}
	})

	t.Run("assoc-const-rejected", func(t *testing.T) {
		// Synthesise an associated const by placing a
		// ConstDecl inside the trait's Items list. The
		// normal trait-decl shape wouldn't generate this
		// from source yet, but defending against a future
		// shape change keeps the rule honest.
		traitID := tab.Nominal(typetable.KindTrait, 502, "m", "C", nil)
		c := &hir.ConstDecl{
			Base: hir.Base{ID: hir.ItemID("m", "C::K")},
			Name: "K",
			Type: tab.I32(),
		}
		tr := &hir.TraitDecl{
			Base:   hir.Base{ID: hir.ItemID("m", "C")},
			Name:   "C",
			TypeID: traitID,
			Items:  []hir.Item{c},
		}
		reason, method := IsObjectSafeWithTab(tr, tab)
		if reason != ObjectSafetyAssocConstItem {
			t.Fatalf("assoc const should be rejected; got %q on %q", reason, method)
		}
		if method != "K" {
			t.Fatalf("expected the offending const to be named K; got %q", method)
		}
	})
}
