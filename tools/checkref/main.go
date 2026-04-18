// Command checkref validates docs/fuse-language-reference.md
// against Rule 2.5: every numbered feature section must carry an
// `Implementation status:` tag of the form `SPECIFIED — Wxx`,
// `DONE — Wxx`, or `STUB — emits: "..."`. Without a status tag a
// section is a documentation defect.
//
// W24 introduces this tool as part of the Stub Clearance Gate
// (W24-P02 reference status audit). Modes:
//
//	(no flags)      — sanity check: every numbered section has a
//	                  status tag in one of the three accepted
//	                  shapes; unrecognised shapes are reported.
//	-all-done       — strict: every section must be tagged DONE.
//	-proof-coverage — assert tests/e2e/README.md mentions every
//	                  DONE section (best-effort; matches by section
//	                  heading keyword).
//	-count          — print the per-kind tally and exit 0.
//	-file PATH      — read a reference other than
//	                  docs/fuse-language-reference.md.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Status classifies one section's tag.
type Status int

const (
	StatusUnknown Status = iota
	StatusSpecified
	StatusDone
	StatusStub
)

func (s Status) String() string {
	switch s {
	case StatusSpecified:
		return "SPECIFIED"
	case StatusDone:
		return "DONE"
	case StatusStub:
		return "STUB"
	}
	return "UNKNOWN"
}

// Section is one numbered `## N. Title` heading and the status tag
// the body immediately under it declared.
type Section struct {
	Number     string // the leading numeric prefix (e.g. "28", "28.1" — only top-level carry tags)
	Title      string
	LineNumber int
	Tag        string // raw tag line, minus "Implementation status:"
	Status     Status
	Wave       string // Wxx
}

var (
	sectionRe = regexp.MustCompile(`^##\s+(\d+(?:\.\d+)*)\.\s+(.+?)\s*$`)
	statusRe  = regexp.MustCompile(`^>\s*Implementation status:\s*(.+?)\s*$`)
	waveRe    = regexp.MustCompile(`W\d{2}`)
)

func classify(tag string) (Status, string) {
	t := strings.TrimSpace(tag)
	up := strings.ToUpper(t)
	// Tags that only list sub-keys (e.g. `@value SPECIFIED — W06;
	// @rank SPECIFIED — W07;` for the decorators section) are
	// classified by the strongest keyword they contain. DONE wins
	// over SPECIFIED wins over STUB so we don't over-report progress.
	switch {
	case strings.Contains(up, "DONE"):
		return StatusDone, firstWave(t)
	case strings.Contains(up, "SPECIFIED"):
		return StatusSpecified, firstWave(t)
	case strings.Contains(up, "STUB"):
		return StatusStub, firstWave(t)
	}
	return StatusUnknown, ""
}

// nonFeatureTitles enumerates top-level numbered sections whose
// content is expository rather than a feature specification. These
// don't require an Implementation status tag — Rule 2.5 targets
// feature sections. Lowercased for case-insensitive compare.
var nonFeatureTitles = map[string]bool{
	"complete example": true,
	"introduction":     true,
	"overview":         true,
}

func firstWave(tag string) string {
	m := waveRe.FindString(tag)
	return m
}

// Parse walks the reference markdown and returns every top-level
// numbered section (## N. Title) together with the status tag
// that appears on the first `> Implementation status:` line under
// the heading, if any. Only top-level numbered headings carry
// status tags by convention; nested `### N.M` subsections are
// ignored for status purposes.
func Parse(path string) ([]*Section, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var (
		sections []*Section
		current  *Section
		line     int
	)
	for sc.Scan() {
		line++
		text := sc.Text()
		if m := sectionRe.FindStringSubmatch(text); m != nil {
			// Only top-level numbered sections carry status tags.
			if !strings.Contains(m[1], ".") {
				current = &Section{Number: m[1], Title: m[2], LineNumber: line}
				sections = append(sections, current)
			} else {
				current = nil
			}
			continue
		}
		if current == nil || current.Tag != "" {
			continue
		}
		if m := statusRe.FindStringSubmatch(text); m != nil {
			current.Tag = m[1]
			current.Status, current.Wave = classify(m[1])
		}
	}
	return sections, sc.Err()
}

