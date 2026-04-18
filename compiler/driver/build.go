package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tembocs/fuse5/compiler/cc"
	"github.com/Tembocs/fuse5/compiler/check"
	"github.com/Tembocs/fuse5/compiler/codegen"
	"github.com/Tembocs/fuse5/compiler/consteval"
	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/liveness"
	"github.com/Tembocs/fuse5/compiler/lower"
	"github.com/Tembocs/fuse5/compiler/monomorph"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/resolve"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// BuildOptions bundles every tunable `fuse build` accepts. Callers
// zero-initialise and then set what they need; the zero value is
// legal and produces a sensible default binary path next to the
// source file.
type BuildOptions struct {
	// Source is the path to the Fuse source file to build. Required.
	Source string

	// Output is the path for the produced binary. When empty, the
	// driver derives it from Source: `foo.fuse` → `foo` (plus
	// `.exe` on Windows).
	Output string

	// KeepC retains the generated C source alongside the binary
	// when true. Used by tests to inspect codegen output.
	KeepC bool

	// WorkDir overrides the temp directory used for the generated
	// C source. When empty, os.MkdirTemp picks one.
	WorkDir string

	// Debug, when true, emits DWARF-style debug info through the
	// host C compiler (-g / /Zi) and injects #line directives in
	// the generated C so native debuggers see Fuse source lines.
	// W17 P12 — bootstrap debug-info rides on the C backend.
	Debug bool

	// Cache, when non-nil, is consulted at the start of Build to
	// short-circuit a redundant rebuild whose inputs — source
	// bytes + Output path + Debug / KeepC flags + detected host C
	// compiler identity — match a prior invocation. A hit whose
	// binary is still on disk returns the recorded BuildResult
	// without walking the pipeline. A miss (or a hit whose binary
	// is gone) runs the full pipeline and repopulates the cache.
	// Nil disables caching (the W18 MVP behaviour), so callers
	// that never opt in pay nothing.
	Cache *Cache
}

// buildCacheSchema versions the BuildResult payload stored under the
// "build" pass key. Bump this any time the cached JSON shape changes
// so older payloads are invalidated rather than silently misread.
const buildCacheSchema = "build-v1"

// buildCachePass is the pass name recorded in the manifest for cache
// entries produced by Build.
const buildCachePass = "build"

// BuildResult reports the outcome of a successful build. BinaryPath
// is the path to the produced executable; CSourcePath is the
// generated `.c` file (populated whenever KeepC or WorkDir was
// set; always populated in the result for diagnostic purposes).
type BuildResult struct {
	BinaryPath  string
	CSourcePath string
}

