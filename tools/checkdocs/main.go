// Command checkdocs verifies that the foundational docs exist and, for the
// -foundational mode, that each is non-empty and readable.
//
// Later waves extend this tool with content-level checks (cross-link
// validation in W30, book structure, etc.). At Wave 00 the scope is narrow:
// confirm the five foundational files and their per-wave companions are
// present, tracked, and non-empty.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var foundationalDocs = []string{
	"docs/fuse-language-reference.md",
	"docs/implementation-plan.md",
	"docs/repository-layout.md",
	"docs/rules.md",
	"docs/learning-log.md",
}

var perWaveDocs = []string{
	"docs/implementation/wave00_governance.md",
	"docs/implementation/wave01_lexer.md",
	"docs/implementation/wave02_parser_and_ast.md",
	"docs/implementation/wave03_resolution.md",
	"docs/implementation/wave04_hir_and_typetable.md",
	"docs/implementation/wave05_minimal_end_to_end_spine.md",
	"docs/implementation/wave06_type_checking.md",
	"docs/implementation/wave07_concurrency_semantics.md",
	"docs/implementation/wave08_monomorphization.md",
	"docs/implementation/wave09_ownership_and_liveness.md",
	"docs/implementation/wave10_pattern_matching.md",
	"docs/implementation/wave11_error_propagation.md",
	"docs/implementation/wave12_closures_and_callable_traits.md",
	"docs/implementation/wave13_trait_objects.md",
	"docs/implementation/wave14_compile_time_evaluation.md",
	"docs/implementation/wave15_lowering_and_mir_consolidation.md",
	"docs/implementation/wave16_runtime_abi.md",
	"docs/implementation/wave17_codegen_c11_hardening.md",
	"docs/implementation/wave18_cli_and_diagnostics.md",
	"docs/implementation/wave19_language_server.md",
	"docs/implementation/wave20_stdlib_core.md",
	"docs/implementation/wave21_custom_allocators.md",
	"docs/implementation/wave22_stdlib_hosted.md",
	"docs/implementation/wave23_package_management.md",
	"docs/implementation/wave24_stub_clearance_gate.md",
	"docs/implementation/wave25_stage2_and_self_hosting.md",
	"docs/implementation/wave26_native_backend_transition.md",
	"docs/implementation/wave27_performance_gate.md",
	"docs/implementation/wave28_retirement_of_go_and_c.md",
	"docs/implementation/wave29_targets_and_native_expansion.md",
	"docs/implementation/wave30_ecosystem_documentation.md",
}

func main() {
	var foundational bool
	flag.BoolVar(&foundational, "foundational", false, "check only the five foundational docs")
	flag.Parse()

	var files []string
	if foundational {
		files = foundationalDocs
	} else {
		files = append(files, foundationalDocs...)
		files = append(files, perWaveDocs...)
		files = append(files, "docs/phase-model.md")
	}

	var problems []string
	for _, f := range files {
		info, err := os.Stat(filepath.FromSlash(f))
		if err != nil {
			problems = append(problems, f+": missing")
			continue
		}
		if info.IsDir() {
			problems = append(problems, f+": is a directory")
			continue
		}
		if info.Size() == 0 {
			problems = append(problems, f+": empty")
		}
	}
	if len(problems) > 0 {
		fmt.Fprintln(os.Stderr, "checkdocs: problems:")
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "  -", p)
		}
		os.Exit(1)
	}
	fmt.Println("checkdocs: ok")
}
