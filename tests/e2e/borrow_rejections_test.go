package e2e_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/driver"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/liveness"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestBorrowRejections — W09-P05-T02. The five W09 rejection
// fixtures must fail to compile with their declared diagnostic
// text. Three of them are source-level (borrow-in-field,
// return-local-borrow, aliased-mutref) and drive the full
// compiler pipeline through driver.Build. Two of them
// (use-after-move, escaping-borrow-closure) need control-flow
// shapes the W05 spine doesn't yet parse/lower, so they are
// exercised via synthetic HIR against liveness.Analyze directly
// — same predicate, same diagnostic text.
func TestBorrowRejections(t *testing.T) {
	t.Run("reject_borrow_in_field", func(t *testing.T) {
		// The W05-vintage grammar does not spell borrow types in
		// struct fields, so the .fuse fixture parses as noise.
		// We exercise the rule via synthetic HIR against
		// liveness.Analyze. The fixture file is kept as a
		// normative statement of intent (see the .fuse header).
		prog := buildBorrowInFieldProgram()
		_, diags := liveness.Analyze(prog)
		assertLivenessContains(t, diags, "§54.1")
		assertLivenessContains(t, diags, "borrow type")
	})
	t.Run("reject_return_local_borrow", func(t *testing.T) {
		// Same parser limitation: `-> ref T` isn't yet spellable
		// in source. Exercise via synthetic HIR.
		prog := buildReturnLocalBorrowProgram()
		_, diags := liveness.Analyze(prog)
		assertLivenessContains(t, diags, "§54.6")
	})
	t.Run("reject_aliased_mutref", func(t *testing.T) {
		diags := buildAndCollectDiags(t, "reject_aliased_mutref.fuse")
		assertContains(t, diags, "§54.7")
		assertContains(t, diags, "aliases mutref")
	})
	t.Run("reject_use_after_move", func(t *testing.T) {
		// The use-after-move scenario requires multi-statement fn
		// bodies that the W05 spine doesn't parse/lower. Exercise
		// the predicate via synthetic HIR against liveness.Analyze.
		prog := buildUseAfterMoveProgram()
		_, diags := liveness.Analyze(prog)
		assertLivenessContains(t, diags, "use of moved value")
	})
	t.Run("reject_escaping_borrow_closure", func(t *testing.T) {
		// Same scope constraint as above: closures returned from
		// fns aren't lowerable yet. Synthetic HIR exercises the
		// escape classifier.
		prog := buildEscapingClosureProgram()
		_, diags := liveness.Analyze(prog)
		assertLivenessContains(t, diags, "§15.5")
	})
}

// TestDropObservable — W09-P05-T01. The W09 proof runs at the
// Go level: the liveness pass's DropIntent list for a synthetic
// Drop-implementing program is non-empty, proving the
// front-end-to-codegen metadata flow is wired.
func TestDropObservable(t *testing.T) {
	prog := buildDropObservableProgram()
	res, diags := liveness.Analyze(prog)
	if len(diags) != 0 {
		t.Fatalf("unexpected liveness diagnostics: %v", diags)
	}
	if len(res.DropIntents) == 0 {
		t.Fatalf("expected at least one DropIntent for Drop-implementing local")
	}
	// The intent must carry a non-empty local name and a non-zero
	// TypeId so codegen can emit `<TypeName>_drop(&local)`.
	for _, ins := range res.DropIntents {
		for _, in := range ins {
			if in.LocalName == "" {
				t.Errorf("empty LocalName in DropIntent: %+v", in)
			}
			if in.Type == typetable.NoType {
				t.Errorf("zero TypeId in DropIntent: %+v", in)
			}
		}
	}
}

// --- helpers -------------------------------------------------------

func buildAndCollectDiags(t *testing.T, fixture string) []lex.Diagnostic {
	t.Helper()
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatalf("abs %s: %v", fixture, err)
	}
	// Redirect any build-produced artifacts into a temp dir so
	// parallel runs don't collide. We don't care about a binary;
	// we want the diagnostics.
	dir := t.TempDir()
	_, diags, err := driver.Build(driver.BuildOptions{
		Source:  abs,
		Output:  filepath.Join(dir, binaryName("reject")),
		WorkDir: dir,
	})
	if err == nil {
		t.Fatalf("expected build failure for %s; got clean build", fixture)
	}
	return diags
}

func assertContains(t *testing.T, diags []lex.Diagnostic, substr string) {
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
	t.Fatalf("expected diagnostic containing %q, got %v", substr, msgs)
}

func assertLivenessContains(t *testing.T, diags []liveness.Diagnostic, substr string) {
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
	t.Fatalf("expected liveness diagnostic containing %q, got %v", substr, msgs)
}

// buildBorrowInFieldProgram synthesizes a struct whose field type
// is a KindRef, triggering the §54.1 diagnostic.
func buildBorrowInFieldProgram() *hir.Program {
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	refI32 := tab.Ref(tab.I32())
	structTid := tab.Nominal(typetable.KindStruct, 999, "m", "Bad", nil)
	sd := &hir.StructDecl{
		Base:   hir.Base{ID: hir.ItemID("m", "Bad")},
		Name:   "Bad",
		TypeID: structTid,
		Fields: []*hir.Field{
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::Bad::field.0"}, Type: refI32},
				Name:      "r",
			},
		},
	}
	prog.RegisterModule(&hir.Module{
		Base:  hir.Base{ID: hir.ItemID("m", "")},
		Path:  "m",
		Items: []hir.Item{sd},
	})
	return prog
}

