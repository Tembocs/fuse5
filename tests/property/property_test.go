// Package property contains MIR-transform property tests for the
// Fuse compiler. Unlike unit tests that pin specific shapes, property
// tests assert invariants that must hold for *every* valid input:
// determinism, idempotency, shape preservation, and full-module
// validator agreement.
//
// The W15 wave-doc Verify command binds here:
//
//	go test ./tests/property/... -run TestMirTransforms -v
//
// A new property added later extends the `cases` table; the test
// harness re-runs every property against every case.
package property

import (
	"fmt"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/consteval"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/liveness"
	"github.com/Tembocs/fuse5/compiler/lower"
	"github.com/Tembocs/fuse5/compiler/mir"
	"github.com/Tembocs/fuse5/compiler/monomorph"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// runPipeline drives lex → parse → resolve → bridge → check →
// consteval → monomorph → liveness → lower, matching the driver's
// production ordering. Each step rejects malformed input via a
// diagnostic; a failure at any step becomes a test fatal so the
// property test does not mask a pipeline bug as a property violation.
func runPipeline(t *testing.T, src string) *mir.Module {
	t.Helper()
	file, pd := parse.Parse("property.fuse", []byte(src))
	if len(pd) != 0 {
		t.Fatalf("parse: %v", pd)
	}
	srcs := []*resolve.SourceFile{{ModulePath: "", File: file}}
	r, rd := resolve.Resolve(srcs, resolve.BuildConfig{})
	if len(rd) != 0 {
		t.Fatalf("resolve: %v", rd)
	}
	tab := typetable.New()
	prog, bd := hir.NewBridge(tab, r, srcs).Run()
	if len(bd) != 0 {
		t.Fatalf("bridge: %v", bd)
	}
	if cd := check.Check(prog); len(cd) != 0 {
		t.Fatalf("check: %v", cd)
	}
	if rd := consteval.CheckRestrictions(prog); len(rd) != 0 {
		t.Fatalf("consteval restrictions: %v", rd)
	}
	result, ced := consteval.Evaluate(prog)
	if len(ced) != 0 {
		t.Fatalf("consteval: %v", ced)
	}
	consteval.Substitute(prog, result)
	mp, md := monomorph.Specialize(prog)
	if len(md) != 0 {
		t.Fatalf("monomorph: %v", md)
	}
	if _, ld := liveness.Analyze(mp); len(ld) != 0 {
		t.Fatalf("liveness: %v", ld)
	}
	mod, ld := lower.Lower(mp)
	_ = tab // tab retained for future property tests that touch TypeTable directly
	if len(ld) != 0 {
		t.Fatalf("lower: %v", ld)
	}
	return mod
}

// serializeModule produces a stable, byte-deterministic
// representation of a MIR Module for shape-equivalence comparisons.
// Two Modules with the same properties must serialize to identical
// strings — if they don't, the serializer is the bug.
//
// Iteration uses sorted function names (Rule 7.1) so map ordering
// does not leak into the comparison.
func serializeModule(m *mir.Module) string {
	if m == nil {
		return "<nil>"
	}
	names := m.SortedFunctionNames()
	byName := map[string]*mir.Function{}
	for _, f := range m.Functions {
		byName[f.Name] = f
	}
	out := ""
	for _, name := range names {
		out += serializeFunction(byName[name])
	}
	return out
}

// serializeFunction writes a stable textual encoding for one fn.
func serializeFunction(f *mir.Function) string {
	out := fmt.Sprintf("fn %s/%s params=%d regs=%d\n", f.Module, f.Name, f.NumParams, f.NumRegs)
	for _, blk := range f.Blocks {
		out += fmt.Sprintf("  blk %d term=%s ret=%d jmp=%d tt=%d ft=%d br=%d bc=%d\n",
			blk.ID, blk.Term, blk.ReturnReg, blk.JumpTarget,
			blk.TrueTarget, blk.FalseTarget, blk.BranchReg, blk.BranchConst)
		for i, in := range blk.Insts {
			out += fmt.Sprintf("    [%d] %s dst=%d lhs=%d rhs=%d iv=%d pi=%d cn=%q cargs=%v mode=%d flag=%v extra=%d fname=%q fields=%v\n",
				i, in.Op, in.Dst, in.Lhs, in.Rhs, in.IntValue, in.ParamIndex,
				in.CallName, in.CallArgs, in.Mode, in.Flag, in.Extra, in.FieldName, in.Fields)
		}
	}
	return out
}

// propertyCases lists the Fuse programs the property tests re-run
// every invariant against. Each entry is a short program that
// reaches codegen-ready MIR through the production pipeline.
var propertyCases = []struct {
	name string
	src  string
}{
	{"literal-return", `fn main() -> I32 { return 0; }`},
	{"arith-return", `fn main() -> I32 { return 1 + 2 * 3 - 4; }`},
	{"add-return-42", `fn main() -> I32 { return 20 + 22; }`},
}

// TestMirTransforms is the umbrella property-test entry point for
// MIR lowering invariants. Each subtest checks one property across
// every case in propertyCases.
func TestMirTransforms(t *testing.T) {
	t.Run("deterministic-lowering", func(t *testing.T) {
		// Property 1: for every input, two independent lowerings
		// produce byte-identical MIR serializations (Rule 7.1).
		for _, tc := range propertyCases {
			t.Run(tc.name, func(t *testing.T) {
				a := serializeModule(runPipeline(t, tc.src))
				b := serializeModule(runPipeline(t, tc.src))
				if a != b {
					t.Fatalf("non-deterministic MIR for %q:\n--- first ---\n%s\n--- second ---\n%s",
						tc.name, a, b)
				}
			})
		}
	})

	t.Run("every-lowered-fn-validates", func(t *testing.T) {
		// Property 2: every function emitted by Lower must pass
		// Function.Validate (the W05+W15 structural invariants).
		for _, tc := range propertyCases {
			t.Run(tc.name, func(t *testing.T) {
				mod := runPipeline(t, tc.src)
				for _, fn := range mod.Functions {
					if err := fn.Validate(); err != nil {
						t.Fatalf("fn %s/%s failed Validate: %v", fn.Module, fn.Name, err)
					}
				}
			})
		}
	})

	t.Run("pass-invariants-accepts-every-output", func(t *testing.T) {
		// Property 3: the full W15 invariant walker — Validate +
		// CheckNoMoveAfterMove — accepts every lowered Module.
		for _, tc := range propertyCases {
			t.Run(tc.name, func(t *testing.T) {
				mod := runPipeline(t, tc.src)
				if err := lower.PassInvariants(mod); err != nil {
					t.Fatalf("PassInvariants on %q: %v", tc.name, err)
				}
			})
		}
	})

	t.Run("roundtrip-lowering-is-idempotent", func(t *testing.T) {
		// Property 4: lowering twice then serializing yields the
		// same bytes. This catches state leakage in the lowerer
		// (a shared map, a counter that carries across runs, etc.).
		for _, tc := range propertyCases {
			t.Run(tc.name, func(t *testing.T) {
				first := runPipeline(t, tc.src)
				firstBytes := serializeModule(first)
				second := runPipeline(t, tc.src)
				secondBytes := serializeModule(second)
				if firstBytes != secondBytes {
					t.Fatalf("MIR roundtrip differs for %q: shapes not idempotent", tc.name)
				}
			})
		}
	})

	t.Run("w15-ops-validate-as-standalone", func(t *testing.T) {
		// Property 5 (structural): every W15 op that the Builder
		// exposes produces a Function that passes Validate when
		// terminated cleanly. Acts as a gate against a future
		// Validate regression that would silently accept malformed
		// MIR.
		type opCase struct {
			name string
			emit func(*mir.Builder) mir.Reg
		}
		cases := []opCase{
			{"cast-widen", func(b *mir.Builder) mir.Reg {
				r := b.ConstInt(1)
				return b.Cast(r, mir.CastWiden)
			}},
			{"borrow", func(b *mir.Builder) mir.Reg {
				r := b.ConstInt(1)
				return b.Borrow(r, false)
			}},
			{"fn-ptr", func(b *mir.Builder) mir.Reg {
				return b.FnPtr("fuse_inc")
			}},
			{"slice-new", func(b *mir.Builder) mir.Reg {
				base := b.ConstInt(0)
				low := b.ConstInt(1)
				high := b.ConstInt(3)
				return b.SliceNew(base, low, high, false)
			}},
			{"struct-new-and-field-write", func(b *mir.Builder) mir.Reg {
				v := b.ConstInt(7)
				s := b.StructNew("Point", []mir.StructField{{Name: "x", Value: v}})
				b.FieldWrite(s, "x", v)
				return s
			}},
			{"method-call", func(b *mir.Builder) mir.Reg {
				recv := b.ConstInt(0)
				arg := b.ConstInt(1)
				return b.MethodCall("Counter__inc", recv, []mir.Reg{arg})
			}},
			{"eq-scalar", func(b *mir.Builder) mir.Reg {
				a := b.ConstInt(1)
				c := b.ConstInt(2)
				return b.EqScalar(a, c)
			}},
			{"eq-call", func(b *mir.Builder) mir.Reg {
				a := b.ConstInt(1)
				c := b.ConstInt(2)
				return b.EqCall("Pair__eq", a, c)
			}},
			{"wrapping-add", func(b *mir.Builder) mir.Reg {
				a := b.ConstInt(1)
				c := b.ConstInt(2)
				return b.OverflowArith(mir.OpWrappingAdd, a, c)
			}},
			{"checked-add", func(b *mir.Builder) mir.Reg {
				a := b.ConstInt(1)
				c := b.ConstInt(2)
				return b.OverflowArith(mir.OpCheckedAdd, a, c)
			}},
			{"saturating-add", func(b *mir.Builder) mir.Reg {
				a := b.ConstInt(1)
				c := b.ConstInt(2)
				return b.OverflowArith(mir.OpSaturatingAdd, a, c)
			}},
		}
		for _, oc := range cases {
			t.Run(oc.name, func(t *testing.T) {
				fn, b := mir.NewFunction("p", "f")
				out := oc.emit(b)
				b.Return(out)
				if err := fn.Validate(); err != nil {
					t.Fatalf("Validate rejected %s: %v", oc.name, err)
				}
			})
		}
	})
}

// ensure imports used; keeps the lex import non-optional for future
// diag-bearing property tests.
var _ = lex.Span{}
