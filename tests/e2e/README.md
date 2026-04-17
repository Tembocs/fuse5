# Fuse End-to-End Proof Programs

> Status: normative root-level infrastructure for Fuse.
>
> This README is the live registry of every end-to-end Fuse proof
> program. A proof program is a `.fuse` file that compiles, links,
> and runs on Linux, macOS, and Windows, and whose expected
> observable output (exit code, stdout) is recorded here.
>
> Every wave that introduces user-visible behavior owns at least one
> proof program starting at Wave 05 (Rule 6.8, implementation-plan
> working principle 7). Adding a row here is part of the wave
> closure checklist; removing a row is not allowed.

## Program inventory

Each row names the source file, the wave that introduced it, the
expected exit code, any expected stdout / stderr content, and the
Go test that executes it.

| Program | Wave | Expected exit | Expected stdout | Driving test |
|---|---|---|---|---|
| `hello_exit.fuse` | W05 | `0` | (empty) | `TestHelloExit` in `tests/e2e/spine_test.go` |
| `exit_with_value.fuse` | W05 | `42` | (empty) | `TestExitWithValue` in `tests/e2e/spine_test.go` |

## Contract

- Every source here must be compilable by the current Stage 1 driver
  without any feature that is still in `STUBS.md` Active state.
- The driving Go test must: compile the program via
  `compiler/driver.Build`, run the produced binary, and assert the
  exit code (and stdout, when the row specifies one).
- Tests skip cleanly when the host has no C compiler; they do not
  fail silently. CI is configured to ensure a C compiler is present
  on every platform we target.
- Adding a proof program means adding: (1) the `.fuse` source, (2) a
  row in this README, (3) a `Test*` in `tests/e2e/spine_test.go`
  that the wave's Verify command exercises.

## Why

Proof programs close the "self-verifying plan" failure mode recorded
in L013: a feature is claimed complete only when its real, compiled
behavior matches the spec. Unit tests prove what the code *can* do
in isolation; proof programs prove the whole pipeline agrees end to
end.