// Build runs the full W05 pipeline on opts.Source and produces a
// native binary. Diagnostics from any stage are returned
// accompanied by an error describing which stage failed. On success
// the BuildResult is non-nil and err is nil.
func Build(opts BuildOptions) (*BuildResult, []lex.Diagnostic, error) {
	if opts.Source == "" {
		return nil, nil, fmt.Errorf("driver.Build: Source is required")
	}
	src, err := os.ReadFile(opts.Source)
	if err != nil {
		return nil, nil, fmt.Errorf("read source: %w", err)
	}

	// Cache consultation. The key composes the content of the source
	// file and the inputs that influence the produced artifact (output
	// path, debug / keep-C flags, host C compiler). A hit short-
	// circuits the full pipeline when the recorded binary still
	// exists on disk — a missing binary degrades to a miss so the
	// caller never receives a stale BuildResult pointing at nothing.
	var cacheKey string
	if opts.Cache != nil {
		cacheKey = buildCacheKey(opts, src)
		if payload, hit := opts.Cache.Get(cacheKey); hit {
			if res, ok := decodeCachedBuild(payload); ok {
				if st, err := os.Stat(res.BinaryPath); err == nil && !st.IsDir() {
					return res, nil, nil
				}
			}
		}
	}

	// Parse.
	file, parseDiags := parse.Parse(opts.Source, src)
	if len(parseDiags) != 0 {
		return nil, parseDiags, fmt.Errorf("parse failed")
	}

	// Resolve. The W05 driver builds a one-file crate: the source
	// file is the root module (module path "").
	sources := []*resolve.SourceFile{{ModulePath: "", File: file}}
	resolved, resolveDiags := resolve.Resolve(sources, resolve.BuildConfig{})
	if len(resolveDiags) != 0 {
		return nil, resolveDiags, fmt.Errorf("resolve failed")
	}

	// Bridge to HIR.
	tab := typetable.New()
	prog, bridgeDiags := hir.NewBridge(tab, resolved, sources).Run()
	if len(bridgeDiags) != 0 {
		return nil, bridgeDiags, fmt.Errorf("HIR bridge failed")
	}

	// Type-check (W06). The checker mutates prog in place,
	// replacing KindInfer TypeIds with concrete types; downstream
	// lowering consults those via TypeOf() without a side table.
	if checkDiags := check.Check(prog); len(checkDiags) != 0 {
		return nil, checkDiags, fmt.Errorf("type checking failed")
	}

	// Compile-time evaluation (W14). Evaluates `const` / `static`
	// initializers, array-length expressions in const contexts,
	// enum discriminants, and enforces `const fn` restrictions
	// from §46.1. Substitutes evaluated literals back into PathExpr
	// references so lowering sees plain literals. Must run after
	// the checker (needs TypeIds) and before monomorphization
	// (array-length substitution can affect specialization).
	restrictDiags := consteval.CheckRestrictions(prog)
	if len(restrictDiags) != 0 {
		return nil, consteval.DiagsToLex(restrictDiags), fmt.Errorf("const fn restrictions rejected the program")
	}
	evalResult, evalDiags := consteval.Evaluate(prog)
	if len(evalDiags) != 0 {
		return nil, consteval.DiagsToLex(evalDiags), fmt.Errorf("compile-time evaluation failed")
	}
	consteval.Substitute(prog, evalResult)

	// Monomorphize (W08). Produces a new Program where generic fns
	// are replaced by their concrete specializations and call sites
	// point at the specialized symbols. Only concrete fns reach
	// lowering (W08 exit criterion).
	monoProg, monoDiags := monomorph.Specialize(prog)
	if len(monoDiags) != 0 {
		return nil, monoDiags, fmt.Errorf("monomorphization failed")
	}
	prog = monoProg

	// Ownership / borrow / liveness / drop-intent (W09). Rejects
	// struct fields with borrow types (§54.1), returns of borrows
	// to locals (§54.6), aliased mutrefs (§54.7), use-after-move,
	// and escaping non-escaping closures. Drop metadata is
	// attached to prog for codegen; W09 does not yet consume it
	// at the MIR emit path — that wiring lands with W15 MIR
	// consolidation.
	if _, livDiags := liveness.Analyze(prog); len(livDiags) != 0 {
		return nil, livDiags, fmt.Errorf("ownership / liveness rejected the program")
	}

	// Lower to MIR.
	mirMod, lowerDiags := lower.Lower(prog)
	if len(lowerDiags) != 0 {
		return nil, lowerDiags, fmt.Errorf("lowering failed")
	}
	if !hasMain(mirMod.SortedFunctionNames()) {
		names := strings.Join(mirMod.SortedFunctionNames(), ", ")
		return nil, nil, fmt.Errorf(
			"no `main` function found; the W05 spine requires `fn main() -> I32 { ... }` (have: %s)",
			names)
	}

	// Codegen. When Debug is set, the emitter prepends a #line
	// directive pointing at the originating .fuse source so gdb
	// / lldb can map native addresses back to Fuse lines.
	cSource, err := codegen.EmitC11(mirMod)
	if err != nil {
		return nil, nil, fmt.Errorf("codegen: %w", err)
	}
	if opts.Debug {
		cSource = codegen.EmitLineDirective(1, opts.Source) + "\n" + cSource
	}

	// Work directory.
	workDir := opts.WorkDir
	cleanup := func() {}
	if workDir == "" {
		workDir, err = os.MkdirTemp("", "fuse-build-*")
		if err != nil {
			return nil, nil, fmt.Errorf("mktemp: %w", err)
		}
		if !opts.KeepC {
			cleanup = func() { os.RemoveAll(workDir) }
		}
	} else {
		if err := os.MkdirAll(workDir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("mkdir: %w", err)
		}
	}

	cPath := filepath.Join(workDir, baseStem(opts.Source)+".c")
	if err := os.WriteFile(cPath, []byte(cSource), 0o644); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("write C source: %w", err)
	}

	// Detect and invoke the host C compiler.
	cc1, err := cc.Detect()
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("detect cc: %w", err)
	}
	binPath := opts.Output
	if binPath == "" {
		binPath = deriveBinaryPath(opts.Source)
	}

	// Programs that emit W16 runtime-ABI calls or TermUnreachable
	// need fuse_rt.h visible and libfuse_rt.a linked. The driver
	// locates the runtime next to the Fuse repo root and builds
	// the archive on demand so e2e tests stay hermetic.
	ccOpts := cc.Options{Debug: opts.Debug}
	if codegen.UsesRuntimeABI(mirMod) {
		rtIncludes, rtObjects, rtLibs, rtErr := locateRuntimeArtifacts()
		if rtErr != nil {
			cleanup()
			return nil, nil, fmt.Errorf("runtime setup: %w", rtErr)
		}
		ccOpts.IncludeDirs = rtIncludes
		ccOpts.ExtraObjects = rtObjects
		ccOpts.ExtraLibs = rtLibs
	}
	if err := cc1.CompileWith(cPath, binPath, ccOpts); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("cc compile: %w", err)
	}

	// Do not remove the work directory when the user asked to keep
	// the C source or when a custom WorkDir was provided; the
	// caller owns the directory in those cases.
	result := &BuildResult{BinaryPath: binPath, CSourcePath: cPath}
	if opts.Cache != nil && cacheKey != "" {
		if payload, err := encodeCachedBuild(result); err == nil {
			_ = opts.Cache.Put(cacheKey, buildCachePass, buildCacheSchema, payload)
			_ = opts.Cache.Flush()
		}
	}
	return result, nil, nil
}

