# E40-F10 Current-HEAD Quality Gate Evidence

**Candidate commit:** `6da3af9d2f8303b7f386e77c82872c4ba1487104`
**Branch:** `E40-F10`
**Captured:** 2026-08-22
**Evidence status:** terminal and reproducible from the committed logs in this directory

The evidence bundle is committed by `ec56f9b2`; that commit adds only this
evidence directory and does not alter executable implementation or tests.

## Results

| Check | Command | Exit | Result |
|---|---|---:|---|
| Go quality gate | `make fmt && make lint && make test` | 0 | PASS |
| Registered regression | `bench/scripts/tests/run-all.sh` | 0 | PASS; 76 wrapper cases, 0 failed wrappers |
| Registration runtime proof | `TC092_RUN_LOG=run-all.log bench/scripts/tests/tc092_full-regression-registration_test.sh` | 0 | PASS; all 62 pre-F10 markers and wrapper PASS lines present |

The registered run includes TC-003 through TC-092, including TC-079 through
TC-092. The separate TC-092 pass proves the runtime-marker assertion that the
default invocation intentionally reports as not exercised.

## Log integrity

| File | SHA-256 |
|---|---|
| `make-fmt-lint-test.log` | `5af5ecabe4d76d99855861bc41ba58a0a3fa28c10038e93432c1fb56b5730bac` |
| `run-all.log` | `22becbfbe7f850ba4ebf61f9e3e45b0ba95e7774193cbf07fb0a364b57714e0f` |
| `tc092-runtime-registration.log` | `58e44223c3c9f684e82e843cd46074971bd96e9aeed9d27c5f7ba649f2916715` |

These logs were recovered from the current-session `/tmp` artifacts, copied
without content changes, and hashed after copy. The evidence is limited to
the quality-gate and registration claims above; it does not independently
clear prior UAT findings about retained lifecycle behavior or the upstream
time/cost contract.
