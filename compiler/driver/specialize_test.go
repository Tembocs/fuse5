package driver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/monomorph"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// binaryName and runBinary live here (shared by driver tests that
// produce and execute binaries). They intentionally duplicate the
// e2e helpers rather than importing them, so driver tests stay
// self-contained.
func binaryName(stem string) string {
	if runtime.GOOS == "windows" {
		return stem + ".exe"
	}
	return stem
}

func runBinary(t *testing.T, binPath string) int {
	t.Helper()
	cmd := exec.Command(binPath)
	_ = cmd.Run()
	return cmd.ProcessState.ExitCode()
}

// TestInstantiationCollection — W08-P02-T01. The driver collects
// every generic call site after checking. Two distinct call sites
// of `identity[I32]` and `identity[Bool]` produce two
// specialization records.
func TestInstantiationCollection(t *testing.T) {
	prog := buildCheckedProgram(t, `
fn first[T](x: T) -> I32 { return 0; }
fn main() -> I32 {
	let a: I32 = first[I32](1);
	let b: I32 = first[Bool](2);
	return a + b;
}
`)
	out, diags := monomorph.Specialize(prog)
	if len(diags) != 0 {
		t.Fatalf("monomorph diags: %v", diags)
	}
	count := 0
	for _, it := range out.Modules["m"].Items {
		if fn, ok := it.(*hir.FnDecl); ok && strings.HasPrefix(fn.Name, "first__") {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 specializations, got %d", count)
	}
}

// TestPartialInstantiationRejected — W08-P02-T02. A generic fn
// called without turbofish is rejected before lowering.
func TestPartialInstantiationRejected(t *testing.T) {
	_, _, err := Build(BuildOptions{
		Source:  writeTempFuse(t, `
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity(42); }
`),
	})
	if err == nil {
		t.Fatalf("expected build failure for partial instantiation")
	}
}

// TestSpecializationInPipeline — W08-P04-T01. The full pipeline
// (parse → resolve → bridge → check → monomorph → lower → codegen
// → cc) produces a binary that runs and returns 42 for a generic
// identity call.
func TestSpecializationInPipeline(t *testing.T) {
	skipIfNoCC(t)
	src := writeTempFuse(t, `
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity[I32](42); }
`)
	dir := filepath.Dir(src)
	result, diags, err := Build(BuildOptions{
		Source:  src,
		Output:  filepath.Join(dir, binaryName("identity_test")),
		WorkDir: dir,
	})
	if err != nil {
		t.Fatalf("Build: %v (diags=%v)", err, diags)
	}
	exit := runBinary(t, result.BinaryPath)
	if exit != 42 {
		t.Fatalf("identity pipeline exit = %d, want 42", exit)
	}
}

// buildCheckedProgram parses + resolves + bridges + checks an
// in-memory source and returns the ready-to-monomorphize Program.
func buildCheckedProgram(t *testing.T, src string) *hir.Program {
	t.Helper()
	f, pd := parse.Parse("m.fuse", []byte(src))
	if len(pd) != 0 {
		t.Fatalf("parse: %v", pd)
	}
	srcs := []*resolve.SourceFile{{ModulePath: "m", File: f}}
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
	return prog
}

// writeTempFuse creates a temp `.fuse` file with the given source
// and returns its absolute path. Each test gets its own tempdir
// so parallel runs don't collide.
func writeTempFuse(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "program.fuse")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}