// buildCacheKey derives the content-addressed key under which a
// BuildResult is stored for `opts`. The key binds the source bytes
// together with every input that influences the produced artifact so
// a cache hit is always safe to reuse:
//
//   - Source bytes (parse / resolve / check / codegen all depend on these).
//   - Output path (a different binary path is a different artifact).
//   - Debug / KeepC flags (Debug flips codegen AND host-compiler flags).
//   - Host C compiler identity (switching gcc↔cl invalidates).
func buildCacheKey(opts BuildOptions, src []byte) string {
	h := sha256.New()
	h.Write([]byte(opts.Source))
	h.Write([]byte{0})
	h.Write(src)
	h.Write([]byte{0})
	h.Write([]byte(opts.Output))
	h.Write([]byte{0})
	if opts.Debug {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	if opts.KeepC {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	h.Write([]byte{0})
	if cc1, err := cc.Detect(); err == nil {
		h.Write([]byte(cc1.Path))
		h.Write([]byte{0})
		h.Write([]byte(cc1.Kind.String()))
	}
	fp := hex.EncodeToString(h.Sum(nil))
	return Key(buildCachePass, []byte(fp))
}

// encodeCachedBuild serialises a BuildResult to the JSON shape the
// cache persists.
func encodeCachedBuild(r *BuildResult) ([]byte, error) {
	return json.Marshal(r)
}

// decodeCachedBuild is the inverse. A decode failure reports `ok=false`
// so the caller treats the entry as a miss rather than a silent error.
func decodeCachedBuild(payload []byte) (*BuildResult, bool) {
	var r BuildResult
	if err := json.Unmarshal(payload, &r); err != nil {
		return nil, false
	}
	if r.BinaryPath == "" {
		return nil, false
	}
	return &r, true
}

// hasMain returns true when `main` is among the function names.
func hasMain(names []string) bool {
	for _, n := range names {
		if n == "main" {
			return true
		}
	}
	return false
}

// baseStem returns the base filename of path with its extension
// removed — used when placing generated files in the work directory.
func baseStem(path string) string {
	b := filepath.Base(path)
	if dot := strings.LastIndex(b, "."); dot >= 0 {
		return b[:dot]
	}
	return b
}

// deriveBinaryPath turns `foo/bar.fuse` into `foo/bar` (or
// `foo/bar.exe` on Windows). Callers that want a different output
// path set opts.Output explicitly.
func deriveBinaryPath(src string) string {
	dir := filepath.Dir(src)
	stem := baseStem(src)
	path := filepath.Join(dir, stem)
	if strings.HasSuffix(strings.ToLower(filepath.Ext(src)), ".fuse") {
		// On Windows the linker expects an .exe extension for
		// executables; Go's `exec.Command` also relies on it.
		if isWindowsHost() {
			path += ".exe"
		}
	}
	return path
}

// isWindowsHost is a small indirection that tests can override;
// callers use runtime.GOOS directly elsewhere.
var isWindowsHost = func() bool {
	return filepath.Separator == '\\' || osIsWindows()
}

// osIsWindows checks GOOS via a helper so the function pointer above
// can be replaced in tests without touching runtime.GOOS directly.
func osIsWindows() bool {
	return pathSeparatorIsWindows()
}

// pathSeparatorIsWindows is the actual OS probe; factored so the
// test file can override it when needed.
var pathSeparatorIsWindows = func() bool {
	return filepath.Separator == '\\'
}
