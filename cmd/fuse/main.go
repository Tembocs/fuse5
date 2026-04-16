// Command fuse is the Stage 1 Fuse compiler CLI entry point.
//
// During Wave 00 the CLI is a stub that reports the project is pre-Wave-01
// and exits with a non-zero status when asked to build or run. Subcommands
// come online in later waves per docs/implementation-plan.md.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is the Stage 1 compiler version string reported by `fuse version`.
// Pre-1.0 waves report the active wave; real version strings land in W18.
const version = "0.0.0-W00"

// run is the testable entry point. It returns the process exit code so that
// tests can drive main without calling os.Exit.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fuse: no subcommand; usage: fuse <version|help>")
		fmt.Fprintln(stderr, "note: the Stage 1 CLI is a stub at Wave 00; most subcommands come online in later waves")
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprintln(stdout, "fuse — Fuse Stage 1 compiler (pre-W01 stub)")
		fmt.Fprintln(stdout, "subcommands: version, help")
		fmt.Fprintln(stdout, "note: build/run/check/test/fmt/doc/repl land in W18")
		return 0
	default:
		fmt.Fprintf(stderr, "fuse: unknown subcommand %q (Wave 00 CLI only provides version and help)\n", args[0])
		return 2
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
