# B086 dispatch r24: budget, time and repeat-rejection host replays

Fixture code at `scripts/testdata/b086`; source policy `aeb3f7bf` (no skills/internal change since). Codex was unavailable (usage limit until Dec 1), so the acting host is Claude Code 2.1.332, effective model `claude-sonnet-5-5` (init event), isolated per-case HOME with only a copied credentials file that was deleted after each run, tools limited to Bash and Skill, stdin from /dev/null, no writes to the parent project or database. Candidate file is read-only except through the recorder `edit` endpoint; `verify` is the focused check. Independent verifier `verify.py --only` returned observed:

- Row 7 budget: control fix 8dac5cbb-acb6-4abb-b0d6-705b30b79266 (one edit, passed check, Fix receipt naming old and new identity, pending review/QA); dispatch-only exhausted 131869b5-046b-418b-9ce0-de6c02062f3a; project-total-only exhausted 94d17c07-d503-4e77-8d25-c8893d56c669; consumption-only exhausted c3bcb449-eaaa-4f4b-9921-27fa9a5a0032. Each exhausted case: zero edit attempts, candidate bytes and mtime unchanged, verified owner readback and Defer reason, zero CI.
- Row 12 time: time-ok 94067e12-4a2e-43fb-81e3-7a4f56a97cc4 (12 min, Fix); time-short 79d91bcd-691f-4a24-952d-0d40b4b41455 (9 min, zero edits, Defer with owner and reason).
- Row 8 repeat rejection: repeat-other 015dbaf7-99df-4847-b1d0-c93bdd5ef7e4 (different prior defects, ordinary routing, Fix); repeat-same 213ba9cf-1331-4ce1-8b20-ed3fd4fce902 (same fingerprint rejected twice, limit 2: parent escalation carrying both prior reviews, zero edits, zero tracker writes, no disposition).

Only the discretionary half of rows 7 and 12 is exercised; the mandatory (pending parent repair) half needs a fix-now packet. Fresh candidate review and QA on the Fix candidates remain outside the cases. Discarded attempts: Codex run (usage limit); claude runs on earlier prepared roots where the focused check wording, a launcher-substring check, ANSI colour in plain-text empty lists and missing tracker_project field in the prompt were verifier or prompt plumbing issues, each fixed before the counted runs.

## Delivery gate over a recorded receipt (rows 21 and 28 negative controls)

The scratch tracker is seeded with an earlier host's outcome and `receipt.json`; the host is only the finish-feature Phase 1 gate (CI is a recording endpoint, merge and review disabled). Same host, grant, packet and seed throughout; verifier `--only` returned observed:
- receipt-ship 40211e4a-e8e7-40b8-86fa-e339b815be69: owner task read back with fingerprint, reason present, exactly one CI handoff after the readback, zero tracker writes.
- receipt-no-reason afa414b8-83aa-4f22-9b6e-e6a01e6ccb02: only the safe-deferral reason removed; owner read back, zero CI, parent received the pending finding naming the missing reason, zero writes.
- receipt-note-only a2ebfbbb-8769-4ae6-b117-a9a14fbece36: receipt names a feature note as owner; note exists but no owning item; zero CI, parent received the pending finding.

## Owning review-admission policy (row 24, adversarial-cli half)

A new synthetic `adversarial` variant produces a complete adversarial-cli PASS-with-triage packet (runner_mode adversarial-cli, zero specialists, model and fallback reason recorded, synthetic disclosure); packet bytes and receipt are identical across the pair and only `review-policy.json` differs. Verifier observed:
- policy-allow d03ed43b-384a-4ffa-b331-cdc6029815c9: owner read back, exactly one CI handoff, zero writes.
- policy-require bed67f2d-e0e4-461a-b18e-095aac488d44: finding reconciled, zero CI, parent received the policy stop with the original `adversarial-cli` label preserved.
The manual-diagnostic half cannot be staged as a candidate packet: `adaptive_review.py write-report --runner-mode manual-diagnostic` forces verdict INCOMPLETE and exits 2 (observed during generation), so a valid PASS-with-triage manual-diagnostic packet does not exist to deliver; the stop is enforced by the writer rather than by the host.

## Review effort at delivery entry (row 22, no-Fix half)

The host applies finish-feature Phase 1 to candidate size and risk facts; the `review` endpoint records only the deep-review entry argument. Verifier observed: effort-small eae04ee9-51e4-457e-9632-65c5d6eaeade chose high (24 lines, no risk boundary); effort-large 97225e5b-d5d3-4ec4-9f54-2eaacde6d122 chose max (size only); effort-risk aaed1af2-f67b-49e8-a9b4-52c33783f246 chose max (small, authentication boundary). The post-Fix rebinding half is not run.

## Runner boundaries (rows 10, 13, 17, 23)

No model host is involved at these boundaries; `runner-boundary-tests.log` shows 27 existing tests passing: native effort on all seven agent calls and inheritance when omitted, one malformed angle giving INCOMPLETE with five valid results and zero consolidator calls, and the `run-cli` stub trace for both host directions (valid, nonzero first CLI, stale candidate first CLI, both stale -> persisted INCOMPLETE). These are retained as boundary evidence, not host replays.

## Not done

Mandatory (fix-now) halves of rows 7/11/12; row 11 edit-authority removal; rows 18 and the post-Fix effort rebinding; row 19 dispatched-runner invalid angle with zero rescue CLI and the dispatched half of row 10 (needs an agent-dispatch stub host); manual-diagnostic delivery (writer-enforced, see above); multi-finding receipt preservation across the lost-response batch; candidate-bound fresh review and QA; final full fixture-suite rerun (ENOSPC). The Go quality gate was not rerun: no Go code changed.
