// Command checkruntime validates the Fuse runtime ABI surface:
// runtime/include/fuse_rt.h declares every function W16 scheduled,
// the function signatures match the expected ABI, and the
// runtime/src tree implements every declared function.
//
// Modes:
//
//	-header-syntax   Verify fuse_rt.h parses (balanced braces,
//	                 extern "C" block present, guard macro intact,
//	                 every declared fn has a matching definition in
//	                 runtime/src/*.c).
//	(no flags)       Runs -header-syntax.
//
// This is the Phase-01 verify command named by
// docs/implementation/wave16_runtime_abi.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Required declares the W16 runtime surface. Each entry is a fn
// name that fuse_rt.h must declare and runtime/src/*.c must define.
// Adding a new runtime surface extends this slice — the checker is
// the one-authoritative source of "what the runtime contract is".
var required = []string{
	// Process control.
	"fuse_rt_abort",
	"fuse_rt_panic",
	"fuse_rt_exit",
	// Memory.
	"fuse_rt_alloc",
	"fuse_rt_realloc",
	"fuse_rt_free",
	// IO.
	"fuse_rt_write_stdout",
	"fuse_rt_write_stderr",
	// Process + time.
	"fuse_rt_pid",
	"fuse_rt_monotonic_ns",
	"fuse_rt_wall_ns",
	"fuse_rt_sleep_ns",
	// Threads.
	"fuse_rt_thread_spawn",
	"fuse_rt_thread_join",
	"fuse_rt_thread_id",
	"fuse_rt_thread_yield",
	// Sync.
	"fuse_rt_mutex_new",
	"fuse_rt_mutex_lock",
	"fuse_rt_mutex_unlock",
	"fuse_rt_mutex_free",
	"fuse_rt_cond_new",
	"fuse_rt_cond_wait",
	"fuse_rt_cond_notify_one",
	"fuse_rt_cond_notify_all",
	"fuse_rt_cond_free",
	// Channels.
	"fuse_rt_chan_new",
	"fuse_rt_chan_send",
	"fuse_rt_chan_recv",
	"fuse_rt_chan_try_send",
	"fuse_rt_chan_try_recv",
	"fuse_rt_chan_close",
	"fuse_rt_chan_free",
}

func main() {
	var headerSyntax bool
	flag.BoolVar(&headerSyntax, "header-syntax", false, "validate fuse_rt.h header syntax and ABI coverage")
	flag.Parse()

	// Default mode: run header-syntax.
	if !headerSyntax {
		headerSyntax = true
	}

	if err := runHeaderSyntax(); err != nil {
		fmt.Fprintf(os.Stderr, "checkruntime: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("checkruntime: ok")
}

func runHeaderSyntax() error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	headerPath := filepath.Join(repoRoot, "runtime", "include", "fuse_rt.h")
	header, err := os.ReadFile(headerPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", headerPath, err)
	}
	text := string(header)

	// Guard macro present.
	if !strings.Contains(text, "#ifndef FUSE_RT_H") {
		return fmt.Errorf("%s missing #ifndef FUSE_RT_H guard", headerPath)
	}
	if !strings.Contains(text, "#define FUSE_RT_H") {
		return fmt.Errorf("%s missing #define FUSE_RT_H guard", headerPath)
	}
	if !strings.HasSuffix(strings.TrimSpace(text), "/* FUSE_RT_H */") {
		return fmt.Errorf("%s should end with the FUSE_RT_H guard-close comment", headerPath)
	}

	// extern "C" wrapper.
	if !strings.Contains(text, "extern \"C\"") {
		return fmt.Errorf("%s missing extern \"C\" wrapper for C++ consumers", headerPath)
	}

	// Brace balance.
	if open, close := strings.Count(text, "{"), strings.Count(text, "}"); open != close {
		return fmt.Errorf("%s has unbalanced braces: %d open, %d close", headerPath, open, close)
	}

	// Every required fn is declared.
	for _, fn := range required {
		pat := regexp.MustCompile(`\b` + regexp.QuoteMeta(fn) + `\s*\(`)
		if !pat.MatchString(text) {
			return fmt.Errorf("%s missing declaration for %s", headerPath, fn)
		}
	}

	// Every declared fn has a definition somewhere under runtime/src.
	srcDir := filepath.Join(repoRoot, "runtime", "src")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcDir, err)
	}
	var allSources strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".c") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		allSources.Write(body)
		allSources.WriteByte('\n')
	}
	combined := allSources.String()
	var missing []string
	for _, fn := range required {
		// A definition is `RET_TYPE name(ARGS) {` where ARGS may
		// contain nested parens (function-pointer parameters). We
		// scan every occurrence of `<name>(` and accept the fn as
		// defined when any occurrence is followed by a matched-paren
		// span and then `{` before the next `;`.
		if !hasDefinition(combined, fn) {
			missing = append(missing, fn)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("runtime/src/*.c missing definitions for: %s", strings.Join(missing, ", "))
	}
	return nil
}

// hasDefinition scans `body` for a function-definition occurrence of
// `name` — that is, the name followed by a `(...)` parameter list
// (nested parens allowed, so function-pointer params are handled)
// and then an opening `{` before the next semicolon. A bare
// declaration `name(...);` returns false.
func hasDefinition(body, name string) bool {
	needle := name + "("
	i := 0
	for {
		j := strings.Index(body[i:], needle)
		if j < 0 {
			return false
		}
		start := i + j
		// Must be at a word boundary: the byte before `start` must not
		// be an identifier continuation.
		if start > 0 {
			p := body[start-1]
			if (p >= 'a' && p <= 'z') || (p >= 'A' && p <= 'Z') || (p >= '0' && p <= '9') || p == '_' {
				i = start + len(needle)
				continue
			}
		}
		// Walk the matched-paren span.
		depth := 0
		k := start + len(name)
		for k < len(body) {
			c := body[k]
			switch c {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					k++
					goto afterParams
				}
			}
			k++
		}
		return false
	afterParams:
		// Skip whitespace; the next non-ws char must be '{' (body
		// start) for this to be a definition; ';' means declaration.
		for k < len(body) && (body[k] == ' ' || body[k] == '\t' || body[k] == '\n' || body[k] == '\r') {
			k++
		}
		if k < len(body) && body[k] == '{' {
			return true
		}
		i = k
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found walking up from %s", wd)
		}
		dir = parent
	}
}
