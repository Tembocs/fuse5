# Wave 01: Lexer

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: build a deterministic lexer that covers the full token set and every
lexical ambiguity listed in the language reference.

Entry criterion: W00 done. Phase 00 of this wave confirms no overdue stubs.

State on entry: `compiler/lex/` is an empty stub package.

Exit criteria:

- every token kind in reference §1 is tested
- BOM is rejected (reference §1.10)
- nested block comments work
- raw strings obey the full-pattern rule (reference §1.10)
- `?.` is emitted as one token (reference §1.10)
- c-string literal `c"..."` lexes correctly (reference §42.5)
- golden tests are byte-stable across three runs

Proof of completion:

```
go test ./compiler/lex/... -v
go test ./compiler/lex/... -run TestGolden -count=3 -v
```

## Phase 00: Stub Audit [W01-P00-STUB-AUDIT]

- Task 01: Audit lex stub [W01-P00-T01-LEX-STUB-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W01 -phase P00`

## Phase 01: Token Model [W01-P01-TOKEN-MODEL]

- Task 01: Define token kinds [W01-P01-T01-TOKEN-KINDS]
  Verify: `go test ./compiler/lex/... -run TestTokenKindCoverage -v`
- Task 02: Define span model [W01-P01-T02-SPAN-MODEL]
  Verify: `go test ./compiler/lex/... -run TestSpanStability -v`

## Phase 02: Scanner Core [W01-P02-SCANNER]

- Task 01: Identifier and keyword scanning [W01-P02-T01-IDENT-KEYWORD]
  Verify: `go test ./compiler/lex/... -run TestKeywords -v`
- Task 02: Literal scanning [W01-P02-T02-LITERALS]
  DoD: integer (all bases), float, string, raw string, and c-string literal
  forms tokenize correctly.
  Verify: `go test ./compiler/lex/... -run TestLiterals -v`
- Task 03: Comments and trivia [W01-P02-T03-TRIVIA]
  Verify: `go test ./compiler/lex/... -run TestNestedBlockComment -v`

## Phase 03: Lexical Edge Cases [W01-P03-EDGES]

- Task 01: Raw string full-pattern rule [W01-P03-T01-RAW-STRING]
  Verify: `go test ./compiler/lex/... -run TestRawStringGuard -v`
- Task 02: `?.` longest-match [W01-P03-T02-OPTIONAL-CHAIN]
  Verify: `go test ./compiler/lex/... -run TestOptionalChainToken -v`
- Task 03: BOM rejection [W01-P03-T03-BOM]
  Verify: `go test ./compiler/lex/... -run TestBomRejection -v`
- Task 04: Golden and fuzz coverage [W01-P03-T04-TESTS]
  Verify: `go test ./compiler/lex/... -run TestLexerFuzz -v`

## Wave Closure Phase [W01-PCL-WAVE-CLOSURE]

- Task 01: Retire lexer stub [W01-PCL-T01-RETIRE-STUB]
  Verify: `go run tools/checkstubs/main.go -wave W01 -retired lexer`
- Task 02: WC001 entry [W01-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC001" docs/learning-log.md`

