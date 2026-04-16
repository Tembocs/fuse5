// Command checklayout verifies the top-level repository tree matches
// docs/repository-layout.md §1.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// requiredDirs is the top-level tree from docs/repository-layout.md §1.
var requiredDirs = []string{
	"cmd/fuse",
	"compiler/diagnostics",
	"compiler/typetable",
	"compiler/ast",
	"compiler/lex",
	"compiler/parse",
	"compiler/resolve",
	"compiler/hir",
	"compiler/check",
	"compiler/liveness",
	"compiler/lower",
	"compiler/mir",
	"compiler/monomorph",
	"compiler/codegen",
	"compiler/cc",
	"compiler/driver",
	"compiler/fmt",
	"compiler/doc",
	"compiler/repl",
	"compiler/testrunner",
	"runtime/include",
	"runtime/src",
	"runtime/platform",
	"runtime/tests",
	"stdlib/core",
	"stdlib/full",
	"stdlib/ext",
	"stage2/src",
	"stage2/tests",
	"stage2/build",
	"tests/fixtures",
	"tests/e2e",
	"tests/bootstrap",
	"tests/property",
	"tests/perf",
	"examples",
	"tools",
	"docs",
	"docs/implementation",
}

// requiredFiles lists root-level files demanded by repository-layout.md §2.
var requiredFiles = []string{
	"STUBS.md",
	"go.mod",
	"Makefile",
	"README.md",
	".gitignore",
	"docs/fuse-language-reference.md",
	"docs/implementation-plan.md",
	"docs/repository-layout.md",
	"docs/rules.md",
	"docs/learning-log.md",
}

func main() {
	var missing []string
	for _, d := range requiredDirs {
		info, err := os.Stat(filepath.FromSlash(d))
		if err != nil || !info.IsDir() {
			missing = append(missing, "directory "+d)
		}
	}
	for _, f := range requiredFiles {
		info, err := os.Stat(filepath.FromSlash(f))
		if err != nil || info.IsDir() {
			missing = append(missing, "file "+f)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "checklayout: repository layout does not match docs/repository-layout.md:")
		for _, m := range missing {
			fmt.Fprintln(os.Stderr, "  missing:", m)
		}
		os.Exit(1)
	}
	fmt.Println("checklayout: ok")
}
