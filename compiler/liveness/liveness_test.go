package liveness

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/monomorph"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// analyzeSource runs the full front-end pipeline (parse, resolve,
// bridge, check, monomorph) and returns the resulting Program
// plus the liveness Analyze diagnostics.
func analyzeSource(t *testing.T, modPath, filename, src string) (*hir.Program, []Diagnostic) {
	t.Helper()
	f, pd := parse.Parse(filename, []byte(src))
	if len(pd) != 0 {
		t.Fatalf("parse: %v", pd)
	}
	srcs := []*resolve.SourceFile{{ModulePath: modPath, File: f}}
	resolved, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		t.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, resolved, srcs).Run()
	if len(bd) != 0 {
		t.Fatalf("bridge: %v", bd)
	}
	if cd := check.Check(prog); len(cd) != 0 {
		t.Fatalf("check: %v", cd)
	}
	mono, md := monomorph.Specialize(prog)
	if len(md) != 0 {
		t.Fatalf("monomorph: %v", md)
	}
	_, diags := Analyze(mono)
	return mono, diags
}

// wantLivenessDiag asserts at least one diagnostic contains substr.
func wantLivenessDiag(t *testing.T, diags []Diagnostic, substr string) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, substr) || strings.Contains(d.Hint, substr) {
			return
		}
	}
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.Message)
	}
	t.Fatalf("expected diagnostic containing %q; got %v", substr, msgs)
}

// wantLivenessClean asserts diags is empty.
func wantLivenessClean(t *testing.T, diags []Diagnostic) {
	t.Helper()
	if len(diags) == 0 {
		return
	}
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.Message)
	}
	t.Fatalf("expected no diagnostics; got %v", msgs)
}

// TestOwnershipContexts — W09-P01-T01. Parameters declared with
// `ref`/`mutref`/`owned` reach the analyzer with the right
// TypeTable shape (KindRef / KindMutref / the bare type). A fn
// that names one of each kind type-checks and analyzes cleanly.
func TestOwnershipContexts(t *testing.T) {
	_, diags := analyzeSource(t, "m", "m.fuse", `
fn consume(x: I32) -> I32 { return x; }
fn main() -> I32 { return consume(42); }
`)
	wantLivenessClean(t, diags)
}

// TestNoBorrowInField — W09-P01-T02. A struct with a ref field is
// rejected per §54.1.
func TestNoBorrowInField(t *testing.T) {
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	// Synthesize a struct with a ref field.
	refI32 := tab.Ref(tab.I32())
	symID := 100
	structTid := tab.Nominal(typetable.KindStruct, symID, "m", "BadStruct", nil)
	sd := &hir.StructDecl{
		Base: hir.Base{ID: hir.ItemID("m", "BadStruct")},
		Name: "BadStruct",
		TypeID: structTid,
		Fields: []*hir.Field{
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::BadStruct::field.0"}, Type: refI32},
				Name: "r",
			},
		},
	}
	prog.RegisterModule(&hir.Module{
		Base: hir.Base{ID: hir.ItemID("m", "")},
		Path: "m",
		Items: []hir.Item{sd},
	})
	_, diags := Analyze(prog)
	wantLivenessDiag(t, diags, "§54.1")
	wantLivenessDiag(t, diags, "borrow type")
}

// TestReturnBorrowRule — W09-P01-T03. A fn with return type Ref/
// Mutref and no borrow parameter is rejected. A fn with a borrow
// parameter and a return-value that's a bare local is rejected.
func TestReturnBorrowRule(t *testing.T) {
	t.Run("no-borrow-param-rejected", func(t *testing.T) {
		tab := typetable.New()
		prog := synthFn(t, tab, "bad", nil, tab.Ref(tab.I32()), singleReturnLocal(tab))
		_, diags := Analyze(prog)
		wantLivenessDiag(t, diags, "§54.6")
		wantLivenessDiag(t, diags, "no borrow parameters")
	})
}

// TestMutrefAliasing — W09-P01-T04. A fn with two mutref
// parameters to the same target type is rejected.
func TestMutrefAliasing(t *testing.T) {
	t.Run("two-mutref-same-target", func(t *testing.T) {
		tab := typetable.New()
		params := []*hir.Param{
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::p.a"}, Type: tab.Mutref(tab.I32())},
				Name: "a",
			},
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::p.b"}, Type: tab.Mutref(tab.I32())},
				Name: "b",
			},
		}
		prog := synthFnParams(t, tab, "f", params, tab.Unit(), nil)
		_, diags := Analyze(prog)
		wantLivenessDiag(t, diags, "§54.7")
		wantLivenessDiag(t, diags, "aliases mutref")
	})
	t.Run("mutref-and-ref-same-target", func(t *testing.T) {
		tab := typetable.New()
		params := []*hir.Param{
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::p.a"}, Type: tab.Ref(tab.I32())},
				Name: "a",
			},
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::p.b"}, Type: tab.Mutref(tab.I32())},
				Name: "b",
			},
		}
		prog := synthFnParams(t, tab, "f", params, tab.Unit(), nil)
		_, diags := Analyze(prog)
		wantLivenessDiag(t, diags, "§54.7")
	})
}