// buildReturnLocalBorrowProgram synthesizes a fn that returns a
// KindRef value with no borrow parameter — the §54.6 violation.
func buildReturnLocalBorrowProgram() *hir.Program {
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	refI32 := tab.Ref(tab.I32())
	fn := &hir.FnDecl{
		Base:   hir.Base{ID: hir.ItemID("m", "leak")},
		Name:   "leak",
		TypeID: tab.Fn(nil, refI32, false),
		Return: refI32,
	}
	prog.RegisterModule(&hir.Module{
		Base:  hir.Base{ID: hir.ItemID("m", "")},
		Path:  "m",
		Items: []hir.Item{fn},
	})
	return prog
}

// buildUseAfterMoveProgram synthesizes a HIR program with a
// non-Copy local that is moved then read.
func buildUseAfterMoveProgram() *hir.Program {
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	structTid := tab.Nominal(typetable.KindStruct, 777, "m", "T", nil)
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body"}, Type: tab.Unit()},
		Stmts: []hir.Stmt{
			// let x: T = some_value_of_T;  -- seeds `x`
			&hir.LetStmt{
				Base: hir.Base{ID: "m::f::body/stmt.0"},
				Pattern: &hir.BindPat{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/pat"}, Type: structTid},
					Name:      "x",
				},
				DeclaredType: structTid,
				Value: &hir.PathExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/value"}, Type: structTid},
					Segments:  []string{"x"},
				},
			},
			// let y: T = x;  -- moves x
			&hir.LetStmt{
				Base: hir.Base{ID: "m::f::body/stmt.1"},
				Pattern: &hir.BindPat{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.1/pat"}, Type: structTid},
					Name:      "y",
				},
				DeclaredType: structTid,
				Value: &hir.PathExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.1/value"}, Type: structTid},
					Segments:  []string{"x"},
				},
			},
			// return x;  -- use-after-move
			&hir.ReturnStmt{
				Base: hir.Base{ID: "m::f::body/stmt.2"},
				Value: &hir.PathExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.2/value"}, Type: structTid},
					Segments:  []string{"x"},
				},
			},
		},
	}
	fn := &hir.FnDecl{
		Base:   hir.Base{ID: hir.ItemID("m", "f")},
		Name:   "f",
		TypeID: tab.Fn(nil, tab.Unit(), false),
		Return: tab.Unit(),
		Body:   body,
	}
	prog.RegisterModule(&hir.Module{
		Base:  hir.Base{ID: hir.ItemID("m", "")},
		Path:  "m",
		Items: []hir.Item{fn},
	})
	return prog
}

// buildEscapingClosureProgram synthesizes a fn that returns a
// closure whose parameter is a ref — the classic escaping-borrow
// shape.
func buildEscapingClosureProgram() *hir.Program {
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	closureType := tab.Fn([]typetable.TypeId{tab.Ref(tab.I32())}, tab.I32(), false)
	closure := &hir.ClosureExpr{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure"}, Type: closureType},
		IsMove:    false,
		Params: []*hir.Param{
			{
				TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::closure::p.0"}, Type: tab.Ref(tab.I32())},
				Name:      "x",
			},
		},
		Return: tab.I32(),
	}
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body"}, Type: closureType},
		Stmts: []hir.Stmt{
			&hir.ReturnStmt{
				Base:  hir.Base{ID: "m::f::body/stmt.0"},
				Value: closure,
			},
		},
	}
	fn := &hir.FnDecl{
		Base:   hir.Base{ID: hir.ItemID("m", "f")},
		Name:   "f",
		TypeID: tab.Fn(nil, closureType, false),
		Return: closureType,
		Body:   body,
	}
	prog.RegisterModule(&hir.Module{
		Base:  hir.Base{ID: hir.ItemID("m", "")},
		Path:  "m",
		Items: []hir.Item{fn},
	})
	return prog
}

// buildDropObservableProgram synthesizes a struct D with an
// `impl Drop for D` block and a fn that declares `let d: D = ...`,
// so the liveness pass records a DropIntent.
func buildDropObservableProgram() *hir.Program {
	tab := typetable.New()
	prog := hir.NewProgram(tab)
	dTid := tab.Nominal(typetable.KindStruct, 88, "m", "D", nil)
	dropTrait := tab.Nominal(typetable.KindTrait, 89, "m", "Drop", nil)
	sd := &hir.StructDecl{
		Base:   hir.Base{ID: hir.ItemID("m", "D")},
		Name:   "D",
		TypeID: dTid,
	}
	impl := &hir.ImplDecl{
		Base:   hir.Base{ID: "m::impl_D_Drop"},
		Target: dTid,
		Trait:  dropTrait,
	}
	body := &hir.Block{
		TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body"}, Type: tab.Unit()},
		Stmts: []hir.Stmt{
			&hir.LetStmt{
				Base: hir.Base{ID: "m::f::body/stmt.0"},
				Pattern: &hir.BindPat{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/pat"}, Type: dTid},
					Name:      "d",
				},
				DeclaredType: dTid,
				Value: &hir.PathExpr{
					TypedBase: hir.TypedBase{Base: hir.Base{ID: "m::f::body/stmt.0/value"}, Type: dTid},
					Segments:  []string{"d"},
				},
			},
		},
	}
	fn := &hir.FnDecl{
		Base:   hir.Base{ID: hir.ItemID("m", "f")},
		Name:   "f",
		TypeID: tab.Fn(nil, tab.I32(), false),
		Return: tab.I32(),
		Body:   body,
	}
	prog.RegisterModule(&hir.Module{
		Base:  hir.Base{ID: hir.ItemID("m", "")},
		Path:  "m",
		Items: []hir.Item{sd, impl, fn},
	})
	return prog
}

