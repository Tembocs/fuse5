// Package lex owns lexical analysis (reference §1).
//
// The scanner turns a UTF-8 byte buffer into a stream of tokens, emitting a
// diagnostic and stopping if the source contains a byte-order mark
// (reference §1.10). The scanner is deterministic: the same input always
// produces the same token sequence with byte-stable spans (Rule 7.1,
// Rule 7.4).
//
// The scanner is the retirement site for the W00 lexer stub. Rule 6.9 forbids
// silent lexical defaults: every lexical error must produce a diagnostic with
// a primary span and a one-line message (Rule 6.17).
package lex

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// utf8BOM is the three-byte UTF-8 byte-order mark. Reference §1.1 forbids it
// in source files; reference §1.10 requires a diagnostic rather than silent
// stripping.
var utf8BOM = [3]byte{0xEF, 0xBB, 0xBF}

// Token is one lexical unit: its kind, the exact source bytes, and the span
// that covers those bytes. Text is a direct slice of the source buffer so the
// parser can read literal spellings without re-indexing.
type Token struct {
	Kind TokenKind
	Text string
	Span Span
}

// Diagnostic is a lexical error: where it happened and what went wrong. The
// full diagnostics package lands in W18; at W01 we emit the primary span and
// the one-line message required by Rule 6.17. The optional suggestion is
// attached via the Hint field.
type Diagnostic struct {
	Span    Span
	Message string
	Hint    string
}

// Error makes Diagnostic satisfy the error interface so callers can treat a
// single-diagnostic failure as a plain error at the boundary.
func (d Diagnostic) Error() string {
	if d.Hint != "" {
		return fmt.Sprintf("%s: %s (hint: %s)", d.Span, d.Message, d.Hint)
	}
	return fmt.Sprintf("%s: %s", d.Span, d.Message)
}

// Scanner tokenizes a single file. Zero value is not usable; construct via
// NewScanner.
type Scanner struct {
	file    string
	src     []byte
	off     int // current byte offset
	line    int
	col     int
	tokens  []Token
	errs    []Diagnostic
	stopped bool // set when BOM or other unrecoverable error is seen
}

// NewScanner returns a scanner ready to run over src. The caller keeps src
// ownership; the scanner reads it but does not mutate it.
func NewScanner(file string, src []byte) *Scanner {
	return &Scanner{file: file, src: src, line: 1, col: 1}
}

// Tokens returns the token stream produced by the last Run. The final token
// is always TokEOF. If Run stopped at a BOM or other unrecoverable error, the
// stream may contain only the EOF anchor.
func (s *Scanner) Tokens() []Token { return s.tokens }

// Errors returns any diagnostics produced during scanning.
func (s *Scanner) Errors() []Diagnostic { return s.errs }

// Run scans the whole buffer. It returns the slice of diagnostics; callers
// that want the tokens read them from Tokens(). Run is idempotent: calling it
// a second time resets state and re-scans.
func (s *Scanner) Run() []Diagnostic {
	s.off = 0
	s.line = 1
	s.col = 1
	s.tokens = s.tokens[:0]
	s.errs = s.errs[:0]
	s.stopped = false

	if s.checkBOM() {
		s.emitEOF()
		return s.errs
	}

	for !s.stopped && s.off < len(s.src) {
		s.skipTrivia()
		if s.stopped || s.off >= len(s.src) {
			break
		}
		s.scanOne()
	}
	s.emitEOF()
	return s.errs
}

