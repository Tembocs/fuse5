package lower

import (
	"reflect"
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// mkClosure is a small constructor for synthetic ClosureExpr
// fixtures used across the W12 analysis tests.
func mkClosure(tab *typetable.Table, isMove bool, body *hir.Block, params ...*hir.Param) *hir.ClosureExpr {
	paramTypes := make([]typetable.TypeId, 0, len(params))
	for _, p := range params {
		paramTypes = append(paramTypes, p.TypeOf())
	}
	return &hir.ClosureExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: "m::f::closure"},
			Type: tab.Fn(paramTypes, tab.I32(), false),
		},
		IsMove: isMove,
		Params: params,
		Return: tab.I32(),
		Body:   body,
	}
}

// mkPath creates a single-segment PathExpr naming `name`.
func mkPath(tab *typetable.Table, name string, tid typetable.TypeId) *hir.PathExpr {
	return &hir.PathExpr{
		TypedBase: hir.TypedBase{
			Base: hir.Base{ID: hir.NodeID("m::f::path:" + name)},
			Type: tid,
		},
		Segments: []string{name},
	}
}

// TestCaptureAnalysis — W12-P01-T01. Reads of a Copy outer
// binding classify as Copy; writes classify as Mutref; reads of
// a non-Copy outer binding classify as Ref.
func TestCaptureAnalysis(t *testing.T) {
	tab := typetable.New()
	outer := map[string]typetable.TypeId{
		"n":   tab.I32(),                                               // Copy primitive
		"owned": tab.Nominal(typetable.KindStruct, 1, "m", "S", nil),   // non-Copy nominal
	}

	t.Run("read-copy-outer", func(t *testing.T) {
		// Body: `return n;`
		body := &hir.Block{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body"}, Type: tab.I32()},
			Stmts: []hir.Stmt{
				&hir.ReturnStmt{
					Base:  hir.Base{ID: "m::f::closure::body/stmt.0"},
					Value: mkPath(tab, "n", tab.I32()),
				},
			},
		}
		c := mkClosure(tab, false, body)
		a := AnalyzeClosure(c, "m::f", outer, tab)
		if len(a.Captures) != 1 || a.Captures[0].Name != "n" {
			t.Fatalf("expected one capture of `n`, got %+v", a.Captures)
		}
		if a.Captures[0].Mode != CaptureCopy {
			t.Fatalf("expected CaptureCopy for I32 read, got %v", a.Captures[0].Mode)
		}
	})

	t.Run("read-non-copy-outer", func(t *testing.T) {
		body := &hir.Block{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body"}, Type: tab.I32()},
			Stmts: []hir.Stmt{
				&hir.ReturnStmt{
					Base:  hir.Base{ID: "m::f::closure::body/stmt.0"},
					Value: mkPath(tab, "owned", outer["owned"]),
				},
			},
		}
		c := mkClosure(tab, false, body)
		a := AnalyzeClosure(c, "m::f", outer, tab)
		if len(a.Captures) != 1 || a.Captures[0].Name != "owned" {
			t.Fatalf("expected one capture of `owned`, got %+v", a.Captures)
		}
		if a.Captures[0].Mode != CaptureRef {
			t.Fatalf("expected CaptureRef for non-Copy read, got %v", a.Captures[0].Mode)
		}
	})

	t.Run("write-outer", func(t *testing.T) {
		// Body: `n = 5;`
		body := &hir.Block{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body"}, Type: tab.Unit()},
			Stmts: []hir.Stmt{
				&hir.ExprStmt{
					Base: hir.Base{ID: "m::f::closure::body/stmt.0"},
					Expr: &hir.AssignExpr{
						TypedBase: hir.TypedBase{
							Base: hir.Base{ID: "m::f::closure::body/stmt.0/assign"},
							Type: tab.Unit(),
						},
						Op:  hir.AssignEq,
						Lhs: mkPath(tab, "n", tab.I32()),
						Rhs: &hir.LiteralExpr{
							TypedBase: hir.TypedBase{
								Base: hir.Base{ID: "m::f::closure::body/stmt.0/assign/rhs"},
								Type: tab.I32(),
							},
							Kind: hir.LitInt,
							Text: "5",
						},
					},
				},
			},
		}
		c := mkClosure(tab, false, body)
		a := AnalyzeClosure(c, "m::f", outer, tab)
		if len(a.Captures) != 1 || a.Captures[0].Mode != CaptureMutref {
			t.Fatalf("expected CaptureMutref on assignment to `n`, got %+v", a.Captures)
		}
	})

	t.Run("closure-param-shadows-outer", func(t *testing.T) {
		// The closure declares its own `n` param; body reads it.
		nParam := &hir.Param{
			TypedBase: hir.TypedBase{
				Base: hir.Base{ID: "m::f::closure::p.n"},
				Type: tab.I32(),
			},
			Name: "n",
		}
		body := &hir.Block{
			TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body"}, Type: tab.I32()},
			Stmts: []hir.Stmt{
				&hir.ReturnStmt{
					Base:  hir.Base{ID: "m::f::closure::body/stmt.0"},
					Value: mkPath(tab, "n", tab.I32()),
				},
			},
		}
		c := mkClosure(tab, false, body, nParam)
		a := AnalyzeClosure(c, "m::f", outer, tab)
		if len(a.Captures) != 0 {
			t.Fatalf("expected zero captures (param shadows outer), got %+v", a.Captures)
		}
	})
}

