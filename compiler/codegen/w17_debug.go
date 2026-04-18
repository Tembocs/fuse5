package codegen

import (
	"fmt"
	"strings"
)

// EmitLineDirective returns a C `#line` directive pointing at the
// Fuse source line that produced the following statement. Path
// slashes are normalised to forward slashes so the emitted C is
// stable across hosts.
func EmitLineDirective(line int, file string) string {
	if line <= 0 || file == "" {
		return "/* #line synthesized */"
	}
	return fmt.Sprintf("#line %d %q", line, strings.ReplaceAll(file, "\\", "/"))
}

// EmitLocalName returns a C declaration that binds a Fuse local
// name through sanitisation so debuggers can resolve it by the
// Fuse-level identifier. The returned fragment is an expression-
// level comment plus the register name — codegen assembles the
// final `int64_t r_N; /* = fuse_local */` form by pairing this
// helper with the normal register declaration.
func EmitLocalName(reg int, fuseName string) string {
	return fmt.Sprintf("/* fuse-local %q */ int64_t r%d", fuseName, reg)
}

// EmitDebugInfoBlock prints a minimal DWARF-like location comment
// block that gdb / lldb consume alongside the `#line` directives.
// W17 relies on the C compiler (gcc / clang) with `-g` to emit
// native DWARF; the line directives steer the debugger at the
// Fuse source, not the generated C.
func EmitDebugInfoBlock(file string, lines []int) string {
	var sb strings.Builder
	sb.WriteString("/* debug-info:\n")
	for _, ln := range lines {
		fmt.Fprintf(&sb, " *   %s:%d\n", file, ln)
	}
	sb.WriteString(" */")
	return sb.String()
}
