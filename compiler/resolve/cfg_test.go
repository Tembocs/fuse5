package resolve

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/lex"
)

// TestCfgEvaluation covers every top-level @cfg form from reference
// §50.1: key/value, feature, not, all, any, plus combinations.
func TestCfgEvaluation(t *testing.T) {
	cases := []struct {
		name    string
		decor   string
		cfg     BuildConfig
		keep    bool
		wantErr string
	}{
		{"kv-match", `@cfg(os = "linux")`,
			BuildConfig{Vars: map[string]string{"os": "linux"}}, true, ""},
		{"kv-miss", `@cfg(os = "windows")`,
			BuildConfig{Vars: map[string]string{"os": "linux"}}, false, ""},
		{"kv-unset", `@cfg(os = "linux")`,
			BuildConfig{}, false, ""},
		{"feature-on", `@cfg(feature = "debug")`,
			BuildConfig{Features: map[string]bool{"debug": true}}, true, ""},
		{"feature-off", `@cfg(feature = "debug")`,
			BuildConfig{}, false, ""},
		{"not-true", `@cfg(not(os = "windows"))`,
			BuildConfig{Vars: map[string]string{"os": "linux"}}, true, ""},
		{"not-false", `@cfg(not(os = "linux"))`,
			BuildConfig{Vars: map[string]string{"os": "linux"}}, false, ""},
		{"all-true", `@cfg(all(os = "linux", feature = "debug"))`,
			BuildConfig{
				Vars:     map[string]string{"os": "linux"},
				Features: map[string]bool{"debug": true},
			}, true, ""},
		{"all-false", `@cfg(all(os = "linux", feature = "debug"))`,
			BuildConfig{Vars: map[string]string{"os": "linux"}}, false, ""},
		{"any-true", `@cfg(any(os = "windows", feature = "debug"))`,
			BuildConfig{Features: map[string]bool{"debug": true}}, true, ""},
		{"any-false", `@cfg(any(os = "windows", feature = "debug"))`,
			BuildConfig{}, false, ""},
		{"nested", `@cfg(all(not(os = "windows"), any(feature = "a", feature = "b")))`,
			BuildConfig{
				Vars:     map[string]string{"os": "linux"},
				Features: map[string]bool{"b": true},
			}, true, ""},
		{"malformed-bare-ident", `@cfg(bogus)`,
			BuildConfig{}, false, "malformed @cfg predicate"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := tc.decor + "\nfn guarded() {}\n"
			srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
			out, diags := Resolve(srcs, tc.cfg)
			if tc.wantErr != "" {
				if !diagContains(diags, tc.wantErr) {
					t.Fatalf("want diag %q, got %v", tc.wantErr, diagMsgs(diags))
				}
				return
			}
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics: %v", diagMsgs(diags))
			}
			m := out.Graph.Modules["m"]
			id := m.Scope.LookupLocal("guarded")
			if tc.keep && id == NoSymbol {
				t.Fatalf("expected item kept; not found in scope")
			}
			if !tc.keep && id != NoSymbol {
				t.Fatalf("expected item filtered out, but it is present")
			}
		})
	}
}

// TestCfgDuplicates asserts that two items with the same name survive
// under mutually exclusive predicates (one per build), and that both
// surviving triggers a diagnostic.
func TestCfgDuplicates(t *testing.T) {
	t.Run("mutually-exclusive-linux", func(t *testing.T) {
		src := `
@cfg(os = "linux")
fn current() {}

@cfg(os = "windows")
fn current() {}
`
		srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
		out, msgs := resolveStrings(t, srcs, BuildConfig{Vars: map[string]string{"os": "linux"}})
		if len(msgs) != 0 {
			t.Fatalf("unexpected diagnostics: %v", msgs)
		}
		if out.Graph.Modules["m"].Scope.LookupLocal("current") == NoSymbol {
			t.Fatalf("linux variant should have survived")
		}
	})
	t.Run("both-survive-is-diagnostic", func(t *testing.T) {
		src := `
@cfg(feature = "a")
fn current() {}

@cfg(feature = "b")
fn current() {}
`
		srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
		_, msgs := resolveStrings(t, srcs, BuildConfig{
			Features: map[string]bool{"a": true, "b": true},
		})
		if !hasSubstring(msgs, "duplicate item") {
			t.Fatalf("expected duplicate-item diagnostic, got %v", msgs)
		}
	})
	t.Run("no-predicate-conflict-still-reported", func(t *testing.T) {
		src := `
fn current() {}
fn current() {}
`
		srcs := []*SourceFile{mkSource(t, "m", "m.fuse", src)}
		_, msgs := resolveStrings(t, srcs, BuildConfig{})
		if !hasSubstring(msgs, "duplicate item") {
			t.Fatalf("expected duplicate-item diagnostic, got %v", msgs)
		}
	})
}

// diagMsgs extracts message strings from diagnostics for debug output.
func diagMsgs(diags []lex.Diagnostic) []string {
	out := make([]string, len(diags))
	for i, d := range diags {
		out[i] = d.Message
	}
	return out
}
