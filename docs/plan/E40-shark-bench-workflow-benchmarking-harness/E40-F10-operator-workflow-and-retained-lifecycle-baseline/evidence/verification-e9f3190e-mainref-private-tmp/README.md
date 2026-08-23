# E40-F10 T-E40-F10-015 verification evidence

**Reviewed HEAD:** `e9f3190e5072663adcdb1a23491a42a6352ffc86`
**Base ref:** `main` at `baf9b021160985ecbd1945820ce1e0431f2bb521`
**Captured:** 2026-08-23
**Environment:** clean clone with bench submodules initialized, clone-local `bin/shark`,
the required `main` ref, and an explicitly pinned private `TMPDIR` to isolate concurrent
benchmark scratch state.

## Terminal results

| Command | Exit | Result |
|---|---:|---|
| `make fmt && make lint && make test` | 0 | PASS |
| `bench/scripts/tests/run-all.sh` | 0 | PASS; 76 wrapper PASS lines, no failures |
| `TC092_RUN_LOG=run-all.log bench/scripts/tests/tc092_full-regression-registration_test.sh` | 0 | PASS; AC-T2 and AC-T3 pass |

The full regression was run after the isolated TC-014, TC-061, TC-062, TC-066, TC-079,
and TC-080 checks passed with the same base ref and private temporary root. The private
temporary root prevents unrelated concurrent benchmark processes from sharing scratch
databases and provisioning paths.

## Evidence files

| File | SHA-256 |
|---|---|
| `make-fmt-lint-test.log` | `50ff09ba6e95fd75a44691e432eadc49a010c0c124ffe1bf78a90f738fb3bf23` |
| `make-fmt-lint-test.exit` | `9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa` |
| `run-all.log` | `1ae2ce562eb67305dd2918a0d73cef2e0e2482acdb885e84034574d46bb7cbaf` |
| `run-all.exit` | `9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa` |
| `tc092-runtime-registration.log` | `6f1c8dc2b41736289f35d659c1c3fa7c45fd8517a10d9cf0d8388f5026325477` |
| `tc092-runtime-registration.exit` | `9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa` |

