package lower

import (
	"testing"

	"github.com/Tembocs/fuse5/compiler/mir"
)

// TestQuestionBranch — W11-P02-T01. A fn using `?` on an enum
// lowers to a MIR block that compares the receiver's discriminant
// to the Err variant's index and early-returns on match. The
// success path reuses the receiver register as the `?` value.
func TestQuestionBranch(t *testing.T) {
	mod, diags := pipeline(t, `
enum Status { Ok, Err }

fn check(b: Bool) -> Status {
	return match b {
		true => Status.Ok,
		false => Status.Err,
	};
}

fn run(b: Bool) -> Status {
	return check(b)?;
}

fn main() -> I32 { return 0; }
`)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	run := findFnByName(t, mod, "run")
	// TermIfEq block compares against Err's index (1). At least
	// one such terminator must appear.
	var haveBranch, haveExtraReturn bool
	returns := 0
	for _, blk := range run.Blocks {
		if blk.Term == mir.TermIfEq && blk.BranchConst == 1 {
			haveBranch = true
		}
		if blk.Term == mir.TermReturn {
			returns++
		}
	}
	if !haveBranch {
		t.Fatalf("expected a TermIfEq against Err's discriminant (1) in fn run")
	}
	if returns < 2 {
		t.Fatalf("expected at least 2 return blocks (early-return + success return), got %d", returns)
	}
	haveExtraReturn = returns >= 2
	_ = haveExtraReturn
}
