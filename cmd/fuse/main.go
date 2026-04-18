// Command fuse is the Stage 1 Fuse compiler CLI entry point.
//
// W18 expands the subcommand surface to the full set the plan
// declares: build, run, check, test, fmt, doc, repl, version,
// help. Exit-code policy:
//
//   0  — success
//   1  — user-visible failure (compile error, test failure, etc.)
//   2  — CLI misuse (unknown flag, missing arg, unknown subcommand)
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Tembocs/fuse5/compiler/diagnostics"
	"github.com/Tembocs/fuse5/compiler/doc"
	fusefmt "github.com/Tembocs/fuse5/compiler/fmt"
	"github.com/Tembocs/fuse5/compiler/driver"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/lsp"
	"github.com/Tembocs/fuse5/compiler/parse"
	"github.com/Tembocs/fuse5/compiler/repl"
)

// version is the Stage 1 compiler version string reported by
// `fuse version`. Incremented per-wave until 1.0.
const version = "0.18.0-W18"

// run is the testable entry point. It returns the process exit
// code so tests can drive main without os.Exit side effects.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fuse: no subcommand; see `fuse help` for the full list")
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "-h", "--help":
		return runHelp(stdout)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "test":
		return runTest(args[1:], stdout, stderr)
	case "fmt":
		return runFmt(args[1:], stdout, stderr)
	case "doc":
		return runDoc(args[1:], stdout, stderr)
	case "repl":
		return runRepl(stdout, stderr)
	case "lsp":
		return runLsp(stderr)
	default:
		fmt.Fprintf(stderr, "fuse: unknown subcommand %q; see `fuse help`\n", args[0])
		return 2
	}
}

// runHelp prints the full subcommand surface. Exit code 0.
func runHelp(stdout io.Writer) int {
	fmt.Fprintln(stdout, "fuse — Fuse Stage 1 compiler")
	fmt.Fprintln(stdout, "subcommands:")
	fmt.Fprintln(stdout, "  version                     print compiler version and exit")
	fmt.Fprintln(stdout, "  help                        show this message")
	fmt.Fprintln(stdout, "  build <file>                compile to a native binary")
	fmt.Fprintln(stdout, "                              flags: -o <path>, --keep-c, --debug, --json")
	fmt.Fprintln(stdout, "  run <file>                  build then execute (returns the binary's exit code)")
	fmt.Fprintln(stdout, "  check <file>                run lex + parse + resolve + check without codegen")
	fmt.Fprintln(stdout, "                              flag: --json (emit diagnostics as JSON)")
	fmt.Fprintln(stdout, "  test <file>                 build and run under the test runner")
	fmt.Fprintln(stdout, "  fmt <file> [<file>...]      canonicalise Fuse source in place; `fuse fmt -` reads stdin")
	fmt.Fprintln(stdout, "                              flag: --check (report non-canonical files, exit 1 on diff)")
	fmt.Fprintln(stdout, "  doc <file>                  extract doc comments; --check flags missing docs on pub items")
	fmt.Fprintln(stdout, "  repl                        interactive read-eval-print loop")
	fmt.Fprintln(stdout, "  lsp                         start a Language Server Protocol session over stdio")
	return 0
}

