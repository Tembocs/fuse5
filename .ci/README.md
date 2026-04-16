# .ci/

This directory holds helper scripts and reusable CI components.

The main GitHub Actions workflow lives at `.github/workflows/ci.yml` —
that is where GitHub looks for workflow definitions — and invokes the
governance tools under `tools/` directly. Per-job helpers (reproducibility
driver scripts, cross-compilation wrappers, perf-suite runners) land here
as later waves need them.

At Wave 00 this directory is a placeholder. Real helper scripts are
added by:

- **W17** (performance baseline driver)
- **W23** (package-manager fetcher smoke test)
- **W25** (reproducibility gate)
- **W27** (performance gate)