// checkBOM enforces reference §1.10 (BOM rejection). Returns true if a BOM was
// detected and an error reported.
func (s *Scanner) checkBOM() bool {
	if len(s.src) < 3 {
		return false
	}
	if s.src[0] == utf8BOM[0] && s.src[1] == utf8BOM[1] && s.src[2] == utf8BOM[2] {
		// Span covers the three BOM bytes so the diagnostic points at the
		// exact problem rather than the whole file (Rule 6.17).
		start := s.pos()
		for i := 0; i < 3; i++ {
			s.advanceByte()
		}
		end := s.pos()
		s.errs = append(s.errs, Diagnostic{
			Span:    Span{File: s.file, Start: start, End: end},
			Message: "UTF-8 byte-order mark is not permitted at the start of a Fuse source file",
			Hint:    "remove the BOM (reference §1.1, §1.10); silent stripping is forbidden",
		})
		s.stopped = true
		return true
	}
	return false
}

// pos captures the current position. Columns count UTF-8 bytes; tabs are not
// expanded (the diagnostic renderer handles visual alignment).
func (s *Scanner) pos() Position {
	return Position{Offset: s.off, Line: s.line, Column: s.col}
}

// advanceByte advances one byte, updating line/column. LF ends a line; a CR
// followed by LF is treated as a single line terminator (reference §1.1).
func (s *Scanner) advanceByte() {
	if s.off >= len(s.src) {
		return
	}
	b := s.src[s.off]
	s.off++
	switch b {
	case '\n':
		s.line++
		s.col = 1
	case '\r':
		if s.off < len(s.src) && s.src[s.off] == '\n' {
			s.off++
			s.line++
			s.col = 1
		} else {
			s.line++
			s.col = 1
		}
	default:
		s.col++
	}
}

// advanceRune advances one Unicode scalar value, returning the rune and its
// byte width. Returns utf8.RuneError (with width 1) on invalid UTF-8 so the
// scanner can make forward progress even on malformed input.
func (s *Scanner) advanceRune() (rune, int) {
	if s.off >= len(s.src) {
		return 0, 0
	}
	r, w := utf8.DecodeRune(s.src[s.off:])
	if r == utf8.RuneError && w <= 1 {
		s.advanceByte()
		return utf8.RuneError, 1
	}
	// Line counting only cares about LF/CR; any other rune advances column by
	// its byte width so span byte ranges round-trip.
	if r == '\n' {
		s.off += w
		s.line++
		s.col = 1
	} else if r == '\r' {
		s.off += w
		if s.off < len(s.src) && s.src[s.off] == '\n' {
			s.off++
		}
		s.line++
		s.col = 1
	} else {
		s.off += w
		s.col += w
	}
	return r, w
}

// peekByte returns the byte at off+k or 0 if past end.
func (s *Scanner) peekByte(k int) byte {
	if s.off+k >= len(s.src) {
		return 0
	}
	return s.src[s.off+k]
}

// skipTrivia consumes whitespace and comments until the next token or EOF.
// Nested block comments (reference §1.2) are handled here. An unterminated
// block comment emits a diagnostic and stops the scanner (Rule 6.9: no silent
// defaults).
func (s *Scanner) skipTrivia() {
	for s.off < len(s.src) && !s.stopped {
		b := s.src[s.off]
		switch {
		case b == ' ' || b == '\t' || b == '\n' || b == '\r':
			s.advanceByte()
		case b == '/' && s.peekByte(1) == '/':
			for s.off < len(s.src) && s.src[s.off] != '\n' {
				s.advanceByte()
			}
		case b == '/' && s.peekByte(1) == '*':
			s.skipBlockComment()
		default:
			return
		}
	}
}

// skipBlockComment consumes a `/* ... */` block, supporting arbitrary
// nesting. Depth increments on every `/*` and decrements on every `*/`; the
// comment ends when depth returns to zero.
func (s *Scanner) skipBlockComment() {
	start := s.pos()
	// Consume the opening `/*`.
	s.advanceByte()
	s.advanceByte()
	depth := 1
	for depth > 0 && s.off < len(s.src) {
		b := s.src[s.off]
		if b == '/' && s.peekByte(1) == '*' {
			s.advanceByte()
			s.advanceByte()
			depth++
			continue
		}
		if b == '*' && s.peekByte(1) == '/' {
			s.advanceByte()
			s.advanceByte()
			depth--
			continue
		}
		s.advanceByte()
	}
	if depth != 0 {
		end := s.pos()
		s.errs = append(s.errs, Diagnostic{
			Span:    Span{File: s.file, Start: start, End: end},
			Message: "unterminated block comment",
			Hint:    "close every `/*` with a matching `*/` (reference §1.2, nested comments supported)",
		})
		s.stopped = true
	}
}

