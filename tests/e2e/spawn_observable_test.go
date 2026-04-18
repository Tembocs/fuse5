package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Tembocs/fuse5/compiler/cc"
	"github.com/Tembocs/fuse5/compiler/codegen"
	"github.com/Tembocs/fuse5/compiler/mir"
)

// TestSpawnObservable is the W16-P05-T01 Verify target. Constructs
// the MIR equivalent of spawn_observable.fuse, emits C, compiles
// with libfuse_rt.a linked, runs the binary, and asserts the exit
// code is 42.
//
// The observable effect: main spawns a worker thread that computes
// 21 + 21 = 42 and returns it via fuse_rt_thread_join. Running the
// binary and reading the process exit code proves every runtime
// entry point along the spawn path — fuse_rt_thread_spawn, the
// trampoline that calls the entry fn, fuse_rt_thread_join's
// platform wait — actually works.
func TestSpawnObservable(t *testing.T) {
	skipIfNoCC(t)
	preferGCCForRuntimeTests(t)
	runtimeDir := findRuntimeDirForTest(t)

	// --- Build MIR: two functions (worker, main). ---
	mod := &mir.Module{}

	// fn worker(arg: I64) -> I64 { return arg + 21; }
	// The `int64_t fn_worker(int64_t)` signature matches the entry
	// shape the runtime trampoline expects when we cast through
	// `int64_t(*)(void*)` — integer parameters pass in the same
	// registers as pointers on x86_64 / MSVC-x64.
	workerFn, wb := mir.NewFunction("", "worker")
	p := wb.Param(0)
	twentyone := wb.ConstInt(21)
	sum := wb.Binary(mir.OpAdd, p, twentyone)
	wb.Return(sum)
	mod.Functions = append(mod.Functions, workerFn)

	// fn main() -> I32 {
	//     let h = spawn worker(21);
	//     return h.join() as I32;
	// }
	mainFn, mb := mir.NewFunction("", "main")
	arg := mb.ConstInt(21)
	handle := mb.Spawn("fuse_worker", arg)
	result := mb.ThreadJoin(handle)
	mb.Return(result)
	mod.Functions = append(mod.Functions, mainFn)

	// Validate every function before emission.
	for _, fn := range mod.Functions {
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate %s: %v", fn.Name, err)
		}
	}

	// Emit C, write to temp, compile with runtime linkage, run.
	cSource, err := codegen.EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	dir := t.TempDir()
	cPath := filepath.Join(dir, "spawn_observable.c")
	if err := os.WriteFile(cPath, []byte(cSource), 0o644); err != nil {
		t.Fatalf("write C source: %v", err)
	}
	binPath := filepath.Join(dir, binaryName("sobs"))
	cc1, err := cc.Detect()
	if err != nil {
		t.Fatalf("detect cc: %v", err)
	}
	libs := []string{}
	if runtime.GOOS != "windows" {
		libs = append(libs, "pthread")
	}
	opts := cc.Options{
		IncludeDirs:  []string{filepath.Join(runtimeDir, "include")},
		ExtraObjects: []string{filepath.Join(runtimeDir, "build", "libfuse_rt.a")},
		ExtraLibs:    libs,
	}
	if err := cc1.CompileWith(cPath, binPath, opts); err != nil {
		t.Fatalf("cc compile: %v\n--- emitted C ---\n%s", err, cSource)
	}

	cmd := exec.Command(binPath)
	runErr := cmd.Run()
	if runErr != nil {
		if _, isExit := runErr.(*exec.ExitError); !isExit {
			t.Fatalf("launch %s: %v", binPath, runErr)
		}
	}
	if cmd.ProcessState == nil {
		t.Fatalf("launch %s: no ProcessState", binPath)
	}
	if code := cmd.ProcessState.ExitCode(); code != 42 {
		t.Fatalf("spawn_observable exit = %d, want 42", code)
	}
}
