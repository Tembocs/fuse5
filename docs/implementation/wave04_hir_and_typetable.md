# Wave 04: HIR and TypeTable

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: establish the typed semantic IR surface, the pass graph, and the
AST-to-HIR bridge with proven type preservation.

Entry criterion: W03 done. Phase 00 confirms no overdue stubs.

State on entry: `compiler/hir/` and `compiler/typetable/` are empty stubs.

Exit criteria:

- HIR exists, is distinct from AST, carries required metadata
- HIR builders enforce required metadata at construction
- TypeTable interns all types; equality is integer comparison
- nominal identity includes defining symbol (reference §2.8)
- pass manifest validates declared metadata dependencies
- invariant walkers run in debug and CI
- AST-to-HIR bridge preserves resolved types — no `Unknown` defaults (L013)
- `KindChannel` and `KindThreadHandle` type kinds defined (used in W07)

Proof of completion:

```
go test ./compiler/hir/... -v
go test ./compiler/typetable/... -v
go test ./compiler/hir/... -run TestInvariantWalkers -v
go test ./compiler/hir/... -run TestBuilderEnforcement -v
go test ./compiler/hir/... -run TestAstToHirTypePreservation -v
go test ./compiler/hir/... -run TestDeterministicOrder -count=3 -v
```

## Phase 00: Stub Audit [W04-P00-STUB-AUDIT]

- Task 01: HIR and TypeTable audit [W04-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W04 -phase P00`

## Phase 01: TypeTable [W04-P01-TYPETABLE]

- Task 01: TypeId and interning [W04-P01-T01-TYPEID]
  Verify: `go test ./compiler/typetable/... -run TestTypeInternEquality -v`
- Task 02: Nominal identity [W04-P01-T02-NOMINAL-IDENTITY]
  Verify: `go test ./compiler/typetable/... -run TestNominalIdentity -v`
- Task 03: Channel type kind stub [W04-P01-T03-CHANNEL-KIND-STUB]
  DoD: `KindChannel` defined; checker integration is in STUBS.md,
  retiring W07.
  Verify: `go test ./compiler/typetable/... -run TestChannelTypeKindExists -v`
- Task 04: Thread handle type kind stub [W04-P01-T04-THREAD-HANDLE-STUB]
  DoD: `KindThreadHandle` defined; checker integration retires W07.
  Verify: `go test ./compiler/typetable/... -run TestThreadHandleKindExists -v`

## Phase 02: HIR Node Set and Metadata [W04-P02-HIR-NODES]

- Task 01: HIR node set [W04-P02-T01-NODES]
  DoD: HIR nodes are semantically oriented; patterns are structured nodes
  (`LiteralPat`, `BindPat`, `ConstructorPat`, `WildcardPat`, `OrPat`,
  `RangePat`, `AtBindPat`), not text (L007).
  Verify: `go test ./compiler/hir/... -run TestHirNodeSet -v`
- Task 02: Metadata fields [W04-P02-T02-METADATA]
  Verify: `go test ./compiler/hir/... -run TestMetadataFields -v`
- Task 03: Builder enforcement [W04-P02-T03-BUILDERS]
  Verify: `go test ./compiler/hir/... -run TestBuilderEnforcement -v`

## Phase 03: AST-to-HIR Bridge [W04-P03-BRIDGE]

- Task 01: Bridge with type propagation [W04-P03-T01-BRIDGE-IMPL]
  DoD: no expression receives `Unknown` as its default type (L013 defense).
  Verify: `go test ./compiler/hir/... -run TestAstToHirTypePreservation -v`
- Task 02: Bridge invariant walker [W04-P03-T02-BRIDGE-INVARIANT]
  Verify: `go test ./compiler/hir/... -run TestBridgeInvariant -v`

## Phase 04: Pass Graph and Determinism [W04-P04-PASS-GRAPH]

- Task 01: Pass manifest [W04-P04-T01-MANIFEST]
  Verify: `go test ./compiler/hir/... -run TestPassManifest -v`
- Task 02: Invariant walkers [W04-P04-T02-INVARIANTS]
  Verify: `go test ./compiler/hir/... -run TestInvariantWalkers -v`
- Task 03: Deterministic IR collections [W04-P04-T03-DETERMINISM]
  Verify: `go test ./compiler/hir/... -run TestDeterministicOrder -count=3 -v`

## Wave Closure Phase [W04-PCL-WAVE-CLOSURE]

- Task 01: Retire HIR/TypeTable/bridge stubs [W04-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W04 -retired hir,typetable,bridge`
- Task 02: WC004 entry [W04-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC004" docs/learning-log.md`

