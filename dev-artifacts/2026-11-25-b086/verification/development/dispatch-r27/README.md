# B086 dispatch r27: row 22 observes high before and max after in one run

Fixes round 6 blocker F-1 (review `dispatch-r26/review/review.md`). Fixture commit `148b3022`; acting host Claude Code (model reported by the init event in each trace), isolated HOME, tools Bash and Skill, no writes to the parent project or database.

Decision: the plan row says an ordinary candidate whose repaired change now touches a high-risk boundary. The plan therefore governs and the starting facts stay "no boundary"; the boundary listing was changed to agree with them. The listed boundary file is now `docs/auth.md`, a base file the starting diff never touched. The finding (same finding text for every case, so pairs differ only in the named inputs) says the wrong default is repeated there; repairing it is what crosses the boundary, under the grant's owner scope authority in the risk case only.
Second change: the starting review is no longer optional for the risk case. The recorder's `review` on the unrepaired candidate is what returns the finding (the prompt no longer points at `finding.json`), and the verifier requires two reviews in one run when the starting and resulting efforts differ (`high` then `max`), the first on the starting identity before any edit. Repair edits may now be several files; every attempted edit must apply.

Observed runs (verifier `observed`, no missing items; `row22-verification.json`):
- e1 small, run `5c3a331c`: review high on start `61268dca`, repair of both files, review high on `cf025c73`.
- e2 large (1800 lines), run `44a7bb3d`: review max, repair, review max.
- e3 risk, run `b96329c6`: review high on start `61268dca` (24 lines, no boundary), repair touching `docs/auth.md` (boundary), recorder verify reports the authentication boundary, review max on `cf025c73`. This is the row 22 high-before then max-after transition in a single run.
Pairs: `row22-pair-small-risk.json` (differ only in boundaries.json, grant.json), `row22-pair-small-large.json` (only candidate-stats.json); both observed. Full ordered tool traces in `traces/`; raw observations in the tarball. Supersedes the r26 row 22 runs (retained, not deleted).

Still open and out of scope (round 6 non-blockers): verifier word-check strictness (F-2, F-3), destructive-repair acceptance (F-4), pair input digests (F-5), `--only` seed digest (F-6), triage unattended rule conflicts (F-7), staged size facts (F-8), README fixture-commit wording (F-9), weak Go assertion (F-10); round 5 QA findings 3 and 5.
Unit suite: 86 tests OK (`python3 -m unittest discover -s scripts/testdata/b086`).