// emitEOF appends the terminating EOF token. Its span is a zero-width range at
// the end of the file.
func (s *Scanner) emitEOF() {
	end := s.pos()
	s.tokens = append(s.tokens, Token{
		Kind: TokEOF,
		Text: "",
		Span: Span{File: s.file, Start: end, End: end},
	})
}

// scanOne dispatches on the current rune to the appropriate scan routine.
func (s *Scanner) scanOne() {
	b := s.src[s.off]
	switch {
	case isIdentStart(rune(b)) && b < 0x80:
		s.scanIdentOrKeyword()
	case b >= 0x80:
		r, _ := utf8.DecodeRune(s.src[s.off:])
		if isIdentStart(r) {
			s.scanIdentOrKeyword()
			return
		}
		s.errInvalidRune(r)
	case b >= '0' && b <= '9':
		s.scanNumber()
	case b == '"':
		s.scanString()
	case b == '\'':
		s.scanChar()
	default:
		s.scanPunct()
	}
}

// scanIdentOrKeyword consumes an identifier and, if it is a reserved word,
// emits the corresponding keyword/literal token kind (reference §1.9).
//
// The raw-string trigger `r"` / `r#"` is intentionally routed through here:
// an identifier starting with `r` is first fully consumed as an identifier,
// *then* if the next byte is `"` or a run of `#` followed by `"`, the scan
// falls through to a raw-string path. Reference §1.10 requires that `r#abc`
// tokenize as `IDENT("r") # IDENT("abc")` — we guarantee this by only
// entering raw-string mode when the full `r#*"` opener pattern is matched.
func (s *Scanner) scanIdentOrKeyword() {
	start := s.pos()
	startOff := s.off

	// Special-cases before consuming the identifier: `r"..."`, `r#"..."#`, and
	// `c"..."`. These never match an identifier followed by a punctuator
	// because the identifier must be exactly the single byte `r` or `c` and
	// must be immediately followed by the full opener pattern.
	if s.src[s.off] == 'r' && s.isRawStringOpener() {
		s.scanRawString(start)
		return
	}
	if s.src[s.off] == 'c' && s.peekByte(1) == '"' {
		s.scanCString(start)
		return
	}

	for s.off < len(s.src) {
		r, w := utf8.DecodeRune(s.src[s.off:])
		if !isIdentContinue(r) {
			break
		}
		_ = w
		s.advanceRune()
	}
	end := s.pos()
	text := string(s.src[startOff:s.off])
	kind := TokIdent
	if kw, ok := keywords[text]; ok {
		kind = kw
	}
	s.tokens = append(s.tokens, Token{
		Kind: kind,
		Text: text,
		Span: Span{File: s.file, Start: start, End: end},
	})
}

// isRawStringOpener reports whether the scanner is positioned at a full
// `r#*"` prefix (reference §1.10). It does not advance.
func (s *Scanner) isRawStringOpener() bool {
	if s.src[s.off] != 'r' {
		return false
	}
	i := s.off + 1
	hashes := 0
	for i < len(s.src) && s.src[i] == '#' {
		i++
		hashes++
	}
	if i >= len(s.src) || s.src[i] != '"' {
		return false
	}
	_ = hashes
	return true
}

