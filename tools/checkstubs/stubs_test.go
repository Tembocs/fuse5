package main

import (
	"strings"
	"testing"
)

const seedExample = `# STUBS

## Active stubs

| Stub | File:Line | Current behavior | Diagnostic emitted | Retiring wave |
|---|---|---|---|---|
| Lexer | compiler/lex/ | compiler not yet implemented | "lexer not yet implemented" | W01 |
| Parser | compiler/parse/ | compiler not yet implemented | "parser not yet implemented" | W02 |

## Stub history

(Stub history is empty at Wave 00; entries are appended at each wave closure.)
`

func TestParseAndAuditSeed(t *testing.T) {
	p, err := Parse(strings.NewReader(seedExample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Active) != 2 {
		t.Fatalf("active rows = %d, want 2", len(p.Active))
	}
	if p.Active[0].RetiringWave != "W01" {
		t.Errorf("first stub retiring wave = %q, want W01", p.Active[0].RetiringWave)
	}
	if !p.HistoryEmpty {
		t.Errorf("history should be empty in seed")
	}
	if err := checkAuditSeed(p); err != nil {
		t.Errorf("checkAuditSeed: %v", err)
	}
}

func TestRejectsNonConcreteRetiringWave(t *testing.T) {
	bad := strings.Replace(seedExample, "W01", "TBD", 1)
	_, err := Parse(strings.NewReader(bad))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// Parse accepts TBD but the main run() rejects it via the baseline check.
	// Re-run the same regex used by run() to confirm the rejection path.
	if waveIDRe.MatchString("TBD") {
		t.Errorf("waveIDRe must reject TBD")
	}
}

func TestRequireEmptyActive(t *testing.T) {
	empty := `## Active stubs

The Active stubs table is empty.

## Stub history

### W24 — Stub Clearance Gate

Added: (none this wave)

Retired: (none this wave)

Rescheduled: (none this wave)
`
	p, err := Parse(strings.NewReader(empty))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !p.ActiveEmpty {
		t.Errorf("ActiveEmpty should be true")
	}
	if err := checkEmpty(p); err != nil {
		t.Errorf("checkEmpty on empty table: %v", err)
	}
}

func TestCheckRetired(t *testing.T) {
	p, _ := Parse(strings.NewReader(seedExample))
	if err := checkRetired(p, "Parser"); err == nil {
		t.Errorf("expected error when named stub is still present")
	}
	if err := checkRetired(p, "ClosuresW12"); err != nil {
		t.Errorf("expected no error for absent stub: %v", err)
	}
}

func TestCheckWaveOverdue(t *testing.T) {
	// Parser retires in W02. If the caller is entering W03 with P00, the
	// Parser stub is overdue and must block.
	p, _ := Parse(strings.NewReader(seedExample))
	err := checkWave(p, "W03", "P00")
	if err == nil {
		t.Fatalf("expected overdue-stub rejection entering W03 P00")
	}
}

func TestCheckWaveSameWaveNotOverdue(t *testing.T) {
	// Lexer retires in W01. Entering W01 P00 is the moment the wave's own
	// retirement work begins, so the stub must not be flagged as overdue.
	// (L016: earlier tool used `<=`, which made every wave unstartable.)
	p, _ := Parse(strings.NewReader(seedExample))
	if err := checkWave(p, "W01", "P00"); err != nil {
		t.Fatalf("stub retiring in entered wave must not be overdue: %v", err)
	}
}

func TestCheckHistoryWave(t *testing.T) {
	hist := `## Active stubs

The Active stubs table is empty.

## Stub history

### W00 — Governance and Phase Model

Added: (none this wave)

Retired: (none this wave)

Rescheduled: (none this wave)
`
	p, _ := Parse(strings.NewReader(hist))
	if err := checkHistoryWave(p, "W00"); err != nil {
		t.Errorf("expected W00 block present: %v", err)
	}
	if err := checkHistoryWave(p, "W01"); err == nil {
		t.Errorf("expected W01 block absent")
	}
}
