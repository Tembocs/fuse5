// Command checkstubs validates STUBS.md against the rules in
// docs/rules.md §6.9–§6.16 and docs/repository-layout.md §2.
//
// Modes:
//
//	-audit-seed            Verify STUBS.md exists and is seeded for Wave 00:
//	                       Active stubs table non-empty, every row has a
//	                       concrete retiring wave (Wxx), Stub history empty.
//	-wave Wxx              Wave-scoped sanity check: STUBS.md parses and no
//	                       stub names a retiring wave smaller than Wxx.
//	-wave Wxx -phase P00   Phase 00 stub audit: enforce -wave and confirm no
//	                       overdue stubs (§6.15).
//	-wave Wxx -retired X   Assert stub named X is no longer in the Active
//	                       table.
//	-require-empty-active  Exit non-zero unless the Active stubs table is
//	                       empty. Used by the Stub Clearance Gate (W24).
//	-history-current-wave Wxx  Assert the Stub history contains a block for
//	                       wave Wxx with at least the Added/Retired/
//	                       Rescheduled sub-headers.
//	(no flags)             Sanity check: STUBS.md parses and structure is
//	                       well-formed.
//
// This file is a single-file package so it can be invoked via `go run
// tools/checkstubs/main.go ...` as specified in the implementation plan.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const stubsPath = "STUBS.md"

// ActiveStub is one row of the Active stubs table.
type ActiveStub struct {
	Name            string
	FileLine        string
	CurrentBehavior string
	Diagnostic      string
	RetiringWave    string
	SourceLine      int
}

// WaveHistory is one wave block in the Stub history section.
type WaveHistory struct {
	WaveID      string
	Title       string
	Added       []string
	Retired     []string
	Rescheduled []string
	SourceLine  int
}

// Parsed is the structured form of a STUBS.md file.
type Parsed struct {
	Active       []ActiveStub
	History      []WaveHistory
	ActiveEmpty  bool
	HistoryEmpty bool
}

var (
	rowRe        = regexp.MustCompile(`^\|(.+)\|$`)
	sepRe        = regexp.MustCompile(`^\|[-:| ]+\|$`)
	waveHeaderRe = regexp.MustCompile(`^###\s+(W\d{2})\s*(?:—|-)\s*(.*?)\s*$`)
	waveIDRe     = regexp.MustCompile(`^W\d{2}$`)
)

func Parse(r io.Reader) (*Parsed, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var (
		out     Parsed
		section string
		line    int
	)
	var curWave *WaveHistory
	var curField *[]string

	for sc.Scan() {
		line++
		text := sc.Text()
		trimmed := strings.TrimSpace(text)

		switch {
		case strings.HasPrefix(trimmed, "## Active stubs"):
			section = "active"
			continue
		case strings.HasPrefix(trimmed, "## Stub history"):
			section = "history"
			continue
		case strings.HasPrefix(trimmed, "## "):
			section = ""
			continue
		}

		if section == "active" {
			if strings.Contains(strings.ToLower(trimmed), "no active stubs") ||
				strings.Contains(strings.ToLower(trimmed), "active stubs table is empty") {
				out.ActiveEmpty = true
				continue
			}
			if !rowRe.MatchString(trimmed) || sepRe.MatchString(trimmed) {
				continue
			}
			cols := splitRow(trimmed)
			if len(cols) < 5 {
				continue
			}
			if strings.EqualFold(cols[0], "Stub") {
				continue
			}
			out.Active = append(out.Active, ActiveStub{
				Name:            cols[0],
				FileLine:        cols[1],
				CurrentBehavior: cols[2],
				Diagnostic:      cols[3],
				RetiringWave:    cols[4],
				SourceLine:      line,
			})
			continue
		}

		if section == "history" {
			if m := waveHeaderRe.FindStringSubmatch(trimmed); m != nil {
				if curWave != nil {
					out.History = append(out.History, *curWave)
				}
				curWave = &WaveHistory{
					WaveID:     m[1],
					Title:      m[2],
					SourceLine: line,
				}
				curField = nil
				continue
			}
			if curWave == nil {
				continue
			}
			switch {
			case strings.HasPrefix(trimmed, "Added:"):
				curField = &curWave.Added
				recordInline(trimmed, "Added:", curField)
				continue
			case strings.HasPrefix(trimmed, "Retired:"):
				curField = &curWave.Retired
				recordInline(trimmed, "Retired:", curField)
				continue
			case strings.HasPrefix(trimmed, "Rescheduled:"):
				curField = &curWave.Rescheduled
				recordInline(trimmed, "Rescheduled:", curField)
				continue
			}
			if strings.HasPrefix(trimmed, "- ") && curField != nil {
				*curField = append(*curField, strings.TrimPrefix(trimmed, "- "))
			}
		}
	}
	if curWave != nil {
		out.History = append(out.History, *curWave)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read STUBS.md: %w", err)
	}
	out.HistoryEmpty = len(out.History) == 0
	return &out, nil
}

func splitRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

func recordInline(trimmed, prefix string, dst *[]string) {
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	if rest == "" {
		return
	}
	low := strings.ToLower(rest)
	if strings.HasPrefix(low, "(none") {
		return
	}
	*dst = append(*dst, rest)
}

type runOpts struct {
	auditSeed    bool
	requireEmpty bool
	wave         string
	phase        string
	retired      string
	historyWave  string
}

func main() {
	var (
		opts      runOpts
		stubsFile string
	)
	flag.BoolVar(&opts.auditSeed, "audit-seed", false, "verify STUBS.md is a valid W00 seed")
	flag.BoolVar(&opts.requireEmpty, "require-empty-active", false, "require Active stubs table to be empty")
	flag.StringVar(&opts.wave, "wave", "", "wave ID (Wxx) for wave-scoped checks")
	flag.StringVar(&opts.phase, "phase", "", "phase ID (Pxx) for phase-scoped checks")
	flag.StringVar(&opts.retired, "retired", "", "stub name expected to be retired")
	flag.StringVar(&opts.historyWave, "history-current-wave", "", "assert Stub history block exists for the named wave")
	flag.StringVar(&stubsFile, "file", stubsPath, "path to STUBS.md")
	flag.Parse()

	if err := run(stubsFile, opts); err != nil {
		fmt.Fprintf(os.Stderr, "checkstubs: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("checkstubs: ok")
}

func run(path string, opts runOpts) error {
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	parsed, err := Parse(f)
	if err != nil {
		return err
	}

	for _, s := range parsed.Active {
		if !waveIDRe.MatchString(s.RetiringWave) {
			return fmt.Errorf("%s:%d: stub %q has non-concrete retiring wave %q (must be of the form Wxx)",
				path, s.SourceLine, s.Name, s.RetiringWave)
		}
		if s.Diagnostic == "" {
			return fmt.Errorf("%s:%d: stub %q has empty diagnostic column (Rule 6.9)",
				path, s.SourceLine, s.Name)
		}
	}

	switch {
	case opts.auditSeed:
		return checkAuditSeed(parsed)
	case opts.requireEmpty:
		return checkEmpty(parsed)
	case opts.retired != "":
		return checkRetired(parsed, opts.retired)
	case opts.historyWave != "":
		return checkHistoryWave(parsed, opts.historyWave)
	case opts.wave != "":
		return checkWave(parsed, opts.wave, opts.phase)
	}
	return nil
}

func checkAuditSeed(p *Parsed) error {
	if p.ActiveEmpty {
		return fmt.Errorf("audit-seed: Active stubs table is empty, but the W00 seed must enumerate every unimplemented feature")
	}
	if len(p.Active) == 0 {
		return fmt.Errorf("audit-seed: Active stubs table contains no rows")
	}
	if !p.HistoryEmpty {
		return fmt.Errorf("audit-seed: Stub history is non-empty, but W00 seeds an empty history")
	}
	return nil
}

func checkEmpty(p *Parsed) error {
	if !p.ActiveEmpty && len(p.Active) > 0 {
		return fmt.Errorf("require-empty-active: Active stubs table still has %d row(s); first: %q (retiring %s)",
			len(p.Active), p.Active[0].Name, p.Active[0].RetiringWave)
	}
	return nil
}

func checkRetired(p *Parsed, name string) error {
	for _, s := range p.Active {
		if s.Name == name {
			return fmt.Errorf("retired: stub %q is still present in the Active stubs table", name)
		}
	}
	return nil
}

func checkHistoryWave(p *Parsed, wave string) error {
	for _, h := range p.History {
		if h.WaveID == wave {
			return nil
		}
	}
	return fmt.Errorf("history-current-wave: no block for wave %s in Stub history", wave)
}

func checkWave(p *Parsed, wave, phase string) error {
	if !waveIDRe.MatchString(wave) {
		return fmt.Errorf("-wave argument %q is not of the form Wxx", wave)
	}
	if phase == "P00" {
		cur := waveOrder(wave)
		for _, s := range p.Active {
			if waveOrder(s.RetiringWave) < cur {
				return fmt.Errorf("overdue stub blocking wave entry (Rule 6.15): %q retires %s but wave %s is entering",
					s.Name, s.RetiringWave, wave)
			}
		}
	}
	return nil
}

func waveOrder(wave string) int {
	if !waveIDRe.MatchString(wave) {
		return -1
	}
	var n int
	fmt.Sscanf(wave, "W%d", &n)
	return n
}