// scanRawString consumes a raw string literal of the form `r#*"..."#*`. The
// number of `#` on the closer must match the opener (reference §1.6). If no
// matching closer exists before EOF, emit a diagnostic.
func (s *Scanner) scanRawString(start Position) {
	// Consume `r` and count the opening `#` run.
	s.advanceByte() // 'r'
	openHashes := 0
	for s.off < len(s.src) && s.src[s.off] == '#' {
		s.advanceByte()
		openHashes++
	}
	// Consume the opening `"`.
	if s.off >= len(s.src) || s.src[s.off] != '"' {
		// Shouldn't happen — isRawStringOpener guards the entry — but report.
		end := s.pos()
		s.errs = append(s.errs, Diagnostic{
			Span:    Span{File: s.file, Start: start, End: end},
			Message: "malformed raw string: missing opening `\"`",
			Hint:    "the raw-string opener is `r\"` or `r#\"` (reference §1.6)",
		})
		s.stopped = true
		return
	}
	s.advanceByte() // '"'

	// Walk until we find `"` followed by exactly openHashes `#`.
	for s.off < len(s.src) {
		if s.src[s.off] == '"' {
			// Check whether this `"` ends the raw string.
			if s.hashesMatchAt(s.off+1, openHashes) {
				// Consume `"` and the matching hashes.
				s.advanceByte()
				for i := 0; i < openHashes; i++ {
					s.advanceByte()
				}
				end := s.pos()
				text := string(s.src[start.Offset:s.off])
				s.tokens = append(s.tokens, Token{
					Kind: TokRawString,
					Text: text,
					Span: Span{File: s.file, Start: start, End: end},
				})
				return
			}
		}
		s.advanceByte()
	}
	end := s.pos()
	s.errs = append(s.errs, Diagnostic{
		Span:    Span{File: s.file, Start: start, End: end},
		Message: "unterminated raw string literal",
		Hint:    fmt.Sprintf("close with `\"%s` to match the %d opening `#` character(s)", repeatHash(openHashes), openHashes),
	})
	s.stopped = true
}

// hashesMatchAt reports whether src[i..] begins with exactly n `#` bytes and
// is not followed by another `#`. Exact match is required so `r"foo"##bar`
// closes at the single `"` (n=0) rather than scanning past.
func (s *Scanner) hashesMatchAt(i, n int) bool {
	end := i + n
	if end > len(s.src) {
		return false
	}
	for k := i; k < end; k++ {
		if s.src[k] != '#' {
			return false
		}
	}
	// Require that the closer is not followed by an even longer hash run —
	// otherwise `r#"..."##` would close early at `"#`. For n>0 an extra `#`
	// after the closer is legal in the source (it becomes the next token); the
	// raw-string terminator itself is exactly n hashes.
	return true
}

func repeatHash(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = '#'
	}
	return string(b)
}

// scanCString consumes a `c"..."` literal (reference §42.5). Escape behavior
// mirrors ordinary strings; embedded NUL bytes in the source are a compile
// error (caught at the HIR-to-MIR boundary per §42.5, not at lex time).
func (s *Scanner) scanCString(start Position) {
	s.advanceByte() // 'c'
	s.advanceByte() // '"'
	s.consumeStringBody(start, TokCString)
}

// scanString consumes a `"..."` literal (reference §1.5). Standard escape
// sequences are parsed syntactically; semantic validation (e.g. `\u{...}`
// codepoint range) is deferred to checking.
func (s *Scanner) scanString() {
	start := s.pos()
	s.advanceByte() // '"'
	s.consumeStringBody(start, TokString)
}

