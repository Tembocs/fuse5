package lex

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// updateGoldens is set via `-update` to regenerate the .tokens golden files
// after an intentional change. Rule 6.2 requires goldens to be byte-stable;
// updates must be a deliberate workflow, not a side effect of running tests.
var updateGoldens = flag.Bool("update", false, "rewrite golden files under testdata/")

// TestGolden scans every testdata/*.fuse file and compares the token stream
// against the committed testdata/*.tokens golden. Running
//
//	go test ./compiler/lex/... -run TestGolden -count=3
//
// proves byte-stability: the same input must produce the same rendered token
// stream across repeated runs. Updates require `-update`.
func TestGolden(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join("testdata", "*.fuse"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no testdata/*.fuse fixtures found")
	}
	for _, in := range entries {
		in := in
		name := strings.TrimSuffix(filepath.Base(in), ".fuse")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatalf("read %s: %v", in, err)
			}
			sc := NewScanner(filepath.Base(in), src)
			diags := sc.Run()
			got := renderGolden(sc.Tokens(), diags)

			goldenPath := strings.TrimSuffix(in, ".fuse") + ".tokens"
			if *updateGoldens {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v (run with -update to create it)", goldenPath, err)
			}
			// Normalize CRLF -> LF on the golden side so a Windows checkout
			// without .gitattributes enforcement does not spuriously fail.
			wantNorm := bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
			if string(wantNorm) != got {
				t.Errorf("golden mismatch for %s\n--- want ---\n%s--- got ---\n%s", name, wantNorm, got)
			}
		})
	}
}

// renderGolden produces a stable textual form of a token stream plus any
// diagnostics. Format:
//
//	KIND L:C-L:C "text"
//	...
//	EOF L:C
//	[optionally: DIAG L:C-L:C "message"]
//
// Offsets are deliberately omitted so the format is invariant under LF vs CRLF
// on disk (CRLF still counts as one newline per reference §1.1). Columns
// count UTF-8 bytes, which is the scanner's contract.
func renderGolden(toks []Token, diags []Diagnostic) string {
	var b strings.Builder
	for _, tk := range toks {
		if tk.Kind == TokEOF {
			fmt.Fprintf(&b, "EOF %d:%d\n", tk.Span.Start.Line, tk.Span.Start.Column)
			continue
		}
		fmt.Fprintf(&b, "%s %d:%d-%d:%d %s\n",
			tk.Kind,
			tk.Span.Start.Line, tk.Span.Start.Column,
			tk.Span.End.Line, tk.Span.End.Column,
			strconv.Quote(tk.Text))
	}
	for _, d := range diags {
		fmt.Fprintf(&b, "DIAG %d:%d-%d:%d %s\n",
			d.Span.Start.Line, d.Span.Start.Column,
			d.Span.End.Line, d.Span.End.Column,
			strconv.Quote(d.Message))
	}
	return b.String()
}