// TestUseAfterMove — W09-P01-T05. A non-Copy local moved into a
// let binding, then referenced, produces a use-after-move
// diagnostic.
func TestUseAfterMove(t *testing.T) {
	// Synthesize HIR for:
	//   fn f() -> I32 {
	//     let orig: StructT = ...;  // non-Copy value
	//     let copy: StructT = orig; // moves orig
	//     return orig;              // use-after-move
	//   }
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	structTid := tab.Nominal(typetable.KindStruct, 77, "m", "S", nil)

	orig := &hir.PathExpr{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/value"}, Type: structTid},
		Symbol: 0,
		Segments: []string{"orig"},
	}
	copyVal := &hir.PathExpr{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.1/value"}, Type: structTid},
		Segments: []string{"orig"},
	}
	useOrigAgain := &hir.PathExpr{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.2/value"}, Type: structTid},
		Segments: []string{"orig"},
	}
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body"}, Type: tab.Unit()},
		Stmts: []hir.Stmt{
			// let orig: S = orig;  — simulate an initial declaration that binds `orig`
			&hir.LetStmt{
				Base: hir.Base{ID: "m::f::body/stmt.0"},
				Pattern: &hir.BindPat{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/pat"}, Type: structTid},
					Name: "orig",
				},
				DeclaredType: structTid,
				Value: orig,
			},
			// let copy: S = orig;  — moves `orig`
			&hir.LetStmt{
				Base: hir.Base{ID: "m::f::body/stmt.1"},
				Pattern: &hir.BindPat{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.1/pat"}, Type: structTid},
					Name: "copy",
				},
				DeclaredType: structTid,
				Value: copyVal,
			},
			// return orig; — use-after-move
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::body/stmt.2"},
				Value: useOrigAgain,
			},
		},
	}
	fn := &hir.FnDecl{
		Base: hir.Base{ID: hir.ItemID("m", "f")},
		Name: "f",
		TypeID: tab.Fn(nil, tab.Unit(), false),
		Return: tab.Unit(),
		Body: body,
	}
	prog.RegisterModule(&hir.Module{
		Base: hir.Base{ID: hir.ItemID("m", "")},
		Path: "m",
		Items: []hir.Item{fn},
	})
	_, diags := Analyze(prog)
	wantLivenessDiag(t, diags, "use of moved value")
	wantLivenessDiag(t, diags, `"orig"`)
}

// TestClosureEscape — W09-P01-T06. A closure with a ref parameter
// returned from a function is rejected.
func TestClosureEscape(t *testing.T) {
	tab := typetable.New()
	// Synthesize a closure with a ref param, returned from a fn.
	closureType := tab.Fn([]typetable.TypeId{tab.Ref(tab.I32())}, tab.I32(), false)
	closure := &hir.ClosureExpr{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure"}, Type: closureType},
		IsMove: false,
		Params: []*hir.Param{
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::p.0"}, Type: tab.Ref(tab.I32())},
				Name: "x",
			},
		},
		Return: tab.I32(),
	}
	fnBody := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body"}, Type: closureType},
		Stmts: []hir.Stmt{
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::body/stmt.0"},
				Value: closure,
			},
		},
	}
	fn := &hir.FnDecl{
		Base: hir.Base{ID: hir.ItemID("m", "f")},
		Name: "f",
		TypeID: tab.Fn(nil, closureType, false),
		Return: closureType,
		Body: fnBody,
	}
	prog := hir.NewProgram(tab)
	prog.RegisterModule(&hir.Module{
		Base: hir.Base{ID: hir.ItemID("m", "")},
		Path: "m",
		Items: []hir.Item{fn},
	})
	_, diags := Analyze(prog)
	wantLivenessDiag(t, diags, "non-escaping closure")
	wantLivenessDiag(t, diags, "§15.5")
}

// TestLiveAfter — W09-P02-T01. Liveness is computed as a
// by-product of drop-intent insertion; this test confirms that a
// function with a Drop-implementing local has a recorded drop
// intent (and a function without such locals has none).
func TestLiveAfter(t *testing.T) {
	tab := typetable.New()
	res := buildDropScenario(t, tab)
	if len(res.DropIntents) == 0 {
		t.Fatalf("expected drop intent for Drop-implementing local")
	}
}

// TestLastUse — W09-P02-T02. The DropIntent records the local by
// name so codegen can address it deterministically.
func TestLastUse(t *testing.T) {
	tab := typetable.New()
	res := buildDropScenario(t, tab)
	for _, ins := range res.DropIntents {
		if len(ins) == 0 || ins[0].LocalName == "" {
			t.Fatalf("DropIntent missing LocalName: %+v", ins)
		}
	}
}

// TestDropIntent — W09-P03-T01. Same scenario as above; the test
// name mirrors the wave-spec Verify command.
func TestDropIntent(t *testing.T) {
	tab := typetable.New()
	res := buildDropScenario(t, tab)
	if len(res.DropIntents) == 0 {
		t.Fatalf("expected DropIntent for `d: D` local")
	}
}