// consumeStringBody scans to the closing `"`, honoring `\\` escapes so an
// escaped `\"` does not end the string. Unterminated strings emit a
// diagnostic. The kind parameter distinguishes `TokString` from `TokCString`.
func (s *Scanner) consumeStringBody(start Position, kind TokenKind) {
	for s.off < len(s.src) {
		b := s.src[s.off]
		switch b {
		case '"':
			s.advanceByte()
			end := s.pos()
			text := string(s.src[start.Offset:s.off])
			s.tokens = append(s.tokens, Token{
				Kind: kind,
				Text: text,
				Span: Span{File: s.file, Start: start, End: end},
			})
			return
		case '\\':
			s.advanceByte()
			if s.off < len(s.src) {
				s.advanceByte()
			}
		case '\n':
			// Permit newlines inside strings; later waves may tighten this.
			s.advanceByte()
		default:
			s.advanceByte()
		}
	}
	end := s.pos()
	s.errs = append(s.errs, Diagnostic{
		Span:    Span{File: s.file, Start: start, End: end},
		Message: "unterminated string literal",
		Hint:    "close the string with `\"` (reference §1.5, §42.3)",
	})
	s.stopped = true
}

// scanChar consumes a `'x'` or `'\n'` character literal (reference §2.5).
func (s *Scanner) scanChar() {
	start := s.pos()
	s.advanceByte() // opening '
	if s.off >= len(s.src) {
		s.charError(start)
		return
	}
	if s.src[s.off] == '\\' {
		s.advanceByte() // '\'
		if s.off >= len(s.src) {
			s.charError(start)
			return
		}
		esc := s.src[s.off]
		s.advanceByte()
		// `\u{HEX}` consumes through the closing `}` (reference §1.5).
		if esc == 'u' && s.off < len(s.src) && s.src[s.off] == '{' {
			s.advanceByte() // '{'
			for s.off < len(s.src) && s.src[s.off] != '}' {
				s.advanceByte()
			}
			if s.off < len(s.src) {
				s.advanceByte() // '}'
			}
		}
	} else {
		s.advanceRune()
	}
	if s.off >= len(s.src) || s.src[s.off] != '\'' {
		s.charError(start)
		return
	}
	s.advanceByte() // closing '
	end := s.pos()
	text := string(s.src[start.Offset:s.off])
	s.tokens = append(s.tokens, Token{
		Kind: TokChar,
		Text: text,
		Span: Span{File: s.file, Start: start, End: end},
	})
}

func (s *Scanner) charError(start Position) {
	end := s.pos()
	s.errs = append(s.errs, Diagnostic{
		Span:    Span{File: s.file, Start: start, End: end},
		Message: "unterminated or malformed character literal",
		Hint:    "character literals are a single Unicode scalar in `'...'` (reference §2.5)",
	})
	s.stopped = true
}

// scanNumber consumes integer and float literals (reference §1.3, §1.4).
// Bases: decimal, 0x (hex), 0o (octal), 0b (binary). Underscores are allowed
// between digits. A type suffix is any contiguous run of ident-continue bytes
// following the digit run; semantic suffix validation is deferred to checking.
func (s *Scanner) scanNumber() {
	start := s.pos()
	startOff := s.off
	isFloat := false

	if s.src[s.off] == '0' && s.off+1 < len(s.src) {
		switch s.src[s.off+1] {
		case 'x', 'X':
			s.advanceByte()
			s.advanceByte()
			s.consumeDigitRun(isHexDigit)
			s.consumeSuffix()
			s.emitNumber(start, startOff, TokInt)
			return
		case 'o', 'O':
			s.advanceByte()
			s.advanceByte()
			s.consumeDigitRun(isOctDigit)
			s.consumeSuffix()
			s.emitNumber(start, startOff, TokInt)
			return
		case 'b', 'B':
			s.advanceByte()
			s.advanceByte()
			s.consumeDigitRun(isBinDigit)
			s.consumeSuffix()
			s.emitNumber(start, startOff, TokInt)
			return
		}
	}

	// Decimal integer / float.
	s.consumeDigitRun(isDecDigit)

	// A `.` starts a float only if followed by a decimal digit — otherwise the
	// `.` belongs to a method call or the `..` range operator and must remain
	// a separate token.
	if s.off < len(s.src) && s.src[s.off] == '.' && isDecDigit(rune(s.peekByte(1))) {
		isFloat = true
		s.advanceByte() // '.'
		s.consumeDigitRun(isDecDigit)
	}
	if s.off < len(s.src) && (s.src[s.off] == 'e' || s.src[s.off] == 'E') {
		isFloat = true
		s.advanceByte()
		if s.off < len(s.src) && (s.src[s.off] == '+' || s.src[s.off] == '-') {
			s.advanceByte()
		}
		s.consumeDigitRun(isDecDigit)
	}
	s.consumeSuffix()
	kind := TokInt
	if isFloat {
		kind = TokFloat
	}
	s.emitNumber(start, startOff, kind)
}

