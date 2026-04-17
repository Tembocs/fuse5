package resolve

import (
	"testing"
)

// TestQualifiedEnumVariant exercises the §11.6 contract: `Enum.Variant`
// and `module.Enum.Variant` both resolve to the hoisted variant symbol.
func TestQualifiedEnumVariant(t *testing.T) {
	t.Run("local-enum", func(t *testing.T) {
		src := `
enum Dir { North, South }

fn at_north() -> Dir {
	return Dir.North;
}
`
		srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
		out, msgs := resolveStrings(t, srcs, BuildConfig{})
		if len(msgs) != 0 {
			t.Fatalf("unexpected diagnostics: %v", msgs)
		}
		// Find the variant symbol for North.
		m := out.Graph.Modules["m"]
		north := m.Scope.LookupLocal("North")
		if north == NoSymbol {
			t.Fatalf("North not hoisted")
		}
		// Confirm at least one PathExpr binding resolved to this variant.
		hit := false
		for _, id := range out.Bindings {
			if id == north {
				hit = true
				break
			}
		}
		if !hit {
			t.Fatalf("no binding resolved to North variant; bindings=%v", out.Bindings)
		}
	})

	t.Run("cross-module-enum", func(t *testing.T) {
		src1 := `pub enum Color { Red, Green, Blue }`
		src2 := `
import pal;

fn red_from_pal() -> pal.Color {
	return pal.Color.Red;
}
`
		srcs := []*SourceFile{
			mkSource(t, "", "lib.fuse", src2),
			mkSource(t, "pal", "pal.fuse", src1),
		}
		out, msgs := resolveStrings(t, srcs, BuildConfig{})
		if len(msgs) != 0 {
			t.Fatalf("unexpected diagnostics: %v", msgs)
		}
		pal := out.Graph.Modules["pal"]
		red := pal.Scope.LookupLocal("Red")
		if red == NoSymbol {
			t.Fatalf("Red not hoisted in pal")
		}
		// Confirm a binding records the Red variant lookup from the
		// root module.
		hit := false
		for key, id := range out.Bindings {
			if id == red && key.Module == "" {
				hit = true
				break
			}
		}
		if !hit {
			t.Fatalf("no root-module binding resolved to pal.Color.Red")
		}
	})

	t.Run("unknown-variant-is-diagnostic", func(t *testing.T) {
		src := `
enum Dir { North, South }
fn bad() -> Dir {
	return Dir.West;
}
`
		srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
		_, msgs := resolveStrings(t, srcs, BuildConfig{})
		if !hasSubstring(msgs, "no variant") {
			t.Fatalf("expected 'no variant' diagnostic, got %v", msgs)
		}
	})
}

// TestVisibilityEnforcement covers the four §53.1 levels: private,
// pub(mod), pub(pkg), pub.
func TestVisibilityEnforcement(t *testing.T) {
	cases := []struct {
		name    string
		srcs    []*SourceFile
		wantErr string // "" means clean
	}{
		{
			name: "private-blocks-cross-module",
			srcs: []*SourceFile{
				makeSrc("util", "util.fuse", "fn secret() {}"),
				makeSrc("user", "user.fuse", "import util;\nfn f() { util.secret(); }"),
			},
			wantErr: "not visible",
		},
		{
			name: "pub-allows-cross-module",
			srcs: []*SourceFile{
				makeSrc("util", "util.fuse", "pub fn greet() {}"),
				makeSrc("user", "user.fuse", "import util;\nfn f() { util.greet(); }"),
			},
			wantErr: "",
		},
		{
			name: "pub-mod-allows-descendant",
			srcs: []*SourceFile{
				makeSrc("util", "util.fuse", "pub(mod) fn helper() {}"),
				makeSrc("util.inner", "inner.fuse", "import util;\nfn g() { util.helper(); }"),
			},
			wantErr: "",
		},
		{
			name: "pub-mod-blocks-non-descendant",
			srcs: []*SourceFile{
				makeSrc("util", "util.fuse", "pub(mod) fn helper() {}"),
				makeSrc("other", "other.fuse", "import util;\nfn g() { util.helper(); }"),
			},
			wantErr: "not visible",
		},
		{
			name: "pub-pkg-allows-arbitrary-module",
			srcs: []*SourceFile{
				makeSrc("util", "util.fuse", "pub(pkg) fn helper() {}"),
				makeSrc("other", "other.fuse", "import util;\nfn g() { util.helper(); }"),
			},
			wantErr: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, msgs := resolveStringsBare(t, tc.srcs)
			if tc.wantErr == "" {
				for _, m := range msgs {
					if indexOf(m, "not visible") >= 0 {
						t.Fatalf("expected clean resolution, got %v", msgs)
					}
				}
				return
			}
			if !hasSubstring(msgs, tc.wantErr) {
				t.Fatalf("want diag containing %q, got %v", tc.wantErr, msgs)
			}
		})
	}
}
