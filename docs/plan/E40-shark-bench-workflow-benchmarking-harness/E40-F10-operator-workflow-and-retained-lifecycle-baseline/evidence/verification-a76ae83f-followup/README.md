# E40-F10 follow-up verification evidence

**Reviewed implementation commit:** `7cae5a7c` (follow-up branch HEAD)
**Captured:** 2026-08-23
**Environment:** primary checkout with unrelated dirty files preserved; all
feature-owned changes were committed at the reviewed implementation commit.

## Terminal results

| Command | Exit | Result |
|---|---:|---|
| `make fmt && make lint && make test` | 0 | PASS |
| `bench/scripts/tests/run-all.sh` | 0 | PASS; 77 wrapper PASS lines, no failures |
| `TC092_RUN_LOG=run-all.log bench/scripts/tests/tc092_full-regression-registration_test.sh` | 0 | PASS; AC-T2 and AC-T3 pass |
| `tc091_static_safety_language_neutrality_test.sh` | 0 | PASS |
| `tc093_digest_authority_test.sh` | 0 | PASS |

## Follow-up fixes covered

- root-scoped single-writer locking and collision-resistant batch IDs;
- streaming file hashing and source copying;
- fail-closed non-finite numeric handling and manifest field typing;
- final-component ledger `O_NOFOLLOW` protection and pre-canonicalization
  retention symlink rejection;
- per-run TC-091 temporary evidence and executable registration assertions;
- bounded deep-review CLI output capture.
- bounded reviewer process-group termination and ledger reference validation.

The implementation commit contains no changes to the unrelated editor or
continuation-prompt files.
