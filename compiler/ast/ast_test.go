package ast

import (
	"reflect"
	"testing"

	"github.com/Tembocs/fuse5/compiler/lex"
)

// TestAstNodeCompleteness asserts that the registry lists cover every
// concrete node kind, every listed node implements the expected marker
// interface, and node names are unique. Adding a new AST struct requires
// appending it to one of the All*Nodes lists — this test is the forcing
// function against silently unregistered nodes.
func TestAstNodeCompleteness(t *testing.T) {
	type check struct {
		name  string
		list  []Node
		iface reflect.Type
	}
	toNodes := func(ns interface{}) []Node {
		v := reflect.ValueOf(ns)
		out := make([]Node, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = v.Index(i).Interface().(Node)
		}
		return out
	}

	cases := []check{
		{"Item", toNodes(AllItemNodes), reflect.TypeOf((*Item)(nil)).Elem()},
		{"Expr", toNodes(AllExprNodes), reflect.TypeOf((*Expr)(nil)).Elem()},
		{"Stmt", toNodes(AllStmtNodes), reflect.TypeOf((*Stmt)(nil)).Elem()},
		{"Pat", toNodes(AllPatNodes), reflect.TypeOf((*Pat)(nil)).Elem()},
		{"Type", toNodes(AllTypeNodes), reflect.TypeOf((*Type)(nil)).Elem()},
	}

	seen := make(map[string]string) // struct name -> registry category
	for _, c := range cases {
		if len(c.list) == 0 {
			t.Errorf("%s registry is empty", c.name)
		}
		for _, n := range c.list {
			rt := reflect.TypeOf(n)
			if !rt.Implements(c.iface) {
				t.Errorf("%s registry: %s does not implement %s", c.name, rt.String(), c.name)
			}
			name := rt.String()
			if prior, dup := seen[name]; dup {
				t.Errorf("%s appears in both %s and %s registries", name, prior, c.name)
				continue
			}
			seen[name] = c.name
			// NodeSpan must not panic on a zero-value instance.
			_ = n.NodeSpan()
		}
	}

	// Grammar-production spot check: every headline grammar production from
	// reference Appendix C must have an AST type. The table below maps each
	// production name to the node type that represents it.
	productions := []struct {
		production string
		haveName   string
	}{
		{"fn_decl", "*ast.FnDecl"},
		{"struct_decl", "*ast.StructDecl"},
		{"enum_decl", "*ast.EnumDecl"},
		{"trait_decl", "*ast.TraitDecl"},
		{"impl_decl", "*ast.ImplDecl"},
		{"const_decl", "*ast.ConstDecl"},
		{"static_decl", "*ast.StaticDecl"},
		{"type_decl", "*ast.TypeDecl"},
		{"extern_decl", "*ast.ExternDecl"},
		{"union_decl", "*ast.UnionDecl"},
		{"block_expr", "*ast.BlockExpr"},
		{"if_expr", "*ast.IfExpr"},
		{"match_expr", "*ast.MatchExpr"},
		{"loop_expr", "*ast.LoopExpr"},
		{"while_expr", "*ast.WhileExpr"},
		{"for_expr", "*ast.ForExpr"},
		{"closure_expr", "*ast.ClosureExpr"},
		{"spawn_expr", "*ast.SpawnExpr"},
		{"unsafe_block", "*ast.UnsafeExpr"},
		{"struct_lit", "*ast.StructLitExpr"},
		{"tuple_expr", "*ast.TupleExpr"},
		{"ctor_pat", "*ast.CtorPat"},
		{"or_pat", "*ast.OrPat"},
		{"range_pat", "*ast.RangePat"},
		{"at_pat", "*ast.AtPat"},
		{"tuple_type", "*ast.TupleType"},
		{"array_type", "*ast.ArrayType"},
		{"slice_type", "*ast.SliceType"},
		{"ptr_type", "*ast.PtrType"},
		{"fn_type", "*ast.FnType"},
		{"dyn_type", "*ast.DynType"},
		{"impl_type", "*ast.ImplType"},
		{"unit_type", "*ast.UnitType"},
	}
	for _, p := range productions {
		if _, ok := seen[p.haveName]; !ok {
			t.Errorf("grammar production %q has no corresponding AST type (looked for %s)", p.production, p.haveName)
		}
	}
}

// TestSpanCorrectness asserts that every node in the registry returns the
// span it was built with. Builders in the parser are covered by the parser
// tests; this test guards against a structural regression where a new node
// type forgets to embed NodeBase.
func TestSpanCorrectness(t *testing.T) {
	want := lex.Span{
		File:  "sentinel.fuse",
		Start: lex.Position{Offset: 1, Line: 2, Column: 3},
		End:   lex.Position{Offset: 10, Line: 2, Column: 12},
	}

	all := []Node{}
	for _, n := range AllItemNodes {
		all = append(all, n)
	}
	for _, n := range AllExprNodes {
		all = append(all, n)
	}
	for _, n := range AllStmtNodes {
		all = append(all, n)
	}
	for _, n := range AllPatNodes {
		all = append(all, n)
	}
	for _, n := range AllTypeNodes {
		all = append(all, n)
	}

	for _, n := range all {
		// Reflectively set NodeBase.Span on each pointer, then read it back.
		v := reflect.ValueOf(n).Elem()
		base := v.FieldByName("NodeBase")
		if !base.IsValid() {
			t.Errorf("%T has no NodeBase field; every node must embed NodeBase", n)
			continue
		}
		spanField := base.FieldByName("Span")
		if !spanField.IsValid() || !spanField.CanSet() {
			t.Errorf("%T: NodeBase.Span is not settable", n)
			continue
		}
		spanField.Set(reflect.ValueOf(want))
		got := n.NodeSpan()
		if got != want {
			t.Errorf("%T: NodeSpan() = %+v, want %+v", n, got, want)
		}
	}

	// Ident uses a plain Span (no NodeBase) — verify it still satisfies Node.
	i := Ident{Span: want, Name: "x"}
	if i.NodeSpan() != want {
		t.Errorf("Ident.NodeSpan() = %+v, want %+v", i.NodeSpan(), want)
	}
}
