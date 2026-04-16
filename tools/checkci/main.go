// Command checkci verifies CI configuration: a workflow file exists and
// references the three tier-1 platforms (Linux, macOS, Windows). The plan
// requires CI on every push and PR for all three (W00 exit criteria).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// workflowCandidates is the set of locations the tool accepts for the main
// CI workflow. GitHub Actions requires .github/workflows/; the project may
// also keep helpers under .ci/.
var workflowCandidates = []string{
	".github/workflows/ci.yml",
	".github/workflows/ci.yaml",
}

var requiredPlatforms = []string{
	"ubuntu-latest",
	"macos-latest",
	"windows-latest",
}

func main() {
	var found string
	for _, p := range workflowCandidates {
		if info, err := os.Stat(filepath.FromSlash(p)); err == nil && !info.IsDir() {
			found = p
			break
		}
	}
	if found == "" {
		fmt.Fprintln(os.Stderr, "checkci: no workflow file at", strings.Join(workflowCandidates, " or "))
		os.Exit(1)
	}
	body, err := os.ReadFile(filepath.FromSlash(found))
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkci: read %s: %v\n", found, err)
		os.Exit(1)
	}
	text := string(body)

	var missing []string
	for _, p := range requiredPlatforms {
		if !strings.Contains(text, p) {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "checkci: workflow", found, "missing required platforms:")
		for _, m := range missing {
			fmt.Fprintln(os.Stderr, "  -", m)
		}
		os.Exit(1)
	}

	// Triggers: push and pull_request are required.
	if !strings.Contains(text, "push:") || !strings.Contains(text, "pull_request:") {
		fmt.Fprintln(os.Stderr, "checkci: workflow missing push and/or pull_request triggers")
		os.Exit(1)
	}

	fmt.Println("checkci: ok (", found, ")")
}
