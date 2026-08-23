# E40-F10 verification evidence — 6b566554

Captured at the exact candidate commit `6b566554` (`fix(E40-F10): reject nested retained symlinks`) on 2026-08-22.

## Results

- `bench/scripts/tests/tc082_retention_layout_test.sh`: PASS, including the round-7 nested retained-artifact symlink counterexample.
- `bench/scripts/tests/run-all.sh`: exit 1, 75/76 registered wrapper tests passed. The sole failure is TC-037's repository-wide `make fmt && make lint && make test` hygiene check.
- `make fmt && make lint && make test` in a clean detached worktree at this commit: exit 1. Go contract packages passed; `TestDeepReviewUsesCompactPassAndDetailedFindingsPolicy` fails because the checked-in deep-review skill text does not contain the test's expected phrase. This is outside the E40-F10 production/test diff.
- TC-092 runtime registration check against the terminal run log: exit 1 only because TC-037 did not emit its required pass marker; AC-T2 registration checks pass.

## Artifact hashes

```text
make-fmt-lint-test.log  1310348413979df449e7d3541787ed8a7daf14638859e157a9c59bee16d7d9cb
run-all.log             38365059dd96f219b454fe101221eb5cc130c502f9db0a878e13c49758290e4a
tc092-runtime-registration.log
                         11ac1862badc89938e5542e501e2fd93dbfaaee03abfc946bb46c789974e57a6
```

This bundle preserves the exact negative gate evidence. It does not claim the repository-wide quality gate is green, and it does not turn the unrelated deep-review contract failure into an E40-F10 product finding.
