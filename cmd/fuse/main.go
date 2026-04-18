// Command fuse is the Stage 1 Fuse compiler CLI entry point.
//
// At W05 the CLI supports `version`, `help`, and `build` — enough to
// compile the minimal end-to-end spine's proof programs. The full
// subcommand surface (`run`, `check`, `test`, `fmt`, `doc`, `repl`,
// incremental driver, audit) comes online in W18.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Tembocs/fuse5/compiler/driver"
)

// version is the Stage 1 compiler version string reported by `fuse version`.
// Pre-1.0 waves report the active wave; real version strings land in W18.
const version = "0.0.0-W05"

// run is the testable entry point. It returns the process exit code so that
// tests can drive main without calling os.Exit.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fuse: no subcommand; usage: fuse <version|help|build>")
		fmt.Fprintln(stderr, "note: most subcommands are W18 work; only version/help/build are live at W05")
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprintln(stdout, "fuse — Fuse Stage 1 compiler (W05 minimal spine)")
		fmt.Fprintln(stdout, "subcommands:")
		fmt.Fprintln(stdout, "  version           print compiler version and exit")
		fmt.Fprintln(stdout, "  help              show this message")
		fmt.Fprintln(stdout, "  build <file>      compile a Fuse source to a native binary")
		fmt.Fprintln(stdout, "                    flags: -o <path>, --keep-c")
		fmt.Fprintln(stdout, "note: run/check/test/fmt/doc/repl and the incremental driver land in W18")
		return 0
	case "build":
		return runBuild(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fuse: unknown subcommand %q (W05 CLI provides version, help, build)\n", args[0])
		return 2
	}
}

// runBuild parses the `build` subcommand's flags and invokes the
// driver. It accepts a simple flag surface (-o PATH, --keep-c) —
// full flag UX and error formatting land in W18.
func runBuild(args []string, stdout, stderr io.Writer) int {
	var (
		outPath string
		keepC   bool
		debug   bool
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
	result, diags, err := driver.Build(driver.BuildOptions{
		Source: srcPath,
		Output: outPath,
		KeepC:  keepC,
		Debug:  debug,
	})
	for _, d := range diags {
		fmt.Fprintf(stderr, "%s: %s\n", d.Span, d.Message)
		if d.Hint != "" {
			fmt.Fprintf(stderr, "  hint: %s\n", d.Hint)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "fuse build: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "fuse: wrote %s\n", result.BinaryPath)
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
