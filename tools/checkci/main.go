// Command checkci verifies CI configuration: a workflow file exists and
// references the three tier-1 platforms (Linux, macOS, Windows). The plan
// requires CI on every push and PR for all three (W00 exit criteria).
package main

import (
	"encoding/json"
	"flag"
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
	var perfBaseline bool
	flag.BoolVar(&perfBaseline, "perf-baseline", false,
		"validate tests/perf/thresholds.json instead of CI workflow config (W17)")
	flag.Parse()

	if perfBaseline {
		if err := runPerfBaseline(); err != nil {
			fmt.Fprintf(os.Stderr, "checkci: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("checkci: perf baseline ok")
		return
	}

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

// runPerfBaseline validates the W17 perf corpus thresholds. The
// file must exist, be parseable, and declare a positive ceiling
// for every required benchmark name.
func runPerfBaseline() error {
	path := filepath.FromSlash("tests/perf/thresholds.json")
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var doc struct {
		Schema     string                    `json:"schema"`
		Benchmarks map[string]map[string]int `json:"benchmarks"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Schema != "fuse-perf/v1" {
		return fmt.Errorf("%s has unexpected schema %q", path, doc.Schema)
	}
	required := []string{
		"BenchmarkLexParseCorpus",
		"BenchmarkMonomorphHeavy",
		"BenchmarkTightArith",
		"BenchmarkChanSpawnRT",
	}
	for _, name := range required {
		entry, ok := doc.Benchmarks[name]
		if !ok {
			return fmt.Errorf("%s missing %q", path, name)
		}
		if len(entry) == 0 {
			return fmt.Errorf("%s:%q has no ceilings", path, name)
		}
		for k, v := range entry {
			if v <= 0 {
				return fmt.Errorf("%s:%q.%q must be positive (got %d)", path, name, k, v)
			}
		}
	}
	return nil
}
