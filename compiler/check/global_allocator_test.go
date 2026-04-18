package check

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestGlobalAllocatorRecognition pins the W24 `@global_allocator`
// recognition contract (reference §46.3). Three shapes:
//
//  1. `@global_allocator` on a `static` → recognized; the lowered
//     hir.StaticDecl.GlobalAllocator flag is true; the checker
//     produces no diagnostic.
//  2. `@global_allocator` on a fn / struct / enum / const / union →
//     the bridge emits "`@global_allocator` only applies to `static`
//     items" (Rule 6.9 — no silent ignore).
//  3. Two `static`s both carrying `@global_allocator` in the same
//     program → the checker emits the "more than one"
//     diagnostic on the second one.
//
// Runtime dispatch (the actual replacement of the default heap
// allocator by the selected static's value) is W26 territory; W24
// only wires recognition, placement, and uniqueness.
func TestGlobalAllocatorRecognition(t *testing.T) {
	t.Run("recognized-on-static", func(t *testing.T) {
		prog, bridgeDiags := runToBridge(t, map[string]string{
			"m": `@global_allocator static MY_ALLOC: I32 = 0;`,
		})
		if len(bridgeDiags) != 0 {
			t.Fatalf("bridge diagnostics on recognized attribute: %v", bridgeDiags)
		}
		if len(Check(prog)) != 0 {
			t.Fatalf("unexpected check diagnostics on single @global_allocator")
		}
		var seen bool
		for _, it := range prog.Modules["m"].Items {
			if st, ok := it.(*hir.StaticDecl); ok && st.Name == "MY_ALLOC" {
				if !st.GlobalAllocator {
					t.Fatalf("StaticDecl.GlobalAllocator not set")
				}
				seen = true
			}
		}
		if !seen {
			t.Fatal("static MY_ALLOC missing from lowered module")
		}
	})

	t.Run("rejected-on-non-static", func(t *testing.T) {
		_, bridgeDiags := runToBridge(t, map[string]string{
			"m": `@global_allocator fn boot() -> I32 { return 0; }`,
		})
		if len(bridgeDiags) == 0 {
			t.Fatal("expected bridge diagnostic for @global_allocator on fn")
		}
		var hit bool
		for _, d := range bridgeDiags {
			if strings.Contains(d.Message, "only applies to `static`") {
				hit = true
				break
			}
		}
		if !hit {
			t.Fatalf("diagnostic does not name placement rule: %v", bridgeDiags)
		}
	})

	t.Run("duplicate-in-same-program", func(t *testing.T) {
		prog, bridgeDiags := runToBridge(t, map[string]string{
			"m": `
				@global_allocator static FIRST: I32 = 0;
				@global_allocator static SECOND: I32 = 0;
			`,
		})
		if len(bridgeDiags) != 0 {
			t.Fatalf("unexpected bridge diagnostics: %v", bridgeDiags)
		}
		diags := Check(prog)
		var hit bool
		for _, d := range diags {
			if strings.Contains(d.Message, "more than one `@global_allocator`") {
				hit = true
				break
			}
		}
		if !hit {
			t.Fatalf("expected duplicate diagnostic; got: %v", diags)
		}
	})

	t.Run("rejects-arguments", func(t *testing.T) {
		_, bridgeDiags := runToBridge(t, map[string]string{
			"m": `@global_allocator(system) static MY_ALLOC: I32 = 0;`,
		})
		var hit bool
		for _, d := range bridgeDiags {
			if strings.Contains(d.Message, "takes no arguments") {
				hit = true
				break
			}
		}
		if !hit {
			t.Fatalf("expected no-arguments diagnostic; got: %v", bridgeDiags)
		}
	})
}

// runToBridge drives parse → resolve → bridge, returning the bridge's
// Program and diagnostics. Unlike checkSource, this helper does NOT
// treat bridge diagnostics as fatal — it's useful for tests that
// specifically exercise bridge-level rejection paths.
func runToBridge(t *testing.T, files map[string]string) (*hir.Program, []Diagnostic) {
	t.Helper()
	var srcs []*resolve.SourceFile
	for modPath, src := range files {
		f, pd := parse.Parse(modPath+".fuse", []byte(src))
		if len(pd) != 0 {
			t.Fatalf("parse %s: %v", modPath, pd)
		}
		srcs = append(srcs, &resolve.SourceFile{ModulePath: modPath, File: f})
	}
	resolved, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		t.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, resolved, srcs).Run()
	return prog, bd
}
