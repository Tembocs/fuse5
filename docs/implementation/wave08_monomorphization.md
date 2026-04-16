# Wave 08: Monomorphization

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: make generic functions and generic types compile through the full
pipeline and produce correct running programs (L015).

Entry criterion: W07 done. Phase 00 confirms no overdue stubs.

Exit criteria:

- generic parameters scope correctly in bodies
- explicit type args at call sites resolve
- driver collects concrete instantiations after checking
- partial specializations rejected
- body duplication produces concrete functions with type substitution
- specialized function names are deterministic and distinct
- call sites rewritten to reference specialized names
- only concrete instantiations reach codegen
- unresolved types are a hard error before codegen
- proof programs: identity[I32], multiple instantiations; Option/Result
  specialization

Proof of completion:

```
go test ./compiler/monomorph/... -v
go test ./compiler/driver/... -run TestSpecializationInPipeline -v
go test ./tests/e2e/... -run TestIdentityGeneric -v
go test ./tests/e2e/... -run TestMultipleInstantiations -v
```

## Phase 00: Stub Audit [W08-P00-STUB-AUDIT]

- Task 01: Monomorphization audit [W08-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W08 -phase P00`

## Phase 01: Generic Parameter Scoping [W08-P01-SCOPING]

- Task 01: Register generic params [W08-P01-T01-REGISTER]
  Verify: `go test ./compiler/check/... -run TestGenericParamScoping -v`
- Task 02: Resolve explicit type args [W08-P01-T02-CALL-SITE-TYPE-ARGS]
  Verify: `go test ./compiler/check/... -run TestCallSiteTypeArgs -v`

## Phase 02: Instantiation Collection [W08-P02-COLLECT]

- Task 01: Scan for generic call sites [W08-P02-T01-SCAN]
  Verify: `go test ./compiler/driver/... -run TestInstantiationCollection -v`
- Task 02: Validate completeness [W08-P02-T02-VALIDATE]
  Verify: `go test ./compiler/driver/... -run TestPartialInstantiationRejected -v`

## Phase 03: Body Specialization [W08-P03-SPECIALIZE]

- Task 01: AST-level body duplication [W08-P03-T01-DUPLICATE]
  Verify: `go test ./compiler/monomorph/... -run TestBodyDuplication -v`
- Task 02: Specialized function names [W08-P03-T02-NAMES]
  Verify: `go test ./compiler/monomorph/... -run TestSpecializedNames -v`
- Task 03: Call-site rewriting [W08-P03-T03-REWRITE]
  Verify: `go test ./compiler/monomorph/... -run TestCallSiteRewrite -v`

## Phase 04: Driver Pipeline Integration [W08-P04-PIPELINE]

- Task 01: Insert specialization step [W08-P04-T01-INSERT]
  Verify: `go test ./compiler/driver/... -run TestSpecializationInPipeline -v`
- Task 02: Skip generic originals in codegen [W08-P04-T02-SKIP-ORIGINALS]
  Verify: `go test ./compiler/codegen/... -run TestGenericOriginalsSkipped -v`

## Phase 05: Generic Types (Option, Result) [W08-P05-GENERIC-TYPES]

- Task 01: Specialize generic enums [W08-P05-T01-SPEC-ENUMS]
  Verify: `go test ./compiler/codegen/... -run TestSpecializedEnumTypes -v`
- Task 02: Specialize generic structs [W08-P05-T02-SPEC-STRUCTS]
  Verify: `go test ./compiler/codegen/... -run TestGenericStructLayout -v`

## Phase 06: Generic Proof Programs [W08-P06-PROOF]

- Task 01: `identity_generic.fuse` [W08-P06-T01-IDENTITY]
  Verify: `go test ./tests/e2e/... -run TestIdentityGeneric -v`
- Task 02: `multiple_instantiations.fuse` [W08-P06-T02-MULTIPLE]
  Verify: `go test ./tests/e2e/... -run TestMultipleInstantiations -v`
- Task 03: Distinct specializations in C [W08-P06-T03-DISTINCT]
  Verify: `go test ./compiler/codegen/... -run TestDistinctSpecializations -v`

## Wave Closure Phase [W08-PCL-WAVE-CLOSURE]

- Task 01: Retire generics stubs [W08-PCL-T01-RETIRE]
  Verify: `go run tools/checkstubs/main.go -wave W08`
- Task 02: WC008 entry [W08-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC008" docs/learning-log.md`

