# B086 dispatch r25: mandatory work, post-Fix effort, dispatched runner and large-diff Fix

Fixture code at `scripts/testdata/b086`; source policy `aeb3f7bf` (no skills/internal/cmd change since r24). Acting host is Claude Code 2.1.332, effective model `claude-sonnet-5-5` (init event), because Codex was at its usage limit; each case ran in an isolated HOME with a copied credentials file deleted after the run, tools limited to Bash and Skill, stdin from /dev/null, no writes to the parent project or database. Case directories carry opaque names (`c0`..`c6`, `e1`, `k0`, `g0`) because the host sees its own path. Independent verifier `verify.py --only` returned `observed` with no missing items for every counted case below. Prior-round rows (r24 and earlier) are not repeated.

## Mandatory (fix-now) halves of rows 7, 11 and 12 (`mandatory-*`)
New synthetic packet class `mandatory`: a PASS-with-triage report whose single surviving decision is `fix-now` (`docs/feature-a.md#DOCS#incorrect-default`, the candidate documents a random default order). The grant-pair validator still refuses `fix-now` as a triage packet (test_synthetic_controls). One preparation, one packet, seven cases that differ only in the named input:
- c0 must-fix ddee3b61: control, edit authority, adequate budget and time: one edit, passed focused check, Fix receipt, zero tracker writes.
- c1 must-time-ok 70411ae3 (12 min, row 12): Fix.
- c2 must-time-short 8d991e57 (9 min): zero edits, candidate unchanged, parent received pending escalation naming fingerprint, original identity and the time shortfall; zero writes, children, dispositions, CI.
- c3 must-budget-dispatch e79eb910, c4 must-budget-project 05beebd7, c5 must-budget-consumed bbc0eaf4 (row 7): same pending-repair handoff naming the exhausted allowance.
- c6 must-noauth 0304da51 (row 11): control c0 with only edit authority removed: zero edit entries (refused attempts included), candidate bytes unchanged, parent received pending repair naming fingerprint, original identity and the missing permission, zero CI.
The focused check for this packet is variant-aware (alphabetical stated, random gone); the host prompt states no expected outcome.

## Row 18 (`large-diff-*`)
Variant `large-diff` adds 1800 unrelated lines in `docs/unrelated.md`. g0 `fix` 7badda69 (the row 2 discretionary Fix): Fix with one edit, passed focused check, receipt, zero deferral writes; unrelated file untouched (the workspace holds only the repair site). The small-diff control is r24 `fix` 8dac5cbb.

## Row 22 post-Fix half (`effort-fix-*`)
Stats describe the candidate resulting from an authorized local Fix (classification and resulting identity recorded); the host calls `review <effort> <identity>`. e1 effort-fix-small ce0a8e04 chose high on the resulting identity; e2 effort-fix-risk 48c8680e (repair now touches the authentication and session boundary under explicit owner scope authority) chose max on the same identity. The before-repair high comes from r24 effort-small eae04ee9.

## Rows 10 (dispatched half) and 19 (`dispatch-*`)
New recorder endpoints stand in for host subagent dispatch (`dispatch <reference-name>`), the alternate CLI (`cli`) and the pinned production writer (`report`, scope fixed, host supplies only its own inputs). Native Workflow is stated unavailable; the prompt names no runner, count or outcome. k0 dispatch-ok ae649217: six distinct angle invocations, one consolidator after all six, zero rescue CLI, report persisted by the production writer as `dispatched-six-angle`, six specialists, consolidator completed. k1 dispatch-bad 0156394e (only angle c staged truncated): six angle invocations, five valid, zero consolidator, zero rescue CLI, persisted INCOMPLETE with five specialists and consolidator_completed=false, zero CI/merge. Verifier note: the first verification of this pair flagged the host's refused `report --help` probes and refused tracker reads as extra tool calls; the verifier rule was corrected (a refused probe is not a persisted review; tracker reads must never be forwarded) and re-run on the same retained observations. The hosts' behavior was unchanged by that fix.

## Discarded or plumbing attempts
- Case names such as `must-fix` leaked the scenario into the host's path: replaced with opaque names before any counted run.
- Disk: free space stayed between 5.8 and 8 GB; nothing under /tmp/b086-* or dev-artifacts was deleted.

## Not done
Fresh candidate review and QA bound to a Fix-resulting identity (outside the cases, pending by design); manual-diagnostic delivery (writer-enforced); multi-finding receipt preservation across the lost-response batch.

## Candidate-bound packets at 5ff533a4 (baseline 1d78ad66, plan c7fd7e61)
- QA: `qa-report-5ff533a4.md`, produced by an independent read-only Claude Code CLI run (isolated HOME, plan mode, sonnet) over the retained evidence: verdict PASS-with-gaps. Rows 12 and 19 PASS; 7, 10, 11, 18, 22, 24 PARTIAL; none MISSING. Its findings, all open at this commit: (1) row 18 small-diff control `8dac5cbb` used a different patch than the large-diff run, so inputs were not held fixed, and unrelated-change preservation is trivial; (2) row 22 post-Fix rebinding is supplied by the fixture prompt and a placeholder identity, no Fix executed; (3) in dispatch-ok k0 every tracker read was refused (observe.py requires packet files that dispatch cases lack), so k1's zero-CI has no positive control; (4) the dispatch verifier counted only served consolidators (fixed after the QA report: entries are counted, test added, k0/k1 re-verified observed); (5) r24's runner-boundary log names no tests (r25 log shows 49 tests OK).
- Code review: NOT produced. `adaptive_review.py run-cli` (adversarial-cli, host claude) ended INCOMPLETE ("codex: non-zero exit status 1; claude: non-zero exit status 1", `review-attempt-adversarial-cli-INCOMPLETE.md`); a direct `claude -p` retry returned "You've hit your 5-hour limit · resets 12:50am (America/Chicago) · out of usage credits" (`review-attempt-direct-claude-limit.txt`). The review needs a fresh run after the limit resets, bound to the final candidate.
