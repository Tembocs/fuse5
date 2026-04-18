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

// TestChannelRoundTrip is the W16-P05-T02 Verify target. Constructs
// the MIR equivalent of channel_round_trip.fuse, emits C, compiles
// with libfuse_rt.a linked, runs the binary, and asserts the exit
// code equals 42 — the value sent through the channel.
//
// Observable effect: main allocates a bounded channel of capacity
// 1, spawns a worker thread that sends 42 and closes the channel,
// main receives from the channel, joins the worker, and returns
// the received value. This exercises every channel entry point
// (fuse_rt_chan_new, _send, _recv, _close, _free via the OS cleanup
// on process exit) plus the thread primitives.
func TestChannelRoundTrip(t *testing.T) {
	skipIfNoCC(t)
	preferGCCForRuntimeTests(t)
	runtimeDir := findRuntimeDirForTest(t)

	mod := &mir.Module{}

	// Shared channel pointer lives in a global so both the worker
	// and main can reach it without passing through the trampoline
	// argument. The W16 proof program declares it as a C global
	// via emit below; here we shape the worker to read it too.
	//
	// fn worker(arg: I64) -> I64 {
	//     // the channel pointer is injected as the `arg` register
	//     let value: I64 = 42;
	//     fuse_rt_chan_send(arg_as_chan, &value);
	//     fuse_rt_chan_close(arg_as_chan);
	//     return 0;
	// }
	workerFn, wb := mir.NewFunction("", "worker")
	chanReg := wb.Param(0)
	value := wb.ConstInt(42)
	_ = wb.ChanSend(chanReg, value)
	wb.ChanClose(chanReg)
	zero := wb.ConstInt(0)
	wb.Return(zero)
	mod.Functions = append(mod.Functions, workerFn)

	// fn main() -> I32 {
	//     let ch = chan_new(capacity=1, elem_bytes=8);
	//     let h = spawn worker(ch);
	//     let got: I64 = 0;
	//     chan_recv(ch, &got);
	//     thread_join(h);
	//     return got as I32;
	// }
	mainFn, mb := mir.NewFunction("", "main")
	capacity := mb.ConstInt(1)
	elemBytes := mb.ConstInt(8)
	ch := mb.ChanNew(capacity, elemBytes)
	handle := mb.Spawn("fuse_worker", ch)
	// Allocate a recv slot register; OpChanRecv writes through
	// `&r_slot` in the emitted C, so the register must be
	// addressable. The W16 codegen emits a local int64_t for every
	// Reg, so any register we designate as the slot works.
	slot := mb.ConstInt(0)
	_ = mb.ChanRecv(ch, slot)
	_ = mb.ThreadJoin(handle)
	// The emitted recv wrote through &r_slot, so `slot` now holds
	// the received value. Return it.
	mb.Return(slot)
	mod.Functions = append(mod.Functions, mainFn)

	for _, fn := range mod.Functions {
		if err := fn.Validate(); err != nil {
			t.Fatalf("Validate %s: %v", fn.Name, err)
		}
	}

	cSource, err := codegen.EmitC11(mod)
	if err != nil {
		t.Fatalf("EmitC11: %v", err)
	}
	dir := t.TempDir()
	cPath := filepath.Join(dir, "channel_round_trip.c")
	if err := os.WriteFile(cPath, []byte(cSource), 0o644); err != nil {
		t.Fatalf("write C source: %v", err)
	}
	binPath := filepath.Join(dir, binaryName("crt"))
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
		t.Fatalf("channel_round_trip exit = %d, want 42", code)
	}
}

// preferGCCForRuntimeTests forces cc.Detect to pick gcc when it is
// available. The runtime archive libfuse_rt.a is built with gcc
// (see runtime/Makefile), and other toolchains on the same host
// cannot read a gcc-produced archive — most visibly TinyCC on
// Windows which reports "invalid object file" when given libfuse_rt.a.
// A test-scoped CC override avoids mixing toolchains.
func preferGCCForRuntimeTests(t *testing.T) {
	t.Helper()
	if os.Getenv("CC") != "" {
		return
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		// No gcc available; fall back to whatever cc.Detect picks.
		// If that compiler cannot link libfuse_rt.a the test will
		// surface an explanatory error.
		return
	}
	t.Setenv("CC", "gcc")
}

// findRuntimeDirForTest locates the runtime/ tree from the test's
// working directory. Duplicates the driver's findRuntimeDir for e2e
// self-containment — the e2e tests explicitly walk their own source
// tree and do not depend on the driver's internal helpers.
func findRuntimeDirForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		candidate := filepath.Join(dir, "runtime", "include", "fuse_rt.h")
		if _, err := os.Stat(candidate); err == nil {
			// Ensure libfuse_rt.a exists; build on demand.
			runtimeDir := filepath.Join(dir, "runtime")
			libPath := filepath.Join(runtimeDir, "build", "libfuse_rt.a")
			if _, err := os.Stat(libPath); err != nil {
				makeBin, lookErr := exec.LookPath("make")
				if lookErr != nil {
					t.Skipf("libfuse_rt.a missing and `make` not on PATH; build runtime first")
				}
				cmd := exec.Command(makeBin, "-C", runtimeDir, "all")
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("build runtime: %v\n%s", err, out)
				}
			}
			return runtimeDir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("runtime/ not found walking up from %s", wd)
			return ""
		}
		dir = parent
	}
}
