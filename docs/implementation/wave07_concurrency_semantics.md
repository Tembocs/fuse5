# Wave 07: Concurrency Semantics

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: enforce `Send`/`Sync`/`Copy` marker traits, `Chan[T]` type checking,
`spawn` semantics (handle-returning), `ThreadHandle[T]`, and `@rank`
lock ordering. Checker-side enforcement only; runtime lowering happens in
W16.

Entry criterion: W06 done. Phase 00 confirms no overdue stubs.

State on entry: channel and thread-handle kind stubs exist in TypeTable
(added W04). Checker does not yet enforce Send/Sync bounds.

Exit criteria:

- `Send`, `Sync`, `Copy` declared as intrinsic marker traits; auto-impl
  rules enforced (reference §47.1)
- negative impls (`impl !Send for T {}`) work in the declaring module
- `Chan[T]` type checking rejects element-type mismatches (reference §17.6)
- `spawn fn() -> T { ... }` typed as `ThreadHandle[T]` (reference §39.1)
- spawn requires `Send` captures
- `@rank(N)` lock ordering enforced structurally (reference §17.6)

Proof of completion:

```
go test ./compiler/check/... -run TestSendSyncMarkerTraits -v
go test ./compiler/check/... -run TestChannelTypecheck -v
go test ./compiler/check/... -run TestSpawnHandleTyping -v
go test ./compiler/check/... -run TestLockRankingEnforcement -v
go test ./tests/e2e/... -run TestConcurrencyRejections -v
```

## Phase 00: Stub Audit [W07-P00-STUB-AUDIT]

- Task 01: Concurrency audit [W07-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W07 -phase P00`

## Phase 01: Marker Traits [W07-P01-MARKERS]

- Task 01: Intrinsic `Send`/`Sync`/`Copy` [W07-P01-T01-DECLARE]
  Verify: `go test ./compiler/check/... -run TestMarkerTraitDeclarations -v`
- Task 02: Auto-impl rules [W07-P01-T02-AUTO-IMPL]
  Verify: `go test ./compiler/check/... -run TestMarkerAutoImpl -v`
- Task 03: Negative impls [W07-P01-T03-NEG-IMPL]
  Verify: `go test ./compiler/check/... -run TestNegativeImpl -v`

## Phase 02: Channel Typecheck [W07-P02-CHANNEL]

- Task 01: `Chan[T]` type operations [W07-P02-T01-CHAN]
  Verify: `go test ./compiler/check/... -run TestChannelTypecheck -v`
- Task 02: Send bound on element type [W07-P02-T02-CHAN-SEND-BOUND]
  Verify: `go test ./compiler/check/... -run TestChannelSendBound -v`

## Phase 03: Spawn Typecheck [W07-P03-SPAWN]

- Task 01: Spawn typing to `ThreadHandle[T]` [W07-P03-T01-SPAWN-TYPING]
  Verify: `go test ./compiler/check/... -run TestSpawnHandleTyping -v`
- Task 02: Send bound on captured environment [W07-P03-T02-SPAWN-SEND]
  Verify: `go test ./compiler/check/... -run TestSpawnSendBound -v`
- Task 03: `Shared[T]` bounds [W07-P03-T03-SHARED-BOUND]
  Verify: `go test ./compiler/check/... -run TestSharedBounds -v`

## Phase 04: Lock Ranking [W07-P04-RANKING]

- Task 01: `@rank(N)` structural enforcement [W07-P04-T01-RANK]
  Verify: `go test ./compiler/check/... -run TestLockRankingEnforcement -v`

## Phase 05: Concurrency Proof Program [W07-P05-PROOF]

- Task 01: Rejection proof [W07-P05-T01-REJECTIONS]
  DoD: checker proof file asserts several rejections fire with the
  expected diagnostic (non-Send spawn, wrong-type send, double-lock rank).
  Verify: `go test ./tests/e2e/... -run TestConcurrencyRejections -v`

## Wave Closure Phase [W07-PCL-WAVE-CLOSURE]

- Task 01: Retire concurrency stubs [W07-PCL-T01-RETIRE]
  DoD: channel type kind, ThreadHandle type kind, marker traits retired.
  Runtime-side lowering (spawn → runtime call, channel ops → runtime call)
  remains stubbed with diagnostics, retiring W16.
  Verify: `go run tools/checkstubs/main.go -wave W07`
- Task 02: WC007 entry [W07-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC007" docs/learning-log.md`

