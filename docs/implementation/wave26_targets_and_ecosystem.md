# Wave 26: Targets and Ecosystem

> Part of the [Fuse implementation plan](../implementation-plan.md).


Goal: resume broader target and library work on top of the native
self-hosted compiler.

Entry criterion: W25 done.

Exit criteria:

- target expansion proceeds without reintroducing bootstrap debt
- `stdlib/ext/` hosts optional libraries

Proof of completion:

```
fuse build --target=<new-target> tests/e2e/hello_exit.fuse -o /tmp/he_<target>
fuse build stdlib/ext/...
```

## Phase 00: Stub Audit [W26-P00-STUB-AUDIT]

- Task 01: Target audit [W26-P00-T01-AUDIT]
  Verify: `go run tools/checkstubs/main.go -wave W26 -phase P00`

## Phase 01: Additional Targets [W26-P01-TARGETS]

- Task 01: Target descriptions [W26-P01-T01-DESCRIPTIONS]
  Verify: `go run tools/checktargets/main.go`
- Task 02: Target CI [W26-P01-T02-TARGET-CI]
  Verify: `go run tools/checkci/main.go -target-matrix`

## Phase 02: Extended Libraries [W26-P02-EXT]

- Task 01: Implement ext stdlib [W26-P02-T01-EXT-STDLIB]
  Verify: `fuse build stdlib/ext/...`
- Task 02: Ecosystem guidance [W26-P02-T02-GUIDANCE]
  Verify: `test -f docs/ecosystem-guide.md`

## Wave Closure Phase [W26-PCL-WAVE-CLOSURE]

- Task 01: Stub history closure [W26-PCL-T01-HISTORY]
  Verify: `go run tools/checkstubs/main.go -wave W26`
- Task 02: WC026 entry [W26-PCL-T02-CLOSURE-LOG]
  Verify: `grep "WC026" docs/learning-log.md`

