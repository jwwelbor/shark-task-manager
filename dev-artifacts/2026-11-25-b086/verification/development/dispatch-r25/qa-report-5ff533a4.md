QA VERDICT: PASS-with-gaps

Only rows 12 and 19 are fully host-evidenced; rows 7, 10, 11, 18, 22 and 24 are PARTIAL. None is MISSING. I found no fabricated run results, but some README statements overstate what the records support.

## Scope

- **Candidate:** HEAD is `5ff533a4…`. The plan's SHA-256 matches `c7fd7e61…`.
- **Source drift:** since policy commit `aeb3f7bf`, the only change outside `scripts/testdata` and `dev-artifacts` is `B086.repair-plan-evidence.md`. The host runs predate HEAD, but the policy sources they exercised are unchanged.
- **Integrity:** every file listed in both `SHA256SUMS.txt` files checks OK.
- **Read:** both READMEs, raw records from the r24 and r25 archives, and parts of `verify.py`, `observe.py`, `dispatch.py` and `prepare.py`.
- **Side effects:** I edited no files (including the plan-mode plan file), ran no `shark` and made no network calls. No `ExitPlanMode` tool is available, so this report is the final output.

## Per-row results

| Row | Status | Runs and what the raw records show |
|---|---|---|
| 7 | **PARTIAL** | **Mandatory:** control `ddee3b61` has 1 edit, a passed check, 1 disposition and 0 creates. `e79eb910` (dispatch 0), `05beebd7` (project 1/1) and `bbc0eaf4` (consumed 2/2) each have 0 edits, 0 verifies, 0 creates, 0 dispositions and 1 parent message naming the exhausted allowance. **Discretionary:** control `8dac5cbb` has 1 edit. `131869b5`, `94d17c07` and `c3bcb449` each have 0 edits, 1 create, a `get T-E01-F01-001 --json` readback and 1 Defer with a reason. **Gaps:** I found no host case for missing budget evidence leading to pending parent action. The dispatch limit is staged as "remaining 1→0", not the plan's "total two, one consumed". |
| 10 | **PARTIAL** | `ae649217` shows 6 distinct angle dispatches, then 1 consolidator, 0 CLI entries and 1 report (`dispatched-six-angle`, 6, true). **Not host-observed:** native effort on or omitted from the seven calls, native completion, both CLI preference and fallback directions, model attribution per accepted packet, and verified Defer plus escalation for each packet. The README relies on unit tests for the native and CLI halves. |
| 11 | **PARTIAL** | `0304da51` differs from the `ddee3b61` grant only in `edit_authority=false`. It has 0 edit entries, 0 verifies, 0 dispositions, 0 CI and 1 parent message naming the fingerprint, identity `e4e688f8…` and missing edit authority. The final file hashes to the same identity, is mode 0444 and keeps the prepared mtime. The control moves it to `50cca8d3…`. **Gap:** the control's fresh review and QA are only reported "pending", and the row requires them on the new identity. |
| 12 | **PASS** | **Mandatory:** `70411ae3` (12 min) has 1 edit, a passed check and a Fix. `8d991e57` (9 min) has 0 edits, 0 creates and a parent message giving 9 < 10. **Discretionary:** `94067e12` has 1 edit, a passed check and a Fix. `79d91bcd` has 0 edits, 1 create, a readback and a Defer with a reason. All four use project 3 total, 1 consumed. |
| 18 | **PARTIAL** | `7badda69` (3 files, 1806 insertions) has 1 edit, a passed check, a Fix and 0 creates. Gaps are in finding 1. |
| 19 | **PASS** (see findings 3–4) | `0156394e` has 6 distinct angle entries. Only angle c served invalid JSON, so 5 were valid. It has 0 consolidator, 0 `cli`, 0 `ci` and 0 merge entries. There are 3 `report` entries (2 refused `--help`, 1 written): `incomplete`, 5, false, INCOMPLETE. The prompts match the control's, and only the angle-c stub differs. |
| 22 | **PARTIAL** | Effort arguments: `eae04ee9` `high`; `97225e5b` `max`; `aaed1af2` `max`; `ce0a8e04` `high a3c1f0e5…`; `48c8680e` `max a3c1f0e5…`. Gaps are in finding 2. Propagation to the six reviewers and consolidator has no host evidence. |
| 24 | **PARTIAL** | `d03ed43b` has a readback, 1 `ci` entry and 0 forwarded writes. `bed67f2d` has 0 `ci`, 0 writes and 1 parent message keeping the `adversarial-cli` label. The packet files are byte-identical. **Gap:** manual-diagnostic delivery under the permissive policy was not run. The writer does force `manual-diagnostic` to INCOMPLETE (`adaptive_review.py:779-781`). That prevents a PASS packet, but does not show the host stopping when handed such evidence. |