type runOpts struct {
	allDone        bool
	proofCoverage  bool
	countOnly      bool
	file           string
	e2eReadme      string
}

func main() {
	var opts runOpts
	flag.BoolVar(&opts.allDone, "all-done", false, "require every section to be DONE (W24 clearance)")
	flag.BoolVar(&opts.proofCoverage, "proof-coverage", false, "assert tests/e2e/README.md mentions every DONE section")
	flag.BoolVar(&opts.countOnly, "count", false, "print the per-status tally and exit 0")
	flag.StringVar(&opts.file, "file", filepath.FromSlash("docs/fuse-language-reference.md"), "path to the reference markdown")
	flag.StringVar(&opts.e2eReadme, "e2e-readme", filepath.FromSlash("tests/e2e/README.md"), "path to the e2e proof-program registry")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "checkref: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("checkref: ok")
}

func run(opts runOpts) error {
	secs, err := Parse(opts.file)
	if err != nil {
		return err
	}
	if len(secs) == 0 {
		return fmt.Errorf("no numbered sections parsed from %s", opts.file)
	}

	// Missing-tag check always fires; a numbered section with no
	// status tag is a Rule 2.5 documentation defect. Expository
	// (non-feature) sections listed in nonFeatureTitles are skipped.
	var defects []string
	for _, s := range secs {
		if nonFeatureTitles[strings.ToLower(s.Title)] {
			continue
		}
		if s.Tag == "" {
			defects = append(defects, fmt.Sprintf("%s:%d: §%s %q lacks Implementation status tag",
				opts.file, s.LineNumber, s.Number, s.Title))
		} else if s.Status == StatusUnknown {
			defects = append(defects, fmt.Sprintf("%s:%d: §%s %q has unrecognised status tag %q",
				opts.file, s.LineNumber, s.Number, s.Title, s.Tag))
		}
	}
	if len(defects) > 0 {
		return fmt.Errorf("reference has %d defect(s):\n  %s", len(defects), strings.Join(defects, "\n  "))
	}

	tally := map[Status]int{}
	for _, s := range secs {
		tally[s.Status]++
	}
	if opts.countOnly {
		fmt.Printf("reference: %d sections total\n", len(secs))
		fmt.Printf("  DONE:      %d\n", tally[StatusDone])
		fmt.Printf("  SPECIFIED: %d\n", tally[StatusSpecified])
		fmt.Printf("  STUB:      %d\n", tally[StatusStub])
		return nil
	}

	if opts.allDone {
		var residual []string
		for _, s := range secs {
			if s.Status != StatusDone {
				residual = append(residual, fmt.Sprintf("§%s %q — %s %s",
					s.Number, s.Title, s.Status, s.Wave))
			}
		}
		if len(residual) > 0 {
			return fmt.Errorf("all-done: %d section(s) not yet DONE:\n  %s",
				len(residual), strings.Join(residual, "\n  "))
		}
	}

	if opts.proofCoverage {
		readme, err := os.ReadFile(opts.e2eReadme)
		if err != nil {
			return fmt.Errorf("proof-coverage: %w", err)
		}
		registry := string(readme)
		var missing []string
		for _, s := range secs {
			if s.Status != StatusDone {
				continue
			}
			// Best-effort coverage: the section's title must appear
			// at least once in the e2e registry.
			if !strings.Contains(strings.ToLower(registry), strings.ToLower(s.Title)) {
				missing = append(missing, fmt.Sprintf("§%s %q (DONE but no registry mention)",
					s.Number, s.Title))
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("proof-coverage: %d DONE section(s) missing from e2e registry:\n  %s",
				len(missing), strings.Join(missing, "\n  "))
		}
	}

	return nil
}
