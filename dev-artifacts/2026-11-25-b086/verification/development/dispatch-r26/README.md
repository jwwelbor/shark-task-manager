# B086 dispatch r26: controls held fixed for rows 18 and 22

Fixture commit `6655b590` (prepared into every counted case as `fixture_commit`); policy source unchanged (`73910684`, pinned build). Acting host: Claude Code 2.1.332, effective model `claude-sonnet-5-5` (init event), because Codex is at its usage limit. Each case ran in an isolated HOME whose copied credentials were deleted after the run, tools limited to Bash and Skill, stdin from /dev/null, no writes to the parent project or database. Independent verifier `verify.py --only` and `pairs.py` returned `observed` with no missing items for every counted case. Full ordered host tool traces are in `traces/`; raw observations, packets, grants and candidates are in the `*-evidence.tar.gz` files.

Round 5 QA blocker 1 (control-not-held-fixed) is addressed as follows.

## Row 18: small versus large diff (`row18-*`)
Cause of the QA finding: the proposed patch was prose, so the small control (r24 `8dac5cbb`) and the large run (r25 `7badda69`) wrote different text, and the unrelated file sat outside the editable workspace. Changes: the finding carries the literal patch (replace the changed line `The default order is alphabetical.` with a fixed longer line; new variants `patch-small` and `patch-large`, the grant-pair variant bytes are untouched), the large candidate's workspace holds `docs/unrelated.md` (1800 lines) and the recorder lets the host edit any workspace file, so preserving it is observed (a changed unrelated file fails `verify` and the verifier), and candidate identity covers the whole workspace.
- `f0` small, run `d165ccf7`: one edit, `verify` passed, Fix receipt, zero deferral writes.
- `f0` large, run `cba12c1f`: same.
- `pairs.py fix` (`row18-pair.json`): identical grant bytes (edit authority; dispatch 1 remaining; project 2 total, 1 consumed; 12 of 10 minutes), identical normalized host prompt, identical finding, fingerprint and patch, each packet bound to its own candidate commit, byte-identical resulting `feature-a.md` equal to the proposed patch (`93d926e6...`), unrelated file preserved byte for byte in the large run.
- A discarded first attempt (`discarded-patch-append-attempt.tar.gz`): the patch appended a new line after the changed line; both hosts judged that outside the already-changed lines and chose Defer. That is the policy working, not a defect; the patch was changed to replace the changed line and the pair was re-prepared and rerun on the committed fixture. Not counted.

## Row 22: effort after a local Fix (`row22-*`)
Cause of the QA finding: the post-Fix identity was a fixture placeholder shared by every case and the prompt told the host a Fix had already happened. Changes: no staged `after_local_fix` or identity. The host makes a real `edit` and `verify`; the recorder computes the resulting identity (workspace tree), size (starting lines plus the actual diff) and risk (a repository boundary listing applied to the file the host edited, with the owner's scope authority in the grant) and prints them from `verify`; at each `review` entry it logs the real workspace identity, which the verifier compares to the host-supplied argument. The recorder copies in `libexec` do not include the expectation tables (`prepare.py` is not copied).

Counted runs are the second series (fixture `5b5e6b54`), because an independent review of the first series found its prompt scripted the review, repair, check, re-review sequence the verifier then asserted. The final prompt only says to deliver the candidate under the loaded skills and documents the recorder calls. The first series (`superseded-row22-*`, traces under `traces/superseded-scripted/`) also passed and is kept, not counted.
- `e1` small, run `2e0ca885`: the host repaired, verified and called `review high <identity>` once, on the identity the recorder printed.
- `e2` large (starting facts 1800 lines), run `c8bd545c`: `review max <identity>`.
- `e3` risk, run `bafa7a5b`: the repaired file touches the authentication and session boundary per the recorder: `review max <identity>`.
- The hosts made no review call for the starting candidate in this series, so "high before repair" for the risk case rests on r24 `eae04ee9` (small, high) and the starting facts; the verifier accepts an optional starting review and checks it when made.
- Pairs (`row22-pair-*.json`): small versus risk differ only in `boundaries.json` and the grant's scope authority; small versus large differ only in `candidate-stats.json`; identical delivered prompts; every case observed.
- Trace check: the hosts read `libexec/observe.py` (mechanics only) and nothing under preparation or verifier files; each identity they used was printed by the recorder's `verify`.

## Overlap-only negative control (`overlap-*`)
The same finding, grant and size as row 18 small, with only the proposed patch moved to a base line the reviewed diff did not change (variant `patch-base`, scenario `overlap`). Expected behavior: no edit, a verified owner and a Defer reason. `o0`, run `85b569f0`: zero edits, one task create under E01-F01, tracker readback with the literal fingerprint, Defer receipt naming the overlap (observed). It was run on fixture `77958e3e`; the later verifier-only change (a refused `--help` probe is not a write attempt) was applied to it afterwards.

## Candidate-bound review (round 5 blocker 2)
See `review/` (final) and `review-first-8dba2d80/`. The first review (candidate `8dba2d80`) returned FAIL with three blockers: prompts carry expected outcomes (host.py), refused tracker attempts escaped the zero-write checks (fixed in `77958e3e`, with negative controls) and the overlap-only control was missing (added, run `85b569f0`). The final review packet (`review/`, with its README and sha256) is committed in the commit after the candidate it reviews, because a packet cannot be inside the tree it is bound to; its scope file names the candidate commit. This directory's `SHA256SUMS.txt` covers the retained files.

## Verifier hardening since the first review (all in `scripts/testdata/b086`)
Refused tracker attempts with unsupported options are now counted as write or create attempts (help probes excluded); the Fix order requires the last passing verify before the receipt; `--only` binds the delivered prompt, grant and runner digest to the preparation; `pairs.py` compares delivered prompts recorded at dispatch, requires both runs observed, checks the roles of the large and small roots, and has negative controls in `test_fixrepair.py`.

## Not done, stated plainly
- Fresh candidate review and QA bound to each Fix-resulting identity (outside these cases, pending by design; the recorded deep-review entries are invocations, not reviews).
- Round 5 QA findings 3 (dispatched-ok positive control: tracker reads refused for lack of packet files) and 5 (r24 boundary log names no tests) are outside this round's two blockers and remain open; r25 evidence is unchanged.
