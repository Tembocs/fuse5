package lex

import (
	"fmt"
	"strings"
	"testing"
)

// scan is a test helper: tokenize src and return the (non-EOF) tokens along
// with any diagnostics.
func scan(t *testing.T, src string) ([]Token, []Diagnostic) {
	t.Helper()
	sc := NewScanner("test.fuse", []byte(src))
	errs := sc.Run()
	toks := sc.Tokens()
	// Trim the trailing TokEOF for convenience; tests that care about EOF
	// position re-call Tokens() directly.
	if n := len(toks); n > 0 && toks[n-1].Kind == TokEOF {
		toks = toks[:n-1]
	}
	return toks, errs
}

// kinds returns just the TokenKind sequence from a token slice.
func kinds(toks []Token) []TokenKind {
	out := make([]TokenKind, len(toks))
	for i, t := range toks {
		out[i] = t.Kind
	}
	return out
}

func expectKinds(t *testing.T, src string, want ...TokenKind) []Token {
	t.Helper()
	toks, errs := scan(t, src)
	if len(errs) > 0 {
		t.Fatalf("unexpected diagnostics scanning %q: %v", src, errs)
	}
	got := kinds(toks)
	if len(got) != len(want) {
		t.Fatalf("scan %q: got %d tokens %v, want %d tokens %v",
			src, len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("scan %q: token %d kind = %s, want %s (got=%v want=%v)",
				src, i, got[i], want[i], got, want)
		}
	}
	return toks
}

// TestKeywords asserts that every reserved word maps to its dedicated token
// kind, and that the raw-string guard from reference §1.10 holds: `r#abc`
// is three tokens (IDENT HASH IDENT), not a raw-string opener.
func TestKeywords(t *testing.T) {
	for _, kw := range keywordList {
		toks, errs := scan(t, kw)
		if len(errs) > 0 {
			t.Errorf("keyword %q produced diagnostics: %v", kw, errs)
			continue
		}
		if len(toks) != 1 {
			t.Errorf("keyword %q produced %d tokens, want 1 (%v)", kw, len(toks), kinds(toks))
			continue
		}
		want := keywords[kw]
		if toks[0].Kind != want {
			t.Errorf("keyword %q lexed as %s, want %s", kw, toks[0].Kind, want)
		}
		if toks[0].Text != kw {
			t.Errorf("keyword %q text = %q, want %q", kw, toks[0].Text, kw)
		}
	}

	// A bare identifier that shadows the keyword prefix must still be an
	// identifier: `func`, `ifdef`, `returns` are not reserved.
	for _, id := range []string{"func", "ifdef", "returns", "structs", "let_var"} {
		toks, _ := scan(t, id)
		if len(toks) != 1 || toks[0].Kind != TokIdent || toks[0].Text != id {
			t.Errorf("identifier %q lexed as %v (text=%q), want single TokIdent", id, kinds(toks), firstText(toks))
		}
	}

	// Reference §1.10: `r#abc` must tokenize as IDENT("r") HASH IDENT("abc").
	// This is the forcing example for the raw-string full-pattern rule.
	toks, errs := scan(t, "r#abc")
	if len(errs) > 0 {
		t.Fatalf("`r#abc` produced diagnostics: %v", errs)
	}
	want := []TokenKind{TokIdent, TokHash, TokIdent}
	if got := kinds(toks); !equalKinds(got, want) {
		t.Fatalf("`r#abc` kinds = %v, want %v", got, want)
	}
	if toks[0].Text != "r" || toks[2].Text != "abc" {
		t.Errorf("`r#abc` idents = (%q, %q), want (\"r\", \"abc\")", toks[0].Text, toks[2].Text)
	}
}