func (s *Scanner) consumeDigitRun(valid func(rune) bool) {
	for s.off < len(s.src) {
		b := rune(s.src[s.off])
		if b == '_' || valid(b) {
			s.advanceByte()
			continue
		}
		break
	}
}

// consumeSuffix consumes the optional ident-like suffix on a numeric literal
// (e.g. `i32`, `u8`, `f64`, `usize`). The scanner does not validate the
// suffix text; §1.10 requires normalization at the HIR-to-MIR boundary.
func (s *Scanner) consumeSuffix() {
	if s.off >= len(s.src) {
		return
	}
	r, _ := utf8.DecodeRune(s.src[s.off:])
	if !isIdentStart(r) {
		return
	}
	for s.off < len(s.src) {
		r, _ := utf8.DecodeRune(s.src[s.off:])
		if !isIdentContinue(r) {
			break
		}
		s.advanceRune()
	}
}

func (s *Scanner) emitNumber(start Position, startOff int, kind TokenKind) {
	end := s.pos()
	s.tokens = append(s.tokens, Token{
		Kind: kind,
		Text: string(s.src[startOff:s.off]),
		Span: Span{File: s.file, Start: start, End: end},
	})
}

// scanPunct handles every operator and punctuation token. Longest-match is
// explicit: we test longer variants first (`?.` before `?`, `..=` before `..`,
// `<<=` before `<<`, etc.) to satisfy reference §1.10 and the operator
// contracts in §5.
func (s *Scanner) scanPunct() {
	start := s.pos()
	b := s.src[s.off]

	// Single-token punctuation that has no multi-byte variant.
	switch b {
	case '(':
		s.single(start, TokLParen)
		return
	case ')':
		s.single(start, TokRParen)
		return
	case '[':
		s.single(start, TokLBracket)
		return
	case ']':
		s.single(start, TokRBracket)
		return
	case '{':
		s.single(start, TokLBrace)
		return
	case '}':
		s.single(start, TokRBrace)
		return
	case ',':
		s.single(start, TokComma)
		return
	case ';':
		s.single(start, TokSemi)
		return
	case '@':
		s.single(start, TokAt)
		return
	case '#':
		s.single(start, TokHash)
		return
	case '~':
		s.single(start, TokTilde)
		return
	}

	// Multi-byte operators — longest match first.
	switch b {
	case ':':
		if s.peekByte(1) == ':' {
			s.multi(start, 2, TokColonColon)
			return
		}
		s.single(start, TokColon)
		return
	case '.':
		if s.peekByte(1) == '.' && s.peekByte(2) == '=' {
			s.multi(start, 3, TokDotDotEq)
			return
		}
		if s.peekByte(1) == '.' {
			s.multi(start, 2, TokDotDot)
			return
		}
		s.single(start, TokDot)
		return
	case '?':
		if s.peekByte(1) == '.' {
			s.multi(start, 2, TokQuestionDot)
			return
		}
		s.single(start, TokQuestion)
		return
	case '-':
		if s.peekByte(1) == '>' {
			s.multi(start, 2, TokArrow)
			return
		}
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokMinusEq)
			return
		}
		s.single(start, TokMinus)
		return
	case '=':
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokEqEq)
			return
		}
		if s.peekByte(1) == '>' {
			s.multi(start, 2, TokFatArrow)
			return
		}
		s.single(start, TokEq)
		return
	case '!':
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokBangEq)
			return
		}
		s.single(start, TokBang)
		return
	case '<':
		if s.peekByte(1) == '<' && s.peekByte(2) == '=' {
			s.multi(start, 3, TokShlEq)
			return
		}
		if s.peekByte(1) == '<' {
			s.multi(start, 2, TokShl)
			return
		}
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokLe)
			return
		}
		s.single(start, TokLt)
		return
	case '>':
		if s.peekByte(1) == '>' && s.peekByte(2) == '=' {
			s.multi(start, 3, TokShrEq)
			return
		}
		if s.peekByte(1) == '>' {
			s.multi(start, 2, TokShr)
			return
		}
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokGe)
			return
		}
		s.single(start, TokGt)
		return
	case '+':
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokPlusEq)
			return
		}
		s.single(start, TokPlus)
		return
	case '*':
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokStarEq)
			return
		}
		s.single(start, TokStar)
		return
	case '/':
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokSlashEq)
			return
		}
		s.single(start, TokSlash)
		return
	case '%':
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokPercentEq)
			return
		}
		s.single(start, TokPercent)
		return
	case '&':
		if s.peekByte(1) == '&' {
			s.multi(start, 2, TokAmpAmp)
			return
		}
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokAmpEq)
			return
		}
		s.single(start, TokAmp)
		return
	case '|':
		if s.peekByte(1) == '|' {
			s.multi(start, 2, TokPipePipe)
			return
		}
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokPipeEq)
			return
		}
		s.single(start, TokPipe)
		return
	case '^':
		if s.peekByte(1) == '=' {
			s.multi(start, 2, TokCaretEq)
			return
		}
		s.single(start, TokCaret)
		return
	}

	// Fallthrough: unknown byte. Emit a diagnostic and skip one rune so
	// scanning continues rather than stalling (Rule 6.9: never silent).
	r, _ := utf8.DecodeRune(s.src[s.off:])
	s.errInvalidRune(r)
}