// TestMoveClosurePrefix — W12-P01-T02. A `move` closure
// reclassifies every capture as Owned.
func TestMoveClosurePrefix(t *testing.T) {
	tab := typetable.New()
	outer := map[string]typetable.TypeId{
		"n":   tab.I32(),
		"box": tab.Nominal(typetable.KindStruct, 2, "m", "B", nil),
	}
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body"}, Type: tab.I32()},
		Stmts: []hir.Stmt{
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::closure::body/stmt.0"},
				Value: &hir.BinaryExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body/stmt.0/bin"}, Type: tab.I32()},
					Op:        hir.BinAdd,
					Lhs:       mkPath(tab, "n", tab.I32()),
					Rhs: &hir.LiteralExpr{
						TypedBase: hir.TypedBase{
							Base: hir.Base{ID: "m::f::closure::body/stmt.0/bin/rhs"},
							Type: tab.I32(),
						},
						Kind: hir.LitInt,
						Text: "1",
					},
				},
			},
		},
	}
	c := mkClosure(tab, true /* move */, body)
	a := AnalyzeClosure(c, "m::f", outer, tab)
	for _, cap := range a.Captures {
		if cap.Mode != CaptureOwned {
			t.Fatalf("move closure should own every capture; got %s for %q", cap.Mode, cap.Name)
		}
	}
}

// TestEscapeClassification — W12-P01-T03. A closure with any
// ref/mutref capture is non-escaping; otherwise escaping.
func TestEscapeClassification(t *testing.T) {
	tab := typetable.New()
	t.Run("owned-captures-escape", func(t *testing.T) {
		caps := []Capture{{Name: "n", Mode: CaptureOwned}}
		if classifyEscape(caps) != EscapeEscaping {
			t.Fatalf("owned-only env must be escaping")
		}
	})
	t.Run("ref-capture-non-escape", func(t *testing.T) {
		caps := []Capture{{Name: "r", Mode: CaptureRef}}
		if classifyEscape(caps) != EscapeNonEscape {
			t.Fatalf("ref capture must be non-escaping")
		}
	})
	t.Run("mutref-capture-non-escape", func(t *testing.T) {
		caps := []Capture{{Name: "m", Mode: CaptureMutref}}
		if classifyEscape(caps) != EscapeNonEscape {
			t.Fatalf("mutref capture must be non-escaping")
		}
	})
	t.Run("copy-only-escapes", func(t *testing.T) {
		caps := []Capture{{Name: "c", Mode: CaptureCopy}}
		if classifyEscape(caps) != EscapeEscaping {
			t.Fatalf("Copy-only env must be escaping")
		}
	})
	_ = tab
}

// TestClosureLifting — W12-P01-T04. The LiftedShape carries the
// env struct + closure params + a derived fn name.
func TestClosureLifting(t *testing.T) {
	tab := typetable.New()
	outer := map[string]typetable.TypeId{"x": tab.I32(), "y": tab.I32()}
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body"}, Type: tab.I32()},
		Stmts: []hir.Stmt{
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::closure::body/stmt.0"},
				Value: &hir.BinaryExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body/stmt.0/bin"}, Type: tab.I32()},
					Op:        hir.BinAdd,
					Lhs:       mkPath(tab, "x", tab.I32()),
					Rhs:       mkPath(tab, "y", tab.I32()),
				},
			},
		},
	}
	c := mkClosure(tab, false, body)
	a := AnalyzeClosure(c, "m::f", outer, tab)
	// Env has two fields, sorted by name.
	wantFields := []EnvField{
		{Name: "x", Mode: CaptureCopy},
		{Name: "y", Mode: CaptureCopy},
	}
	if !reflect.DeepEqual(a.Lifted.Env.Fields, wantFields) {
		t.Fatalf("lifted env fields = %+v, want %+v", a.Lifted.Env.Fields, wantFields)
	}
	if a.Lifted.FnName == "" {
		t.Fatalf("lifted fn name is empty")
	}
}

// TestClosureConstruction — W12-P01-T05. Two passes over the
// same closure body produce identical analyses (determinism).
func TestClosureConstruction(t *testing.T) {
	tab := typetable.New()
	outer := map[string]typetable.TypeId{"a": tab.I32(), "b": tab.I32()}
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body"}, Type: tab.I32()},
		Stmts: []hir.Stmt{
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::closure::body/stmt.0"},
				Value: &hir.BinaryExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::body/stmt.0/bin"}, Type: tab.I32()},
					Op:        hir.BinAdd,
					Lhs:       mkPath(tab, "a", tab.I32()),
					Rhs:       mkPath(tab, "b", tab.I32()),
				},
			},
		},
	}
	c := mkClosure(tab, false, body)
	a1 := AnalyzeClosure(c, "m::f", outer, tab)
	a2 := AnalyzeClosure(c, "m::f", outer, tab)
	if !reflect.DeepEqual(a1.Captures, a2.Captures) {
		t.Fatalf("capture set non-deterministic: %v vs %v", a1.Captures, a2.Captures)
	}
	if a1.Escape != a2.Escape {
		t.Fatalf("escape class non-deterministic: %v vs %v", a1.Escape, a2.Escape)
	}
}

// TestCallDesugar — W12-P02-T03. The call-desugaring map picks
// `call` / `call_mut` / `call_once` based on the tightest
// callable trait.
func TestCallDesugar(t *testing.T) {
	cases := []struct {
		trait CallableTrait
		want  string
	}{
		{CallableFn, "call"},
		{CallableFnMut, "call_mut"},
		{CallableFnOnce, "call_once"},
	}
	for _, tc := range cases {
		if got := DesugarCall(tc.trait); got != tc.want {
			t.Errorf("DesugarCall(%v) = %q, want %q", tc.trait, got, tc.want)
		}
	}
}
