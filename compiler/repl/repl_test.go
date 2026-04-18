package repl

import (
	"bytes"
	"strings"
	"testing"
)

// TestReplRoundTrip exercises the REPL's Eval path across the
// arithmetic / comparison / bool-logic surface the W18 scope
// declares. A round-trip is "input → canonical output string".
//
// Bound by:
//
//	go test ./compiler/repl/... -run TestReplRoundTrip -v
func TestReplRoundTrip(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"0", "0"},
		{"42", "42"},
		{"6 * 7", "42"},
		{"2 + 3 * 4", "14"},
		{"(2 + 3) * 4", "20"},
		{"100 / 7", "14"},
		{"100 % 7", "2"},
		{"-5 + 3", "-2"},
		{"1 == 1", "true"},
		{"1 == 2", "false"},
		{"1 < 2 && 3 > 2", "true"},
		{"!(1 == 1)", "false"},
		{"true || false", "true"},
		{"true && !false", "true"},
		{"0x10 + 0b10", "18"},
	}
	r := NewRepl(bytes.NewReader(nil), &bytes.Buffer{})
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := r.Eval(c.in)
			if err != nil {
				t.Fatalf("Eval(%q): %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("Eval(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestReplErrors confirms the evaluator surfaces well-formed
// diagnostics for bad input rather than panicking.
func TestReplErrors(t *testing.T) {
	r := NewRepl(bytes.NewReader(nil), &bytes.Buffer{})
	badCases := []struct {
		in, wantContain string
	}{
		{"1 / 0", "divide by zero"},
		{"1 @ 0", "unexpected character"},
		{"1 +", "unexpected end of input"},
		{"(1 + 2", "expected `)`"},
		{"1 + true", "operands must be int"},
	}
	for _, c := range badCases {
		t.Run(c.in, func(t *testing.T) {
			_, err := r.Eval(c.in)
			if err == nil {
				t.Fatalf("Eval(%q) unexpectedly succeeded", c.in)
			}
			if !strings.Contains(err.Error(), c.wantContain) {
				t.Errorf("Eval(%q) err = %q, want to contain %q", c.in, err.Error(), c.wantContain)
			}
		})
	}
}

// TestReplSessionTermination exercises the main loop quit shapes.
func TestReplSessionTermination(t *testing.T) {
	t.Run(":quit-exits-cleanly", func(t *testing.T) {
		in := bytes.NewReader([]byte(":quit\n"))
		out := &bytes.Buffer{}
		r := NewRepl(in, out)
		if err := r.Run(); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})
	t.Run("eof-exits-cleanly", func(t *testing.T) {
		in := bytes.NewReader([]byte("1 + 1\n")) // no :quit; EOF after one eval
		out := &bytes.Buffer{}
		r := NewRepl(in, out)
		if err := r.Run(); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if !strings.Contains(out.String(), "=> 2") {
			t.Errorf("expected `=> 2` in output, got %q", out.String())
		}
	})
}