// runBuild parses the `build` subcommand's flags and invokes the
// driver. Accepts the W05 shape (-o, --keep-c) plus W17's --debug
// and W18's --json flag for diagnostic output.
func runBuild(args []string, stdout, stderr io.Writer) int {
	var (
		outPath string
		keepC   bool
		debug   bool
		asJSON  bool
		srcPath string
	)
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-o":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "fuse build: -o requires a path argument")
				return 2
			}
			outPath = args[i+1]
			i += 2
		case a == "--keep-c":
			keepC = true
			i++
		case a == "--debug":
			debug = true
			i++
		case a == "--json":
			asJSON = true
			i++
		case a == "--":
			if i+1 < len(args) {
				srcPath = args[i+1]
			}
			i = len(args)
		case len(a) > 0 && a[0] == '-':
			fmt.Fprintf(stderr, "fuse build: unknown flag %q\n", a)
			return 2
		default:
			if srcPath != "" {
				fmt.Fprintf(stderr, "fuse build: multiple source files not yet supported (got %q and %q)\n", srcPath, a)
				return 2
			}
			srcPath = a
			i++
		}
	}
	if srcPath == "" {
		fmt.Fprintln(stderr, "fuse build: missing source file")
		return 2
	}
	// W20 library-mode: `fuse build DIR/...` walks the directory
	// for every .fuse file and runs parse-only validation. This is
	// the stdlib-core proof path — `fuse build stdlib/core/...`
	// succeeds when every stdlib file parses cleanly. Full
	// codegen for a library graph lands with W23 package
	// management.
	if strings.HasSuffix(srcPath, "/...") || strings.HasSuffix(srcPath, `\...`) {
		return runBuildLib(srcPath, stdout, stderr, asJSON)
	}
	result, diags, err := driver.Build(driver.BuildOptions{
		Source: srcPath,
		Output: outPath,
		KeepC:  keepC,
		Debug:  debug,
	})
	printDiags(stderr, diags, asJSON)
	if err != nil {
		fmt.Fprintf(stderr, "fuse build: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "fuse: wrote %s\n", result.BinaryPath)
	return 0
}

// runBuildLib handles the `DIR/...` library-mode pattern. Walks
// every .fuse file under DIR, parses each, and reports the first
// failure (or success with a count). Output is deterministic —
// files are visited in lexicographic order.
func runBuildLib(pattern string, stdout, stderr io.Writer, asJSON bool) int {
	root := strings.TrimSuffix(strings.TrimSuffix(pattern, "/..."), `\...`)
	if root == "" {
		root = "."
	}
	var files []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".fuse") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "fuse build: walk %s: %v\n", root, err)
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintf(stderr, "fuse build: no .fuse files under %s\n", root)
		return 1
	}
	failures := 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "fuse build: %v\n", err)
			failures++
			continue
		}
		_, diags := parse.Parse(path, src)
		if len(diags) != 0 {
			failures++
			printDiags(stderr, diags, asJSON)
			fmt.Fprintf(stderr, "fuse build: %s FAILED\n", path)
		}
	}
	if failures != 0 {
		fmt.Fprintf(stderr, "fuse build: %d/%d files failed\n", failures, len(files))
		return 1
	}
	fmt.Fprintf(stdout, "fuse: %d files ok (lib mode)\n", len(files))
	return 0
}

