package hir

import (
	"bytes"
	"reflect"
	"testing"
)

// simplePass is a minimal test pass used throughout the manifest
// and fingerprint tests.
type simplePass struct {
	name      string
	inputs    []string
	outputKey string
	fn        func(ctx *PassContext) error
}

func (s *simplePass) Name() string                            { return s.name }
func (s *simplePass) Inputs() []string                        { return s.inputs }
func (s *simplePass) OutputKey() string                       { return s.outputKey }
func (s *simplePass) Fingerprint(in map[string][]byte) []byte { return ComputeFingerprint(s.name, in) }
func (s *simplePass) Run(ctx *PassContext) error {
	if s.fn != nil {
		return s.fn(ctx)
	}
	ctx.Outputs[s.outputKey] = []byte("ran:" + s.name)
	return nil
}

// TestPassManifest exercises registration, validation, topological
// order, and error handling.
func TestPassManifest(t *testing.T) {
	t.Run("simple-chain-validates", func(t *testing.T) {
		m := NewManifest()
		m.Register(&simplePass{name: "parse", outputKey: "ast"})
		m.Register(&simplePass{name: "resolve", inputs: []string{"parse"}, outputKey: "resolved"})
		m.Register(&simplePass{name: "bridge", inputs: []string{"resolve"}, outputKey: "hir"})
		if err := m.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		want := []string{"parse", "resolve", "bridge"}
		if !reflect.DeepEqual(m.Order(), want) {
			t.Fatalf("Order = %v, want %v", m.Order(), want)
		}
	})
	t.Run("unknown-input-is-error", func(t *testing.T) {
		m := NewManifest()
		m.Register(&simplePass{name: "a", inputs: []string{"missing"}, outputKey: "a-out"})
		if err := m.Validate(); err == nil {
			t.Fatalf("expected error for missing input")
		}
	})
	t.Run("cycle-is-error", func(t *testing.T) {
		m := NewManifest()
		m.Register(&simplePass{name: "a", inputs: []string{"b"}, outputKey: "a-out"})
		m.Register(&simplePass{name: "b", inputs: []string{"a"}, outputKey: "b-out"})
		if err := m.Validate(); err == nil {
			t.Fatalf("expected cycle error")
		}
	})
	t.Run("duplicate-name-panics", func(t *testing.T) {
		m := NewManifest()
		m.Register(&simplePass{name: "a", outputKey: "a-out"})
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic on duplicate name")
			}
		}()
		m.Register(&simplePass{name: "a", outputKey: "a-alt"})
	})
	t.Run("run-executes-in-order", func(t *testing.T) {
		var trace []string
		recordPass := func(name string, inputs []string) *simplePass {
			p := &simplePass{name: name, inputs: inputs, outputKey: name + "-out"}
			p.fn = func(ctx *PassContext) error {
				trace = append(trace, name)
				ctx.Outputs[p.outputKey] = []byte(name)
				return nil
			}
			return p
		}
		m := NewManifest()
		m.Register(recordPass("z", []string{"y"}))
		m.Register(recordPass("y", []string{"x"}))
		m.Register(recordPass("x", nil))
		if err := m.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if err := m.Run(NewPassContext(nil)); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if !reflect.DeepEqual(trace, []string{"x", "y", "z"}) {
			t.Fatalf("trace = %v, want [x y z]", trace)
		}
	})
}

// TestDeterministicOrder proves that validating and iterating a
// fixed set of passes yields the same order every time across
// multiple runs of the test (W04-P04-T03). Running with `-count=3`
// exercises the contract explicitly.
func TestDeterministicOrder(t *testing.T) {
	newManifest := func() *Manifest {
		m := NewManifest()
		// Diamond graph: a → b, a → c, b → d, c → d.
		m.Register(&simplePass{name: "a", outputKey: "a"})
		m.Register(&simplePass{name: "b", inputs: []string{"a"}, outputKey: "b"})
		m.Register(&simplePass{name: "c", inputs: []string{"a"}, outputKey: "c"})
		m.Register(&simplePass{name: "d", inputs: []string{"b", "c"}, outputKey: "d"})
		if err := m.Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
		return m
	}
	first := newManifest().Order()
	for i := 0; i < 5; i++ {
		got := newManifest().Order()
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("order %d differs:\n  got  %v\n  want %v", i+1, got, first)
		}
	}
	// The expected order: a, then b and c lexicographically, then d.
	want := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("order = %v, want %v", first, want)
	}
}

// TestPassFingerprintStable proves fingerprints are byte-identical
// across runs (W04-P05-T01). The -count=3 run exercises the
// determinism contract.
func TestPassFingerprintStable(t *testing.T) {
	inputs := map[string][]byte{
		"resolve": []byte("resolved-v1"),
		"parse":   []byte("parsed-v1"),
		"other":   []byte("other-v2"),
	}
	p := &simplePass{name: "my-pass", outputKey: "my-out"}
	first := p.Fingerprint(inputs)
	for i := 0; i < 10; i++ {
		got := p.Fingerprint(inputs)
		if !bytes.Equal(got, first) {
			t.Fatalf("fingerprint differs on run %d:\n  got  %x\n  first %x", i+1, got, first)
		}
	}
	// Different inputs must produce a different digest.
	other := map[string][]byte{"parse": []byte("parsed-v2")}
	if bytes.Equal(p.Fingerprint(other), first) {
		t.Fatalf("fingerprints of different inputs collided")
	}
	// Pass name is folded into the hash: two passes with the same
	// inputs must still produce distinct fingerprints.
	q := &simplePass{name: "other-pass", outputKey: "my-out"}
	if bytes.Equal(p.Fingerprint(inputs), q.Fingerprint(inputs)) {
		t.Fatalf("different pass names must yield distinct fingerprints")
	}
}