// TestLiterals covers reference §1.3–§1.6 and §42.5 end-to-end: integer bases
// and suffixes, float forms, string/raw-string/c-string literals.
func TestLiterals(t *testing.T) {
	cases := []struct {
		name string
		src  string
		kind TokenKind
		text string
	}{
		{"int-decimal", "42", TokInt, "42"},
		{"int-underscore", "1_000_000", TokInt, "1_000_000"},
		{"int-suffix-i32", "42i32", TokInt, "42i32"},
		{"int-hex", "0xff_u8", TokInt, "0xff_u8"},
		{"int-oct", "0o77_usize", TokInt, "0o77_usize"},
		{"int-bin", "0b1010_i64", TokInt, "0b1010_i64"},
		{"float-plain", "1.0", TokFloat, "1.0"},
		{"float-suffix", "3.14f32", TokFloat, "3.14f32"},
		{"float-exp", "6.02e23", TokFloat, "6.02e23"},
		{"float-exp-signed-suffix", "1.5e-4f64", TokFloat, "1.5e-4f64"},
		{"string-simple", `"hello"`, TokString, `"hello"`},
		{"string-escape", `"tab\there"`, TokString, `"tab\there"`},
		{"string-escaped-quote", `"say \"hi\""`, TokString, `"say \"hi\""`},
		{"string-unicode-escape", `"u: \u{1F600}"`, TokString, `"u: \u{1F600}"`},
		{"raw-string-basic", `r"no \n escapes"`, TokRawString, `r"no \n escapes"`},
		{"raw-string-one-hash", `r#"can contain " freely"#`, TokRawString, `r#"can contain " freely"#`},
		{"raw-string-two-hash", `r##"can contain "# freely"##`, TokRawString, `r##"can contain "# freely"##`},
		{"cstring", `c"hello\n"`, TokCString, `c"hello\n"`},
		{"char-ascii", `'A'`, TokChar, `'A'`},
		{"char-escape", `'\n'`, TokChar, `'\n'`},
		{"char-unicode", `'\u{1F600}'`, TokChar, `'\u{1F600}'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			toks, errs := scan(t, tc.src)
			if len(errs) > 0 {
				t.Fatalf("scan %q: %v", tc.src, errs)
			}
			if len(toks) != 1 {
				t.Fatalf("scan %q: got %d tokens, want 1 (%v)", tc.src, len(toks), kinds(toks))
			}
			if toks[0].Kind != tc.kind {
				t.Errorf("scan %q: kind = %s, want %s", tc.src, toks[0].Kind, tc.kind)
			}
			if toks[0].Text != tc.text {
				t.Errorf("scan %q: text = %q, want %q", tc.src, toks[0].Text, tc.text)
			}
			if toks[0].Span.Start.Offset != 0 || toks[0].Span.End.Offset != len(tc.src) {
				t.Errorf("scan %q: span covers %d..%d, want 0..%d",
					tc.src, toks[0].Span.Start.Offset, toks[0].Span.End.Offset, len(tc.src))
			}
		})
	}

	// True and false are both reserved words and boolean literals — assert the
	// dedicated kinds so the parser can distinguish them from arbitrary
	// identifiers without an extra table lookup.
	if toks := expectKinds(t, "true false", TokTrue, TokFalse); len(toks) != 2 {
		t.Fatalf("true/false: got %d tokens", len(toks))
	}
}

// TestNestedBlockComment covers reference §1.2: block comments nest, and the
// outer comment only closes when its matching `*/` is found at depth one.
func TestNestedBlockComment(t *testing.T) {
	src := `/* outer /* inner */ still outer */ 1`
	toks, errs := scan(t, src)
	if len(errs) > 0 {
		t.Fatalf("nested block comment produced diagnostics: %v", errs)
	}
	if len(toks) != 1 {
		t.Fatalf("got %d tokens, want 1 (%v)", len(toks), kinds(toks))
	}
	if toks[0].Kind != TokInt || toks[0].Text != "1" {
		t.Errorf("post-comment token = (%s, %q), want (INT, \"1\")", toks[0].Kind, toks[0].Text)
	}

	// Doubly nested.
	src2 := `/* a /* b /* c */ b */ a */ 42`
	toks2, errs2 := scan(t, src2)
	if len(errs2) > 0 {
		t.Fatalf("doubly nested produced diagnostics: %v", errs2)
	}
	if len(toks2) != 1 || toks2[0].Kind != TokInt {
		t.Fatalf("doubly nested: got %v, want [INT]", kinds(toks2))
	}

	// Unterminated: explicit diagnostic, not silent EOF.
	_, errs3 := scan(t, `/* never closed`)
	if len(errs3) == 0 {
		t.Fatalf("unterminated block comment must produce a diagnostic")
	}
	if !strings.Contains(errs3[0].Message, "unterminated block comment") {
		t.Errorf("unterminated diagnostic = %q, want mention of unterminated block comment", errs3[0].Message)
	}
}

// TestRawStringGuard exercises reference §1.10: raw strings are only
// recognized on the full `r#*"..."#*` pattern. Counter-examples ensure the
// lexer does not greedily enter raw-string mode on `r` + `#`.
func TestRawStringGuard(t *testing.T) {
	// Forcing counter-example from the reference: `r#abc` is three tokens.
	expectKinds(t, "r#abc", TokIdent, TokHash, TokIdent)

	// `r"..."` is a raw string with zero hashes.
	expectKinds(t, `r"x"`, TokRawString)

	// `r#"..."#` — one hash.
	expectKinds(t, `r#"x"#`, TokRawString)

	// `r##"..."##` — two hashes, closer must match.
	expectKinds(t, `r##"x"##`, TokRawString)

	// A raw string that contains `"#` internally must not close early when
	// the opener has two hashes.
	toks, errs := scan(t, `r##"can contain "# freely"##`)
	if len(errs) > 0 {
		t.Fatalf("diagnostics: %v", errs)
	}
	if len(toks) != 1 || toks[0].Kind != TokRawString {
		t.Fatalf("raw-string `#` guard: kinds = %v, want [RAWSTRING]", kinds(toks))
	}

	// Unterminated raw string is a diagnostic, not silent.
	_, errs = scan(t, `r#"never closed`)
	if len(errs) == 0 {
		t.Fatalf("unterminated raw string must produce a diagnostic")
	}
}

// TestOptionalChainToken exercises reference §1.10: `?.` is one token.
// `??` is two `?` tokens, `? .` (with a space) is two tokens as well.
func TestOptionalChainToken(t *testing.T) {
	// Forcing example: `expr?.field`.
	toks, errs := scan(t, "x?.y")
	if len(errs) > 0 {
		t.Fatalf("diagnostics: %v", errs)
	}
	want := []TokenKind{TokIdent, TokQuestionDot, TokIdent}
	if got := kinds(toks); !equalKinds(got, want) {
		t.Fatalf("`x?.y` kinds = %v, want %v", got, want)
	}
	// The `?.` token must span exactly two bytes.
	if toks[1].Span.Len() != 2 || toks[1].Text != "?." {
		t.Errorf("`?.` token: text=%q, len=%d, want text=\"?.\" len=2", toks[1].Text, toks[1].Span.Len())
	}

	// `? .` — a space breaks the longest match.
	expectKinds(t, "x? .y", TokIdent, TokQuestion, TokDot, TokIdent)

	// `??` — error propagation followed by another question mark. Two QUESTION
	// tokens, *not* a single `??`.
	expectKinds(t, "x??", TokIdent, TokQuestion, TokQuestion)
}

// TestBomRejection covers reference §1.10: a UTF-8 BOM at the start of the
// file is a lexical error, not a silent strip.
func TestBomRejection(t *testing.T) {
	// BOM + `fn`.
	src := append([]byte{0xEF, 0xBB, 0xBF}, []byte("fn")...)
	sc := NewScanner("test.fuse", src)
	errs := sc.Run()
	if len(errs) == 0 {
		t.Fatalf("BOM must produce a diagnostic, got none")
	}
	if !strings.Contains(errs[0].Message, "byte-order mark") {
		t.Errorf("BOM diagnostic = %q, want mention of byte-order mark", errs[0].Message)
	}
	if errs[0].Hint == "" {
		t.Errorf("BOM diagnostic must carry a suggestion (Rule 6.17), got empty hint")
	}
	// Scanner must refuse to produce tokens (beyond EOF) after BOM rejection.
	toks := sc.Tokens()
	if len(toks) != 1 || toks[0].Kind != TokEOF {
		t.Errorf("after BOM, tokens = %v, want [EOF] only", kinds(toks))
	}

	// A BOM in the middle of a file is not a file-start BOM; it should not
	// trigger the file-start check. (It may produce a different error as an
	// unknown rune, but the BOM-specific diagnostic should not fire.)
	src2 := append([]byte("fn "), 0xEF, 0xBB, 0xBF)
	sc2 := NewScanner("mid.fuse", src2)
	errs2 := sc2.Run()
	for _, e := range errs2 {
		if strings.Contains(e.Message, "start of a Fuse source file") {
			t.Errorf("mid-file BOM triggered file-start diagnostic: %v", e)
		}
	}
}

// TestSpanStability: scanning the same input twice yields identical token
// streams with identical spans. Line/column advance correctly across LF and
// CRLF.
func TestSpanStability(t *testing.T) {
	src := "fn main() {\n    return 42;\n}\r\nstruct S {}"
	sc1 := NewScanner("a.fuse", []byte(src))
	_ = sc1.Run()
	sc2 := NewScanner("a.fuse", []byte(src))
	_ = sc2.Run()

	a := sc1.Tokens()
	b := sc2.Tokens()
	if len(a) != len(b) {
		t.Fatalf("stability: token counts differ (%d vs %d)", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("stability: token %d differs\n  run1 = %+v\n  run2 = %+v", i, a[i], b[i])
		}
	}

	// Line/column checks against the known layout.
	// `return` is on line 2 starting at column 5.
	var ret Token
	for _, tk := range a {
		if tk.Kind == TokKwReturn {
			ret = tk
			break
		}
	}
	if ret.Span.Start.Line != 2 || ret.Span.Start.Column != 5 {
		t.Errorf("return: span=%s, want line 2 col 5", ret.Span)
	}

	// The `struct` keyword follows a CRLF-terminated line: should be line 4.
	var strct Token
	for _, tk := range a {
		if tk.Kind == TokKwStruct {
			strct = tk
			break
		}
	}
	if strct.Span.Start.Line != 4 {
		t.Errorf("struct: span=%s, want line 4 (CRLF should be one newline)", strct.Span)
	}

	// Spans must be contiguous with increasing offsets.
	prev := -1
	for i, tk := range a {
		if tk.Span.Start.Offset < prev {
			t.Errorf("span %d starts at %d, before previous end %d", i, tk.Span.Start.Offset, prev)
		}
		prev = tk.Span.End.Offset
	}
}

// TestLexerFuzz is a deterministic corpus of pathological inputs. The lexer
// must never panic and must either emit a token stream or a diagnostic — the
// forbidden state is silent tokens (Rule 6.9). A true fuzz harness will be
// added later via `go test -fuzz`; this test is the regression anchor.
func TestLexerFuzz(t *testing.T) {
	seeds := []string{
		"",
		" \t\r\n\r\n  \t",
		"///",
		"/* /* */",
		"/* */ /* */",
		"fn main() {}",
		"let x = 0xDEADBEEFu32;",
		"let y = 0b1010_1010_i8;",
		"let z = 3.14e+10f64;",
		`"hello\n" "unterminated`,
		`r"x" r#"y"# r##"z"## r#abc`,
		"x?.y?.z?.w",
		"????",
		"..=",
		"<<=>>=",
		"match x { 1 => 2, _ => 3 }",
		"fn f[T, U, V](x: T) -> Result[U, V] {}",
		"impl<T> Trait for Box[T] where T: Send + Sync {}",
		"spawn do_work(chan[Int])",
		"c\"null\" c\"\"",
		// Garbage that must not panic.
		"`~@#$%^&*()",
		string([]byte{0xFF, 0xFE, 0xFD}),
		"\x00\x01\x02",
	}
	for i, s := range seeds {
		t.Run(fmt.Sprintf("seed-%02d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("scanner panicked on %q: %v", s, r)
				}
			}()
			sc := NewScanner("fuzz.fuse", []byte(s))
			_ = sc.Run()
			// Every scan must terminate with an EOF token.
			toks := sc.Tokens()
			if len(toks) == 0 || toks[len(toks)-1].Kind != TokEOF {
				t.Errorf("seed %q: missing terminal EOF", s)
			}
		})
	}
}

func firstText(toks []Token) string {
	if len(toks) == 0 {
		return ""
	}
	return toks[0].Text
}

func equalKinds(a, b []TokenKind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
