package check

import "testing"

// TestQuestionTypecheck — W11-P01-T01. `?` on a Result-shaped
// enum type-checks when the enclosing fn returns the same enum.
// An enum without an `Err` variant is rejected. A mismatched
// enclosing return type is rejected.
func TestQuestionTypecheck(t *testing.T) {
	t.Run("result-shape-ok", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
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
		wantClean(t, diags)
	})

	t.Run("no-err-variant-rejected", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
enum Color { Red, Blue }

fn oops(c: Color) -> Color {
	return c?;
}

fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "Err")
	})

	t.Run("non-enum-rejected", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
fn oops(x: I32) -> I32 {
	return x?;
}

fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "enum scrutinee")
	})

	t.Run("mismatched-enclosing-return-rejected", func(t *testing.T) {
		_, diags := checkSource(t, "m", "m.fuse", `
enum Status { Ok, Err }

fn check(b: Bool) -> Status {
	return match b {
		true => Status.Ok,
		false => Status.Err,
	};
}

fn oops(b: Bool) -> I32 {
	return check(b)?;
}

fn main() -> I32 { return 0; }
`)
		wantDiag(t, diags, "enclosing fn")
	})
}

// TestQuestionOptionTypecheck — W11-P01-T02. `?` works on an
// Option-shaped enum carrying an error variant named `Err` (the
// Fuse lexer reserves `None` as a keyword for LitNone, so
// user-defined `None` variants aren't spellable in source at
// W11; the checker still recognizes the name for forward-
// compatibility when the grammar opens it up). This sub-test
// uses the Err-marker form to exercise the same code path.
func TestQuestionOptionTypecheck(t *testing.T) {
	_, diags := checkSource(t, "m", "m.fuse", `
enum Maybe { Present, Err }

fn probe(b: Bool) -> Maybe {
	return match b {
		true => Maybe.Present,
		false => Maybe.Err,
	};
}

fn chain(b: Bool) -> Maybe {
	return probe(b)?;
}

fn main() -> I32 { return 0; }
`)
	wantClean(t, diags)
}