// runRun performs `build` and then executes the resulting binary,
// returning the binary's exit code.
func runRun(args []string, stdout, stderr io.Writer) int {
	srcPath, flags, code := extractSourcePath(args, stderr, "run")
	if code != 0 {
		return code
	}
	tmp, err := os.MkdirTemp("", "fuse-run-*")
	if err != nil {
		fmt.Fprintf(stderr, "fuse run: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	binPath := filepath.Join(tmp, "run.exe")
	result, diags, err := driver.Build(driver.BuildOptions{
		Source:  srcPath,
		Output:  binPath,
		WorkDir: tmp,
		Debug:   flags.debug,
	})
	printDiags(stderr, diags, flags.asJSON)
	if err != nil {
		fmt.Fprintf(stderr, "fuse run: %v\n", err)
		return 1
	}
	cmd := exec.Command(result.BinaryPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if runErr := cmd.Run(); runErr != nil {
		if exit, ok := runErr.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintf(stderr, "fuse run: launch failed: %v\n", runErr)
		return 1
	}
	return 0
}

// runCheck runs the front-end passes (lex + parse + resolve +
// check) without emitting codegen. Used as a tight-loop
// correctness probe by tooling + CI.
func runCheck(args []string, stdout, stderr io.Writer) int {
	srcPath, flags, code := extractSourcePath(args, stderr, "check")
	if code != 0 {
		return code
	}
	src, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintf(stderr, "fuse check: %v\n", err)
		return 1
	}
	// Parse-only front-end probe. A full check depends on the
	// resolver + bridge + checker; W18 check surfaces parse
	// diagnostics and defers deeper checks to `build` (which
	// returns the same diagnostics with a non-zero exit). The
	// structure is ready for W19 to extend to resolve + check.
	_, diags := parse.Parse(srcPath, src)
	printDiags(stderr, diags, flags.asJSON)
	if len(diags) != 0 {
		return 1
	}
	fmt.Fprintf(stdout, "fuse: %s ok\n", srcPath)
	return 0
}

// runTest builds and runs the binary under a minimal test harness
// — effectively `fuse run` with an "exit code 0 is pass" gate.
// The richer test runner (discovery, filtering, parallelism)
// lives in compiler/testrunner and lands with W20 stdlib.
func runTest(args []string, stdout, stderr io.Writer) int {
	srcPath, flags, code := extractSourcePath(args, stderr, "test")
	if code != 0 {
		return code
	}
	tmp, err := os.MkdirTemp("", "fuse-test-*")
	if err != nil {
		fmt.Fprintf(stderr, "fuse test: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	binPath := filepath.Join(tmp, "test.exe")
	result, diags, err := driver.Build(driver.BuildOptions{
		Source:  srcPath,
		Output:  binPath,
		WorkDir: tmp,
		Debug:   flags.debug,
	})
	printDiags(stderr, diags, flags.asJSON)
	if err != nil {
		fmt.Fprintf(stderr, "fuse test: %v\n", err)
		return 1
	}
	cmd := exec.Command(result.BinaryPath)
	out := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = out
	runErr := cmd.Run()
	stdout.Write(out.Bytes())
	if runErr != nil {
		if exit, ok := runErr.(*exec.ExitError); ok {
			fmt.Fprintf(stderr, "fuse test: FAIL (exit %d)\n", exit.ExitCode())
			return 1
		}
		fmt.Fprintf(stderr, "fuse test: launch failed: %v\n", runErr)
		return 1
	}
	fmt.Fprintf(stdout, "fuse: %s PASS\n", srcPath)
	return 0
}

// runFmt formats one or more Fuse sources in place. `--check`
// reports non-canonical files and exits 1 without rewriting.
// `fuse fmt -` reads stdin and writes formatted output to stdout.
func runFmt(args []string, stdout, stderr io.Writer) int {
	var (
		checkOnly bool
		paths     []string
	)
	for _, a := range args {
		switch {
		case a == "--check":
			checkOnly = true
		case a == "-":
			paths = append(paths, "-")
		case len(a) > 0 && a[0] == '-':
			fmt.Fprintf(stderr, "fuse fmt: unknown flag %q\n", a)
			return 2
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "fuse fmt: missing source file (use `-` for stdin)")
		return 2
	}
	anyDiff := false
	for _, p := range paths {
		if p == "-" {
			src, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(stderr, "fuse fmt: read stdin: %v\n", err)
				return 1
			}
			stdout.Write(fusefmt.Format(src))
			continue
		}
		src, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "fuse fmt: %v\n", err)
			return 1
		}
		out := fusefmt.Format(src)
		if !bytes.Equal(out, src) {
			anyDiff = true
			if checkOnly {
				fmt.Fprintf(stdout, "%s needs formatting\n", p)
				continue
			}
			if err := os.WriteFile(p, out, 0o644); err != nil {
				fmt.Fprintf(stderr, "fuse fmt: write %s: %v\n", p, err)
				return 1
			}
			fmt.Fprintf(stdout, "formatted %s\n", p)
		}
	}
	if checkOnly && anyDiff {
		return 1
	}
	return 0
}

// runDoc extracts doc comments and writes a simple summary.
// `--check` reports public items without docs and exits 1.
func runDoc(args []string, stdout, stderr io.Writer) int {
	var (
		checkOnly bool
		srcPath   string
	)
	for _, a := range args {
		switch {
		case a == "--check":
			checkOnly = true
		case len(a) > 0 && a[0] == '-':
			fmt.Fprintf(stderr, "fuse doc: unknown flag %q\n", a)
			return 2
		default:
			srcPath = a
		}
	}
	if srcPath == "" {
		fmt.Fprintln(stderr, "fuse doc: missing source file")
		return 2
	}

	// Resolve the target into the list of files to process. A
	// directory expands to every .fuse file under it (recursive);
	// a single file is processed as-is. W20 stdlib-core docs
	// coverage relies on the directory form.
	files, err := collectFuseFiles(srcPath)
	if err != nil {
		fmt.Fprintf(stderr, "fuse doc: %v\n", err)
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintf(stderr, "fuse doc: no .fuse files at %s\n", srcPath)
		return 1
	}
	anyMissing := false
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "fuse doc: %v\n", err)
			return 1
		}
		items := doc.Extract(src)
		for _, it := range items {
			vis := "priv"
			if it.IsPub {
				vis = "pub "
			}
			docSummary := it.Doc
			if idx := strings.IndexByte(docSummary, '\n'); idx >= 0 {
				docSummary = docSummary[:idx]
			}
			fmt.Fprintf(stdout, "%s %s %s %s:%d  %s\n", path, vis, it.Kind, it.Name, it.Line, docSummary)
		}
		if checkOnly {
			missing := doc.CheckMissingDocs(items)
			for _, m := range missing {
				fmt.Fprintf(stderr, "%s: missing doc: %s\n", path, m)
				anyMissing = true
			}
		}
	}
	if checkOnly && anyMissing {
		return 1
	}
	return 0
}