## README claims re-checked

| Claim | Result |
|---|---|
| No-authority case has zero edit entries and unchanged bytes | Confirmed (`0304da51`). `observe.py` logs the entry before authorizing, so refused attempts would count. |
| Invalid-angle case has zero consolidator and zero rescue CLI | Confirmed (`0156394e`). I also found no direct `claude` or `codex` launch pattern in its Bash commands. |
| Post-Fix effort and identity | Arguments confirmed. Their evidential value is limited (finding 2). |
| Exhausted budget cases have zero edits | Confirmed for all six. |
| Large diff leaves the unrelated file untouched | True only trivially (finding 1). |
| Policy pair: "receipt identical", only `review-policy.json` differs | Minor mismatch: `receipt.json` also differs in `tracker_project`, a per-case path. |

## Findings

1. **Row 18 control does not hold the inputs fixed (important).**
   - The README names r24 `8dac5cbb` as the small-diff control. Its patch is "alphabetical; for example, an entry named apple … banana", giving `6a372dc5…`. The large-diff run wrote "alphabetical. For example … pear and apple …", giving `461f70c4…`. The plan requires the same local repair.
   - r24 `94067e12` produced the byte-identical `461f70c4…`, but its project total is 3 rather than 2, and the README does not pair it.
   - "Unrelated changes preserved" is vacuous. The editable workspace holds only `docs/feature-a.md`, and the verifier checks only that file.
   - The overlap-only negative control and the gates on each resulting identity are absent.

2. **Row 22 post-Fix rebinding is largely supplied by the fixture (important).**
   - The prompt tells the host an authorized Fix happened and to call `review` once "with the effort word and the resulting candidate identity from the stats file". Renewed invocation and identity binding therefore cannot show host judgment. Only the effort word does.
   - No Fix was executed. The identity is a hard-coded placeholder (`prepare.py:98`, a repeated 16-character string), shared by e1 and e2.
   - e2's "before: high" comes from a separate case with different stats (2 files versus 3) and no identity link.

3. **The dispatched positive control hit what looks like a harness defect, which the README does not disclose.**
   - All 12 tracker calls in `ae649217` were refused with `No such file or directory: …/k0/packet`. `observe.py:189-194` requires packet files that dispatch cases don't have.
   - So the control could never verify an owner or reach CI. This leaves k1's zero-CI result without a positive control.
   - The README says only "refused tracker reads". `verify.py` was relaxed after the results were seen (`e058e166`), which the README does disclose. The relaxed rule still requires zero forwarded calls.

4. **Verifier and isolation weaknesses.**
   - `inspect_dispatch` counts only *served* consolidators, and excludes consolidator entries from the angle-entry count. A refused or failed consolidator invocation in the bad case would pass. The raw k1 trace has none, so this result stands.
   - The k0 host read harness source (`libexec/dispatch.py`) outside the recorder. Case isolation relies on host behavior. I saw no k0 command matching a stub-file read.
   - Large-diff and mandatory dispositions report fresh review and QA as pending. An `observed` status does not cover those gates.

5. **Boundary test logs cannot support their descriptions.** `runner-boundary-tests.log` in r24 shows only dots, "Ran 27 tests" and OK. It names no tests, so the claimed assertions (effort on seven calls, CLI traces) can't be confirmed from it.

## Not verified

- I did not rerun `verify.py`, the fixture suite or the Go quality gate.
- I checked rows 7, 10, 11, 12, 18, 19, 22 and 24 only. The other 24 rows are unassessed.
- I did not examine the raw archives from r20–r23, `r18`, `approved-r14` or `host-r14-final`. A missing-budget or manual-diagnostic case there would be outside this check.
- I did not independently confirm credential deletion, closed stdin, or that nothing wrote to the parent project or database.
- The pinned `finish-feature` skill comes from outside the repository. I saw that `prepare.py` enforces a digest, but did not compare it with an installed source.
- I did not recompute the store snapshots or open the SQLite stores. "Zero writes" rests on recorder forward and argument records.
- No fresh candidate-bound review or QA exists for any Fix-resulting identity.
