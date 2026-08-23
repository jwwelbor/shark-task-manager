# E40-F10 T-E40-F10-015 verification evidence

**Reviewed HEAD:** `e9f3190e5072663adcdb1a23491a42a6352ffc86`
**Captured:** 2026-08-23
**Environment:** clean non-worktree clone at the reviewed HEAD, initialized
bench submodules, and locally built `bin/shark`.

## Terminal results

| Command | Exit | Result |
|---|---:|---|
| `make fmt && make lint && make test` | 0 | PASS; terminal log and exit file retained |
| `bench/scripts/tests/run-all.sh` | 1 | FAIL; 70 wrapper PASS lines and 6 wrapper FAIL lines |
| `TC092_RUN_LOG=run-all.log bench/scripts/tests/tc092_full-regression-registration_test.sh` | 1 | AC-T2 passes; AC-T3 fails for markers from failed suites |

The quality gate was run in a clean clone so unrelated dirty files in the
shared checkout could not affect the result. The registered bench suite was
run only after initializing `bench/fixture-py` and `bench/fixture-repo` and
building the clone-local Shark binary.

## Regression failures

The terminal `run-all.sh` log records failures in:

- TC-014 (`run-one.sh` provisioning sub-case)
- TC-061 (lifecycle runner loop)
- TC-062 (lifecycle runner limits)
- TC-066 (question handoff)
- TC-079 (preview Shark-call count)
- TC-080 (UAT-R2 retention classification)

The exact stdout/stderr and exit statuses are retained in the sibling files.
TC-092 confirms that the pre-F10 registration list still contains all 62
tests in relative order and that TC-079 through TC-092 are registered, but it
correctly rejects AC-T3 because failed suites did not emit their pass markers
or wrapper PASS lines.

## Evidence files

| File | SHA-256 |
|---|---|
| `make-fmt-lint-test.log` | `2dea7193cb8960fd68e9930c1a983e160afd2eb31bc847ef026151541ccdf80a` |
| `make-fmt-lint-test.exit` | `9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa` |
| `run-all.log` | `726d87a029dfe5af056e38dfafc8944f0c241ac1dbfee43c1fcf6e8a260e8fbf` |
| `run-all.exit` | `4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865` |
| `tc092-runtime-registration.log` | `72ea4797a1f62ea5e7af38ad944ec8e91ec47576869141b75c96a98c9dee1088` |
| `tc092-runtime-registration.exit` | `4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865` |

## Disposition

The implementation already present for this task retains the required suite
registration and README operator documentation. The reviewed HEAD does not
meet AC-T1/AC-T3 because the registered regression is red. This artifact is
durable terminal evidence for the parent loop; the task is not ready to be
advanced to completed until the failing regression classes are repaired and
the three commands above are rerun successfully at the resulting HEAD.
