# T-E40-F10-015 verification evidence

Reviewed HEAD: `e9f3190e5072663adcdb1a23491a42a6352ffc86`

This bundle was captured from a real local clone with both bench submodules
initialized and a task-scoped private `TMPDIR`. The checkout itself was clean.

## Terminal results

| Command | Exit | Result |
|---|---:|---|
| `make fmt` | 0 | PASS |
| `make lint` | 0 | PASS |
| `make test` | 0 | PASS |
| `make shark` | 0 | PASS |
| `bench/scripts/tests/run-all.sh` | 1 | FAIL: TC-061, TC-062, TC-066, TC-079, and TC-080 |
| `TC092_RUN_LOG=run-all.log bench/scripts/tests/tc092_full-regression-registration_test.sh` | 1 | AC-T2 passes; AC-T3 fails because the five failed tests have no pass markers |

The failed regression tests are runtime/environment evidence, not registration
failures. TC-092 independently confirms all 62 pre-F10 registrations remain in
relative order and TC-079..TC-092 are registered.

## Evidence-bundle sweep

The following final-gate bundles were inspected for reviewed-SHA binding and
terminal exit capture:

- `quality-gate-6da3af9d`: stale reviewed SHA; no exit files.
- `verification-6b566554`: reviewed SHA `6b566554`; no exit files; negative
  quality-gate result recorded in its README.
- `verification-e9f3190e`: reviewed SHA `e9f3190e`; terminal exit files present;
  `run-all` and TC-092 failed.
- `verification-e9f3190e-mainref-private-tmp`: reviewed SHA `e9f3190e`; terminal
  exit files present; prior green result is not treated as current because its
  capture date is later than this verification session.

This report preserves the current exact-HEAD result rather than claiming AC-T1
through AC-T3 are complete.
