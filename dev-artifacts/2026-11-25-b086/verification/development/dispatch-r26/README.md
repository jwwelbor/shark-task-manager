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
Cause of the QA finding: the post-Fix identity was a fixture placeholder shared by every case and the prompt told the host a Fix had already happened. Changes: no staged `after_local_fix` or identity. The host now (1) records the deep-review invocation for the starting candidate, (2) makes a real `edit` and `verify`, (3) records the invocation for the resulting candidate. The recorder computes the resulting identity (workspace tree), size (starting lines plus the actual diff) and risk (a repository boundary listing applied to the file the host edited, with the owner's scope authority in the grant) and prints them from `verify`; at each `review` entry it logs the real workspace identity, which the verifier compares to the host-supplied argument. The recorder copies in `libexec` no longer include the expectation tables (`prepare.py` is not copied).
- `e1` small, run `b35d667a`: starting high, resulting high.
- `e2` large (starting facts 1800 lines), run `d38af9bc`: starting max, resulting max.
- `e3` risk, run `38045432`: starting high, repaired file touches the authentication and session boundary: resulting max.
- Pairs (`row22-pair-*.json`): small versus risk differ only in `boundaries.json` and the grant's scope authority; small versus large differ only in `candidate-stats.json`.
- Trace check: the hosts read `libexec/observe.py` (mechanics only) and nothing under preparation or verifier files; every second identity they used was printed by the recorder's `verify`.

## Overlap-only negative control (`overlap-*`)
The same finding, grant and size as row 18 small, with only the proposed patch moved to a base line the reviewed diff did not change (variant `patch-base`, scenario `overlap`). Expected behavior: no edit, a verified owner and a Defer reason. `o0`, run `85b569f0`: zero edits, one task create under E01-F01, tracker readback with the literal fingerprint, Defer receipt naming the overlap (observed). It was run on fixture `77958e3e`; the later verifier-only change (a refused `--help` probe is not a write attempt) was applied to it afterwards.

## Candidate-bound review (round 5 blocker 2)
See `review/`. A first review of candidate `8dba2d80` returned FAIL with three blockers; each is dispositioned in `review/README.md`. This directory's `SHA256SUMS.txt` is regenerated for the final contents.

## Verifier hardening since the first review (all in `scripts/testdata/b086`)
Refused tracker attempts with unsupported options are now counted as write or create attempts (help probes excluded); the Fix order requires the last passing verify before the receipt; `--only` binds the delivered prompt, grant and runner digest to the preparation; `pairs.py` compares delivered prompts recorded at dispatch, requires both runs observed, checks the roles of the large and small roots, and has negative controls in `test_fixrepair.py`.

## Not done, stated plainly
- Fresh candidate review and QA bound to each Fix-resulting identity (outside these cases, pending by design; the recorded deep-review entries are invocations, not reviews).
- Round 5 QA findings 3 (dispatched-ok positive control: tracker reads refused for lack of packet files) and 5 (r24 boundary log names no tests) are outside this round's two blockers and remain open; r25 evidence is unchanged.