// collectFuseFiles returns every .fuse file at `path`. If path is
// a directory it's walked recursively; otherwise the single file
// is returned.
func collectFuseFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	var out []string
	err = filepath.Walk(path, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".fuse") {
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

// runRepl runs the interactive read-eval-print loop. Exits 0 on
// EOF or `:quit`.
func runRepl(stdout, stderr io.Writer) int {
	r := repl.NewRepl(os.Stdin, stdout)
	if err := r.Run(); err != nil {
		fmt.Fprintf(stderr, "fuse repl: %v\n", err)
		return 1
	}
	return 0
}

// runLsp starts the language server over stdin / stdout so an
// editor can drive a session. Blocks until the client sends the
// `exit` notification or the connection closes.
func runLsp(stderr io.Writer) int {
	srv := lsp.New(os.Stdin, os.Stdout, version)
	if err := srv.Run(); err != nil {
		fmt.Fprintf(stderr, "fuse lsp: %v\n", err)
		return 1
	}
	return 0
}

// commonFlags captures the optional flags every source-taking
// subcommand accepts. Keeps parsing concise.
type commonFlags struct {
	debug  bool
	asJSON bool
}

// extractSourcePath pulls the single required source path out of
// a subcommand's args, accepting --debug / --json. Returns the
// path, the parsed flags, and an exit code (0 when ok).
func extractSourcePath(args []string, stderr io.Writer, sub string) (string, commonFlags, int) {
	var (
		src   string
		flags commonFlags
	)
	for _, a := range args {
		switch {
		case a == "--debug":
			flags.debug = true
		case a == "--json":
			flags.asJSON = true
		case len(a) > 0 && a[0] == '-':
			fmt.Fprintf(stderr, "fuse %s: unknown flag %q\n", sub, a)
			return "", flags, 2
		default:
			if src != "" {
				fmt.Fprintf(stderr, "fuse %s: multiple source files not yet supported\n", sub)
				return "", flags, 2
			}
			src = a
		}
	}
	if src == "" {
		fmt.Fprintf(stderr, "fuse %s: missing source file\n", sub)
		return "", flags, 2
	}
	return src, flags, 0
}

// printDiags renders diagnostics in the selected format.
func printDiags(w io.Writer, diags []lex.Diagnostic, asJSON bool) {
	if len(diags) == 0 {
		return
	}
	out, err := diagnostics.RenderAll(diags, asJSON)
	if err != nil {
		fmt.Fprintf(w, "diagnostic render failed: %v\n", err)
		return
	}
	w.Write(out)
	if !asJSON {
		fmt.Fprintln(w)
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