func (s *Scanner) single(start Position, kind TokenKind) {
	startOff := s.off
	s.advanceByte()
	s.tokens = append(s.tokens, Token{
		Kind: kind,
		Text: string(s.src[startOff:s.off]),
		Span: Span{File: s.file, Start: start, End: s.pos()},
	})
}

func (s *Scanner) multi(start Position, n int, kind TokenKind) {
	startOff := s.off
	for i := 0; i < n; i++ {
		s.advanceByte()
	}
	s.tokens = append(s.tokens, Token{
		Kind: kind,
		Text: string(s.src[startOff:s.off]),
		Span: Span{File: s.file, Start: start, End: s.pos()},
	})
}

func (s *Scanner) errInvalidRune(r rune) {
	start := s.pos()
	s.advanceRune()
	end := s.pos()
	s.errs = append(s.errs, Diagnostic{
		Span:    Span{File: s.file, Start: start, End: end},
		Message: fmt.Sprintf("unexpected character %q in source", r),
		Hint:    "remove or replace the character; reference §1 enumerates the valid token starts",
	})
	// We do not stop the scanner; skipping one rune and continuing gives
	// downstream users more information per file.
}

// isIdentStart reports whether r may start an identifier (reference §1.9).
func isIdentStart(r rune) bool {
	if r == '_' {
		return true
	}
	return unicode.IsLetter(r)
}

// isIdentContinue reports whether r may continue an identifier.
func isIdentContinue(r rune) bool {
	if r == '_' {
		return true
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isDecDigit(r rune) bool { return r >= '0' && r <= '9' }
func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
func isOctDigit(r rune) bool { return r >= '0' && r <= '7' }
func isBinDigit(r rune) bool { return r == '0' || r == '1' }