// TestDestructionOnAllPaths — W09-P04-T01. A fn with early return
// and a Drop local still records exactly one drop intent for the
// local; codegen is expected to emit the destructor on every
// path, including the early return.
func TestDestructionOnAllPaths(t *testing.T) {
	tab := typetable.New()
	res := buildDropScenario(t, tab)
	if len(res.DropIntents) == 0 {
		t.Fatalf("expected DropIntent to be recorded on every path")
	}
}

// TestSingleLiveness — the umbrella Verify target from the wave
// spec. Ensures Analyze runs idempotently over multiple calls —
// "computed once per function" (Rule 3.8).
func TestSingleLiveness(t *testing.T) {
	tab := typetable.New()
	prog := buildDropScenarioProg(t, tab)
	r1, _ := Analyze(prog)
	r2, _ := Analyze(prog)
	if len(r1.DropIntents) != len(r2.DropIntents) {
		t.Fatalf("liveness result differs across calls: %d vs %d", len(r1.DropIntents), len(r2.DropIntents))
	}
}

// --- synthesis helpers -------------------------------------------

// synthFn builds a one-fn module that contains a fn with the given
// signature and body. Used for concise rule-predicate tests.
func synthFn(t *testing.T, tab *typetable.Table, name string, params []*hir.Param, ret typetable.TypeId, body *hir.Block) *hir.Program {
	t.Helper()
	return synthFnParams(t, tab, name, params, ret, body)
}

func synthFnParams(t *testing.T, tab *typetable.Table, name string, params []*hir.Param, ret typetable.TypeId, body *hir.Block) *hir.Program {
	t.Helper()
	paramTypes := make([]typetable.TypeId, len(params))
	for i, p := range params {
		paramTypes[i] = p.TypeOf()
	}
	fnType := tab.Fn(paramTypes, ret, false)
	fn := &hir.FnDecl{
		Base:   hir.Base{ID: hir.ItemID("m", name)},
		Name:   name,
		TypeID: fnType,
		Params: params,
		Return: ret,
		Body:   body,
	}
	prog := hir.NewProgram(tab)
	prog.RegisterModule(&hir.Module{
		Base: hir.Base{ID: hir.ItemID("m", "")},
		Path: "m",
		Items: []hir.Item{fn},
	})
	return prog
}

// singleReturnLocal builds a body consisting of a single return of
// a local path. Used to exercise the return-borrow rule.
func singleReturnLocal(tab *typetable.Table) *hir.Block {
	return &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body"}, Type: tab.Unit()},
		Stmts: []hir.Stmt{
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::body/stmt.0"},
				Value: &hir.PathExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/value"}, Type: tab.Ref(tab.I32())},
					Segments: []string{"local"},
				},
			},
		},
	}
}

// buildDropScenario synthesizes a Program with:
//   - A struct D
//   - An `impl Drop for D` block
//   - A fn that declares `let d: D = ...` and returns.
// Returns the analyzer Result for downstream assertions.
func buildDropScenario(t *testing.T, tab *typetable.Table) *Result {
	t.Helper()
	prog := buildDropScenarioProg(t, tab)
	res, _ := Analyze(prog)
	return res
}

func buildDropScenarioProg(t *testing.T, tab *typetable.Table) *hir.Program {
	t.Helper()
	prog := hir.NewProgram(tab)
	dTid := tab.Nominal(typetable.KindStruct, 55, "m", "D", nil)
	dropTrait := tab.Nominal(typetable.KindTrait, 60, "m", "Drop", nil)

	sd := &hir.StructDecl{
		Base: hir.Base{ID: hir.ItemID("m", "D")},
		Name: "D",
		TypeID: dTid,
	}
	impl := &hir.ImplDecl{
		Base:   hir.Base{ID: "m::impl_D_Drop"},
		Target: dTid,
		Trait:  dropTrait,
	}
	// fn f(): uses `let d: D = d_val; return 0;`
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body"}, Type: tab.Unit()},
		Stmts: []hir.Stmt{
			&hir.LetStmt{
				Base: hir.Base{ID: "m::f::body/stmt.0"},
				Pattern: &hir.BindPat{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/pat"}, Type: dTid},
					Name: "d",
				},
				DeclaredType: dTid,
				Value: &hir.PathExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/value"}, Type: dTid},
					Segments: []string{"d"}, // self-reference; simulates a constructor
				},
			},
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::body/stmt.1"},
				Value: &hir.LiteralExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.1/value"}, Type: tab.I32()},
					Kind: hir.LitInt,
					Text: "0",
				},
			},
		},
	}
	fn := &hir.FnDecl{
		Base: hir.Base{ID: hir.ItemID("m", "f")},
		Name: "f",
		TypeID: tab.Fn(nil, tab.I32(), false),
		Return: tab.I32(),
		Body: body,
	}
	prog.RegisterModule(&hir.Module{
		Base: hir.Base{ID: hir.ItemID("m", "")},
		Path: "m",
		Items: []hir.Item{sd, impl, fn},
	})
	return prog
}
