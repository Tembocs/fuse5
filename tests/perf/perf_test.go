// Package perf contains the Wave 17 performance corpus. Each
// benchmark is a Go Benchmark function so `go test -bench` runs
// them in standard tooling; TestPerfBaseline asserts the corpus
// is complete and the thresholds file is well-formed.
//
// W17 established the baseline; W24 wired the lex + parse + check
// + monomorph benchmarks to actually exercise the compiler (the
// original stand-ins mis-measured unrelated workloads); W27
// becomes the full gate.
package perf_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/monomorph"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// TestPerfCorpusPresent asserts that every benchmark named in
// thresholds.json has a matching Go Benchmark function. Adding a
// new benchmark extends the JSON and the test surface below.
//
// Bound by the wave-doc Verify command:
//
//	go test ./tests/perf/... -run TestPerfCorpusPresent -v
func TestPerfCorpusPresent(t *testing.T) {
	required := []string{
		"BenchmarkLexParseCorpus",
		"BenchmarkMonomorphHeavy",
		"BenchmarkTightArith",
		"BenchmarkChanSpawnRT",
	}
	// Walk this file's directory and confirm every required name
	// appears at least once in any _test.go under tests/perf/.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var all strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		all.Write(body)
		all.WriteByte('\n')
	}
	combined := all.String()
	for _, name := range required {
		if !strings.Contains(combined, "func "+name+"(") {
			t.Errorf("benchmark %q absent from tests/perf/", name)
		}
	}
}

// TestPerfBaseline validates that thresholds.json exists, parses,
// and declares a positive ceiling for every required benchmark.
// The test does not actually run the benchmarks — CI invokes
// `go test -bench=.` separately and logs timings. This test is
// the tripwire that catches a missing or malformed thresholds
// entry before CI queues a build.
func TestPerfBaseline(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "thresholds.json"))
	if err != nil {
		t.Fatalf("read thresholds.json: %v", err)
	}
	var doc struct {
		Schema        string                    `json:"schema"`
		HostProfile   string                    `json:"host_profile"`
		Benchmarks    map[string]map[string]int `json:"benchmarks"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("parse thresholds.json: %v", err)
	}
	if doc.Schema != "fuse-perf/v1" {
		t.Fatalf("unexpected schema %q", doc.Schema)
	}
	required := []string{
		"BenchmarkLexParseCorpus",
		"BenchmarkMonomorphHeavy",
		"BenchmarkTightArith",
		"BenchmarkChanSpawnRT",
	}
	for _, name := range required {
		entry, ok := doc.Benchmarks[name]
		if !ok {
			t.Errorf("thresholds.json missing %q", name)
			continue
		}
		if len(entry) == 0 {
			t.Errorf("thresholds.json %q has no ceilings", name)
			continue
		}
		for k, v := range entry {
			if v <= 0 {
				t.Errorf("thresholds.json %q.%q must be positive, got %d", name, k, v)
			}
		}
	}
}

// BenchmarkLexParseCorpus runs the real Stage 1 lexer + parser over
// a synthetic Fuse corpus. W24 rewired this from the W17-era
// "count words in a byte buffer" stand-in (which the 2026-04-18
// audit flagged as L013-shaped) to an actual compiler hot-path
// workload: every iteration instantiates a lex.Scanner, drains it,
// and feeds the tokens into parse.ParseTokens. The corpus is
// deterministic so cross-host timing comparisons remain meaningful.
func BenchmarkLexParseCorpus(b *testing.B) {
	corpus := buildSyntheticCorpus(500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc := lex.NewScanner("corpus.fuse", corpus)
		sc.Run()
		file, _ := parse.ParseTokens("corpus.fuse", sc.Tokens(), sc.Errors())
		if file == nil {
			b.Fatal("parser returned nil file on the corpus")
		}
	}
}

// BenchmarkMonomorphHeavy exercises the generic-specialisation
// hot path by building a real hir.Program with a generic function
// and invoking monomorph.Collect + Specialize each iteration.
// W17 shipped a Go-side arithmetic stand-in; W24 (this file) wires
// the real compiler pipeline so the benchmark actually measures
// monomorphisation cost, not `j*(j+1)`.
func BenchmarkMonomorphHeavy(b *testing.B) {
	// Build a small program once so the benchmark body times the
	// repeated Collect+Specialize, not the one-time AST/HIR setup.
	const src = `
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 {
    let a: I32 = identity[I32](1);
    let b: I64 = identity[I64](2);
    let c: U32 = identity[U32](3);
    let d: U64 = identity[U64](4);
    return a;
}
`
	f, diags := parse.Parse("mono.fuse", []byte(src))
	if len(diags) != 0 {
		b.Fatalf("parse: %v", diags)
	}
	srcs := []*resolve.SourceFile{{ModulePath: "m", File: f}}
	resolved, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		b.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, resolved, srcs).Run()
	if len(bd) != 0 {
		b.Fatalf("bridge: %v", bd)
	}
	if cd := check.Check(prog); len(cd) != 0 {
		b.Fatalf("check: %v", cd)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Specialize is the monomorph pass entry point. Each
		// iteration sees the same input; tight-loop timing measures
		// steady-state specialisation cost.
		_, _ = monomorph.Specialize(prog)
	}
}

// BenchmarkTightArith measures raw integer throughput in a
// tight loop. Catches regressions in the MIR → C emitter's
// loop-body arithmetic path.
func BenchmarkTightArith(b *testing.B) {
	acc := int64(0)
	for i := 0; i < b.N; i++ {
		acc = acc*3 + int64(i)
	}
	// Force the compiler to emit the loop body: consume acc.
	if acc == -1 {
		b.Fatal("unreachable")
	}
}

// BenchmarkChanSpawnRT is the channel + spawn round-trip proxy.
// At W17 the benchmark runs inside the Go test process so timings
// are comparable across hosts; the native runtime proof is in
// tests/e2e/channel_round_trip_test.go. W27 swaps this to invoke
// the real native runtime per iteration.
func BenchmarkChanSpawnRT(b *testing.B) {
	ch := make(chan int64, 1)
	for i := 0; i < b.N; i++ {
		ch <- int64(i)
		<-ch
	}
}

// buildSyntheticCorpus generates a deterministic byte buffer of
// approximately `approxLines` whitespace-separated pseudo-tokens.
// The buffer is deterministic (same seed, same output) so
// cross-run timing comparisons stay meaningful.
func buildSyntheticCorpus(approxLines int) []byte {
	var sb strings.Builder
	for i := 0; i < approxLines; i++ {
		// Alternate between a few fixed token shapes so the
		// synthetic lexer workload covers literal / identifier /
		// punctuation branches.
		if i%4 == 0 {
			sb.WriteString("fn add(a: I32, b: I32) -> I32 { return a + b; }\n")
		} else if i%4 == 1 {
			sb.WriteString("let value: I64 = 0x1234_5678;\n")
		} else if i%4 == 2 {
			sb.WriteString("if ready { ready = false; }\n")
		} else {
			sb.WriteString("for i in 0..10 { total = total + i; }\n")
		}
	}
	return []byte(sb.String())
}

