package check

import (
	"testing"
)

// TestExhaustivenessChecking — W10-P01-T01. A match over Bool that
// only covers `true` is non-exhaustive; a match over an enum that
// omits a variant is non-exhaustive. Total matches and matches
// that include a wildcard are clean.
func TestExhaustivenessChecking(t *testing.T) {
	t.Run("bool-both-arms-clean", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn pick(b: Bool) -> I32 {
	return match b {
		true => 1,
		false => 0,
	};
}
fn main() -> I32 { return 0; }
`)
		wantClean(t, diags)
	})

	t.Run("bool-only-true-rejected", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn pick(b: Bool) -> I32 {
	return match b {
		true => 1,
	};
}
fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "non-exhaustive")
		wantDiag(t, diags, "false")
	})

	t.Run("bool-with-wildcard-clean", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn pick(b: Bool) -> I32 {
	return match b {
		true => 1,
		_ => 0,
	};
}
fn main() -> I32 { return 0; }
`)
		wantClean(t, diags)
	})

	t.Run("enum-all-variants-clean", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
enum Dir { North, South }
fn pick(d: Dir) -> I32 {
	return match d {
		Dir.North => 0,
		Dir.South => 1,
	};
}
fn main() -> I32 { return 0; }
`)
		wantClean(t, diags)
	})

	t.Run("enum-missing-variant-rejected", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
enum Dir { North, South, East }
fn pick(d: Dir) -> I32 {
	return match d {
		Dir.North => 0,
		Dir.South => 1,
	};
}
fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "non-exhaustive")
		wantDiag(t, diags, "East")
	})
}

// TestUnreachableArmDetection — W10-P01-T02. An arm that follows
// a total arm is unreachable.
func TestUnreachableArmDetection(t *testing.T) {
	t.Run("arm-after-wildcard-is-unreachable", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn pick(b: Bool) -> I32 {
	return match b {
		_ => 0,
		true => 1,
	};
}
fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "unreachable match arm")
	})

	t.Run("duplicate-variant-is-unreachable", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
enum Dir { North, South }
fn pick(d: Dir) -> I32 {
	return match d {
		Dir.North => 0,
		Dir.North => 1,
		Dir.South => 2,
	};
}
fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "unreachable")
		wantDiag(t, diags, "North")
	})
}
