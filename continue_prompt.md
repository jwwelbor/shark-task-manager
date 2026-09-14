---
timestamp: 2026-09-13T22:00:00Z
branch: fix/e40-benchmark-capture (pushed as fix/e40-benchmark-capture-v2)
worktree: /home/jwwel/projects/shark-task-manager/.worktrees/e40-benchmark-capture
pr: superseded #216 -> new PR from fix/e40-benchmark-capture-v2
bug: B064 (shark get B064 --json; status=completed, epic E40)
---

# B064 rebase reconciliation — closed out

## State: done, independently verified, pushed as a new PR

The 17-commit rebase of `fix/e40-benchmark-capture` onto current `main`
(past #213/#214) plus 5 follow-on commits (67c44dfa, 9884ff10, 04186163,
d6aa0b00, 5a590683) is complete. Both open items from the prior checkpoint
were resolved and committed:

- **Infinite-requeue bug** (`9884ff10`): added signature-based guards to
  `run-lifecycle.sh`'s dispatch loop so a repeat `(entity_key, status)` pair
  or an unchanged fork candidate set no longer re-queues forever.
- **tc080 contract gap** (`04186163`): dropped tc080's private
  `expected_entity_graph` boilerplate block and declared it in AC-F11-27's
  allowlist with an honest rationale.

Two more commits followed: `d6aa0b00` repaired 8 pre-existing (unrelated)
bench-test failures, and `5a590683` restored a tiered exit-code contract
for `run-lifecycle.sh` that an earlier commit in this same branch
(`5f80d23c`) had accidentally flattened to always-0.

**Independently re-verified on 2026-09-13** (not just trusted from commit
messages): `make fmt` (clean), `make lint` (0 issues), `make test` incl.
`go test ./tests/contracts/...` (0 failures), and the full bench battery
`bench/scripts/tests/run-all.sh` (101/101 PASS, exit 0). Also read the diffs
of `5a590683` and `d6aa0b00` directly: both are legitimate, paired
script+test fixes with specific per-case technical justification (e.g. the
TC-015 SIGTERM fix corrected a real latent test bug — signaling the wrong
PID because the test assumed `fork` when `run-lifecycle.sh` actually uses
`exec`) — not test-bending to hide a regression.

## What happened to PR #216

#216 was stale (built from the pre-rebase tip `8cbd82b1`), had zero CI runs
and zero formal reviews, and its only engagement was a self-contradicting
close/reopen comment thread from 2026-09-11 that promised re-verification
and never delivered it. Per Architect review (fable-architect, 2026-09-13):
force-pushing over it would destroy that thin-but-real audit trail for no
benefit. Instead: the rebased branch was pushed under a new ref
(`fix/e40-benchmark-capture-v2`), a fresh PR was opened from it describing
what it adds beyond main's independently-built #213/#214, and #216 was
closed with a comment pointing to the new PR. `b064-pre-rebase` tag and the
old `fix/e40-benchmark-capture` ref on origin are left untouched.

## Remaining before merge

- Code review must be re-run on the new PR (not skipped) — ~4k lines
  changed post-rebase, none reviewed since.
- Standard `/finish-feature` flow from PR creation onward: resolve PR
  comments, verify CI green (CI only runs `go test ./...` + lint — it does
  NOT cover the bench battery, so that stays a local/manual gate before any
  merge), squash merge, branch/worktree cleanup.
- Do not delete `b064-pre-rebase` tag or old `fix/e40-benchmark-capture`
  ref until the new PR merges.
