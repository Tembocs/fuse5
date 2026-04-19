# Wave 02: Parser and AST

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: build an AST-only parser that accepts the full surface grammar
defined in reference Appendix C without semantic shortcuts.

Entry criterion: W01 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/parse/` and `compiler/ast/` are empty stubs.

Exit criteria:

- parser handles every grammar construct in reference Appendix C, including
  `dyn Trait`, `impl Trait`, `union`, decorators (`@value`, `@rank`,
  `@repr`, `@align`, `@inline`, `@cold`, `@cfg`), `const fn`, or-patterns,
  range patterns, `@`-binding patterns, slice range indexing, and struct
  update syntax
- parser does not panic on malformed input
- AST remains syntax-only (no resolved symbols, no types, no metadata)
- struct literal disambiguation works per reference §10.7
- optional chaining parses per reference §1.10
- golden tests are byte-stable

Proof of completion:

```
go test ./compiler/parse/... -v
go test ./compiler/ast/... -v
go test ./compiler/parse/... -run TestGolden -count=3 -v
go test ./compiler/parse/... -run TestNopanicOnMalformed -v
```

## Phase 00: Stub Audit [W02-P00-STUB-AUDIT]

- Task 01: Audit [W02-P00-T01-PARSE-STUB-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W02 -phase P00`

## Phase 01: AST Surface [W02-P01-AST]

- Task 01: Define AST node set [W02-P01-T01-NODES]
  Verify: `go test ./compiler/ast/... -run TestAstNodeCompleteness -v`
- Task 02: Builders and span correctness [W02-P01-T02-BUILDERS]
  Verify: `go test ./compiler/ast/... -run TestSpanCorrectness -v`

## Phase 02: Core Parsing [W02-P02-CORE]

- Task 01: Items and declarations [W02-P02-T01-ITEMS]
  DoD: functions (including `const fn`), structs, enums, traits, impls,
  consts, statics, type aliases, externs (including variadic externs),
  unions parse correctly.
  Verify: `go test ./compiler/parse/... -run TestItemParsing -v`
- Task 02: Expressions and statements [W02-P02-T02-EXPRS]
  DoD: precedence per reference Appendix B; expression goldens pass; struct
  update syntax (`..base`), slice range indexing parse correctly.
  Verify: `go test ./compiler/parse/... -run TestExprPrecedence -v`
- Task 03: Type expressions [W02-P02-T03-TYPES]
  DoD: tuples, arrays, slices, raw pointers, generics, fn-types, `dyn
  Trait`, `impl Trait`, unions all parse.
  Verify: `go test ./compiler/parse/... -run TestTypeExprs -v`
- Task 04: Patterns [W02-P02-T04-PATTERNS]
  DoD: literal, wildcard, bind, tuple, struct, constructor, or, range,
  `@`-binding patterns all parse.
  Verify: `go test ./compiler/parse/... -run TestPatternParsing -v`
- Task 05: Decorators [W02-P02-T05-DECORATORS]
  DoD: `@value`, `@rank(N)`, `@repr(C)`, `@repr(packed)`, `@repr(Uxx)`,
  `@repr(Ixx)`, `@align(N)`, `@inline`, `@inline(always)`, `@inline(never)`,
  `@cold`, `@cfg(...)` parse with predicate arguments.
  Verify: `go test ./compiler/parse/... -run TestDecoratorParsing -v`

## Phase 03: Ambiguity Control [W02-P03-AMBIGUITY]

- Task 01: Struct-literal disambiguation [W02-P03-T01-STRUCT-LITERAL]
  Verify: `go test ./compiler/parse/... -run TestStructLiteralDisambig -v`
- Task 02: Optional chaining parse [W02-P03-T02-OPTIONAL-CHAIN]
  Verify: `go test ./compiler/parse/... -run TestOptionalChainParse -v`
- Task 03: Malformed input corpus [W02-P03-T03-MALFORMED]
  Verify: `go test ./compiler/parse/... -run TestNopanicOnMalformed -v`

## Wave Closure Phase [W02-PCL-WAVE-CLOSURE]

- Task 01: Retire parse/ast stubs [W02-PCL-T01-RETIRE-STUB]
  Verify: `go run tools/checkstubs/main.go -wave W02 -retired parse,ast`
- Task 02: WC002 entry [W02-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC002" docs/learning-log.md`

