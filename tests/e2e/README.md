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
| `checker_basic.fuse` | W06 | `42` | (empty) | `TestCheckerBasicProof` in `tests/e2e/spine_test.go` |
| (rejection proof — no `.fuse` source) | W07 | N/A (test asserts diagnostics) | N/A | `TestConcurrencyRejections` in `tests/e2e/concurrency_rejections_test.go` |
| `identity_generic.fuse` | W08 | `42` | (empty) | `TestIdentityGeneric` in `tests/e2e/spine_test.go` |
| `multiple_instantiations.fuse` | W08 | `42` | (empty) | `TestMultipleInstantiations` in `tests/e2e/spine_test.go` |
| `drop_observable.fuse` | W09 | N/A (test asserts DropIntent metadata) | N/A | `TestDropObservable` in `tests/e2e/borrow_rejections_test.go` |
| `reject_borrow_in_field.fuse` | W09 | N/A (must fail to compile) | N/A | `TestBorrowRejections/reject_borrow_in_field` |
| `reject_return_local_borrow.fuse` | W09 | N/A (must fail to compile) | N/A | `TestBorrowRejections/reject_return_local_borrow` |
| `reject_aliased_mutref.fuse` | W09 | N/A (must fail to compile) | N/A | `TestBorrowRejections/reject_aliased_mutref` |
| `reject_use_after_move.fuse` | W09 | N/A (synthetic HIR assertion) | N/A | `TestBorrowRejections/reject_use_after_move` |
| `reject_escaping_borrow_closure.fuse` | W09 | N/A (synthetic HIR assertion) | N/A | `TestBorrowRejections/reject_escaping_borrow_closure` |
| `match_enum_dispatch.fuse` | W10 | `42` | (empty) | `TestMatchEnumDispatch` in `tests/e2e/match_test.go` |
| `error_propagation_err.fuse` | W11 | `43` | (empty) | `TestErrorPropagation/run-false-propagates-err` |
| `error_propagation_ok.fuse` | W11 | `0` | (empty) | `TestErrorPropagation/run-true-continues-ok` |
| `closure_capture.fuse` | W12 | `42` | (empty) | `TestClosureCaptureRuns` in `tests/e2e/closure_capture_test.go` |

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
