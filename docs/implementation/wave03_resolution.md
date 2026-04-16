# Wave 03: Resolution

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: resolve symbols, imports, and the module graph; evaluate `@cfg`
predicates; enforce visibility levels.

Entry criterion: W02 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/resolve/` is an empty stub.

Exit criteria:

- module discovery deterministic
- imports resolve with module-first fallback (reference §18.7)
- import cycles diagnosed without hang
- qualified enum variant access resolves (reference §11.6)
- `@cfg` evaluated at resolve time; items with false predicate removed
  from the HIR-input stream (reference §50.1)
- four visibility levels enforced: private / `pub(mod)` / `pub(pkg)` / `pub`
  (reference §53.1)

Proof of completion:

```
go test ./compiler/resolve/... -v
go test ./compiler/resolve/... -run TestImportCycleDetection -v
go test ./compiler/resolve/... -run TestQualifiedEnumVariant -v
go test ./compiler/resolve/... -run TestModuleFirstFallback -v
go test ./compiler/resolve/... -run TestCfgEvaluation -v
go test ./compiler/resolve/... -run TestVisibilityEnforcement -v
```

## Phase 00: Stub Audit [W03-P00-STUB-AUDIT]

- Task 01: Resolve audit [W03-P00-T01-RESOLVE-STUB-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W03 -phase P00`

## Phase 01: Module Graph [W03-P01-MODULE-GRAPH]

- Task 01: Discover modules [W03-P01-T01-DISCOVER]
  Verify: `go test ./compiler/resolve/... -run TestModuleDiscovery -count=3 -v`
- Task 02: Build module graph [W03-P01-T02-GRAPH]
  Verify: `go test ./compiler/resolve/... -run TestModuleGraph -v`

## Phase 02: Symbols and Scopes [W03-P02-SYMBOLS]

- Task 01: Symbol table and scopes [W03-P02-T01-SYMBOLS]
  Verify: `go test ./compiler/resolve/... -run TestScopeLookup -v`
- Task 02: Top-level indexing [W03-P02-T02-INDEX]
  Verify: `go test ./compiler/resolve/... -run TestTopLevelIndex -v`

## Phase 03: Import and Path Resolution [W03-P03-IMPORTS]

- Task 01: Module-first fallback [W03-P03-T01-MODULE-FIRST]
  Verify: `go test ./compiler/resolve/... -run TestModuleFirstFallback -v`
- Task 02: Qualified enum variants [W03-P03-T02-QUALIFIED-VARIANTS]
  Verify: `go test ./compiler/resolve/... -run TestQualifiedEnumVariant -v`
- Task 03: Import cycles [W03-P03-T03-CYCLES]
  Verify: `go test ./compiler/resolve/... -run TestImportCycleDetection -v`

## Phase 04: Conditional Compilation [W03-P04-CFG]

- Task 01: `@cfg` evaluator [W03-P04-T01-CFG-EVAL]
  DoD: `@cfg(os = "linux")`, `not`, `all`, `any`, `feature = "name"` all
  evaluate; items with false predicate are removed from the HIR input.
  Verify: `go test ./compiler/resolve/... -run TestCfgEvaluation -v`
- Task 02: Duplicate-item resolution [W03-P04-T02-CFG-DUPLICATES]
  Verify: `go test ./compiler/resolve/... -run TestCfgDuplicates -v`

## Phase 05: Visibility Enforcement [W03-P05-VISIBILITY]

- Task 01: Enforce four visibility levels [W03-P05-T01-VIS-LEVELS]
  Verify: `go test ./compiler/resolve/... -run TestVisibilityEnforcement -v`

## Wave Closure Phase [W03-PCL-WAVE-CLOSURE]

- Task 01: Retire resolve stubs [W03-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W03 -retired resolve,cfg,visibility`
- Task 02: WC003 entry [W03-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC003" docs/learning-log.md`

