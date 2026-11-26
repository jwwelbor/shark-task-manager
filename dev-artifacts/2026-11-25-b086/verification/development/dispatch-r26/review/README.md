# B086 r26 candidate-bound code review

- Candidate: `568a5d00457ba717fa25e83cdbe2d3b4256f7ce9` (tree `29d43c71e39906f74ca5538407cb6d41e40970c5`), base `1d78ad660f2c55dd188465785b61723a3b7e6a6c`, plan sha256 `c7fd7e61...e33aa`, diff sha256 `4bd0eb93fe4901ea86ef6ed6787ad83518dbf5e0ac184573385173d9fb1c5059` (diff not stored; 56 MB compressed).
- Packet commit: the commit that follows `568a5d00` on `fix/B086-unattended-triage` and adds only this directory. A packet cannot be inside the tree it is bound to.
- Packet: `review.md` sha256 `33b988c45b45074cd4e6cd49fb5262a1b8c829510759a199bc0d79a8e594704c`; decisions `f64a7ad7...7a77`; scope `9d3c5c78...271d`; all listed in `SHA256SUMS.txt`.
- Runner: `adaptive_review.py run-cli --host claude` (adversarial-cli, `specialists_completed=0`). Codex was the preferred CLI and failed non-zero (usage limit), so the recorded fallback reason is `codex: non-zero exit status 1; using claude CLI`. Effective model is unknown (the tool records none); this is a single-reviewer CLI review, not a six-angle review. Requested model: unknown.
- Verdict: **FAIL**, one blocker, seven non-blockers, two nits.

## Attempts, all retained
1. `../review-first-8dba2d80/`: FAIL, 3 blockers (prompt content, refused attempts escaping zero-write checks, missing overlap control). Addressed in later commits.
2. `../review-second-ac80a46b/`: FAIL, 2 blockers (scripted row 22 sequence, `--only` rejecting retained staged cases). Both addressed in `5b5e6b54`.
3. `../review-incomplete-attempt-568a5d00/`: the first run on the final candidate returned INCOMPLETE (the CLI output failed report validation, "malformed report"); retried on the same candidate with an output-format reminder, giving this packet.

## Dispositions of this packet's findings (none was fixed after the review: any change would be a new candidate)
- F-1 blocker, row 22 never observes `high` before the repair: OPEN. The starting facts for the risk case say no boundary while `boundaries.json` lists the edited file; the counted hosts skipped the optional starting review, so the `high`-then-`max` transition within one run is not evidenced (r24 `eae04ee9` covers the small starting case only). Needs a design decision: derive the starting classification from the same boundary listing, or require the host to review the starting candidate through the skill rather than the prompt.
- F-2, F-3, F-4, F-5, F-6, F-8: open non-blockers on verifier strictness (word checks matching keys, destructive repair accepted by the effort verifier, pair inputs not digest-bound, `--only` skipping the seed digest, staged size facts).
- F-7: open non-blocker; triage Step 0 and Step 5 are not scoped around the unattended branch. Changing the skill would change the pinned policy source and invalidate the host evidence; left for the parent.
- F-9, F-10: nits (README fixture-commit wording corrected in the per-section statements; weak Go `parent` assertion).
- Earlier, from review 1 and 2, still open: host credentials copied into the case HOME stay readable by the Bash-enabled host during the run (deleted afterwards); the receipt-gate prompt dictates the pending-state action (r24 evidence).
