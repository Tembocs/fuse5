// Command checkgov verifies governance artifacts.
//
// At Wave 00 the scope is .claude/current-wave.json — a tracked coordination
// file that names the active wave and phase so multi-machine agents and
// contributors stay aligned (Rule 11.1, 11.2).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type currentWave struct {
	CurrentWave  string `json:"current_wave"`
	CurrentPhase string `json:"current_phase"`
	Updated      string `json:"updated"`
	Notes        string `json:"notes,omitempty"`
}

var (
	waveIDRe  = regexp.MustCompile(`^W\d{2}$`)
	phaseIDRe = regexp.MustCompile(`^(P\d{2}|PCL)$`)
	dateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func main() {
	var checkCurrent bool
	flag.BoolVar(&checkCurrent, "current-wave", false, "check .claude/current-wave.json")
	flag.Parse()

	if !checkCurrent {
		// Default: run all governance checks we know about.
		checkCurrent = true
	}

	if checkCurrent {
		if err := validateCurrentWave(".claude/current-wave.json"); err != nil {
			fmt.Fprintln(os.Stderr, "checkgov:", err)
			os.Exit(1)
		}
	}
	fmt.Println("checkgov: ok")
}

func validateCurrentWave(path string) error {
	data, err := os.ReadFile(filepath.FromSlash(path))
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var cw currentWave
	if err := json.Unmarshal(data, &cw); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if !waveIDRe.MatchString(cw.CurrentWave) {
		return fmt.Errorf("current_wave %q is not Wxx", cw.CurrentWave)
	}
	if !phaseIDRe.MatchString(cw.CurrentPhase) {
		return fmt.Errorf("current_phase %q is not Pxx or PCL", cw.CurrentPhase)
	}
	if !dateRe.MatchString(cw.Updated) {
		return fmt.Errorf("updated %q is not YYYY-MM-DD", cw.Updated)
	}
	return nil
}
